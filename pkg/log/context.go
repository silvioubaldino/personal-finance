package log

import (
	"context"
)

type logCtxKey struct{}

func Context(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, logCtxKey{}, logger)
}

func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return Global
	}

	l, ok := ctx.Value(logCtxKey{}).(Logger)
	if ok {
		return l
	}
	return Global
}

func DebugContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Debug(msg, fields...)
}

func InfoContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Info(msg, fields...)
}

func WarnContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Warn(msg, fields...)
}

func ErrorContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Error(msg, fields...)
}

func FatalContext(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Fatal(msg, fields...)
}
