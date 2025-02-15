package utils

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type LogLevel int

const (
	DEBUG = iota
	INFO
	WARN
	ERROR
	FATAL
)

var (
	LogLevelNames = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
		FATAL: "FATAL",
	}
)

type Logger struct {
	*log.Logger
	mtx   sync.Mutex
	name  string
	level LogLevel
}

func (lg *Logger) SetLevel(level LogLevel) *Logger {
	lg.level = level
	return lg
}

func (lg *Logger) SetLevelName(level string) *Logger {
	for k, v := range LogLevelNames {
		if v == strings.ToUpper(level) {
			lg.SetLevel(k)
		}
	}
	return lg
}

func NewLogger(name string, writer io.Writer, level LogLevel, flags int) *Logger {
	lg := &Logger{
		Logger: log.New(writer, name, flags),
		name:   name,
		level:  level,
	}
	return lg
}

//func insert(array []int, element int, i int) []int {
//return append(array[:i], append([]int{element}, array[i:]…)…)

func (lg *Logger) write(level LogLevel, vars ...any) {
	if lg.level > level {
		return
	}
	defer lg.mtx.Unlock()
	lg.mtx.Lock()
	prefix := lg.Prefix()
	lg.SetPrefix(LogLevelNames[level] + " ")
	if _, file, line, ok := runtime.Caller(2); ok {
		file = filepath.Base(file)
		fl := fmt.Sprintf("%s:%d", file, line)
		vars = append(vars[:0], append([]any{fl}, vars...)...)

	}
	lg.Println(vars...)

	lg.SetPrefix(prefix)
}

func (lg *Logger) Debug(vars ...any) {
	lg.write(DEBUG, vars...)
}

func (lg *Logger) Info(vars ...any) {
	lg.write(INFO, vars...)
}

func (lg *Logger) Warn(vars ...any) {
	lg.write(WARN, vars...)

}

func (lg *Logger) Error(vars ...any) {
	lg.write(ERROR, vars...)
}

func (lg *Logger) Fatal(vars ...any) {
	lg.write(FATAL, vars...)
}
