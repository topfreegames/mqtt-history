package app

import (
	"fmt"

	"github.com/uber-go/zap"
	"github.com/uber/jaeger-client-go"
)

func WrapZapLogger(zl zap.Logger) jaeger.Logger {
	return &jaegerZapLogger{zl}
}

type jaegerZapLogger struct {
	logger zap.Logger
}

func (l *jaegerZapLogger) Error(msg string) {
	l.logger.Error(msg)
}

func (l *jaegerZapLogger) Infof(msg string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(msg, args...))
}
