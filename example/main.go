// Package example Create at 2021-01-19 18:20
package main

import (
	"fmt"
	"log"
	"net/http"

	"zephyr"
)

func TestHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello"))
}

func main() {
	z := zephyr.NewZephyr(nil)
	z.AddHandler(zephyr.Action{
		URL:             "/test",
		Method:          []string{zephyr.GET},
		Handler:         TestHandler,
		PreHandler:      nil,
		AfterCompletion: nil,
		Wrapper:         nil,
	})

	http.HandleFunc("/", z.ServeHTTP)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 2000), nil))
}
