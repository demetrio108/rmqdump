# rmqdump

Simple CLI tool for dumping messages from RabbitMQ to stdout. or publishing from stdin to RabbitMQ.

**This is an alpha software**

## Installing (assuming go environment is set)
```
go install github.com/demetrio108/rmqdump
```

## Usage examples

Dump all messages from exchange:
```
rmqdump -x amq.topic amqp://localhost
```

Dump all messages from exchange with routing key:
```
rmqdump -x amq.topic -k "my.key.#" amqp://localhost
```

Dump all messages from existing queue:
```
rmqdump -q my_queue amqp://localhost
```

Read messages from stdin (JSON) and push to exhange. Reads one message per line:
```
cat messages.json | rmqdump -p -x amq.topic -k my.key.#  amqp://localhost
```

Proxy messages from one RabbitMQ to another:
```
rmqdump -q my_queue1 amqp://user:pass@host1 | rmqdump -p -x amq.topic -k from.host1.# amqp://localhost 
```
