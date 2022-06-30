// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"log"
    "fmt"
	"net/http"
	"time"
    "encoding/json"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}

    clientNo = 0
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

    // Client name
    name string

    // Client id
    id int

    // Client termination time
    timeEnd time.Time

    // Client time remaining
    timeLeft int64
}

// Associte message with client for sending the name
type ClientMessage struct{
    c *Client

    message []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
        if(c.hub != nil){
		    c.hub.unregister <- c
        }
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
     for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

        c.handleReadMessage(message)
	}
}

func (c *Client) handleReadMessage(message []byte){
    fmt.Printf("%s\n",string(message))

    msgJSON := map[string]interface{}{}
    err := json.Unmarshal(message,&msgJSON)

    if(err != nil){
        return
    }

    switch(msgJSON["type"].(string)){
        case "chat":
            if(c.timeLeft <= 0 || c.hub == nil){
                break
            }

            message := []byte(msgJSON["text"].(string))
            // Package websocket message and client message
            cmessage := ClientMessage{
                c: c,
                message: bytes.TrimSpace(bytes.Replace(message, newline, space, -1)),
            }

            c.hub.broadcast <- &cmessage
        break
        case "join":

            fmt.Println("a")
            hubName, valid := msgJSON["room"]
            if(!valid){
                c.sendJSON(map[string]interface{}{
                    "type": "join",
                    "status": false,
                })

                break
            }

            fmt.Println("b")
            hub, validHub := hubs[hubName.(string)]
            if (!validHub){
                c.sendJSON(map[string]interface{}{
                    "type": "join",
                    "status": false,
                })

                break
            }
            fmt.Println("c")
            c.sendJSON(map[string]interface{}{
                "type": "join",
                "status": true,
            })

            fmt.Println("d")

            c.hub = hub
	        c.hub.register <- c
        break
        case "connect":
            clientNo++
            c.id = clientNo
            c.name = fmt.Sprintf("User %d",clientNo)

            c.sendJSON(map[string]interface{}{
                "name": c.name,
                "id": c.id,
                "type": "connect",
            })


        default:
    }
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: nil, conn: conn, send: make(chan []byte, 256)}

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}

func (c *Client) sendJSON(jsonObj map[string]interface{}){
    msgJSON, _ := json.Marshal(jsonObj)

    select {
    case c.send <- msgJSON:
    default:
        close(c.send)
        if(c.hub != nil){
            delete(c.hub.clients, c)
        }
    }
}
