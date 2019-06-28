package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/morfien101/go-metrics-reciever/config"
	"github.com/morfien101/go-metrics-reciever/influxpump"
	"github.com/morfien101/go-metrics-reciever/webengine"
)

var (
	// VERSION stores the version of the application
	VERSION = "0.0.2"

	flagVersion = flag.Bool("v", false, "Shows the version")
	flagHelp    = flag.Bool("h", false, "Shows the help menu")
)

func main() {
	flag.Parse()
	if *flagHelp {
		flag.PrintDefaults()
		return
	}
	if *flagVersion {
		fmt.Println(VERSION)
		return
	}

	config, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	pumpingChan := make(chan []byte, 1000)

	influxPump := influxpump.NewPump(config.Influx, pumpingChan)
	influxPump.Start()

	webserver := webengine.New(config.WebServer, config.Auth.AuthHost, pumpingChan)
	<-webserver.Start()
}
