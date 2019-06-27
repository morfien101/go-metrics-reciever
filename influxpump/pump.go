package influxpump

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/morfien101/go-metrics-reciever/config"
)

const (
	maxIdleConnections int = 20
	requestTimeout     int = 5
)

type Pump struct {
	input          chan []byte
	writer         chan []byte
	address        string
	wg             *sync.WaitGroup
	httpclient     *http.Client
	stop           chan bool
	config         config.InfluxConfig
	payloadManager *payloadManager
}

func NewPump(config config.InfluxConfig, input chan []byte) *Pump {
	return &Pump{
		config:         config,
		input:          input,
		writer:         make(chan []byte, 10000),
		address:        fmt.Sprintf("%s:%s", config.InfluxHost, config.InfluxPort),
		payloadManager: newPayloadManager(config),
		wg:             &sync.WaitGroup{},
	}
}

func (p *Pump) Start() error {
	p.wg.Add(1)
	if err := p.connect(); err != nil {
		return err
	}
	log.Print("Connected to Influx")
	go p.readOnChan()
	go p.shipper()
	return nil
}

func (p *Pump) Stop() chan error {
	p.stop <- true
	c := make(chan error, 1)
	go func() {
		p.wg.Wait()
		c <- nil
	}()
	return c
}

func (p *Pump) readOnChan() {
	for {
		select {
		case <-p.stop:
			fmt.Println("Stopping influx pump.")
			close(p.writer)
			return
		case msg := <-p.input:
			metric, err := convert(msg)
			if err != nil {
				// Print it and toss it out for now
				fmt.Println(err)
				continue
			}
			p.writer <- []byte(metric.OutputWithTimestamp())
		}
	}
}

func (p *Pump) shipper() {
	shipTicker := time.NewTicker(time.Second * time.Duration(p.config.SendIntervalSeconds))
	for {
		select {
		case msg, ok := <-p.writer:
			if !ok {
				// Shutdown
				p.writeout()
				p.wg.Done()
				log.Println("Shutdown Influx Writer")
				return
			}
			if err := p.payloadManager.add(msg); err != nil {
				p.writeout()
				p.payloadManager.add(msg)
			}
		case t := <-shipTicker.C:
			if t.UnixNano()-int64((time.Second*time.Duration(p.config.SendIntervalSeconds))) > p.payloadManager.getLastFlushtime() {
				p.writeout()
			}
		}
	}
}

func (p *Pump) writeout() {
	if p.payloadManager.currentSize() < 1 {
		return
	}
	out := p.payloadManager.read()
	p.flush(out)
	p.payloadManager.reset()
}

func (p *Pump) pingURL() string {
	return fmt.Sprintf("%s/ping", p.address)
}

func (p *Pump) writeURL() string {
	return fmt.Sprintf("%s/write?db=%s", p.address, p.config.InfluxDB)
}

func urlencodedContentType(req *http.Request) {
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
}

func (p *Pump) ping() error {
	req, err := http.NewRequest("GET", p.pingURL(), nil)
	if err != nil {
		return err
	}
	urlencodedContentType(req)

	resp, err := p.httpclient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to ping Influx. Error: %s", err)
	}
	resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("Got a bad status code from influx when pinging. Code: %d", resp.StatusCode)
	}
	return nil
}

func (p *Pump) connect() error {

	p.httpclient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: maxIdleConnections,
		},
		Timeout: time.Duration(requestTimeout) * time.Second,
	}

	return p.ping()
}

func (p *Pump) flush(payload []byte) {
	writeOk := false
	wait := func(sleepTime int) {
		time.Sleep(time.Duration(sleepTime*200) * time.Millisecond)
	}

	for try := 1; try <= 3; try++ {
		if try != 1 {
			wait(try)
		}

		req, err := http.NewRequest("POST", p.writeURL(), bytes.NewReader(payload))
		if err != nil {
			log.Println("Failed to create request. Error:", err)
			continue
		}
		req.Header.Add("Content-Type", "application/octet-stream")

		resp, err := p.httpclient.Do(req)
		if err != nil {
			log.Printf("Failed to write to influx. Error: %s", err)
			continue
		}

		if resp.StatusCode > 300 {
			log.Printf("Got a bad status code while writing to influx. Status Code: %d", resp.StatusCode)
			continue
		}

		// If we get to here it means that the write was successful and we can move on from the loop
		writeOk = true
		break
	}

	if !writeOk {
		log.Println("Failed to write to influx. Shrug!")
		return
	}

	log.Printf("Flushed %d bytes successfully", len(payload))
}
