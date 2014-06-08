package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type ChannelMessage struct {
	content string
	author  string
	time    time.Time
}

type ChannelBroker struct {
	name          string
	clients       map[chan ChannelMessage]bool
	newClients    chan chan ChannelMessage
	closedClients chan chan ChannelMessage
	messages      chan ChannelMessage
}

func NewChannelBroker(name string) *ChannelBroker {
	return &ChannelBroker{
		name:          name,
		clients:       make(map[chan ChannelMessage]bool),
		newClients:    make(chan chan ChannelMessage),
		closedClients: make(chan chan ChannelMessage),
		messages:      make(chan ChannelMessage),
	}
}

func (b *ChannelBroker) Start() {
	log.Print("Start channel broker for channel ", b.name)
	for {
		select {
		case clientChannel := <-b.newClients:
			b.clients[clientChannel] = true
		case closedClient := <-b.closedClients:
			delete(b.clients, closedClient)
		case message := <-b.messages:
			for clientChannel, _ := range b.clients {
				clientChannel <- message
			}
		}
	}
}

func (b *ChannelBroker) AddClient(client chan ChannelMessage) {
	log.Print("Client added")
	b.newClients <- client
}

func (b *ChannelBroker) CloseClient(client chan ChannelMessage) {
	log.Print("Client removed")
	b.closedClients <- client
}

func (b *ChannelBroker) PushMessage(message ChannelMessage) {
	log.Printf("Pushing message %s to %d clients.", message, len(b.clients))
	b.messages <- message
}

// Closes this channel
func (b *ChannelBroker) CloseChannel() {
	for channel, _ := range b.clients {
		close(channel)
	}
	log.Printf("Channel %s closed.", b.name)
}

// Posts a new message over http
func (b *ChannelBroker) PostHTTPMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var message ChannelMessage
	error := decoder.Decode(&message)
	if error != nil {
		http.Error(w, error.Error(), http.StatusBadRequest)
		return
	}

	b.PushMessage(message)

	fmt.Fprint(w, "ok")
}

// Serves an event stream over http
func (b *ChannelBroker) ServeHTTPEventStream(w http.ResponseWriter, r *http.Request) {
	// See if this writer supports flishing
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streamming not supported", http.StatusInternalServerError)
		return
	}

	clientChannel := make(chan ChannelMessage)
	closedConnectionChannel := f.(http.CloseNotifier).CloseNotify()
	// Add client to the broker
	b.AddClient(clientChannel)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", " no-cache")
	w.Header().Set("Connection", "keep-alive")

	// create a json encoder so we can transform messages into JSON
	encoder := json.NewEncoder(w)

	// Loop until connection is closed and pool for messages
	for {
		select {
		case message, closed := <-clientChannel:
			if closed {
				f.Flush()
				return
			}
			encoder.Encode(message)
			fmt.Fprint(w, "\n\n")
			f.Flush()
		case <-closedConnectionChannel:
			b.CloseClient(clientChannel)
			return
		}
	}
}
