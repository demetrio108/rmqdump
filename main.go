package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/streadway/amqp"
)

type BinaryMessage struct {
	CType   string `json:"ctype"`
	B64Body string `json:"b64body"`
}

var (
	publisherModeFlag = flag.Bool("p", false, "publisher mode, read from stdin and publish to rabbitmq")
	queueFlag         = flag.String("q", "", "queue to attach to")
	exchangeFlag      = flag.String("x", "", "exchange to bind to")
	routingKeyFlag    = flag.String("k", "#", "routing key")
	binaryFlag        = flag.Bool("b", false, "assume binary messages, output/input format is {\"ctype\": \"x\", \"b64body\": \"base64 body\"}")

	Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <options> <amq-uri>\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\namqp-uri\trabbitmq uri like amqp://rabbitmq:5672/\n")

		fmt.Fprintf(flag.CommandLine.Output(),
			`
This tool works in consumer mode (default) or publisher mode (-p option).

In consumer mode tehre is 2 cases:
* -x specified, -q is not: autodeclare new queue with random name and bind to exchange by routing key
* -x specified, -q specified: autodeclare new queue with name from config and bind to exchanhe by routing key
* -x unspecified, -q specified: consume from existing queue, fail if it doesn't exist

In publisher mode you just have to specify an exchange.
`)
	}
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func randomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	flag.Usage = Usage
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	conn, err := amqp.Dial(flag.Args()[0])
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// Connecting to existing queue in this case
	if *exchangeFlag == "" {
		if *queueFlag == "" {
			log.Fatal("Existing queue name have to be specified when running without -x flag")
		}
	} else {
		if *queueFlag == "" {
			*queueFlag = fmt.Sprintf("rmqdump-%s", randomString(16))
		}
	}

	if *publisherModeFlag {
		if *exchangeFlag == "" {
			log.Fatal("Exchange flag have to be specified when running in publisher mode")
		}

		buf := make([]byte, 512*1024)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(buf, 512*1024)

		for scanner.Scan() {
			if scanner.Text() == "" {
				continue
			}

			var ctype string
			var body []byte

			if *binaryFlag {
				var msg BinaryMessage

				err := json.Unmarshal(scanner.Bytes(), &msg)
				if err != nil {
					log.Printf("Failed to unmarshal JSON: %s\n, %s", err, scanner.Bytes())
					continue
				}

				body_dec, err := base64.StdEncoding.DecodeString(msg.B64Body)
				if err != nil {
					log.Printf("Failed to decode message: %s\n, %s", err, msg.B64Body)
					continue
				}

				ctype = msg.CType
				body = []byte(body_dec)
			} else {
				ctype = "application/json"
				body = scanner.Bytes()
			}

			msg := amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				Timestamp:    time.Now(),
				ContentType:  ctype,
				Body:         body,
			}

			// This is not a mandatory delivery, so it will be dropped if there are no
			// queues bound to the logs exchange.
			err = ch.Publish(*exchangeFlag, *routingKeyFlag, false, false, msg)
			if err != nil {
				log.Printf("Failed to publish message: %s", err)
			}
		}
	} else {
		q := amqp.Queue{Name: *queueFlag}

		if *exchangeFlag != "" {
			args := amqp.Table{"x-message-ttl": int32(300000)}
			q, err = ch.QueueDeclare(
				*queueFlag, // name
				false,      //durable
				true,       //autodelete
				false,      //exclusive
				false,      // no-wait
				args,       // arguments
			)
			failOnError(err, "Failed to declare a queue")
			err = ch.QueueBind(
				*queueFlag, // name
				*routingKeyFlag,
				*exchangeFlag,
				false, // no-wait
				nil,   // arguments
			)
			failOnError(err, "Failed to declare a queue")
		}

		msgs, err := ch.Consume(
			q.Name, // queue
			"",     // consumer
			true,   // auto-ack
			false,  // exclusive
			false,  // no-local
			false,  // no-wait
			nil,    // args
		)
		failOnError(err, "Failed to register a consumer")

		for d := range msgs {
			if *binaryFlag {
				b64body := base64.StdEncoding.EncodeToString(d.Body)
				fmt.Printf("{\"ctype\": \"%s\", \"b64body\": \"%s\"}\n", d.ContentType, b64body)
			} else {
				fmt.Printf("%s\n", d.Body)
			}
		}
	}
}
