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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {

	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}
func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {

	return func(arm gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		moveOutcome := gs.HandleMove(arm)

		switch moveOutcome {
		case gamelogic.MoveOutComeSafe, gamelogic.MoveOutcomeMakeWar:
			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
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

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal(err)
	}

	pauseKey := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, pauseKey, routing.PauseKey, pubsub.TRANSIENT)

	gameState := gamelogic.NewGameState(username)

	handler := handlerPause(gameState)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, pauseKey, routing.PauseKey, pubsub.TRANSIENT, handler)

	armyMovesKey := fmt.Sprintf("%s.%s", routing.ArmyMovesKey, username)
	armyMovesRoutingKey := fmt.Sprintf("%s.*", routing.ArmyMovesKey)
	hm := handlerMove(gameState)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, armyMovesKey, armyMovesRoutingKey, pubsub.TRANSIENT, hm)

repl:
	for {
		command := gamelogic.GetInput()
		if command == nil {
			continue
		}
		switch command[0] {
		case "spawn":
			gameState.CommandSpawn(command)
		case "move":
			arm, err := gameState.CommandMove(command)
			if err != err {
				fmt.Println(err)
				continue
			}
			pubsub.PublishJSON(ch, routing.ExchangePerilTopic, armyMovesKey, arm)
			fmt.Println("Movement pusblished succesfully")
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			break repl
		default:
			fmt.Println("Command not found read the fucking manuel dumb...")
		}
	}

	if err != nil {
		log.Fatal(err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Clientt gracefully stopped")
}
