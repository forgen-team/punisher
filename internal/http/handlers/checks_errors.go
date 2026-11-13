package handlers

import (
	"errors"
	"net/http"

	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/services"
)

// ErrorLectureNotReviewed: '확인하기' on a lecture without a pass review (queued for the backfill review).
const ErrorLectureNotReviewed ErrorCode = "LECTURE_NOT_REVIEWED"

func writeCheckError(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidCheck):
		writeError(w, r, http.StatusBadRequest, checkErrorCode(err), err.Error())
	case errors.Is(err, services.ErrLectureNotReviewed):
		writeError(w, r, http.StatusConflict, ErrorLectureNotReviewed, err.Error())
	case errors.Is(err, services.ErrCheckBusy), errors.Is(err, services.ErrCheckState):
		writeError(w, r, http.StatusConflict, ErrorConflict, err.Error())
	default:
		writeServiceError(w, r, op, err)
	}
}

// checkErrorCode is the cause's code of a bad check request (BAD_REQUEST when unknown).
func checkErrorCode(err error) ErrorCode {
	var ie *services.CheckInputError
	if errors.As(err, &ie) && ie.Code != "" {
		return ErrorCode(ie.Code)
	}
	return ErrorBadRequest
}

// writeCheck answers 202 while a tutor turn or the item generation runs (poll GET), else 200.
func writeCheck(w http.ResponseWriter, d models.CheckDetail) {
	status := http.StatusOK
	if d.Pending {
		status = http.StatusAccepted
	}
	writeJSON(w, status, d)
}
