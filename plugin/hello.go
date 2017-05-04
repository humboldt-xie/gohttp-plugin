package main

import (
	"io"
	"log"
	"net/http"
)

// hello world, the web server
func HelloServer(w http.ResponseWriter, req *http.Request) {
	log.Printf("hello")
	io.WriteString(w, "hello, world v4!\n")
}

func GetRouter() map[string]http.Handler {
	return map[string]http.Handler{
		"/hello": http.HandlerFunc(HelloServer),
	}
}
