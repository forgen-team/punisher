package handlers

import (
	"errors"
	"net/http"

	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/services"
)

// ErrorLectureNotReviewed: '확인하기' on a lecture without a pass review (queued for the backfill review).
const ErrorLectureNotReviewed ErrorCode = "LECTURE_NOT_REVIEWED"

// 409 codes the web translates (the message stays the server's Korean text for older clients).
const (
	ErrorCheckBusy  ErrorCode = "CHECK_BUSY"  // a tutor turn is being graded
	ErrorCheckState ErrorCode = "CHECK_STATE" // the action does not fit the session state
)

func writeCheckError(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidCheck):
		writeError(w, r, http.StatusBadRequest, checkErrorCode(err), err.Error())
	case errors.Is(err, services.ErrLectureNotReviewed):
		writeError(w, r, http.StatusConflict, ErrorLectureNotReviewed, err.Error())
	case errors.Is(err, services.ErrCheckBusy):
		writeError(w, r, http.StatusConflict, ErrorCheckBusy, err.Error())
	case errors.Is(err, services.ErrCheckState):
		writeError(w, r, http.StatusConflict, ErrorCheckState, err.Error())
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
