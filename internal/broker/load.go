package broker

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func (b *Broker) Load(queueName, contentType string, data []byte) error {
	err := b.channel.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: contentType,
			Body:        data,
		},
	)
	if err != nil {
		return err
	}
	return nil
}
