package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

func newZapLogger(cfg *config) Logger {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.format == JSONFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	writeSyncer, err := cfg.getWriteSyncer()
	if err != nil {
		writeSyncer = os.Stderr
	}

	level := getZapLevel(cfg.level)

	core := zapcore.NewCore(encoder, writeSyncer.(zapcore.WriteSyncer), level)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))

	return &zapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
	}
}

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, fieldsToZapFields(fields)...)
}

func (l *zapLogger) With(fields ...Field) Logger {
	newLogger := l.logger.With(fieldsToZapFields(fields)...)
	return &zapLogger{
		logger: newLogger,
		sugar:  newLogger.Sugar(),
	}
}

func fieldsToZapFields(fields []Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		if zapField, ok := field.(zapField); ok {
			zapFields[i] = zapField.field
		}
	}
	return zapFields
}

func getZapLevel(level LogLevel) zapcore.Level {
	switch level {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
