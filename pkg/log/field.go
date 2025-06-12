package log

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapField struct {
	field zapcore.Field
}

func (f zapField) Key() string {
	return f.field.Key
}

func (f zapField) Value() interface{} {
	return f.field.Interface
}

func String(key, value string) Field {
	return zapField{field: zap.String(key, value)}
}

func Int(key string, value int) Field {
	return zapField{field: zap.Int(key, value)}
}

func Int64(key string, value int64) Field {
	return zapField{field: zap.Int64(key, value)}
}

func Float64(key string, value float64) Field {
	return zapField{field: zap.Float64(key, value)}
}

func Bool(key string, value bool) Field {
	return zapField{field: zap.Bool(key, value)}
}

func Time(key string, value time.Time) Field {
	return zapField{field: zap.Time(key, value)}
}

func Duration(key string, value time.Duration) Field {
	return zapField{field: zap.Duration(key, value)}
}

func Any(key string, value interface{}) Field {
	return zapField{field: zap.Any(key, value)}
}

func Err(err error) Field {
	return zapField{field: zap.Error(err)}
}
