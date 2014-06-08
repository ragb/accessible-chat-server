// This module implements an HTTP chat server using server sent events.
package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func main() {
	channel := NewChannelBroker("main")
	go channel.Start()
	router := mux.NewRouter()

	router.HandleFunc("/post/", channel.PostHTTPMessage).Methods("POST").Headers("Content-Type", "application/json")
	router.HandleFunc("/events/", channel.ServeHTTPEventStream)
	log.Print("Starting http server")
	http.Handle("/", router)
	http.ListenAndServe(":8000", nil)
	log.Print("Closed http server")
}
