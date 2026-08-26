package services

import (
	"context"

	"github.com/LibreDental/libredental/internal/domain"
)

type AuditRepository interface {
	Query(ctx context.Context, patientID string, limit int, offset int) ([]*domain.AuditLogEntry, error)
	Log(ctx context.Context, entry *domain.AuditLogEntry) error
}

type AuditService struct {
	repo AuditRepository
}

func NewAuditService(repo AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) GetAuditLogs(patientID string, limit int, offset int) ([]*domain.AuditLogEntry, error) {
	return s.repo.Query(context.Background(), patientID, limit, offset)
}

func (s *AuditService) LogEvent(entry *domain.AuditLogEntry) error {
	return s.repo.Log(context.Background(), entry)
}
