package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("bots service started")
	for {
		time.Sleep(time.Hour)
	}
}
