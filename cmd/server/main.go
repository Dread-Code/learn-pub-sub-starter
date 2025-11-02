package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("Connection was successful")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
