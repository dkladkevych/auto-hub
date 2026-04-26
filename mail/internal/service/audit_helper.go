package service

import "auto-hub/mail/internal/models"

// actorAuditUserID returns nil for operator (actorID == 0) so that the audit
// log foreign key stays clean, otherwise it returns a pointer to the ID.
func actorAuditUserID(actorID int) *int {
	if actorID == 0 {
		return nil
	}
	return &actorID
}

// enrichAuditPayload marks the payload with actor_type=operator when the
// actor is the system operator account.
func enrichAuditPayload(actorID int, payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		payload = make(map[string]interface{})
	}
	if actorID == 0 {
		payload["actor_type"] = "operator"
	}
	return payload
}

// buildAuditLog is a convenience constructor that applies operator-aware
// defaults for ActorUserID and Payload.
func buildAuditLog(actorID int, action, entityType string, entityID *int, payload map[string]interface{}) *models.AuditLog {
	return &models.AuditLog{
		ActorUserID: actorAuditUserID(actorID),
		Action:      action,
		EntityType:  entityType,
		EntityID:    entityID,
		Payload:     enrichAuditPayload(actorID, payload),
	}
}
