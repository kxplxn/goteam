package login

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"server/db"
	"server/relay"

	"github.com/google/uuid"
)

// Handler is the HTTP handler for the login route.
type Handler struct {
	readerUser      db.Reader[*db.User]
	comparerHash    Comparer
	upserterSession db.Upserter[*db.Session]
}

// NewHandler is the constructor for Handler.
func NewHandler(
	readerUser db.Reader[*db.User],
	comparerHash Comparer,
	upserterSession db.Upserter[*db.Session],
) *Handler {
	return &Handler{
		readerUser:      readerUser,
		comparerHash:    comparerHash,
		upserterSession: upserterSession,
	}
}

// ServeHTTP responds to requests made to the login route. Unlike the register
// handler where we tell the user exactly what's wrong with their credentials,
// we instead just want to return a 400 Bad Request, which the client should
// use to display a boilerplate "Invalid credentials." error.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	reqBody := &ReqBody{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		relay.ServerErr(w, err.Error())
		return
	}

	if reqBody.Username == "" || reqBody.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := h.readerUser.Read(reqBody.Username)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if err != nil {
		relay.ServerErr(w, err.Error())
		return
	}

	isMatch, err := h.comparerHash.Compare(user.Password, reqBody.Password)
	if err != nil {
		relay.ServerErr(w, err.Error())
		return
	}
	if !isMatch {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	session := db.NewSession(
		uuid.NewString(), reqBody.Username, time.Now().Add(1*time.Hour),
	)
	if err = h.upserterSession.Upsert(session); err != nil {
		relay.ServerErr(w, err.Error())
		return
	} else {
		// Register succes, session creator success, all good...
		http.SetCookie(w, &http.Cookie{
			Name:    "sessionToken",
			Value:   session.ID,
			Expires: session.Expiry,
		})
		w.WriteHeader(http.StatusOK)
		return
	}
}
