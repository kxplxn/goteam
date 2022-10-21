package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kxplxn/goteam/server/v2/relay"
)

// HandlerRegister is a HTTP handler for the "/register" endpoint.
type HandlerRegister struct {
	log relay.ErrMsger
}

// NewHandlerRegister is the constructor for HandlerRegister.
func NewHandlerRegister(log relay.ErrMsger) *HandlerRegister {
	return &HandlerRegister{log: log}
}

// ServeHTTP responds to requests made to the "/register" endpoint.
func (h *HandlerRegister) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// accept only POST
	if r.Method != "POST" {
		status := http.StatusMethodNotAllowed
		h.log.Err(w, http.StatusText(status), status)
		return
	}

	// read body into map
	dec := make(map[string]string, 3)
	if err := json.NewDecoder(r.Body).Decode(&dec); err != nil {
		status := http.StatusInternalServerError
		h.log.Err(w, http.StatusText(status), status)
	}

	// rly decoded body
	h.log.Msg(w, fmt.Sprintf(
		"usn: %s\npwd: %s\nref: %s\n",
		dec["usn"], dec["pwd"], dec["ref"],
	))
}
