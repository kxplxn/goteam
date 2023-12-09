//go:build utest

package task

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kxplxn/goteam/internal/api"
	"github.com/kxplxn/goteam/pkg/assert"
	"github.com/kxplxn/goteam/pkg/db"
	taskTable "github.com/kxplxn/goteam/pkg/db/task"
	pkgLog "github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/token"
)

// TestPOSTHandler tests the Handle method of POSTHandler to assert that it
// behaves correctly in all possible scenarios.
func TestPOSTHandler(t *testing.T) {
	decodeAuth := &token.FakeDecode[token.Auth]{}
	decodeState := &token.FakeDecode[token.State]{}
	titleVdtor := &api.FakeStringValidator{}
	subtTitleVdtor := &api.FakeStringValidator{}
	colNoVdtor := &api.FakeIntValidator{}
	taskInserter := &db.FakeInserter[taskTable.Task]{}
	encodeState := &token.FakeEncode[token.State]{}
	log := &pkgLog.FakeErrorer{}
	sut := NewPostHandler(
		decodeAuth.Func,
		decodeState.Func,
		titleVdtor,
		subtTitleVdtor,
		colNoVdtor,
		taskInserter,
		encodeState.Func,
		log,
	)

	for _, c := range []struct {
		name                 string
		reqBody              string
		authToken            string
		authDecoded          token.Auth
		errDecodeAuth        error
		inStateToken         string
		inStateDecoded       token.State
		errDecodeInState     error
		errValidateColNo     error
		errValidateTaskTitle error
		errValidateSubtTitle error
		errInsertTask        error
		outStateToken        token.State
		outStateEncoded      string
		errEncodeState       error
		wantStatus           int
		assertFunc           func(*testing.T, *http.Response, string)
	}{
		{
			name:       "NoAuth",
			wantStatus: http.StatusUnauthorized,
			assertFunc: assert.OnResErr("Auth token not found."),
		},
		{
			name:                 "InvalidAuth",
			authToken:            "nonempty",
			errDecodeAuth:        token.ErrInvalid,
			inStateToken:         "",
			inStateDecoded:       token.State{},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusUnauthorized,
			assertFunc:           assert.OnResErr("Invalid auth token."),
		},
		{
			name:                 "NotAdmin",
			reqBody:              "",
			authToken:            "nonempty",
			authDecoded:          token.Auth{},
			errDecodeAuth:        nil,
			inStateToken:         "",
			inStateDecoded:       token.State{},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusForbidden,
			assertFunc: assert.OnResErr(
				"Only team admins can create tasks.",
			),
		},
		{
			name:                 "NoState",
			reqBody:              "",
			authToken:            "nonempty",
			authDecoded:          token.Auth{IsAdmin: true},
			errDecodeAuth:        nil,
			inStateToken:         "",
			inStateDecoded:       token.State{},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusBadRequest,
			assertFunc:           assert.OnResErr("State token not found."),
		},
		{
			name:                 "InvalidState",
			reqBody:              "",
			authToken:            "nonempty",
			authDecoded:          token.Auth{IsAdmin: true},
			errDecodeAuth:        nil,
			inStateToken:         "nonempty",
			inStateDecoded:       token.State{},
			errDecodeInState:     token.ErrInvalid,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusBadRequest,
			assertFunc:           assert.OnResErr("Invalid state token."),
		},
		{
			name:          "ColNoOutOfBounds",
			reqBody:       `{"board": "boardid"}`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{
				Boards: []token.Board{{ID: "boardid"}},
			},
			errDecodeInState:     nil,
			errValidateColNo:     api.ErrOutOfBounds,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Column number out of bounds.",
			),
		},
		{
			name:          "NoBoardAccess",
			reqBody:       `{"board": "boardid"}`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{
				Boards: []token.Board{{ID: "foo"}},
			},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusForbidden,
			assertFunc: assert.OnResErr(
				"You do not have access to this board.",
			),
		},
		{
			name:          "TaskTitleEmpty",
			reqBody:       `{"board": "boardid", "column": 0}`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: api.ErrEmpty,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Task title cannot be empty.",
			),
		},
		{
			name:          "TaskTitleTooLong",
			reqBody:       `{"board": "boardid", "column": 0}`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: api.ErrTooLong,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Task title cannot be longer than 50 characters.",
			),
		},
		{
			name:          "TaskTitleUnexpectedErr",
			reqBody:       `{"board": "boardid", "column": 0}`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: api.ErrNotInt,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				api.ErrNotInt.Error(),
			),
		},
		{
			name: "SubtTitleEmpty",
			reqBody: `{
                "board": "boardid", "column": 0, "subtasks": ["foo"]
            }`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: api.ErrEmpty,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Subtask title cannot be empty.",
			),
		},
		{
			name: "SubtTitleTooLong",
			reqBody: `{
                "board": "boardid", "column": 0, "subtasks": ["foo"]
            }`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: api.ErrTooLong,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Subtask title cannot be longer than 50 characters.",
			),
		},
		{
			name: "ValidateSubtTitleErr",
			reqBody: `{
                "board": "boardid", "column": 0, "subtasks": ["foo"]
            }`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: api.ErrNotInt,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				api.ErrNotInt.Error(),
			),
		},
		{
			name:          "ErrPutTask",
			reqBody:       `{"board": "boardid", "column": 0}`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        errors.New("failed to put task"),
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       nil,
			wantStatus:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				"failed to put task",
			),
		},
		{
			name:          "ErrEncodeState",
			reqBody:       `{"board": "boardid", "column": 0}`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "",
			errEncodeState:       errors.New("encode state failed"),
			wantStatus:           http.StatusInternalServerError,
			assertFunc:           assert.OnLoggedErr("encode state failed"),
		},
		{
			name:          "OK",
			reqBody:       `{"board": "boardid"}`,
			authToken:     "nonempty",
			authDecoded:   token.Auth{IsAdmin: true},
			errDecodeAuth: nil,
			inStateToken:  "nonempty",
			inStateDecoded: token.State{Boards: []token.Board{{
				ID: "boardid",
				Columns: []token.Column{{Tasks: []token.Task{{
					ID: "taskid", Order: 0,
				}}}},
			}}},
			errDecodeInState:     nil,
			errValidateColNo:     nil,
			errValidateTaskTitle: nil,
			errValidateSubtTitle: nil,
			errInsertTask:        nil,
			outStateToken:        token.State{},
			outStateEncoded:      "foobarbazbang",
			errEncodeState:       nil,
			wantStatus:           http.StatusOK,
			assertFunc: func(t *testing.T, resp *http.Response, _ string) {
				// assert on set state
				ck := resp.Cookies()[0]
				assert.Equal(t.Error, ck.Name, "state-token")
				assert.Equal(t.Error, ck.Value, "foobarbazbang")
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			decodeAuth.Decoded = c.authDecoded
			decodeAuth.Err = c.errDecodeAuth
			decodeState.Decoded = c.inStateDecoded
			decodeState.Err = c.errDecodeInState
			colNoVdtor.Err = c.errValidateColNo
			titleVdtor.Err = c.errValidateTaskTitle
			subtTitleVdtor.Err = c.errValidateSubtTitle
			taskInserter.Err = c.errInsertTask
			encodeState.Encoded = c.outStateEncoded
			encodeState.Err = c.errEncodeState

			req := httptest.NewRequest(
				http.MethodPost, "/", bytes.NewReader([]byte(c.reqBody)),
			)
			if c.authToken != "" {
				req.AddCookie(&http.Cookie{
					Name: "auth-token", Value: c.authToken,
				})
			}
			if c.inStateToken != "" {
				req.AddCookie(&http.Cookie{
					Name: "state-token", Value: c.inStateToken,
				})
			}

			w := httptest.NewRecorder()

			sut.Handle(w, req, "")
			res := w.Result()

			assert.Equal(t.Error, res.StatusCode, c.wantStatus)

			c.assertFunc(t, w.Result(), log.InMessage)
		})
	}
}
