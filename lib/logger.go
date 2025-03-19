// Atomic Installer
// Copyright (C) 2025 Дмитрий Удалов dmitry@udalov.online
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package lib

import (
	"bytes"
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// Log — глобальный logrus
var Log = logrus.New()

// SafeBuffer — потокобезопасный буфер с каналом уведомлений.
type SafeBuffer struct {
	mu sync.Mutex
	*bytes.Buffer
	notify chan struct{}
}

// NewSafeBuffer создаёт новый SafeBuffer с инициализированным каналом.
func NewSafeBuffer() *SafeBuffer {
	return &SafeBuffer{
		Buffer: new(bytes.Buffer),
		notify: make(chan struct{}, 1),
	}
}

// Write реализует интерфейс io.Writer с блокировкой.
// После записи, выполняется неблокирующая отправка в канал notify.
func (b *SafeBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	n, err = b.Buffer.Write(p)
	b.mu.Unlock()
	// Неблокирующая отправка сигнала:
	select {
	case b.notify <- struct{}{}:
	default:
	}
	return n, err
}

// NotifyChan возвращает канал уведомлений об изменениях.
func (b *SafeBuffer) NotifyChan() <-chan struct{} {
	return b.notify
}

// LogBuffer — глобальный буфер, куда будут копироваться логи.
var LogBuffer = NewSafeBuffer()

// GetLogText возвращает содержимое буфера логов в виде строки.
func GetLogText() string {
	LogBuffer.mu.Lock()
	defer LogBuffer.mu.Unlock()
	return LogBuffer.Buffer.String()
}

// InitLogger инициализирует логгер logrus, направляя вывод в файл (или stdout)
func InitLogger() {
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   false,
	})

	// Пытаемся открыть файл для логирования
	file, err := os.OpenFile(Env.PathLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		Log.Warn("Failed to open log file, logging to stdout and buffer only: ", err)
	}

	var writers []io.Writer
	if file != nil {
		writers = append(writers, file)
	}
	if LogBuffer != nil {
		writers = append(writers, LogBuffer)
	}
	writers = append(writers, os.Stdout)

	multi := io.MultiWriter(writers...)
	Log.SetOutput(multi)
	Log.SetLevel(logrus.DebugLevel)
}
