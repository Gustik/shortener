package zaplog_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/shortener/internal/zaplog"
)

func TestNew_ValidLevel(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		logger, err := zaplog.New(level)
		require.NoError(t, err, "уровень %s должен быть валидным", level)
		assert.NotNil(t, logger)
	}
}

func TestNew_InvalidLevel(t *testing.T) {
	logger, err := zaplog.New("notavalidlevel")
	assert.Error(t, err)
	assert.Nil(t, logger)
}

func TestNewNoop(t *testing.T) {
	logger := zaplog.NewNoop()
	assert.NotNil(t, logger)
	// No-op логгер не должен паниковать при использовании
	logger.Info("test message")
}
