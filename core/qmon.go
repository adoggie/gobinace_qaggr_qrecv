package core

import (
	"context"
	czmq "github.com/go-zeromq/goczmq/v4"
	"github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)
import "encoding/json"

type ServiceMonConfigVars struct {
	Enable         bool   `toml:"enable"`
	ReportDest     string `toml:"report_dest"`
	ReportInterval int64  `toml:"report_interval"`
	AuthKey        string `toml:"auth_key"`
}

type ServiceMonStatus struct {
	MsgType     string `json:"msg_type"`
	ServiceId   string `json:"service_id"`
	ServiceType string `json:"service_type"`
	Version     string `json:"version"`
	Ip          string `json:"ip"`
	Pid         int64  `json:"pid"`
	Start       int64  `json:"start"`
	Now         int64  `json:"now"`
	AuthKey     string `json:"auth_key"`
	Others      string `json:"others"`
}

type ServiceMonLogEntry struct {
	Level    string `json:"level"`
	Category string `json:"category"`
	Action   string `json:"action"`
	Detail   string `json:"detail"`
}

type ServiceMonLog struct {
	MsgType     string             `json:"msg_type"`
	ServiceId   string             `json:"service_id"`
	ServiceType string             `json:"service_type"`
	Version     string             `json:"version"`
	Ip          string             `json:"ip"`
	Pid         int64              `json:"pid"`
	Start       int64              `json:"start"`
	Now         int64              `json:"now"`
	AuthKey     string             `json:"auth_key"`
	Logs        ServiceMonLogEntry `json:"logs"`
}

type ServiceMonManager struct {
	config *ServiceMonConfigVars
	start  int64
	status ServiceMonStatus
	sock   *czmq.Sock
	logger *logrus.Logger
	mtx    sync.Mutex
}

func NewServiceMonManager(config *ServiceMonConfigVars, logger *logrus.Logger) *ServiceMonManager {
	return &ServiceMonManager{config: config, logger: logger}
}
func (m *ServiceMonManager) ServiceId(id string) *ServiceMonManager {
	m.status.ServiceId = id
	return m
}

func (m *ServiceMonManager) ServiceType(tp string) *ServiceMonManager {
	m.status.ServiceType = tp
	return m
}

func (m *ServiceMonManager) Version(v string) *ServiceMonManager {
	m.status.Version = v
	return m
}

func (m *ServiceMonManager) Ip(ip string) *ServiceMonManager {
	m.status.Ip = ip
	return m
}

func (m *ServiceMonManager) Pid(pid int64) *ServiceMonManager {
	m.status.Pid = pid
	return m
}

func (m *ServiceMonManager) Start(start int64) *ServiceMonManager {
	m.status.Start = start
	return m
}

func (m *ServiceMonManager) Now(now int64) *ServiceMonManager {
	m.status.Now = now
	return m
}

func (m *ServiceMonManager) AuthKey(key string) *ServiceMonManager {
	m.status.AuthKey = key
	return m
}

func (m *ServiceMonManager) Init() (err error) {
	m.Start(time.Now().UTC().Unix())
	m.Pid(int64(os.Getpid()))
	m.AuthKey(m.config.AuthKey)

	m.sock = czmq.NewSock(czmq.Pub)
	if err = m.sock.Connect(m.config.ReportDest); err != nil {
		m.logger.Errorln("ServiceMonManager Init Error:", err.Error())
	}
	return
}

func (m *ServiceMonManager) Open(ctx context.Context) error {
	m.SendStatus()
	go func() {
		timer := time.After(time.Second * time.Duration(m.config.ReportInterval))
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer:
				m.SendStatus()
				timer = time.After(time.Second * time.Duration(m.config.ReportInterval))
			}
		}
	}()
	return nil
}

func (m *ServiceMonManager) SendStatus() {

	m.Now(time.Now().UTC().Unix())
	log := ServiceMonStatus{}
	log.MsgType = "cc_status"
	log.ServiceId = m.status.ServiceId
	log.ServiceType = m.status.ServiceType
	log.Version = m.status.Version
	log.Ip = m.status.Ip
	log.Pid = m.status.Pid
	log.Start = m.status.Start
	log.Now = m.status.Now
	log.AuthKey = m.status.AuthKey

	m.logger.Traceln("SendStatus:", log)
	if jsondata, err := json.Marshal(log); err != nil {
		m.logger.Errorln("SendStatus Error:", err.Error())
		return
	} else {
		if err = m.sock.SendFrame([]byte("cc_status"), czmq.FlagMore); err != nil {
			m.logger.Errorln("SendStatus Error:", err.Error())
			return
		}
		if err = m.sock.SendFrame([]byte(jsondata), czmq.FlagNone); err != nil {
			m.logger.Errorln("SendStatus Error:", err.Error())
			return
		}
	}

}

// SendLog send log to monitor ，wait for delay seconds , no wait if delay is 0

func (m *ServiceMonManager) SendLog(level, category, action, detail string, delay int64) {
	go func() {
		if !m.mtx.TryLock() {
			return
		}
		defer m.mtx.Unlock()
		m.Now(time.Now().UTC().Unix())
		log := ServiceMonLog{}
		log.MsgType = "cc_log"
		log.ServiceId = m.status.ServiceId
		log.ServiceType = m.status.ServiceType
		log.Version = m.status.Version
		log.Ip = m.status.Ip
		log.Pid = m.status.Pid
		log.Start = m.status.Start
		log.Now = m.status.Now
		log.AuthKey = m.status.AuthKey
		log.Logs.Level = level
		log.Logs.Category = category
		log.Logs.Action = action
		log.Logs.Detail = detail

		m.logger.Traceln("SendLog:", log)
		if jsondata, err := json.Marshal(log); err != nil {
			m.logger.Errorln("SendLog Error:", err.Error())
			return
		} else {
			if err = m.sock.SendFrame([]byte("cc_log"), czmq.FlagMore); err != nil {
				m.logger.Errorln("SendLog Error:", err.Error())
				return
			}
			if err = m.sock.SendFrame([]byte(jsondata), czmq.FlagNone); err != nil {
				m.logger.Errorln("SendLog Error:", err.Error())
				return
			}
		}
		if delay > 0 {
			time.Sleep(time.Second * time.Duration(delay))
		}
	}()

}

func (m *ServiceMonManager) Close() error {
	return nil
}
