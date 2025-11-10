package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerGameLogs() func(routing.GameLog) pubsub.AckType {
	return func(gl routing.GameLog) pubsub.AckType {
		defer fmt.Print("> ")
		gamelogic.WriteLog(gl)
		return pubsub.Ack
	}
}

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("Connection was successful")

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}

	routingKey := fmt.Sprintf("%s.*", routing.GameLogSlug)
	hgl := handlerGameLogs()
	err = pubsub.SubscribeGob(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routingKey, pubsub.DURABLE, hgl)
	fmt.Println(err)
repl:
	for {
		command := gamelogic.GetInput()
		if command == nil {
			continue
		}
		switch command[0] {
		case "pause":
			fmt.Println("Sending pause message...")
			pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
		case "resume":
			fmt.Println("Sending resume message...")
			pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
		case "quit":
			fmt.Println("Closing the game...")
			break repl
		case "help":
			gamelogic.PrintServerHelp()
		default:
			fmt.Println("Command not found read the fucking manuel dumb...")
		}
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
