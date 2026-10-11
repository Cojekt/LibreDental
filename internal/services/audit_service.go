package services

import (
	"context"
	"errors"
	"fmt"
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

	configService := NewPracticeConfigService(s.providerRepo, nil)
	provider, err := configService.VerifyProviderPin(id, pin)
	if err != nil {
		return "", err
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
	return s.logPatientActionAs(auditActor{id: user.ID, name: user.Name}, action, patientID, resource, "", details)
}

// auditActor is who an audit entry is attributed to: a logged-in staff member, or a system
// actor (see domain.SystemActorPrefix).
type auditActor struct {
	id   string
	name string
}

func (s *AuditService) logPatientActionAs(actor auditActor, action domain.AuditAction, patientID, resource, resourceID, details string) error {
	entry := &domain.AuditLogEntry{
		ID:         uuid.NewString(),
		Timestamp:  time.Now().UTC(),
		UserID:     actor.id,
		UserName:   actor.name,
		PatientID:  patientID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
	}

	return s.repo.Log(context.Background(), entry)
}

// logSystemPatientAction records an action LibreDental took on its own, with no staff member
// logged in. It is unexported so Wails does not bind it: in LAN server mode any client could
// otherwise write audit entries attributed to the system.
func (s *AuditService) logSystemPatientAction(actorID string, action domain.AuditAction, patientID, resource, resourceID, details string) error {
	if !domain.IsSystemActorID(actorID) {
		return fmt.Errorf("%w: %q is not a system actor ID", storage.ErrInvalidInput, actorID)
	}
	return s.logPatientActionAs(auditActor{id: actorID, name: domain.SystemActorName}, action, patientID, resource, resourceID, details)
}

func (s *AuditService) GetAuditLogs(token string, patientID string, limit int, offset int) ([]*domain.AuditLogEntry, error) {
	if s.GetSessionUser(token) == nil {
		return nil, ErrUnauthorized
	}
	entries, err := s.repo.Query(context.Background(), patientID, limit, offset)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		e.SystemActor = domain.IsSystemActorID(e.UserID)
	}
	return entries, nil
}
