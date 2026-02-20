package audit

import (
	"os"
	"sync"

	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/model"
	"go.uber.org/zap"
)

// Publisher рассылает события аудита зарегистрированным наблюдателям.
type Publisher interface {
	// Register добавляет наблюдателя, который будет получать будущие события.
	Register(AuditObserver)
	// Publish отправляет событие всем зарегистрированным наблюдателям.
	Publish(model.AuditEvent)
}

// AuditPublisher — Publisher по умолчанию, рассылающий события набору
// зарегистрированных экземпляров AuditObserver.
type AuditPublisher struct {
	mu        sync.RWMutex
	observers map[string]AuditObserver
	logger    *zap.Logger
}

// NewPublisher создаёт Publisher, сконфигурированный из cfg. Возвращает функцию
// очистки, которую следует вызвать при завершении. Если назначения аудита
// не настроены, возвращается DummyPublisher.
func NewPublisher(cfg *config.Config, logger *zap.Logger) (Publisher, func()) {
	noop := func() {}

	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		return DummyPublisher{}, noop
	}

	auditPublisher := &AuditPublisher{
		observers: make(map[string]AuditObserver),
		logger:    logger,
	}

	var auditFile *os.File

	if cfg.AuditFile != "" {
		var err error
		auditFile, err = os.OpenFile(cfg.AuditFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			logger.Error("Ошибка открытия файла аудита", zap.Error(err))
		} else {
			auditPublisher.Register(NewFileObserver(auditFile))
		}
	}

	if cfg.AuditURL != "" {
		auditPublisher.Register(NewHTTPObserver(cfg.AuditURL))
	}

	cleanup := func() {
		if auditFile != nil {
			if err := auditFile.Close(); err != nil {
				logger.Error("Ошибка закрытия файла аудита", zap.Error(err))
			}
		}
	}

	return auditPublisher, cleanup
}

// Register добавляет наблюдателя в издатель, используя его ID как ключ.
func (p *AuditPublisher) Register(o AuditObserver) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.observers[o.GetID()] = o
}

// Publish отправляет событие каждому зарегистрированному наблюдателю. Ошибки
// логируются, но не останавливают доставку остальным наблюдателям.
func (p *AuditPublisher) Publish(event model.AuditEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, obs := range p.observers {
		if err := obs.Notify(event); err != nil {
			p.logger.Error("audit notify failed", zap.String("observer", obs.GetID()), zap.Error(err))
		}
	}
}

// DummyPublisher — заглушка Publisher, используемая при отключённом аудите.
type DummyPublisher struct{}

func (DummyPublisher) Register(AuditObserver)   {}
func (DummyPublisher) Publish(model.AuditEvent) {}
