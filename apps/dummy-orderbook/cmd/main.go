package main

import (
	"log"
	"net/http"

	"github.com/tradebench/dummy-orderbook/api"
)

func main() {
	mux := api.SetupRouter()

	log.Println("dummy-orderbook listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
