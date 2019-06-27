package webengine

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/morfien101/go-metrics-reciever/config"
	"github.com/morfien101/go-metrics-reciever/redisengine"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  10240,
	WriteBufferSize: 10240,
}

type WebEngine struct {
	config        config.WebServerConfig
	router        *mux.Router
	server        *http.Server
	redisEngine   *redisengine.RedisEngine
	influxChannel chan []byte
}

func New(config config.WebServerConfig, redis *redisengine.RedisEngine, influxChannel chan []byte) *WebEngine {
	we := &WebEngine{
		config:        config,
		router:        mux.NewRouter(),
		redisEngine:   redis,
		influxChannel: influxChannel,
	}

	we.router.HandleFunc("/_status", we.getStatus).Methods("GET")
	we.router.Handle("/metrics", we.authRequired(we.metrics)).Methods("GET")

	listenerAddress := we.config.ListenAddress + ":" + we.config.Port
	we.server = &http.Server{Addr: listenerAddress, Handler: we.router}
	return we
}

// ServeHTTP is used to allow the router to start accepting requests before the start is started up. This will help with testing.
func (we *WebEngine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	we.router.ServeHTTP(w, r)
}

func setContentJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
}

func jsonMarshal(x interface{}) ([]byte, error) {
	return json.MarshalIndent(x, "", "  ")
}

func printJSON(w http.ResponseWriter, jsonbytes []byte) (int, error) {
	return fmt.Fprint(w, string(jsonbytes), "\n")
}

// Start will start the web server using the configuration provided.
// It returns a channel that will give the error if there is one
func (we *WebEngine) Start() <-chan error {
	c := make(chan error, 1)
	startfunc := we.startClear
	if we.config.UseTLS {
		startfunc = we.startTLS
	}
	go func() {
		c <- startfunc()
	}()

	return c
}

func (we *WebEngine) startTLS() error {
	return we.server.ListenAndServeTLS(we.config.CertPath, we.config.KeyPath)
}

func (we *WebEngine) startClear() error {
	return we.server.ListenAndServe()
}

func (we *WebEngine) getStatus(w http.ResponseWriter, r *http.Request) {
	// Make this better.
	w.Write([]byte("OK"))
}

func (we *WebEngine) authRequired(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			setContentJSON(w)
			w.Write([]byte("{\"error\":\"Credentials not supplied\"}"))
			return
		}
		ok, err := we.redisEngine.ValidateAuth(user, pass)
		if err != nil {
			log.Printf("Error validating creds, Error: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !ok {
			setContentJSON(w)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{\"error\":\"Credentials invalid\"}"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (we *WebEngine) metrics(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	ws := &webSocketHandler{
		conn:    conn,
		influx:  we.influxChannel,
		maxLife: we.config.SocketLifetime,
	}

	go ws.readPump()
	go ws.writePump()
}
