package webengine

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 3 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 5120
)

// Client reads from the websocket.
type webSocketHandler struct {
	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Buffered channel for writing to metrics pump
	outbound chan []byte

	// InfluxPump send the received data out to influx pumping package
	influx chan []byte

	// How long can the socket stay alive in seconds
	maxLife int
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (wsh *webSocketHandler) readPump() {
	stop := make(chan struct{}, 1)
	triggeredStop := false
	stopTimer := time.AfterFunc(time.Second*time.Duration(wsh.maxLife), func() {
		triggeredStop = true
		stop <- struct{}{}
	})
	defer func() {
		if !triggeredStop {
			stopTimer.Stop()
		}
		wsh.conn.Close()
	}()
	wsh.conn.SetReadLimit(maxMessageSize)
	wsh.conn.SetReadDeadline(time.Now().Add(pongWait))
	wsh.conn.SetPongHandler(func(string) error {
		wsh.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-stop:
			// return will call the deferred function to close the socket
			return
		default:
			_, message, err := wsh.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("error: %v", err)
				}
				break
			}
			select {
			case wsh.influx <- message:
			default:
				log.Println("Influx queue is full, dropping message.")
			}
		}
	}
}

// writePump is used to send pings to the client to keep connections alive.
func (wsh *webSocketHandler) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		wsh.conn.Close()
	}()
	for {
		select {
		case <-ticker.C:
			wsh.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := wsh.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
