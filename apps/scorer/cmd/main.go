package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("scorer service started")
	for {
		time.Sleep(time.Hour)
	}
}
