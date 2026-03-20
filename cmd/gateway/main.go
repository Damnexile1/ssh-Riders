package main

import (
	"log"

	"github.com/example/ssh-riders/internal/app/gateway"
)

func main() {
	if err := gateway.RunMain(); err != nil {
		log.Fatal(err)
	}
}
