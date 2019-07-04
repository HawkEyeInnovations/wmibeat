package beater

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/eskibars/wmibeat/config"

	"github.com/go-ole/go-ole"
)

// Wmibeat implements the Beater interface for wmibeat
type Wmibeat struct {
	queries []*Query
	done    chan struct{}
}

// New creates the beater
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

// Run starts wmibeat.
func (bt *Wmibeat) Run(b *beat.Beat) error {
	var wg sync.WaitGroup

	// Initialise this once for all threads
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

// Stop signals to the done channel so all queries stop
func (bt *Wmibeat) Stop() {
	close(bt.done)
}
