package main

import (
	"log"

	"github.com/morfien101/go-metrics-reciever/config"
	"github.com/morfien101/go-metrics-reciever/influxpump"
	"github.com/morfien101/go-metrics-reciever/redisengine"
	"github.com/morfien101/go-metrics-reciever/webengine"
)

func main() {
	config, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	redis := redisengine.New(config.Redis)
	err = redis.Start()
	if err != nil {
		log.Fatal(err)
	}

	pumpingChan := make(chan []byte, 1000)

	influxPump := influxpump.NewPump(config.Influx, pumpingChan)
	influxPump.Start()

	webserver := webengine.New(config.WebServer, redis, pumpingChan)
	<-webserver.Start()
}
