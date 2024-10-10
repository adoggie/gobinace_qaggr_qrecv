package main

import (
	"bytes"
	"compress/zlib"
	"container/list"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/adoggie/ccsuites/message"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type WsUserAccount struct {
	User   string   `json:"user"`
	Passwd string   `json:"passwd"`
	Enable bool     `json:"enable"`
	Ips    []string `json:"ips"`
}

type WsServerConfigVars struct {
	Enable         bool   `toml:"enable"`
	PassUsers      string `toml:"passusers"`
	UpdateUserTime int64  `toml:"update_user_time"`
	ListenAddr     string `toml:"listen_addr"`
	MaxConns       int64  `toml:"max_conns"`
	IpLimited      bool   `toml:"ip_limited"`
	AuthTimeWait   int64  `toml:"auth_time_wait"`
	DataCacheTime  int64  `toml:"data_cache_time"`
	CompressData   bool   `toml:"compress_data"`
	SendMsgHdr     bool   `toml:"send_msg_hdr"`
	Users          []WsUserAccount
}

type WsSendData struct {
	Period int64
	Ts     int64
	Size   int64 // uncompressed size
	PostTs int64
	Data   []byte
}

func NewWsSendData(pm *message.PeriodMessage) *WsSendData {
	data := &WsSendData{Period: pm.Period, Ts: pm.Ts}
	var bs []byte
	var err error
	if bs, err = proto.Marshal(pm); err == nil {
		datasize := len(bs)
		data.Size = int64(datasize)
		if controller.config.WsServerConfig.CompressData {
			var buffer bytes.Buffer
			var writer *zlib.Writer
			writer = zlib.NewWriter(&buffer)
			if writer != nil {
				var compressed int
				if compressed, err = writer.Write(bs); err == nil {
					_ = compressed
					_ = writer.Close()
					bs = buffer.Bytes()
				}
			} else {
				logger.Errorln("NewWsSendData Compress error:", err.Error())
				return nil
			}
		}
		//logger.Traceln("NewWsSendData send data size:", datasize, " compressed size:", len(bs))
		data.Data = bs
	} else {
		logger.Errorln("NewWsSendData Marshal error:", err.Error())
		return nil
	}
	return data
}

type WsClient struct {
	conn        *websocket.Conn
	born        time.Time
	user        WsUserAccount
	cur         *list.Element // 数据游标
	server      *WsServer
	start       int64
	msgCh       chan *WsSendData
	msgLiveCh   chan *WsSendData
	pre_sync_ok atomic.Bool
	accmsg      []*WsSendData
	mtx         sync.RWMutex
}

func NewWsClient(conn *websocket.Conn, server *WsServer, start int64, user WsUserAccount) *WsClient {
	client := &WsClient{conn: conn, born: time.Now(), cur: nil, start: start, server: server, user: user}
	client.msgCh = make(chan *WsSendData, server.config.DataCacheTime*60/3)
	client.msgLiveCh = make(chan *WsSendData, 1024)
	client.accmsg = make([]*WsSendData, 0)
	return client
}

type WsServer struct {
	dataQueue list.List
	config    *WsServerConfigVars
	mtx       sync.RWMutex
	clients   map[string]*WsClient
	ctx       context.Context
	upgrade   websocket.Upgrader
}

func NewWsServer(config *WsServerConfigVars) *WsServer {
	server := &WsServer{config: config}
	server.dataQueue.Init()
	server.upgrade = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	server.clients = make(map[string]*WsClient)

	return server

}

func (s *WsServer) enqueue(pm *message.PeriodMessage) {
	senddata := NewWsSendData(pm)
	if senddata == nil {
		return
	}

	s.mtx.Lock()
	s.dataQueue.PushBack(senddata)
	if int64(s.dataQueue.Len()) > s.config.DataCacheTime*60/3 {
		s.dataQueue.Remove(s.dataQueue.Front())
	}
	for name, client := range s.clients {
		_ = name
		//client.msgLiveCh <- pm
		client.delivery(senddata)
	}
	s.mtx.Unlock()

}

// func (c *WsClient) delivery(pm *message.PeriodMessage) {
func (c *WsClient) delivery(pm *WsSendData) {
	//c.accmsg = append(c.accmsg,pm)
	c.msgLiveCh <- pm
}

func (c *WsClient) get_id() string {
	return fmt.Sprintf("%s_%s", c.user.User, c.conn.RemoteAddr().String())
}

func (c *WsClient) close() {
	if err := c.conn.Close(); err != nil {
		logger.Warnln(err.Error())
	}
}

func (c *WsClient) run() {
	logger.Infoln("WsClient Running..", c.get_id())
	c.pre_sync_ok.Store(true)
	go c.handleMessages()

	if c.start != 0 {
		//c.start = c.start * 1000
		c.pre_sync_ok.Store(false)

		go func() {
			c.server.mtx.RLock()
			pms := make([]*WsSendData, 0)
			for e := c.server.dataQueue.Front(); e != nil; e = e.Next() {
				// 查找start 标记记录
				pm := e.Value.(*WsSendData)
				if c.start <= pm.Ts {
					pms = append(pms, pm)
				}
			}
			c.server.mtx.RUnlock()

			for _, pm := range pms {
				c.msgCh <- pm
			}
			c.mtx.Lock()
			for _, pm := range c.accmsg {
				c.msgCh <- pm
			}
			c.accmsg = make([]*WsSendData, 0) // free memory
			c.mtx.Unlock()
			c.pre_sync_ok.Store(true)

		}()
	}

	timer_hb := time.After(time.Second * 10)
	for {
		select {
		//case <-stop:
		//	logger.Infoln("WsClient Exiting..")
		//	return
		case pm := <-c.msgCh:
			go c.send(pm)

		case pm := <-c.msgLiveCh:
			if c.pre_sync_ok.Load() {
				c.msgCh <- pm
			} else {
				c.mtx.Lock()
				c.accmsg = append(c.accmsg, pm)
				c.mtx.Unlock()

			}

		case <-timer_hb:
			logger.Debugln("Heartbeat..", c.get_id())
			err := c.conn.WriteMessage(websocket.PingMessage, nil)

			if err != nil {
				logger.Errorln(c.get_id(), "Heartbeat error:", err.Error())
				c.conn.Close()
			}
		}
	}
}

// 定义结构体
type MessageHeaderFixed struct {
	Version       int8
	Compress      int8
	Period        int64
	Timestamp     int64
	PostTimestamp int64
}

func (c *WsClient) sendHdr(data *WsSendData) (hdr []byte, err error) {
	header := MessageHeaderFixed{
		Version:       1,
		Compress:      1, // 1: zlib
		Period:        data.Period,
		Timestamp:     data.Ts,
		PostTimestamp: time.Now().UTC().UnixMilli(),
	}
	buf := new(bytes.Buffer)
	// 采用网络字节序（大端序）
	err = binary.Write(buf, binary.BigEndian, header.Version)
	if err != nil {
		return
	}

	err = binary.Write(buf, binary.BigEndian, header.Compress)
	if err != nil {
		return
	}

	err = binary.Write(buf, binary.BigEndian, header.Period)
	if err != nil {
		return
	}

	err = binary.Write(buf, binary.BigEndian, header.Timestamp)
	if err != nil {
		return
	}

	err = binary.Write(buf, binary.BigEndian, header.PostTimestamp)
	if err != nil {
		return
	}

	// 将 buf 转换为固定长度的字节数组（26字节）
	hdr = buf.Bytes()
	//var serializedHeader [26]byte
	//copy(serializedHeader[:], buf.Bytes())
	//fmt.Println(len(buf.Bytes()))
	//err = c.conn.WriteMessage(websocket.BinaryMessage, serializedHeader[:])
	return
}

func (c *WsClient) send(pm *WsSendData) (err error) {
	var hdr []byte
	var body []byte = make([]byte, 0)
	if c.server.config.SendMsgHdr {
		hdr, err = c.sendHdr(pm)
		if err != nil {
			logger.Errorln("WsServer Send MsgHdr Error:", err.Error(), c.get_id())
			return
		}
		body = append(hdr, pm.Data...)
		//body = hdr
		err = c.conn.WriteMessage(websocket.BinaryMessage, body)
		if err != nil {
			logger.Errorln("WsServer Send Error:", err.Error(), c.get_id())
			return
		}
	} else {
		body = pm.Data
		err = c.conn.WriteMessage(websocket.BinaryMessage, body)
		if err != nil {
			logger.Errorln("WsServer Send Error:", err.Error(), c.get_id())
			return
		}
	}

	//body = append(body, pm.Data...)
	//err = c.conn.WriteMessage(websocket.BinaryMessage, body)
	//if err != nil {
	//	logger.Errorln("WsServer Send Error:", err.Error(), c.get_id())
	//	return
	//}
	logger.Debugln("WsServer data sent ", c.get_id(), " period:", pm.Period, " ts:", pm.Ts, "data size:", pm.Size, " compressed size:", len(pm.Data))

	return
}

func (c *WsClient) _send(pm *message.PeriodMessage) (err error) {
	var bs []byte
	if bs, err = proto.Marshal(pm); err == nil {
		datasize := len(bs)
		if c.server.config.CompressData {
			var buffer bytes.Buffer
			var writer *zlib.Writer
			writer = zlib.NewWriter(&buffer)
			if writer != nil {
				var compressed int
				if compressed, err = writer.Write(bs); err == nil {
					_ = compressed
					_ = writer.Close()
					bs = buffer.Bytes()

				}
			} else {
				logger.Errorln("WsServer Compress error:", err.Error())
				return
			}
		}

		err = c.conn.WriteMessage(websocket.BinaryMessage, bs)
		if err != nil {
			logger.Errorln("WsServer Send Error:", err.Error())
			return
		}
		logger.Traceln("WsServer send data size:", datasize, " compressed size:", len(bs))
	}
	return
}

func (c *WsClient) handleMessages() {
	var (
		err  error
		data []byte
	)

	for {
		if _, data, err = c.conn.ReadMessage(); err != nil {
			_ = data
			logger.Errorln(c.get_id(), err.Error())
			_ = c.conn.Close()
			break
		}
	}
	logger.Infoln("WsClient Exiting..", c.get_id())
	c.server.onclose(c)
}

func (s *WsServer) onclose(client *WsClient) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if c, ok := s.clients[client.user.User]; ok {
		if c == client {
			delete(s.clients, client.user.User)
		}
	}
	logger.Infoln("WsClient Offline:", client.get_id())
}

func (s *WsServer) open(ctx context.Context, stop <-chan struct{}) {
	s.ctx = ctx
	s.reloadUsers()

	go func() {
		timer := time.After(time.Second * time.Duration(s.config.UpdateUserTime))
		for {
			select {
			case <-stop:
				logger.Info("WsServer will exit..")
				return
			case <-timer:
				s.reloadUsers()
				timer = time.After(time.Second * time.Duration(s.config.UpdateUserTime))
			}
		}
	}()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {

		name := r.FormValue("user")
		var start int64 = 0
		start, _ = strconv.ParseInt(r.FormValue("start"), 10, 64)
		param_start := start
		var useracc *WsUserAccount = nil
		for _, user := range s.config.Users {
			if user.User == name {
				logger.Infoln("incoming user is in list..", name)
				if len(user.Ips) != 0 {
					logger.Traceln("checking ip access..", r.RemoteAddr, user.Ips)
					fs := Filter(user.Ips, func(ip string) bool {
						if ip == strings.Split(r.RemoteAddr, ":")[0] {
							return true
						}
						return false
					})
					if len(fs) != 0 {
						useracc = &user
						break
					}
				} else {
					useracc = &user
					break
				}
			}
		}
		//s.register <- conn
		if useracc != nil {
			logger.Infoln("WsClient Access Okay: ", name)
		} else {
			logger.Warnln("WsClient Access Denied: ", name, r.RemoteAddr, s.config.Users)
			return
		}
		conn, err := s.upgrade.Upgrade(w, r, nil)
		if err != nil {
			logger.Fatalln(err.Error())
			return
		}
		//fmt.Println(time.Now().UTC().UnixMilli())
		start = time.Now().UTC().UnixMilli() - int64(start*1000)
		client := NewWsClient(conn, s, start, *useracc)
		logger.Info("New Wsclient Incoming.", client.get_id(), " param_start:", param_start)
		logger.Info("New Wsclient Incoming.", client.get_id(), " start:", start, " ", time.UnixMilli(start).UTC().String())
		s.mtx.Lock()
		if c, ok := s.clients[name]; ok {
			logger.Infoln("User is online , kick it off :", c.get_id())
			c.close()
		}
		s.clients[name] = client
		s.mtx.Unlock()
		go client.run()
	})
	addr := s.config.ListenAddr
	logger.Infoln("WebSocket server listening on ", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err.Error())
	}

}

func (s *WsServer) reloadUsers() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.config.Users = make([]WsUserAccount, 0)

	data, err := ioutil.ReadFile(s.config.PassUsers)
	if err != nil {
		logger.Warnln("Read User Account, ", err.Error())
		return
	}

	err = json.Unmarshal(data, &s.config.Users)
	if err != nil {
		logger.Warnln("Read User Account, ", err.Error())
		return
	}
	logger.Traceln("Load Users:", len(s.config.Users))
}

func (s *WsServer) close() {

}
