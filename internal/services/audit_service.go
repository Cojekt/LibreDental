package services

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/google/uuid"
)

var ErrUnauthorized = errors.New("unauthorized: must be logged in to perform this action")

type AuditRepository interface {
	Query(ctx context.Context, patientID string, limit int, offset int) ([]*domain.AuditLogEntry, error)
	Log(ctx context.Context, entry *domain.AuditLogEntry) error
}

type AuditService struct {
	repo         AuditRepository
	providerRepo storage.PracticeConfigRepository
	mu           sync.RWMutex
	sessions     map[string]*domain.Provider
}

func NewAuditService(repo AuditRepository, providerRepo storage.PracticeConfigRepository) *AuditService {
	return &AuditService{
		repo:         repo,
		providerRepo: providerRepo,
		sessions:     make(map[string]*domain.Provider),
	}
}

func (s *AuditService) CreateSession(id string, pin string) (string, error) {
	if id == "" || pin == "" {
		return "", errors.New("id and pin are required")
	}
	if s.providerRepo == nil {
		return "", errors.New("provider repository not configured")
	}

	providers, err := s.providerRepo.ListProviders(context.Background())
	if err != nil {
		return "", err
	}

	var provider *domain.Provider
	for _, p := range providers {
		if p.ID == id {
			if !p.IsActive {
				return "", errors.New("provider is inactive")
			}
			if p.Pin == "" || pin == "" {
				return "", errors.New("pin not set or empty")
			}
			if p.Pin != pin {
				return "", errors.New("incorrect pin")
			}

			copiedProvider := *p
			copiedProvider.Pin = "****"
			provider = &copiedProvider
			break
		}
	}

	if provider == nil {
		return "", errors.New("provider not found")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	token := uuid.NewString()
	s.sessions[token] = provider
	return token, nil
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
	provider := s.sessions[token]
	s.mu.RUnlock()

	if provider == nil {
		return nil
	}

	if s.providerRepo != nil {
		providers, err := s.providerRepo.ListProviders(context.Background())
		if err == nil {
			foundAndActive := false
			for _, p := range providers {
				if p.ID == provider.ID && p.IsActive {
					foundAndActive = true
					break
				}
			}
			if !foundAndActive {
				s.DestroySession(token)
				return nil
			}
		}
	}

	return provider
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
