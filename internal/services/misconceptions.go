package services

import (
	"context"
	"strings"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// Codes of the misconception API's 400 responses.
const (
	CodeTurnRequired  = "TURN_REQUIRED"
	CodeInvalidStatus = "INVALID_STATUS"
)

// maxMisconceptionList bounds GET /v1/misconceptions.
const maxMisconceptionList = 100

// AcknowledgeMisconception is '맞아요, 그렇게 알았어요' on a check result: the
// server stores the turn's misconception with its objective and concept (the
// learner types nothing). created=false when it was already acknowledged.
func (s *Service) AcknowledgeMisconception(ctx context.Context, userID, turnID string) (models.Misconception, bool, error) {
	turnID = strings.TrimSpace(turnID)
	if turnID == "" {
		return models.Misconception{}, false, invalidCheck(CodeTurnRequired, "turnId is required")
	}
	if !validID(turnID) {
		return models.Misconception{}, false, store.ErrNotFound
	}
	return s.Store.AcknowledgeMisconception(ctx, userID, turnID)
}

// ListMisconceptions returns the user's acknowledged misconceptions of a status
// (open | resolved; "" = open), latest first.
func (s *Service) ListMisconceptions(ctx context.Context, userID, status string) ([]models.Misconception, error) {
	if status == "" {
		status = learning.MisconceptionOpen
	}
	if !learning.ValidMisconceptionStatus(status) {
		return nil, invalidCheck(CodeInvalidStatus, "status must be open|resolved")
	}
	return s.Store.ListMisconceptions(ctx, userID, status, maxMisconceptionList)
}
