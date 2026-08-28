package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/google/uuid"
)

var ErrUnauthorized = errors.New("unauthorized: must be logged in to perform this action")

type AuditRepository interface {
	Query(ctx context.Context, patientID string, limit int, offset int) ([]*domain.AuditLogEntry, error)
	Log(ctx context.Context, entry *domain.AuditLogEntry) error
}

type AuditService struct {
	repo     AuditRepository
	mu       sync.RWMutex
	sessions map[string]*domain.Provider
}

func NewAuditService(repo AuditRepository) *AuditService {
	return &AuditService{
		repo:     repo,
		sessions: make(map[string]*domain.Provider),
	}
}

func (s *AuditService) CreateSession(provider *domain.Provider) string {
	if provider == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	token := uuid.NewString()
	s.sessions[token] = provider
	return token
}

func (s *AuditService) DestroySession(token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *AuditService) GetSessionUser(token string) *domain.Provider {
	if token == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[token]
}

func (s *AuditService) LogAction(token string, action domain.AuditAction, resource string, details string) error {
	user := s.GetSessionUser(token)
	if user == nil {
		return ErrUnauthorized
	}

	entry := &domain.AuditLogEntry{
		ID:        uuid.NewString(),
		Timestamp: time.Now().UTC(),
		UserID:    user.ID,
		UserName:  user.Name,
		Action:    action,
		Resource:  resource,
		Details:   details,
	}

	return s.repo.Log(context.Background(), entry)
}

func (s *AuditService) LogPatientAction(token string, action domain.AuditAction, patientID string, resource string, details string) error {
	user := s.GetSessionUser(token)
	if user == nil {
		return ErrUnauthorized
	}

	entry := &domain.AuditLogEntry{
		ID:        uuid.NewString(),
		Timestamp: time.Now().UTC(),
		UserID:    user.ID,
		UserName:  user.Name,
		PatientID: patientID,
		Action:    action,
		Resource:  resource,
		Details:   details,
	}

	return s.repo.Log(context.Background(), entry)
}

func (s *AuditService) GetAuditLogs(patientID string, limit int, offset int) ([]*domain.AuditLogEntry, error) {
	return s.repo.Query(context.Background(), patientID, limit, offset)
}

func (s *AuditService) LogEvent(entry *domain.AuditLogEntry) error {
	return s.repo.Log(context.Background(), entry)
}
