package broker

import amqp "github.com/rabbitmq/amqp091-go"

type Broker struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	QueueName string
}

func NewBroker(queueName string) (*Broker, error) {
	url, err := getAmqpUrl()
	if err != nil {
		return nil, err
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	broker := &Broker{
		conn:      conn,
		channel:   ch,
		QueueName: queueName,
	}
	err = broker.makeQueue(queueName)
	if err != nil {
		broker.Close()
		return nil, err
	}

	return broker, nil
}
