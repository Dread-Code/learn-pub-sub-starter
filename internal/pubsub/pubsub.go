package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	ctx := context.Background()
	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	}
	err = ch.PublishWithContext(ctx, exchange, key, false, false, msg)
	if err != nil {
		return err
	}
	return nil
}

type AckType string

var (
	Ack         AckType = "ack"
	NackRequeue AckType = "nack_requeue"
	NackDiscard AckType = "nack_discard"
)

type SimpleQueueType string

const (
	DURABLE   SimpleQueueType = "durable"
	TRANSIENT SimpleQueueType = "transient"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	var queue amqp.Queue
	if queueType == DURABLE {
		queue, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	}
	if queueType == TRANSIENT {
		queue, err = ch.QueueDeclare(queueName, false, true, true, false, nil)
	}
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	deliveryChan, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for message := range deliveryChan {
			var msg T
			err := json.Unmarshal(message.Body, &msg)
			if err != nil {
				log.Printf("subscribe: unmarshal failed for queue %s: %v", queue.Name, err)
				message.Nack(false, false)
				continue
			}
			ackType := handler(msg)

			switch ackType {
			case Ack:
				message.Ack(false)
				fmt.Println("Ack")
			case NackRequeue:
				message.Nack(false, true)
				fmt.Println("Nack reueue")
			default:
				message.Nack(false, false)
				fmt.Println("Nack discard")
			}
		}
	}()

	return nil
}
