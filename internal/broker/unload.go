package broker

import amqp "github.com/rabbitmq/amqp091-go"

func (b *Broker) Unload() (<-chan amqp.Delivery, error) {
	msgs, err := b.channel.Consume(
		b.QueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return msgs, nil
}
