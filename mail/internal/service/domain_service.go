package service

import (
	"context"
	"fmt"

	"auto-hub/mail/internal/models"
	"auto-hub/mail/internal/repo"
)

// DomainService manages DNS domains that can be used when creating users
// or mailboxes.
type DomainService struct {
	domainRepo  *repo.DomainRepo
	mailboxRepo *repo.MailboxRepo
	auditRepo   *repo.AuditRepo
}

// NewDomainService creates a DomainService with the required repositories.
func NewDomainService(domainRepo *repo.DomainRepo, mailboxRepo *repo.MailboxRepo, auditRepo *repo.AuditRepo) *DomainService {
	return &DomainService{domainRepo: domainRepo, mailboxRepo: mailboxRepo, auditRepo: auditRepo}
}

// Create inserts a new domain.  When makeDefault is true the default flag
// is first cleared from all other domains so that only one domain is the
// default at any time.
func (s *DomainService) Create(ctx context.Context, actorID int, domain string, makeDefault bool) (*models.Domain, error) {
	d := &models.Domain{
		Domain:    domain,
		IsDefault: makeDefault,
		IsActive:  true,
	}

	if makeDefault {
		_ = s.domainRepo.ClearDefault(ctx)
	}

	if err := s.domainRepo.Create(ctx, d); err != nil {
		return nil, err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "domain_created", "domain", &d.ID, map[string]interface{}{"domain": domain, "is_default": makeDefault}))

	return d, nil
}

// ListAll returns every domain including inactive ones.
func (s *DomainService) ListAll(ctx context.Context) ([]models.Domain, error) {
	return s.domainRepo.ListAll(ctx)
}

// ListActive returns only the domains that are currently active.
func (s *DomainService) ListActive(ctx context.Context) ([]models.Domain, error) {
	return s.domainRepo.ListActive(ctx)
}

// GetDefaultDomain returns the domain string of the current default domain.
// An error is returned when no default domain has been configured.
func (s *DomainService) GetDefaultDomain(ctx context.Context) (string, error) {
	d, err := s.domainRepo.GetDefault(ctx)
	if err != nil {
		return "", err
	}
	if d == nil {
		return "", fmt.Errorf("no default domain configured")
	}
	return d.Domain, nil
}

// SetDefault marks the given domain as the new default.
func (s *DomainService) SetDefault(ctx context.Context, actorID, id int) error {
	if err := s.domainRepo.ClearDefault(ctx); err != nil {
		return err
	}
	if err := s.domainRepo.SetDefault(ctx, id); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "domain_set_default", "domain", &id, map[string]interface{}{}))

	return nil
}

// Delete removes a domain.  If mailboxes still exist on the domain the
// operation is rejected to prevent orphaned addresses.
func (s *DomainService) Delete(ctx context.Context, actorID, id int) error {
	d, err := s.domainRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("domain not found")
	}
	count, err := s.mailboxRepo.CountByDomain(ctx, d.Domain)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("cannot delete domain with existing mailboxes")
	}

	if err := s.domainRepo.Delete(ctx, id); err != nil {
		return err
	}

	_ = s.auditRepo.Log(ctx, buildAuditLog(actorID, "domain_deleted", "domain", &id, map[string]interface{}{}))

	return nil
}
