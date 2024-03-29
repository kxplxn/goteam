package boardapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kxplxn/goteam/pkg/cookie"
	"github.com/kxplxn/goteam/pkg/db"
	"github.com/kxplxn/goteam/pkg/db/teamtbl"
	"github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/validator"
)

// PatchReq defines the body of PATCH board requests.
type PatchReq teamtbl.Board

// PatchResp defines the body of PATCH board responses.
type PatchResp struct {
	Error string `json:"error,omitempty"`
}

// PatchHandler can be used to handle PATCH board requests.
type PatchHandler struct {
	authDecoder   cookie.Decoder[cookie.Auth]
	idValidator   validator.String
	nameValidator validator.String
	boardUpdater  db.UpdaterDualKey[teamtbl.Board]
	log           log.Errorer
}

// DeleteHandler is an api.MethodHandler that can be used to handle DELETE board
// requests.
func NewPatchHandler(
	authDecoder cookie.Decoder[cookie.Auth],
	idValidator validator.String,
	nameValidator validator.String,
	boardUpdater db.UpdaterDualKey[teamtbl.Board],
	log log.Errorer,
) *PatchHandler {
	return &PatchHandler{
		authDecoder:   authDecoder,
		idValidator:   idValidator,
		nameValidator: nameValidator,
		boardUpdater:  boardUpdater,
		log:           log,
	}
}

// Handle handles PATCH board requests.
func (h *PatchHandler) Handle(
	w http.ResponseWriter, r *http.Request, username string,
) {
	// get auth token
	ckAuth, err := r.Cookie(cookie.AuthName)
	if err == http.ErrNoCookie {
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(
			PatchResp{Error: "Auth token not found."},
		); err != nil {
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
		if err := json.NewEncoder(w).Encode(
			PatchResp{Error: "Invalid auth token."},
		); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	}

	// validate user is admin
	if !auth.IsAdmin {
		w.WriteHeader(http.StatusForbidden)
		if err = json.NewEncoder(w).Encode(PatchResp{
			Error: "Only team admins can edit boards.",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	}

	// decode board
	var req PatchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err)
		return
	}

	// validate board ID
	if err := h.idValidator.Validate(req.ID); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		var msg string
		if errors.Is(err, validator.ErrEmpty) {
			msg = "Board ID cannot be empty."
		} else if errors.Is(err, validator.ErrWrongFormat) {
			msg = "Board ID must be a UUID."
		}

		if err = json.NewEncoder(w).Encode(PatchResp{Error: msg}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	}
	if err := h.nameValidator.Validate(req.Name); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		var msg string
		if errors.Is(err, validator.ErrEmpty) {
			msg = "Board name cannot be empty."
		} else if errors.Is(err, validator.ErrTooLong) {
			msg = "Board name cannot be longer than 35 characters."
		}

		if err := json.NewEncoder(w).Encode(
			PatchResp{Error: msg},
		); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	}

	// update the board for the team
	if err := h.boardUpdater.Update(
		r.Context(), auth.TeamID, teamtbl.Board(req),
	); errors.Is(err, db.ErrNoItem) {
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(
			PatchResp{Error: "Board not found."},
		); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err)
		}
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.log.Error(err)
		return
	}
}
