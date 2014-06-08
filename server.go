// This module implements an HTTP chat server using server sent events.
package main

import (
	"github.com/gorilla/mux"
	"net/http"
)

func main() {
	channel := NewChannelBroker("main")
	go channel.Start()
	router := mux.NewRouter()
	router.Headers("Content-Type", "application/json")
	router.Path("/post/").HandlerFunc(channel.PostHTTPMessage)
	router.Path("/events/").HandlerFunc(channel.ServeHTTPEventStream)
	http.Handle("/", router)
	http.ListenAndServe(":8000", nil)
}
