package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/adoggie/ccsuites/core"
	"github.com/adoggie/ccsuites/utils"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gookit/goutil/cliutil"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	controller     Controller
	configFileName string
	logger         *logrus.Logger
	config         Config
)

func init() {
	gob.Register(core.QuoteDepth{})
	gob.Register(core.QuoteTrade{})
	gob.Register(core.QuoteMessage{})
	gob.Register(core.QuoteOrbIncr{})
	gob.Register(core.QuoteSnapShot{})
	gob.Register(core.QuoteKline{})
	gob.Register(core.QuoteAggTrade{})

}

// https://learnxinyminutes.com/docs/toml/

func normalize_config(cfg *Config) {
	if cfg.SigPosMultiplier <= 0 {
		cfg.SigPosMultiplier = 1
	}
	cfg.SymbolBlackList = utils.Map(cfg.SymbolBlackList, func(s string) string {
		return strings.ToUpper(s)
	})
	if cfg.RecvOrbIncReconnectTimeout == 0 {
		cfg.RecvOrbIncReconnectTimeout = 120
	}
	if cfg.RecvTradeReconnectTimeout == 0 {
		cfg.RecvTradeReconnectTimeout = 120
	}
}

func main() {
	if runtime.GOOS == "linux" {
		identifier := os.Getenv("AMBER")
		if identifier != "shakira" {
			panic("0x88765244442")
		}
	}

	var version = flag.Bool("v", false, "version")
	flag.StringVar(&configFileName, "config", "settings.toml", "configuration files")

	flag.Parse()
	if *version {
		cliutil.Redln("Author:", AUTHOR)
		cliutil.Redln("Version:", VERSION)
		cliutil.Redln("Date:", DATE)
		return
	}

	//color.ForceColor()
	data, _ := os.ReadFile(configFileName)
	if _, e := toml.Decode(string(data), &config); e != nil {
		fmt.Println(e)
		return
	}
	normalize_config(&config)

	cliutil.Blueln("BN-DRS start..")
	cliutil.Blueln("Id:", config.ThunderId, " Account:", config.Account)

	logger = logrus.New()
	logger.Formatter = new(logrus.TextFormatter) //default
	logger.Formatter = &nested.Formatter{
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
		TimestampFormat: time.DateTime,
		NoColors:        false,
	}
	logger.Level = logrus.InfoLevel
	logger.Out = os.Stdout

	logdir := config.LogDir
	os.MkdirAll(logdir, 0755)
	logfile := path.Join(logdir, fmt.Sprintf("bn_drs_%s_%s.log", config.ThunderId, config.Account))

	lumberjackLogger := &lumberjack.Logger{
		// Log file abbsolute path, os agnostic
		Filename:   filepath.ToSlash(logfile),
		MaxSize:    5, // MB
		MaxBackups: 10,
		MaxAge:     30,   // days
		Compress:   true, // disabled by default
	}

	logger.SetOutput(io.MultiWriter(os.Stdout, lumberjackLogger))
	logger.Level, _ = logrus.ParseLevel(config.LogLevel)

	// log error file recorder
	{
		errlog := path.Join(logdir, fmt.Sprintf("bnthunder_%s_%s.err", config.ThunderId, config.Account))
		writer, err := os.OpenFile(errlog, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
		if err != nil {
			log.Fatalf("Create Log File Failed: %v \n", err)
			return
		}
		//logger.SetOutput(io.MultiWriter(writer))
		logger.AddHook(utils.NewLogLevelHook(writer, logrus.ErrorLevel, logrus.PanicLevel, logrus.FatalLevel))
	}

	////////////////////
	pidfile := fmt.Sprintf("/tmp/bn_drs_%s_%s.pid", config.ThunderId, config.Account)
	po := utils.ProcessOnce{}
	if err := po.Lock(pidfile); err != nil {
		fmt.Println(err.Error())
		return
	}
	defer po.Unlock()

	_ = controller
	controller.init(&config)
	controller.open()
	logger.Info("BN-DRS Start..")
	waitCh := make(chan struct{})
	<-waitCh

}
