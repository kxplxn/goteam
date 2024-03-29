package taskapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kxplxn/goteam/pkg/cookie"
	"github.com/kxplxn/goteam/pkg/db"
	"github.com/kxplxn/goteam/pkg/db/tasktbl"
	"github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/validator"
)

// PatchReq defines the body of PATCH task requests.
type PatchReq tasktbl.Task

// PatchResp defines the body of PATCH task responses.
type PatchResp struct {
	Error string `json:"error"`
}

// PatchHandler is an api.MethodHandler that can handle PATCH requests sent to
// the task route.
type PatchHandler struct {
	authDecoder        cookie.Decoder[cookie.Auth]
	titleValidator     validator.String
	subtTitleValidator validator.String
	taskUpdater        db.Updater[tasktbl.Task]
	log                log.Errorer
}

// NewPatchHandler returns a new PatchHandler.
func NewPatchHandler(
	authDecoder cookie.Decoder[cookie.Auth],
	taskTitleValidator validator.String,
	subtaskTitleValidator validator.String,
	taskUpdater db.Updater[tasktbl.Task],
	log log.Errorer,
) *PatchHandler {
	return &PatchHandler{
		authDecoder:        authDecoder,
		titleValidator:     taskTitleValidator,
		subtTitleValidator: subtaskTitleValidator,
		taskUpdater:        taskUpdater,
		log:                log,
	}
}

// Handle handles PATCH requests sent to the task route.
func (h *PatchHandler) Handle(
	w http.ResponseWriter, r *http.Request, username string,
) {
	// get auth token
	ckAuth, err := r.Cookie(cookie.AuthName)
	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusUnauthorized)
		if encodeErr := json.NewEncoder(w).Encode(PatchResp{
			Error: "Auth token not found.",
		}); encodeErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err)
		return
	}

	// decode auth token
	auth, err := h.authDecoder.Decode(*ckAuth)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		if err = json.NewEncoder(w).Encode(PatchResp{
			Error: "Invalid auth token.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	}

	// validate user is admin
	if !auth.IsAdmin {
		w.WriteHeader(http.StatusForbidden)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: "Only team admins can edit tasks.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	}

	// read request body
	var req PatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err)
		return
	}

	// validate task title
	if err := h.titleValidator.Validate(req.Title); err != nil {
		var errMsg string
		if errors.Is(err, validator.ErrEmpty) {
			errMsg = "Task title cannot be empty."
		} else if errors.Is(err, validator.ErrTooLong) {
			errMsg = "Task title cannot be longer than 50 characters."
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: errMsg,
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	}

	// validate subtask titles
	for _, subtask := range req.Subtasks {
		if err := h.subtTitleValidator.Validate(subtask.Title); err != nil {
			var errMsg string
			if errors.Is(err, validator.ErrEmpty) {
				errMsg = "Subtask title cannot be empty."
			} else if errors.Is(err, validator.ErrTooLong) {
				errMsg = "Subtask title cannot be longer than 50 characters."
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error(err)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(PatchResp{
				Error: errMsg,
			}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error(err)
			}
			return
		}
	}

	// update task in task table
	task := tasktbl.Task(req)
	task.TeamID = auth.TeamID
	err = h.taskUpdater.Update(r.Context(), task)
	if errors.Is(err, db.ErrNoItem) {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(PatchResp{
			Error: "Task not found.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err)
		return
	}

	// no need to update state token as it does not store any of the updated
	// fields and the frontend will have updated its own state already
}
