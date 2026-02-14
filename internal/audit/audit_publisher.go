package audit

import (
	"os"

	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/model"
	"go.uber.org/zap"
)

type Publisher interface {
	Register(AuditObserver)
	Publish(model.AuditEvent)
}

type AuditPublisher struct {
	observers map[string]AuditObserver
	logger    *zap.Logger
}

func NewPublisher(cfg *config.Config, logger *zap.Logger) (Publisher, func()) {
	noop := func() {}

	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		return DummyPublisher{}, noop
	}

	auditPublisher := &AuditPublisher{
		logger: logger,
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

func (p *AuditPublisher) Register(o AuditObserver) {
	if p.observers == nil {
		p.observers = make(map[string]AuditObserver)
	}
	p.observers[o.GetID()] = o
}

func (p *AuditPublisher) Publish(event model.AuditEvent) {
	for _, obs := range p.observers {
		if err := obs.Notify(event); err != nil {
			p.logger.Error("audit notify failed", zap.String("observer", obs.GetID()), zap.Error(err))
		}
	}
}

type DummyPublisher struct{}

func (DummyPublisher) Register(AuditObserver)   {}
func (DummyPublisher) Publish(model.AuditEvent) {}
