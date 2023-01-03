package register

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"server/assert"
	"server/db"
)

func TestHandler(t *testing.T) {
	// handler setup
	var (
		validatorReq   = &fakeValidator{}
		existorUser    = &db.FakeReaderUser{}
		hasherPwd      = &fakeHasherPwd{}
		creatorUser    = &db.FakeCreatorUser{}
		creatorSession = &db.FakeCreatorSession{}
	)
	sut := NewHandler(validatorReq, existorUser, hasherPwd, creatorUser, creatorSession)

	for _, c := range []struct {
		name                 string
		httpMethod           string
		reqBody              *ReqBody
		outErrValidatorReq   *Errs
		outResReaderUser     *db.User
		outErrReaderUser     error
		outResHasherPwd      []byte
		outErrHasherPwd      error
		outErrCreatorUser    error
		outErrCreatorSession error
		wantStatusCode       int
		wantFieldErrs        *Errs
	}{
		{
			name:                 "ErrHttpMethod",
			httpMethod:           http.MethodGet,
			reqBody:              &ReqBody{Username: "bob2121", Password: "Myp4ssword!"},
			outErrValidatorReq:   nil,
			outResReaderUser:     nil,
			outErrReaderUser:     nil,
			outResHasherPwd:      nil,
			outErrHasherPwd:      nil,
			outErrCreatorUser:    nil,
			outErrCreatorSession: nil,
			wantStatusCode:       http.StatusMethodNotAllowed,
			wantFieldErrs:        nil,
		},
		{
			name:                 "ErrValidator",
			httpMethod:           http.MethodPost,
			reqBody:              &ReqBody{Username: "bobobobobobobobob", Password: "myNOdigitPASSWORD!"},
			outErrValidatorReq:   &Errs{Username: []string{usnTooLong}, Password: []string{pwdNoDigit}},
			outResReaderUser:     nil,
			outErrReaderUser:     nil,
			outResHasherPwd:      nil,
			outErrHasherPwd:      nil,
			outErrCreatorUser:    nil,
			outErrCreatorSession: nil,
			wantStatusCode:       http.StatusBadRequest,
			wantFieldErrs:        &Errs{Username: []string{usnTooLong}, Password: []string{pwdNoDigit}},
		},
		{
			name:                 "ResExistorTrue",
			httpMethod:           http.MethodPost,
			reqBody:              &ReqBody{Username: "bob21", Password: "Myp4ssword!"},
			outErrValidatorReq:   nil,
			outResReaderUser:     nil,
			outErrReaderUser:     nil,
			outResHasherPwd:      nil,
			outErrHasherPwd:      nil,
			outErrCreatorUser:    nil,
			outErrCreatorSession: nil,
			wantStatusCode:       http.StatusBadRequest,
			wantFieldErrs:        &Errs{Username: []string{errFieldUsernameTaken}},
		},
		{
			name:                 "ErrExistor",
			httpMethod:           http.MethodPost,
			reqBody:              &ReqBody{Username: "bob2121", Password: "Myp4ssword!"},
			outErrValidatorReq:   nil,
			outResReaderUser:     nil,
			outErrReaderUser:     errors.New("existor fatal error"),
			outResHasherPwd:      nil,
			outErrHasherPwd:      nil,
			outErrCreatorUser:    nil,
			outErrCreatorSession: nil,
			wantStatusCode:       http.StatusInternalServerError,
			wantFieldErrs:        nil,
		},
		{
			name:                 "ErrHasher",
			httpMethod:           http.MethodPost,
			reqBody:              &ReqBody{Username: "bob2121", Password: "Myp4ssword!"},
			outErrValidatorReq:   nil,
			outResReaderUser:     nil,
			outErrReaderUser:     sql.ErrNoRows,
			outResHasherPwd:      nil,
			outErrHasherPwd:      errors.New("hasher fatal error"),
			outErrCreatorUser:    nil,
			outErrCreatorSession: nil,
			wantStatusCode:       http.StatusInternalServerError,
			wantFieldErrs:        nil,
		},
		{
			name:                 "ErrCreatorUser",
			httpMethod:           http.MethodPost,
			reqBody:              &ReqBody{Username: "bob2121", Password: "Myp4ssword!"},
			outErrValidatorReq:   nil,
			outResReaderUser:     nil,
			outErrReaderUser:     sql.ErrNoRows,
			outResHasherPwd:      nil,
			outErrHasherPwd:      nil,
			outErrCreatorUser:    errors.New("creator fatal error"),
			outErrCreatorSession: nil,
			wantStatusCode:       http.StatusInternalServerError,
			wantFieldErrs:        nil,
		},
		{
			name:                 "ErrCreatorSession",
			httpMethod:           http.MethodPost,
			reqBody:              &ReqBody{Username: "bob2121", Password: "Myp4ssword!"},
			outErrValidatorReq:   nil,
			outResReaderUser:     nil,
			outErrReaderUser:     sql.ErrNoRows,
			outResHasherPwd:      nil,
			outErrHasherPwd:      nil,
			outErrCreatorUser:    nil,
			outErrCreatorSession: errors.New("session creator error"),
			wantStatusCode:       http.StatusUnauthorized,
			wantFieldErrs:        &Errs{Session: errSession},
		},
		{
			name:                 "ResHandlerOK",
			httpMethod:           http.MethodPost,
			reqBody:              &ReqBody{Username: "bob2121", Password: "Myp4ssword!"},
			outErrValidatorReq:   nil,
			outResReaderUser:     nil,
			outErrReaderUser:     sql.ErrNoRows,
			outResHasherPwd:      nil,
			outErrHasherPwd:      nil,
			outErrCreatorUser:    nil,
			outErrCreatorSession: nil,
			wantStatusCode:       http.StatusOK,
			wantFieldErrs:        nil,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			// Set pre-determinate return values for Handler dependencies.
			validatorReq.outErrs = c.outErrValidatorReq
			existorUser.OutRes = c.outResReaderUser
			existorUser.OutErr = c.outErrReaderUser
			hasherPwd.outHash = c.outResHasherPwd
			hasherPwd.outErr = c.outErrHasherPwd
			creatorUser.OutErr = c.outErrCreatorUser
			creatorSession.OutErr = c.outErrCreatorSession

			// Parse request body.
			reqBody, err := json.Marshal(c.reqBody)
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest(c.httpMethod, "/register", bytes.NewReader(reqBody))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Send request (act).
			sut.ServeHTTP(w, req)

			// Input-based assertions to be run up onto the point where handler
			// stops execution. Conditionals serve to determine which
			// dependencies should have received their function arguments.
			if c.httpMethod == http.MethodPost {
				assert.Equal(t, c.reqBody.Username, validatorReq.inReqBody.Username)
				assert.Equal(t, c.reqBody.Password, validatorReq.inReqBody.Password)
				if c.outErrValidatorReq == nil {
					// validator.Validate doesn't error – readerUser.Exists is called.
					assert.Equal(t, c.reqBody.Username, existorUser.InArg)
					if c.outErrReaderUser == sql.ErrNoRows {
						// readerUser.Exists returns sql.ErrNoRows - hasher.Hash is called.
						assert.Equal(t, c.reqBody.Password, hasherPwd.inPlaintext)
						if c.outErrHasherPwd == nil {
							// hasher.Hash doesn't error – creatorUser.Create is called.
							assert.Equal(t, c.reqBody.Username, creatorUser.InArg.Username)
							assert.Equal(t, string(c.outResHasherPwd), string(creatorUser.InArg.Password))
							if c.outErrCreatorUser == nil {
								// creatorUser.Create doesn't error – creatorSession.Create is called.
								assert.Equal(t, c.reqBody.Username, creatorSession.InArg.Username)
							}
						}
					}
				}
			}

			// Assert on status code.
			res := w.Result()
			assert.Equal(t, c.wantStatusCode, res.StatusCode)

			// Assert on response body – however, there are some cases such as
			// internal server errors where an empty res body is returned and
			// these assertions are not run.
			if c.httpMethod != http.MethodPost ||
				c.outErrReaderUser != nil ||
				c.outErrHasherPwd != nil ||
				c.outErrCreatorUser != nil ||
				c.wantStatusCode == http.StatusOK {
				return
			}

			resBody := &ResBody{}
			if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
				t.Fatal(err)
			}

			if c.wantFieldErrs != nil {
				// field errors - assert on them
				assert.EqualArr(t, c.wantFieldErrs.Username, resBody.Errs.Username)
				assert.EqualArr(t, c.wantFieldErrs.Password, resBody.Errs.Password)
				assert.Equal(t, c.wantFieldErrs.Session, resBody.Errs.Session)
			} else {
				// no field errors - assert on session token
				foundSessionToken := false
				for _, cookie := range res.Cookies() {
					if cookie.Name == "sessionToken" {
						foundSessionToken = true
					}
				}
				assert.Equal(t, true, foundSessionToken)
			}
		})
	}
}