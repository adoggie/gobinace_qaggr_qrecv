module github.com/adoggie/ccsuites

go 1.21.4

toolchain go1.22.3

require (
	github.com/BurntSushi/toml v1.4.0
	github.com/adshao/go-binance/v2 v2.0.0-00010101000000-000000000000
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/binance/binance-connector-go v0.0.0-00010101000000-000000000000
	github.com/bits-and-blooms/bloom/v3 v3.7.0
	github.com/confluentinc/confluent-kafka-go/v2 v2.5.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-zeromq/goczmq/v4 v4.2.2
	github.com/gookit/goutil v0.6.16
	github.com/gorilla/websocket v1.5.0
	github.com/pebbe/zmq4 v1.2.11
	github.com/segmentio/kafka-go v0.4.47
	github.com/shopspring/decimal v1.4.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
//github.com/confluentinc/confluent-kafka-go
//go.nanomsg.org/mangos/v3 v3.4.2
)

require (
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/term v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)

replace github.com/binance/binance-connector-go => ./pkg/binance-connector-go-0.5.2

replace github.com/adshao/go-binance/v2 => ./pkg/go-binance-2.5.0/v2
