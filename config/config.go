package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	hjson "github.com/hjson/hjson-go"
)

//

const (
	// defaultAuthEnv is the ENV var that will be looked up if the configuration is
	// not in the default location.
	defaultAuthEnv = "METRIC_RECEIVER_CONFIG"
)

var (
	basePath = "/etc/metric-receiver/"
	// defaultConfigPath is the default location to look for the configuration file.
	defaultConfigPath = basePath + "metric-receiver.conf"

	defaultWebServer = WebServerConfig{
		ListenAddress:  "0.0.0.0",
		Port:           "80",
		UseTLS:         false,
		CertPath:       basePath + "cert.cert",
		KeyPath:        basePath + "cert.key",
		SocketLifetime: 900,
	}

	defaultInfluxServer = InfluxConfig{
		InfluxHost:          "http://172.17.0.1",
		InfluxPort:          "8086",
		InfluxDB:            "externalMetrics",
		WriteBuffer:         10000,
		BatchSize:           5000,
		SendIntervalSeconds: 2,
	}

	defaultAuthServer = AuthConfig{
		AuthHost: "http://172.17.0.1",
	}
)

// Config is a struct that holds all the configuration options for the application.
type Config struct {
	Auth      AuthConfig      `json:"auth_server"`
	WebServer WebServerConfig `json:"web_server"`
	Influx    InfluxConfig    `json:"influx_server"`
}

// WebServerConfig holds the configuration for the http web server
type WebServerConfig struct {
	ListenAddress  string `json:"listen_address"`
	Port           string `json:"listen_port"`
	UseTLS         bool   `json:"use_tls"`
	CertPath       string `json:"cert_path"`
	KeyPath        string `json:"key_path"`
	SocketLifetime int    `json:"max_sockets_lifetime_seconds"`
}

type InfluxConfig struct {
	InfluxHost          string `json:"host"`
	InfluxPort          string `json:"port"`
	InfluxDB            string `json:"database"`
	WriteBuffer         int    `json:"write_buffer"`
	BatchSize           int    `json:"batch_size"`
	SendIntervalSeconds int    `json:"send_interval"`
}

type AuthConfig struct {
	AuthHost string `json:"host"`
}

// New creates a new configuration struct and returns to to the caller.
// The configuration file location is either defaultConfigPath const or
// METRIC_AUTH_CONFIG environment variable.
func New() (*Config, error) {
	configBytes, err := readConfigFile(determinConfigPath())
	if err != nil {
		return nil, err
	}

	configBytes, err = digestHJSONConfig(configBytes)
	if err != nil {
		return nil, err
	}

	c, err := hydrateConfig(configBytes)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func digestHJSONConfig(rawConfig []byte) ([]byte, error) {
	// Change to proper JSON, then convert to Go Struct
	hjs := make(map[string]interface{})
	if err := hjson.Unmarshal(rawConfig, &hjs); err != nil {
		return nil, fmt.Errorf("Failed digest HJSON configuration. Error: %s", err)
	}

	return json.Marshal(hjs)
}

func determinConfigPath() string {
	path := defaultConfigPath
	if value, ok := os.LookupEnv(defaultAuthEnv); ok {
		path = value
	}

	return path
}

func readConfigFile(path string) ([]byte, error) {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return configBytes, nil
}

func hydrateConfig(cb []byte) (*Config, error) {
	// Make new config and set defaults
	config := new(Config)
	config.WebServer = defaultWebServer
	config.Influx = defaultInfluxServer
	config.Auth = defaultAuthServer

	err := json.Unmarshal(cb, config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
