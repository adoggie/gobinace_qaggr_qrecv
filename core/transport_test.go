package core

import (
	"bytes"
	"compress/flate"
	"encoding/gob"
	"fmt"
	"testing"
)

func TestData(t *testing.T) {
	//
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	gob.Register(QuoteDepth{})
	gob.Register(QuoteTrade{})
	gob.Register(QuoteMessage{})

	//msg := QuoteMessage{Type: QuoteDataType_Depth, Exchange: ExchangeType_Binance,
	//	Message: Message{Type: MessageType_Quote, Ts: 0}, Data: &QuoteDepth{}}

	msg := Message{Type: MessageType_Quote, Payload: QuoteMessage{Type: QuoteDataType_Depth, Exchange: ExchangeType_Binance, Data: QuoteDepth{}}}

	if err := enc.Encode(msg); err == nil {
		fmt.Println(data.Len(), " ", data)

		buffer := bytes.NewBuffer(nil)
		if writer, err := flate.NewWriter(buffer, flate.DefaultCompression); err == nil {
			writer.Write(data.Bytes())
			//writer.Flush()
			fmt.Println(buffer.Len())
			fmt.Println(buffer)
		}

	} else {
		fmt.Println(err.Error())
	}
	dec := gob.NewDecoder(&data)
	{
		var msg Message
		if err := dec.Decode(&msg); err != nil {
			fmt.Println(err.Error())
		} else {
			if msg.Type == MessageType_Quote {
				if quote, ok := msg.Payload.(QuoteMessage); ok {
					if quote.Type == QuoteDataType_Depth {
						if depth, ok := quote.Data.(QuoteDepth); ok {
							fmt.Println("", depth)
						}
					}

				}

			}
		}
	}
}
