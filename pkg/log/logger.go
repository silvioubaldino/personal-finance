package log

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	With(fields ...Field) Logger
}

type Field interface {
	Key() string
	Value() interface{}
}

type LoggerOption func(*config)

func New(options ...LoggerOption) Logger {
	cfg := defaultConfig()
	for _, opt := range options {
		opt(cfg)
	}
	return newZapLogger(cfg)
}

var Global Logger

func SetGlobalLogger(logger Logger) {
	Global = logger
}

func Initialize(options ...LoggerOption) {
	Global = New(options...)
}

func Debug(msg string, fields ...Field) {
	if Global != nil {
		Global.Debug(msg, fields...)
	}
}

func Info(msg string, fields ...Field) {
	if Global != nil {
		Global.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...Field) {
	if Global != nil {
		Global.Warn(msg, fields...)
	}
}

func Error(msg string, fields ...Field) {
	if Global != nil {
		Global.Error(msg, fields...)
	}
}

func Fatal(msg string, fields ...Field) {
	if Global != nil {
		Global.Fatal(msg, fields...)
	}
}

func With(fields ...Field) Logger {
	if Global != nil {
		return Global.With(fields...)
	}
	return nil
}
