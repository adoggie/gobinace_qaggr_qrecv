package utils

import (
	"github.com/sirupsen/logrus"
	"os"
)

type LogLevelHook struct {
	levels []logrus.Level
	writer *os.File
}

func NewLogLevelHook(writer *os.File, levels ...logrus.Level) *LogLevelHook {
	return &LogLevelHook{levels, writer}
}

func (hook *LogLevelHook) Fire(entry *logrus.Entry) error {
	for _, level := range hook.levels {
		if entry.Level == level {
			_, err := hook.writer.Write([]byte(entry.Message + "\n"))
			return err
		}
	}
	return nil
}

// Levels 实现logrus.Hook接口的Levels方法
func (hook *LogLevelHook) Levels() []logrus.Level {
	return hook.levels
}
