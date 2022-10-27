package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn      *amqp.Connection
	queueName string
}

const LogTopicName = "logs_topic"

func NewConsumer(conn *amqp.Connection) (Consumer, error) {

	consumer := Consumer{
		conn: conn,
	}

	err := consumer.setup()
	if err != nil {
		return Consumer{}, err
	}

	return consumer, nil
}

func (consumer *Consumer) setup() error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		return err
	}

	return declareExchange(channel, LogTopicName)
}

// Declare multiple exchanges
func (consumer *Consumer) setUpMultipleQueues(queues []string) []error {
	errors := [...]error{}
	count := 0

	for _, queue := range queues {
		fmt.Println(queue)
		channel, err := consumer.conn.Channel()
		if err != nil {
			errors[count] = err
			continue
		}
		count++
		declareQueue(channel, queue)
	}

	if len(errors) > 0 {
		return errors[:]
	}

	return nil

}

type Payload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (consumer *Consumer) Listen(topics []string) error {
	ch, err := consumer.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := declareRandomQueue(ch)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		ch.QueueBind(
			q.Name,
			topic,
			LogTopicName,
			false,
			nil,
		)

		if err != nil {
			return err
		}
	}

	messages, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	// https://go.dev/tour/concurrency/2
	forever := make(chan bool)
	go func() {
		for message := range messages {
			var payload Payload
			_ = json.Unmarshal(message.Body, &payload)

			go handlePayload(payload)
		}
	}()

	fmt.Printf("Waiting for message on exchange [Exchange, Queue] [%s, %s]", LogTopicName, q.Name)
	<-forever

	return nil
}

func handlePayload(payload Payload) {
	switch payload.Name {
	case "log", "event":
		// log
		err := logEvent(payload)
		if err != nil {
			log.Println(err)
		}

	case "auth":
	default:
		err := logEvent(payload)
		if err != nil {
			log.Println(err)
		}

	}
}

func logEvent(entry Payload) error {
	// create json send to auth microservice
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	client := http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		return err
	}

	return nil
}
