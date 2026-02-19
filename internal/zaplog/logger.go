package zaplog

import (
	"go.uber.org/zap"
)

// New создаёт production zap.Logger с указанным уровнем логирования
// (например "info", "debug", "error").
func New(level string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil, err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return zl, nil
}

// NewNoop возвращает no-op логгер, отбрасывающий весь вывод. Полезен в тестах.
func NewNoop() *zap.Logger {
	return zap.NewNop()
}
