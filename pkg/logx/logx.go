package logx

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Logger struct{ *log.Logger }

func New(service string) *Logger {
	return &Logger{Logger: log.New(os.Stdout, "", 0)}
}

func (l *Logger) Info(msg string, fields map[string]any)  { l.write("info", msg, fields) }
func (l *Logger) Error(msg string, fields map[string]any) { l.write("error", msg, fields) }

func (l *Logger) write(level, msg string, fields map[string]any) {
	m := map[string]any{"ts": time.Now().UTC().Format(time.RFC3339), "level": level, "msg": msg}
	for k, v := range fields {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	l.Println(string(b))
}
