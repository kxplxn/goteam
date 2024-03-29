//go:build utest

package taskapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kxplxn/goteam/pkg/api"
	"github.com/kxplxn/goteam/pkg/assert"
	"github.com/kxplxn/goteam/pkg/cookie"
	"github.com/kxplxn/goteam/pkg/db"
	"github.com/kxplxn/goteam/pkg/db/tasktbl"
	"github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/validator"
)

// TestPatchHandler tests the PATCH handler.
func TestPatchHandler(t *testing.T) {
	decodeAuth := &cookie.FakeDecoder[cookie.Auth]{}
	titleValidator := &api.FakeStringValidator{}
	subtTitleValidator := &api.FakeStringValidator{}
	taskUpdater := &db.FakeUpdater[tasktbl.Task]{}
	log := &log.FakeErrorer{}
	sut := NewPatchHandler(
		decodeAuth,
		titleValidator,
		subtTitleValidator,
		taskUpdater,
		log,
	)

	for _, c := range []struct {
		name                 string
		authToken            string
		authDecoded          cookie.Auth
		errDecodeAuth        error
		errValidateTitle     error
		errValidateSubtTitle error
		taskUpdaterErr       error
		wantStatusCode       int
		assertFunc           func(*testing.T, *http.Response, []any)
	}{
		{
			name:                 "NoAuth",
			authToken:            "",
			authDecoded:          cookie.Auth{},
			errDecodeAuth:        nil,
			errValidateTitle:     nil,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusUnauthorized,
			assertFunc:           assert.OnRespErr("Auth token not found."),
		},
		{
			name:                 "ErrDecodeAuth",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{},
			errDecodeAuth:        cookie.ErrInvalid,
			errValidateTitle:     nil,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusUnauthorized,
			assertFunc:           assert.OnRespErr("Invalid auth token."),
		},
		{
			name:                 "NotAdmin",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: false},
			errDecodeAuth:        nil,
			errValidateTitle:     nil,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusForbidden,
			assertFunc: assert.OnRespErr(
				"Only team admins can edit tasks.",
			),
		},
		{
			name:                 "TaskTitleEmpty",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     validator.ErrEmpty,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusBadRequest,
			assertFunc: assert.OnRespErr(
				"Task title cannot be empty.",
			),
		},
		{
			name:                 "TaskTitleTooLong",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     validator.ErrTooLong,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusBadRequest,
			assertFunc: assert.OnRespErr(
				"Task title cannot be longer than 50 characters.",
			),
		},
		{
			name:                 "TaskTitleErr",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     validator.ErrWrongFormat,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				validator.ErrWrongFormat.Error(),
			),
		},
		{
			name:                 "SubtaskTitleEmpty",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     nil,
			errValidateSubtTitle: validator.ErrEmpty,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusBadRequest,
			assertFunc: assert.OnRespErr(
				"Subtask title cannot be empty.",
			),
		},
		{
			name:                 "SubtaskTitleTooLong",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     nil,
			errValidateSubtTitle: validator.ErrTooLong,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusBadRequest,
			assertFunc: assert.OnRespErr(
				"Subtask title cannot be longer than 50 characters.",
			),
		},
		{
			name:                 "SubtaskTitleErr",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     nil,
			errValidateSubtTitle: validator.ErrWrongFormat,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				validator.ErrWrongFormat.Error(),
			),
		},
		{
			name:                 "TaskNotFound",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     nil,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       db.ErrNoItem,
			wantStatusCode:       http.StatusNotFound,
			assertFunc:           assert.OnRespErr("Task not found."),
		},
		{
			name:                 "TaskUpdaterErr",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     nil,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       errors.New("update task failed"),
			wantStatusCode:       http.StatusInternalServerError,
			assertFunc:           assert.OnLoggedErr("update task failed"),
		},
		{
			name:                 "Success",
			authToken:            "nonempty",
			authDecoded:          cookie.Auth{IsAdmin: true, TeamID: "21"},
			errDecodeAuth:        nil,
			errValidateTitle:     nil,
			errValidateSubtTitle: nil,
			taskUpdaterErr:       nil,
			wantStatusCode:       http.StatusOK,
			assertFunc:           func(*testing.T, *http.Response, []any) {},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			decodeAuth.Res = c.authDecoded
			decodeAuth.Err = c.errDecodeAuth
			titleValidator.Err = c.errValidateTitle
			subtTitleValidator.Err = c.errValidateSubtTitle
			taskUpdater.Err = c.taskUpdaterErr
			w := httptest.NewRecorder()
			r := httptest.NewRequest("", "/?id=qwerty", strings.NewReader(`{
				"column":      0,
				"title":       "",
				"description": "",
				"subtasks":    [{"title": ""}]
			}`))
			if c.authToken != "" {
				r.AddCookie(&http.Cookie{
					Name:  "auth-token",
					Value: c.authToken,
				})
			}

			sut.Handle(w, r, "")

			resp := w.Result()
			assert.Equal(t.Error, resp.StatusCode, c.wantStatusCode)
			c.assertFunc(t, resp, log.Args)
		})
	}
}
