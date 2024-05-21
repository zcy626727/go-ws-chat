package chat

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

const (
	ExchangeName  = "chat"
	connectionURI = "amqp://zcy:123456@localhost:5672/"
)

var MQChannel *amqp.Channel
var MQQueue amqp.Queue

func init() {
	// 创建连接
	conn, err := amqp.Dial(connectionURI)
	if err != nil {
		log.Fatalf("%s: %s", "Failed to connect to RabbitMQ", err)
	}
	// 获取管道
	MQChannel, err = conn.Channel()
	if err != nil {
		log.Fatalf("%s: %s", "Failed to open a channel", err)
		return
	}

	// 创建临时队列
	MQQueue, err = MQChannel.QueueDeclare(
		"",    // 临时队列
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("%s: %s", "Failed to declare a queue", err)
		return
	}

	// 创建交换机
	err = MQChannel.ExchangeDeclare(
		ExchangeName, // name
		"fanout",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		log.Fatalf("%s: %s", "Failed to declare an exchange", err)
		return
	}

	// 绑定交换机和队列
	err = MQChannel.QueueBind(
		MQQueue.Name, // queue name
		"",           // routing key
		ExchangeName, // exchange
		false,
		nil,
	)

}

func pushMessageToMQ(message []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := MQChannel.PublishWithContext(ctx,
		ExchangeName, // exchange
		"",           // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		})
	if err != nil {
		println(err)
		log.Fatalf("Error producing: %s", err)
	}
}
