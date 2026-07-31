package broker

import (
	"fmt"
	"os"
)

func getAmqpUrl() (string, error) {
	brokerHost := os.Getenv("RABBIT_HOST")
	brokerPwd := os.Getenv("RABBIT_PWD")
	brokerUser := os.Getenv("RABBIT_USER")
	brokerPort := os.Getenv("RABBIT_PORT")

	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%s/", brokerUser, brokerPwd, brokerHost, brokerPort)
	return amqpURL, nil
}

func (b *Broker) makeQueue(queueName string) error {
	_, err := b.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (b *Broker) Close() {
	if b.channel != nil {
		b.channel.Close()
	}
	if b.conn != nil {
		b.conn.Close()
	}
}
