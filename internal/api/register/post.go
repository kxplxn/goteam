package register

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/kxplxn/goteam/internal/api"
	"github.com/kxplxn/goteam/pkg/auth"
	"github.com/kxplxn/goteam/pkg/dbaccess"
	teamTable "github.com/kxplxn/goteam/pkg/dbaccess/team"
	userTable "github.com/kxplxn/goteam/pkg/dbaccess/user"
	pkgLog "github.com/kxplxn/goteam/pkg/log"
)

// POSTReq defines the body of POST register requests.
type POSTReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// POSTResp defines the body of POST register responses.
type POSTResp struct {
	Err            string           `json:"error,omitempty"`
	ValidationErrs ValidationErrors `json:"validationErrors,omitempty"`
}

// ValidationErrors defines the validation errors returned in POSTResp.
type ValidationErrors struct {
	Username []string `json:"username,omitempty"`
	Password []string `json:"password,omitempty"`
}

// Any checks whether there are any validation errors within the ValidationErrors.
func (e ValidationErrors) Any() bool {
	return len(e.Username) > 0 || len(e.Password) > 0
}

// POSTHandler is a http.POSTHandler that can be used to handle register requests.
type POSTHandler struct {
	userValidator       ReqValidator
	inviteCodeValidator api.StringValidator
	teamSelector        dbaccess.Selector[teamTable.Record]
	userSelector        dbaccess.Selector[userTable.Record]
	hasher              Hasher
	userInserter        dbaccess.Inserter[userTable.Record]
	authTokenGenerator  auth.TokenGenerator
	log                 pkgLog.Errorer
}

// NewPOSTHandler is the constructor for Handler.
func NewPOSTHandler(
	userValidator ReqValidator,
	inviteCodeValidator api.StringValidator,
	teamSelector dbaccess.Selector[teamTable.Record],
	userSelector dbaccess.Selector[userTable.Record],
	hasher Hasher,
	userInserter dbaccess.Inserter[userTable.Record],
	authTokenGenerator auth.TokenGenerator,
	log pkgLog.Errorer,
) POSTHandler {
	return POSTHandler{
		userValidator:       userValidator,
		inviteCodeValidator: inviteCodeValidator,
		teamSelector:        teamSelector,
		userSelector:        userSelector,
		hasher:              hasher,
		userInserter:        userInserter,
		authTokenGenerator:  authTokenGenerator,
		log:                 log,
	}
}

// ServeHTTP responds to requests made to the register route.
func (h POSTHandler) Handle(
	w http.ResponseWriter, r *http.Request, _ string,
) {
	// Read request body.
	reqBody := POSTReq{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Validate request body, write errors if any occur.
	if validationErrs := h.userValidator.Validate(
		reqBody,
	); validationErrs.Any() {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(
			POSTResp{ValidationErrs: validationErrs},
		); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
		}
		return
	}

	// Define a user record to progressively populate the fields of.
	var user userTable.Record

	// Validate invite code if found.
	inviteCode := r.URL.Query().Get("inviteCode")
	if inviteCode != "" {
		if err := h.inviteCodeValidator.Validate(inviteCode); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(
				POSTResp{Err: "Invalid invite code."},
			); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error(err.Error())
			}
			return
		}
		team, err := h.teamSelector.Select(inviteCode)
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(
				POSTResp{Err: "Team not found."},
			); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				h.log.Error(err.Error())
			}
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(err.Error())
			return
		}
		user.TeamID = team.ID
		user.IsAdmin = false
	} else {
		// This indicates that the user is non-invited and a new team will be
		// created for them, which they will be the admin of.
		user.TeamID = -1
		user.IsAdmin = true
	}

	// Check whether the username is taken.
	_, err := h.userSelector.Select(reqBody.Username)
	if err == nil {
		w.WriteHeader(http.StatusBadRequest)
		if errEncode := json.NewEncoder(w).Encode(
			POSTResp{ValidationErrs: ValidationErrors{
				Username: []string{"Username is already taken."},
			}},
		); errEncode != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.log.Error(errEncode.Error())
		}
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		h.log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user.Username = reqBody.Username

	// Hash password and create user.
	pwdHash, err := h.hasher.Hash(reqBody.Password)
	if err != nil {
		h.log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user.Password = pwdHash

	if err = h.userInserter.Insert(user); err != nil {
		h.log.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Generate an authentication token for the user that is valid for an hour
	// and return it within a Set-Cookie header.
	expiry := time.Now().Add(auth.Duration).UTC()
	if authToken, err := h.authTokenGenerator.Generate(
		reqBody.Username, expiry,
	); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(
			POSTResp{
				Err: "You have been registered successfully but something " +
					"went wrong. Please log in using the credentials you " +
					"registered with.",
			},
		); err != nil {
			h.log.Error(err.Error())
		}
	} else {
		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    authToken,
			Expires:  expiry,
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		})
		w.WriteHeader(http.StatusOK)
	}
}