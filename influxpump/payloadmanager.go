package influxpump

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/morfien101/go-metrics-reciever/config"
)

type payloadManager struct {
	lasttick int64
	payloads [][]byte
	index    int
	maxSize  int
	sync.RWMutex
}

func newPayloadManager(config config.InfluxConfig) *payloadManager {
	return &payloadManager{
		maxSize:  config.BatchSize,
		payloads: make([][]byte, config.BatchSize),
		index:    0,
	}
}

func (pl *payloadManager) add(input []byte) error {
	if pl.index == pl.maxSize-1 {
		return fmt.Errorf("Full")
	}
	pl.Lock()
	pl.payloads[pl.index] = input
	pl.index++
	pl.Unlock()
	return nil
}

func (pl *payloadManager) reset() {
	pl.Lock()
	pl.index = 0
	pl.Unlock()
}

func (pl *payloadManager) read() []byte {
	pl.RLock()
	out := bytes.Join(pl.payloads[:pl.index], []byte("\n"))
	pl.RUnlock()
	return out
}

func (pl *payloadManager) currentSize() int {
	pl.RLock()
	s := pl.index - 1
	pl.RUnlock()
	return s
}

func (pl *payloadManager) setLastFlushtime(t int64) {
	pl.Lock()
	pl.lasttick = t
	pl.Unlock()
}

func (pl *payloadManager) getLastFlushtime() int64 {
	pl.RLock()
	defer pl.RUnlock()
	return pl.lasttick

}
