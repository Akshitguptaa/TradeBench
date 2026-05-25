package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("ingester service started")
	for {
		time.Sleep(time.Hour)
	}
}
