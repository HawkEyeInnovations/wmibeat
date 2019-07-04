package beater

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/eskibars/wmibeat/config"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

type Wmibeat struct {
	queries []*Query
	done    chan struct{}
}

type Query struct {
	query  string
	config config.QueryConfig
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.Config{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}
	bt := &Wmibeat{
		done: make(chan struct{}),
	}

	for _, queryConfig := range config.Queries {
		query, err := NewQuery(queryConfig)
		if err == nil {
			bt.queries = append(bt.queries, query)
		} else {
			logp.Warn(err.Error())
		}
	}

	return bt, nil
}

func NewQuery(config config.QueryConfig) (*Query, error) {
	if len(config.Fields) == 0 {
		return nil, fmt.Errorf("No fields defined for class %v. Skipping", config.Class)
	}

	var query bytes.Buffer
	query.WriteString("SELECT ")
	query.WriteString(strings.Join(config.Fields, ","))
	query.WriteString(" FROM ")
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

// Run starts wmibeat.
func (bt *Wmibeat) Run(b *beat.Beat) error {
	var wg sync.WaitGroup
	ole.CoInitializeEx(0, 0)
	defer ole.CoUninitialize()

	for _, q := range bt.queries {
		client, err := b.Publisher.Connect()
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(query *Query) {
			defer wg.Done()
			query.Run(bt.done, client)
		}(q)
	}

	logp.Info("wmibeat is running! Hit CTRL-C to stop it.")

	wg.Wait()
	return nil
}

func (query *Query) Run(done <-chan struct{}, client beat.Client) error {
	var err error

	ticker := time.NewTicker(query.config.Period)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return nil
		case <-ticker.C:
		}

		err := query.RunOnce(client)
		if err != nil {
			logp.Err("Unable to run WMI queries: %v", err)
			break
		}
	}

	return err
}

func (query *Query) RunOnce(client beat.Client) error {
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
				logp.Err("Unable to get propery by name: %v", err)
				return err
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

func (bt *Wmibeat) Stop() {
	close(bt.done)
}
