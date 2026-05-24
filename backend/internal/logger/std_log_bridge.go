package logger

import (
	"log"
	"strings"
	"sync"
)

type standardLogWriter struct {
	component string
	mu        sync.Mutex
}

func (w *standardLogWriter) Write(payload []byte) (int, error) {
	text := strings.TrimSpace(string(payload))
	if text == "" {
		return len(payload), nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		New(w.component).Info(line)
	}
	return len(payload), nil
}

func InstallStandardLogBridge(component string) {
	component = strings.TrimSpace(component)
	if component == "" {
		component = "Runtime"
	}
	log.SetFlags(0)
	log.SetOutput(&standardLogWriter{component: component})
}
