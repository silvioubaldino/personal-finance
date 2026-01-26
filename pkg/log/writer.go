package log

import "strings"

type LoggerWriter struct {
	logger Logger
	level  LogLevel
}

func NewLoggerWriter(logger Logger, level LogLevel) *LoggerWriter {
	if logger == nil {
		logger = Global
	}
	if logger == nil {
		logger = New()
	}
	return &LoggerWriter{
		logger: logger,
		level:  level,
	}
}

func (w *LoggerWriter) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	if message == "" {
		return len(p), nil
	}

	switch w.level {
	case DebugLevel:
		w.logger.Debug(message, String("component", "gin"))
	case WarnLevel:
		w.logger.Warn(message, String("component", "gin"))
	case ErrorLevel:
		w.logger.Error(message, String("component", "gin"))
	case FatalLevel:
		w.logger.Fatal(message, String("component", "gin"))
	default:
		w.logger.Info(message, String("component", "gin"))
	}

	return len(p), nil
}
