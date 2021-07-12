package amqp

import (
	"time"

	"github.com/spiral/errors"
	"github.com/spiral/roadrunner/v2/pkg/events"
)

func (j *JobsConsumer) initRabbitMQ() error {
	const op = errors.Op("jobs_plugin_rmq_init")
	// Channel opens a unique, concurrent server channel to process the bulk of AMQP
	// messages.  Any error from methods on this receiver will render the receiver
	// invalid and a new Channel should be opened.
	channel, err := j.conn.Channel()
	if err != nil {
		return errors.E(op, err)
	}

	// declare an exchange (idempotent operation)
	err = channel.ExchangeDeclare(
		j.exchangeName,
		j.exchangeType,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return errors.E(op, err)
	}

	// verify or declare a queue
	q, err := channel.QueueDeclare(
		j.queue,
		false,
		false,
		true,
		false,
		nil,
	)
	if err != nil {
		return errors.E(op, err)
	}

	// bind queue to the exchange
	err = channel.QueueBind(
		q.Name,
		j.routingKey,
		j.exchangeName,
		false,
		nil,
	)
	if err != nil {
		return errors.E(op, err)
	}

	j.eh.Push(events.JobEvent{
		Event:  events.EventInitialized,
		Driver: "amqp",
		Start:  time.Now(),
	})
	return channel.Close()
}
