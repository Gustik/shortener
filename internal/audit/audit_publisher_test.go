package audit

import (
	"testing"

	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockObserver struct {
	id     string
	events []model.AuditEvent
}

func (m *mockObserver) Notify(event model.AuditEvent) error {
	m.events = append(m.events, event)
	return nil
}

func (m *mockObserver) GetID() string {
	return m.id
}

func TestAuditPublisher_Register_And_Publish(t *testing.T) {
	pub := &AuditPublisher{logger: zap.NewNop()}

	obs := &mockObserver{id: "mock"}
	pub.Register(obs)

	event := testEvent()
	pub.Publish(event)

	require.Len(t, obs.events, 1)
	assert.Equal(t, event, obs.events[0])
}

func TestAuditPublisher_Publish_MultipleObservers(t *testing.T) {
	pub := &AuditPublisher{logger: zap.NewNop()}

	obs1 := &mockObserver{id: "obs1"}
	obs2 := &mockObserver{id: "obs2"}
	pub.Register(obs1)
	pub.Register(obs2)

	event := testEvent()
	pub.Publish(event)

	require.Len(t, obs1.events, 1)
	require.Len(t, obs2.events, 1)
	assert.Equal(t, event, obs1.events[0])
	assert.Equal(t, event, obs2.events[0])
}

func TestAuditPublisher_Register_SameID_Replaces(t *testing.T) {
	pub := &AuditPublisher{logger: zap.NewNop()}

	obs1 := &mockObserver{id: "same"}
	obs2 := &mockObserver{id: "same"}
	pub.Register(obs1)
	pub.Register(obs2)

	pub.Publish(testEvent())

	assert.Empty(t, obs1.events)
	require.Len(t, obs2.events, 1)
}

func TestNewPublisher_NoConfig_ReturnsDummy(t *testing.T) {
	cfg := &config.Config{}
	pub, cleanup := NewPublisher(cfg, zap.NewNop())
	defer cleanup()

	_, ok := pub.(DummyPublisher)
	assert.True(t, ok)
}

func TestNewPublisher_WithAuditFile(t *testing.T) {
	tmpFile := t.TempDir() + "/audit.log"
	cfg := &config.Config{AuditFile: tmpFile}

	pub, cleanup := NewPublisher(cfg, zap.NewNop())
	defer cleanup()

	_, ok := pub.(*AuditPublisher)
	assert.True(t, ok)

	pub.Publish(testEvent())
}

func TestNewPublisher_WithAuditURL(t *testing.T) {
	cfg := &config.Config{AuditURL: "http://localhost:9999"}

	pub, cleanup := NewPublisher(cfg, zap.NewNop())
	defer cleanup()

	_, ok := pub.(*AuditPublisher)
	assert.True(t, ok)
}
