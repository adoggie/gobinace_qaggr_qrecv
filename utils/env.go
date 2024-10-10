package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

func ExecPath() string {
	if path, err := os.Getwd(); err == nil {
		return path
	}
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	fmt.Println(exPath)
	return exPath
}

type ProcessOnce struct {
	file *os.File
	fn   string
}

func (po *ProcessOnce) Lock(fn string) error {
	var err error
	po.fn = fn
	po.file, err = po.createPIDFile(po.fn)
	return err
}

func (po *ProcessOnce) Unlock() {
	if po.file != nil {
		po.file.Close()
		os.Remove(po.fn)
	}
}

func (po *ProcessOnce) createPIDFile(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open or create PID file: %w", err)
	}

	// 尝试获取文件锁
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("Another instance is already running: %w", err)
	}

	// 写入PID到文件
	pid := os.Getpid()
	_, err = f.Write([]byte(strconv.Itoa(pid)))
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to write PID to file: %w", err)
	}

	// 将文件指针移动到文件开头
	_, err = f.Seek(0, 0)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to seek to the beginning of file: %w", err)
	}
	return f, nil
}
