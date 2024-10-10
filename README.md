## CC suites

cc 交易套件

sudo apt install libczmq4 -y
sudo apt install libczmq-dev libczmq4 -y
apt instal ntpdate -y
ntpdate ntp.aliyun.com

```shell
静态编译
CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' .

#### install切换时区
/usr/share/zoneinfo/Etc/UTC
ln -s /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

> go get github.com/go-zeromq/zmq4
> go get github.com/pebbe/zmq4
> go get go.nanomsg.org/mangos/v3
> go get github.com/rodaine/table
>

```js

apikey = "36de1b5f-3576-4e12-83a0-1e111ce67342"
secretkey = ""
IP = ""
备注名 = "test"
权限 = "读取/交易"

```

#### nanomsg

https://github.com/nanomsg/mangos/tree/master/examples
https://pkg.go.dev/go.nanomsg.org/mangos/v3

#### kafka

https://kafka.apache.org/downloads
https://docs.confluent.io/kafka-clients/python/current/overview.html
https://github.com/confluentinc/confluent-kafka-go
go get -u github.com/confluentinc/confluent-kafka-go/v2/kafka
https://kafka.apache.org/documentation/#quickstart

https://downloads.apache.org/kafka/3.8.0/kafka_2.12-3.8.0.tgz

$ bin/kafka-topics.sh --create --topic quickstart-events --bootstrap-server 172:9092
https://github.com/confluentinc/librdkafka/blob/master/CONFIGURATION.md

https://github.com/segmentio/kafka-go

#### zmq

go get github.com/zeromq/goczmq

#### protobuf

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
https://protobuf.dev/getting-started/gotutorial/

#### okex

> https://www.okx.com/docs-v5/zh/#websocket-api-trade-place-order

## Build and Install

go build github.com/adoggie/ccsuites/cmd/QuoteRecvSerivce
go run github.com/adoggie/ccsuites/cmd/QuoteRecvSerivce
go build github.com/adoggie/ccsuites/cmd/QuoteAggrService 

