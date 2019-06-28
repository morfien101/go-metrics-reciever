package webengine

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/morfien101/go-metrics-reciever/config"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  10240,
	WriteBufferSize: 10240,
}

type WebEngine struct {
	config        config.WebServerConfig
	router        *mux.Router
	server        *http.Server
	authClient    *http.Client
	authEndpoint  string
	influxChannel chan []byte
}

func New(config config.WebServerConfig, authEndpoint string, influxChannel chan []byte) *WebEngine {
	we := &WebEngine{
		config:        config,
		authEndpoint:  authEndpoint,
		influxChannel: influxChannel,
	}
	we.loadRoutes()
	we.loadHTTPClient()
	listenerAddress := we.config.ListenAddress + ":" + we.config.Port
	we.server = &http.Server{Addr: listenerAddress, Handler: we.router}
	return we
}

func (we *WebEngine) loadHTTPClient() {
	tr := &http.Transport{
		MaxIdleConns:    20,
		IdleConnTimeout: 30 * time.Second,
	}
	client := &http.Client{
		Timeout:   time.Second * 3,
		Transport: tr,
	}
	we.authClient = client
}

func (we *WebEngine) loadRoutes() {
	we.router = mux.NewRouter()
	we.router.HandleFunc("/_status", we.getStatus).Methods("GET")
	we.router.Handle("/metrics", we.authRequired(we.metrics)).Methods("GET")
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
		errStatement := struct {
			ErrorText string `json:"Error"`
		}{
			ErrorText: "",
		}
		user, pass, ok := r.BasicAuth()
		if !ok {
			setContentJSON(w)
			errStatement.ErrorText = "Credentials not supplied"
			b, _ := jsonMarshal(errStatement)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(b)
			return
		}

		ok, err := we.validateAuth(user, pass)
		if err != nil {
			errStatement.ErrorText = "Internal server error"
			b, _ := jsonMarshal(errStatement)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(b)
			log.Printf("Error validating credentials. Error: %s", err)
			return
		}

		if !ok {
			setContentJSON(w)
			errStatement.ErrorText = "Credentials invalid"
			b, _ := jsonMarshal(errStatement)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(b)
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

func (we *WebEngine) validateAuth(username, password string) (bool, error) {
	req, err := http.NewRequest("POST", we.authURL(), nil)
	if err != nil {
		return false, err
	}
	q := req.URL.Query()
	q.Set("username", username)
	q.Set("password", password)
	req.URL.RawQuery = q.Encode()

	resp, err := we.authClient.Do(req)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 200 {
		return true, nil
	}
	return false, nil

}

func (we *WebEngine) authURL() string {
	return fmt.Sprintf("%s/auth", we.authEndpoint)
}
