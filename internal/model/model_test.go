package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Gustik/shortener/internal/model"
)

func TestURLRecord_NextID(t *testing.T) {
	var r model.URLRecord

	r.NextID()
	first := r.UUID

	r.NextID()
	second := r.UUID

	assert.NotEmpty(t, first)
	assert.NotEmpty(t, second)
	assert.NotEqual(t, first, second, "каждый вызов NextID должен давать новый UUID")
}
