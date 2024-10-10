package core

import (
	"github.com/gorilla/websocket"
)

type MessageType int //消息类型

const (
	MessageType_Unknown MessageType = iota
	MessageType_Quote
	MessageType_Command
)

type CommandType int //命令
const (
	Command_Unknown    = "unknown"
	Command_Login      = "login"
	Command_Accept     = "accept"
	Command_Reject     = "reject"
	Command_Logout     = "logout"
	Command_Heart      = "heart"
	Command_DataResend = "data_resend"
	Command_BackFill   = "data_back_fill"
)

type Message struct {
	Type     MessageType
	Ts       uint64
	SeqNo    uint64
	ReqNo    string
	Sender   string // 发送者标示
	LiveTime uint64 //消息有效时间
	Payload  interface{}
}

type CommandMessage struct {
	Message
	Type CommandType
}

type CommandLogin struct {
	CommandMessage
	Username string
	Password string
}

type CommandAccept struct {
	CommandMessage
}

type CommandReject struct {
	CommandMessage
	Reason string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
