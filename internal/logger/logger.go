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

func (l Logger) Info(v ...any) {
	if l.level == LevelInfo || l.level == LevelDebug {
		l.infoLog.Println(v...)
	}
}

func (l Logger) Error(v ...any) {
	l.errorLog.Println(v...)
}

func (l Logger) Warning(v ...any) {
	if l.level != LevelError {
		l.warningLog.Println(v...)
	}
}

func (l Logger) Debug(v ...any) {
	if l.level == LevelDebug {
		l.debugLog.Println(v...)
	}
}
