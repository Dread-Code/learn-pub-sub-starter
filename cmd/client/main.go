package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(arm gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		moveOutcome := gs.HandleMove(arm)
		fmt.Println("handlermove return", moveOutcome)
		switch moveOutcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			username := gs.GetUsername()
			units := gs.Player.Units
			row := gamelogic.RecognitionOfWar{Attacker: arm.Player, Defender: gamelogic.Player{Username: username, Units: units}}
			warKey := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, username)
			err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, warKey, row)
			if err != nil {
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
	}
}

func PublishGameLog(ch *amqp.Channel, username string, msg string) pubsub.AckType {
	gameLogKey := fmt.Sprintf("%s.%s", routing.GameLogSlug, username)
	gl := routing.GameLog{
		Message:     msg,
		Username:    username,
		CurrentTime: time.Now(),
	}

	err := pubsub.PublishGob(ch, routing.ExchangePerilTopic, gameLogKey, gl)
	if err != nil {
		return pubsub.NackRequeue
	}
	fmt.Println("Publishin gamelog...")
	return pubsub.Ack
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(row gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, winner, losser := gs.HandleWar(row)
		fmt.Println("handlerwar return", outcome, winner, losser)
		username := gs.GetUsername()
		var log string
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			log = fmt.Sprintf("{%s} won a war against {%s}", winner, losser)
			return PublishGameLog(ch, username, log)
		case gamelogic.WarOutcomeYouWon:
			log = fmt.Sprintf("{%s} won a war against {%s}", winner, losser)
			return PublishGameLog(ch, username, log)
		case gamelogic.WarOutcomeDraw:
			log = fmt.Sprintf("A war between {%s} and {%s} resulted in a draw", winner, losser)
			return PublishGameLog(ch, username, log)
		default:
			fmt.Println("An error occur")
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
	hm := handlerMove(gameState, ch)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, armyMovesKey, armyMovesRoutingKey, pubsub.TRANSIENT, hm)

	hw := handlerWar(gameState, ch)
	warKey := "war"
	warRoutingKey := fmt.Sprintf("%s.*", warKey)
	pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, warKey, warRoutingKey, pubsub.DURABLE, hw)
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
