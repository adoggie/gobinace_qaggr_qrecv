package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// WsHandler handle raw websocket message
type WsHandler func(message []byte)

// ErrHandler handles errors
type ErrHandler func(err error)

type WebsocketAPIClient struct {
	APIKey         string
	APISecret      string
	Endpoint       string
	Conn           *websocket.Conn
	Dialer         *websocket.Dialer
	ReqResponseMap map[string]chan []byte
	mtxReqResp     sync.Mutex
	mtx            sync.Mutex
}

type WsAPIRateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
	Count         int    `json:"count"`
}

type WsAPIErrorResponse struct {
	Code    int    `json:"code"`
	ID      string `json:"id"`
	Message string `json:"msg"`
}

var (
	// WebsocketAPITimeout is an interval for sending ping/pong messages if WebsocketKeepalive is enabled
	WebsocketAPITimeout = time.Second * 60
	// WebsocketAPIKeepalive enables sending ping/pong messages to check the connection stability
	WebsocketAPIKeepalive = true
)

func NewWebsocketAPIClient(apiKey string, apiSecret string, baseURL ...string) *WebsocketAPIClient {
	// Set default base URL to production WS URL
	url := "wss://ws-api.binance.com:443/ws-api/v3"

	if len(baseURL) > 0 {
		url = baseURL[0]
	}

	return &WebsocketAPIClient{
		APIKey:    apiKey,
		APISecret: apiSecret,
		Endpoint:  url,
		Dialer: &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  45 * time.Second,
			EnableCompression: false,
		},
	}
}

func (c *WebsocketAPIClient) AddReqRespChan(id string, ch chan []byte) {
	c.mtxReqResp.Lock()
	defer c.mtxReqResp.Unlock()
	c.ReqResponseMap[id] = ch
}

func (c *WebsocketAPIClient) RemoveReqRespChan(id string) {
	c.mtxReqResp.Lock()
	defer c.mtxReqResp.Unlock()

	if _, ok := c.ReqResponseMap[id]; ok {
		delete(c.ReqResponseMap, id)
	}
}
func (c *WebsocketAPIClient) Connect() error {
	if c.Dialer == nil {
		return fmt.Errorf("dialer not initialized")
	}
	headers := http.Header{}
	headers.Add("User-Agent", fmt.Sprintf("%s/%s", "Name", "Version"))
	conn, _, err := c.Dialer.Dial(c.Endpoint, headers)
	if err != nil {
		return err
	}

	fmt.Println("Connected to Binance Websocket API")
	c.Conn = conn

	c.ReqResponseMap = make(map[string]chan []byte)
	c.startReader() // start reader again
	return nil
}

func (c *WebsocketAPIClient) startReader() {
	go func() {
		for {
			_, message, err := c.Conn.ReadMessage()
			if err != nil {
				log.Println("Error reading:", err)
				return
			}
			c.Handler(message)
		}
	}()
}

// Handler function to handle responses
func (c *WebsocketAPIClient) Handler(message []byte) {
	var response WsAPIErrorResponse
	err := json.Unmarshal(message, &response)
	if err != nil {
		log.Println("Error unmarshaling:", err)
		return
	}
	// Send the message to the corresponding request
	//if channel, ok := c.ReqResponseMap[response.ID]; ok {
	//	channel <- message
	//}

	// scott fixed thread safe issue
	c.mtxReqResp.Lock()
	channel, ok := c.ReqResponseMap[response.ID]
	c.mtxReqResp.Unlock()
	if ok {
		channel <- message
	}
}

func (c *WebsocketAPIClient) WaitForCloseSignal() {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	<-stopCh
}

func (c *WebsocketAPIClient) Close() error {
	return c.Conn.Close()
}

func (c *WebsocketAPIClient) SendMessage(msg interface{}) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	return c.Conn.WriteJSON(msg)
}

func (c *WebsocketAPIClient) RequestHandler(req interface{}, handler WsHandler, errHandler ErrHandler) (stopCh chan struct{}, err error) {
	err = c.SendMessage(req)
	if err != nil {
		return nil, err
	}
	stopCh, err = wsApiServe(c.Conn, handler, errHandler)
	if err != nil {
		return nil, err
	}
	return stopCh, nil
}

func keepAlive(c *websocket.Conn, timeout time.Duration) {
	ticker := time.NewTicker(timeout)

	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		defer ticker.Stop()
		for {
			deadline := time.Now().Add(10 * time.Second)
			err := c.WriteControl(websocket.PingMessage, []byte{}, deadline)
			if err != nil {
				return
			}
			<-ticker.C
			if time.Since(lastResponse) > timeout {
				return
			}
		}
	}()
}

func wsApiServe(c *websocket.Conn, handler WsHandler, errHandler ErrHandler) (stopCh chan struct{}, err error) {
	stopCh = make(chan struct{})
	go func() {
		if WebsocketAPIKeepalive {
			keepAlive(c, WebsocketAPITimeout)
		}
		silent := false
		for {
			select {
			case <-stopCh:
				return
			default:
				_, message, err := c.ReadMessage()
				if err != nil {
					if !silent {
						fmt.Println(err)
						errHandler(err)
					}
					continue
				}
				handler(message)
			}
		}
	}()
	return stopCh, nil
}

func websocketAPISignature(apiKey string, apiSecret string, parameters map[string]string) (map[string]string, error) {
	if apiKey == "" || apiSecret == "" {
		return nil, &WebsocketClientError{
			Message: "api_key and api_secret are required for websocket API signature",
		}
	}

	parameters["timestamp"] = strconv.FormatInt(time.Now().Unix()*1000, 10)
	parameters["apiKey"] = apiKey

	// Sort parameters by key
	keys := make([]string, 0, len(parameters))
	for key := range parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build sorted query string
	var sortedParams []string
	for _, key := range keys {
		sortedParams = append(sortedParams, key+"="+parameters[key])
	}

	// Calculate signature
	queryString := strings.Join(sortedParams, "&")
	signature := hmacHashing(apiSecret, queryString)

	parameters["signature"] = signature

	return parameters, nil
}

func hmacHashing(apiSecret string, data string) string {
	mac := hmac.New(sha256.New, []byte(apiSecret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func getUUID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", randomHex(8), randomHex(4), randomHex(4), randomHex(4), randomHex(12))
}

func randomHex(n int) string {
	rand.Seed(time.Now().UnixNano())
	hexChars := "0123456789abcdef"
	bytes := make([]byte, n)
	for i := 0; i < n; i++ {
		bytes[i] = hexChars[rand.Intn(len(hexChars))]
	}
	return string(bytes)
}

type WebsocketClientError struct {
	Message string
}

func (e *WebsocketClientError) Error() string {
	return e.Message
}
