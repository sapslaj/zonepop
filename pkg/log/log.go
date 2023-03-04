package log

import (
	"fmt"

	"go.uber.org/zap"
)

func NewLogger() (*zap.Logger, error) {
	return zap.NewProduction(
		zap.AddCaller(),
		zap.AddStacktrace(zap.DPanicLevel),
	)
}

func MustNewLogger() *zap.Logger {
	l, err := NewLogger()
	if err != nil {
		panic(fmt.Errorf("could not create new logger: %w", err))
	}
	return l
}
