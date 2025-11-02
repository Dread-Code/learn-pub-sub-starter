#!/bin/bash

start_or_run () {
    docker inspect peril_rabbitmq > /dev/null 2>&1

    if [ $? -eq 0 ]; then
        echo "Starting Peril RabbitMQ container..."
        docker start peril_rabbitmq
    else
        echo "Peril RabbitMQ container not found, creating a new one..."
        docker run -d --name peril_rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:3.13-management
    fi
}

stomp_or_run () {
    docker inspect rabbitmq_stomp > /dev/null 2>&1

    if [ $? -eq 0 ]; then
        echo "Starting Stomp RabbitMQ container..."
        docker start rabbitmq_stomp
    else
        echo "Stomp RabbitMQ container not found, creating a new one..."
        docker run -d --name rabbitmq_stomp -p 5672:5672 -p 15672:15672 -p 61613:61613 rabbitmq-stomp
    fi
}

case "$1" in
    start)
        start_or_run
        ;;
    stomp)
        stomp_or_run
        ;;
    stop)
        echo "Stopping Peril RabbitMQ container..."
        docker stop peril_rabbitmq
        ;;
    stompstop)
        echo "Stopping Stomp RabbitMQ container..."
        docker stop rabbitmq_stomp
        ;;
    logs)
        echo "Fetching logs for Peril RabbitMQ container..."
        docker logs -f peril_rabbitmq
        ;;
    *)
        echo "Usage: $0 {start|stop|logs}"
        exit 1
esac
