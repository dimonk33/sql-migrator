package logger

import (
	"log"
	"os"
)

const (
	LevelError = "error"
	LevelInfo  = "info"
	LevelDebug = "debug"
)

type Logger struct {
	level      string
	errorLog   *log.Logger
	warningLog *log.Logger
	infoLog    *log.Logger
	debugLog   *log.Logger
}

func New(level string) *Logger {
	return &Logger{
		level:      level,
		errorLog:   log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile),
		warningLog: log.New(os.Stdout, "WARNING\t", log.Ldate|log.Ltime|log.Lshortfile),
		infoLog:    log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		debugLog:   log.New(os.Stdout, "DEBUG\t", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

func (l Logger) Info(msg string) {
	if l.level == LevelInfo || l.level == LevelDebug {
		l.infoLog.Println(msg)
	}
}

func (l Logger) Error(msg string) {
	l.errorLog.Println(msg)
}

func (l Logger) Warning(msg string) {
	if l.level != LevelError {
		l.warningLog.Println(msg)
	}
}

func (l Logger) Debug(msg string) {
	if l.level == LevelDebug {
		l.debugLog.Println(msg)
	}
}
