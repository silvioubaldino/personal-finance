package log

import (
	"os"
)

type LogLevel string

type LogFormat string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"

	JSONFormat LogFormat = "json"
	TextFormat LogFormat = "text"
)

type config struct {
	level  LogLevel
	format LogFormat
	output string
}

func defaultConfig() *config {
	return &config{
		level:  InfoLevel,
		format: JSONFormat,
		output: "stdout",
	}
}

func WithLevel(level string) LoggerOption {
	return func(c *config) {
		c.level = LogLevel(level)
	}
}

func WithFormat(format string) LoggerOption {
	return func(c *config) {
		c.format = LogFormat(format)
	}
}

func WithOutput(output string) LoggerOption {
	return func(c *config) {
		c.output = output
	}
}

func (c *config) getWriteSyncer() (interface{}, error) {
	switch c.output {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		file, err := os.OpenFile(c.output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		return file, nil
	}
}
