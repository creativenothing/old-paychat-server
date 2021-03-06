// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
    "fmt"
    "time"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan *ClientMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan *ClientMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
    timeTicker := time.NewTicker(time.Second)

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
            client.timeEnd = time.Now().Add(time.Second * 30)

        case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case cmessage := <-h.broadcast:
            h.handleBroadcast(*cmessage)

        case <-timeTicker.C:
            h.handleTime()

        }
	}
}

func (h *Hub) handleTime(){
    for client := range h.clients {
        timeLeftMillis := time.Until(client.timeEnd).Milliseconds()
        if timeLeftMillis < 0 {
            timeLeftMillis = 0
        }

        client.timeLeft = timeLeftMillis

        client.sendJSON(
            map[string]interface{}{
                "time": timeLeftMillis,
                "type": "time",
            } )

    }
}

func (h *Hub) handleBroadcast(cmessage ClientMessage){
        message:= cmessage.message
        name:= cmessage.c.name
        id:= cmessage.c.id

        fmt.Printf("%s: %s\n", name,string(message))

        msgJSON := map[string]interface{}{
                "text": string(message),
                "username": name,
                "userid": id,
                "type": "chat",
            }

        for client := range h.clients {
            client.sendJSON(msgJSON)
        }
}
