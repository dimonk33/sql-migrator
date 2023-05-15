package logger

import "testing"

func TestLogger(t *testing.T) {
	t.Run("error level log", func(t *testing.T) {
		logger := New(LevelError)
		logMessage := "test 12345"
		logger.Error(logMessage)
	})
}
