package beater

import (
	"bytes"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/eskibars/wmibeat/config"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type Query struct {
	query  string
	config config.QueryConfig
}

// NewQuery constructs a new query
func NewQuery(config config.QueryConfig) (*Query, error) {
	if len(config.Fields) == 0 {
		return nil, fmt.Errorf("No fields defined for class %v. Skipping", config.Class)
	}

	var query bytes.Buffer
	query.WriteString("SELECT * FROM ")
	query.WriteString(config.Class)
	if config.WhereClause != "" {
		query.WriteString(" WHERE ")
		query.WriteString(config.WhereClause)
	}

	q := &Query{
		query:  query.String(),
		config: config,
	}

	logp.Info("Created query %v", q.query)

	return q, nil
}

// Run loop for the query
func (query *Query) Run(done <-chan struct{}, client beat.Client) error {
	ticker := time.NewTicker(query.config.Period)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return nil
		case <-ticker.C:
		}

		if err := query.RunQuery(client); err != nil {
			logp.Err("Unable to run WMI query: %v", err)
		}
	}
}

// RunQuery runs one instance of the query
func (query *Query) RunQuery(client beat.Client) error {
	events := []common.MapStr{}

	wmiscriptObj, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		logp.Err("Unable to create object: %v", err)
		return err
	}
	defer wmiscriptObj.Release()

	wmiqi, err := wmiscriptObj.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		logp.Err("Unable to get locator query interface: %v", err)
		return err
	}
	defer wmiqi.Release()

	serviceObj, err := oleutil.CallMethod(wmiqi, "ConnectServer", ".", query.config.Namespace)
	if err != nil {
		logp.Err("Unable to connect to server: %v", err)
		return err
	}
	defer serviceObj.Clear()

	service := serviceObj.ToIDispatch()

	logp.Info("Query: " + query.query)

	resultObj, err := oleutil.CallMethod(service, "ExecQuery", query.query, "WQL")
	if err != nil {
		logp.Err("Unable to execute query: %v", err)
		return err
	}
	defer resultObj.Clear()

	result := resultObj.ToIDispatch()
	countObj, err := oleutil.GetProperty(result, "Count")
	if err != nil {
		logp.Err("Unable to get result count: %v", err)
		return err
	}
	defer countObj.Clear()

	count := int(countObj.Val)

	for i := 0; i < count; i++ {
		rowObj, err := oleutil.CallMethod(result, "ItemIndex", i)
		if err != nil {
			logp.Err("Unable to get result item by index: %v", err)
			return err
		}
		defer rowObj.Clear()

		row := rowObj.ToIDispatch()

		event := common.MapStr{
			"class": query.config.Class,
			"type":  "wmibeat",
		}

		for _, fieldName := range query.config.Fields {
			wmiObj, err := oleutil.GetProperty(row, fieldName)
			if err != nil {
				logp.Err("Unable to get property %v: %v", fieldName, err)
				continue
			}
			defer wmiObj.Clear()

			var objValue = wmiObj.Value()
			event[fieldName] = objValue
		}

		events = append(events, event)
	}

	for _, wmievent := range events {
		if wmievent != nil {
			event := beat.Event{
				Timestamp: time.Now(),
				Fields:    wmievent,
			}

			client.Publish(event)
		}
	}

	return err
}
