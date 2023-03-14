package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	assert.NotPanics(t, func() {
		logger, err := NewLogger()
		assert.NoError(t, err)
		assert.NotNil(t, logger)
		assert.IsType(t, &zap.Logger{}, logger)
	})
}

func TestMustNewLogger(t *testing.T) {
	assert.NotPanics(t, func() {
		logger := MustNewLogger()
		assert.NotNil(t, logger)
		assert.IsType(t, &zap.Logger{}, logger)
	})
}
