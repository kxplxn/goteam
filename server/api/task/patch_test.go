//go:build utest

package task

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kxplxn/goteam/server/api"
	"github.com/kxplxn/goteam/server/assert"
	columnTable "github.com/kxplxn/goteam/server/dbaccess/column"
	taskTable "github.com/kxplxn/goteam/server/dbaccess/task"
	userboardTable "github.com/kxplxn/goteam/server/dbaccess/userboard"
	pkgLog "github.com/kxplxn/goteam/server/log"
)

// TestPATCHHandler tests the Handle method of PATCHHandler to assert that it
// behaves correctly in all possible scenarios.
func TestPATCHHandler(t *testing.T) {
	taskIDValidator := &api.FakeStringValidator{}
	taskTitleValidator := &api.FakeStringValidator{}
	subtaskTitleValidator := &api.FakeStringValidator{}
	taskSelector := &taskTable.FakeSelector{}
	columnSelector := &columnTable.FakeSelector{}
	userBoardSelector := &userboardTable.FakeSelector{}
	taskUpdater := &taskTable.FakeUpdater{}
	log := &pkgLog.FakeErrorer{}
	sut := NewPATCHHandler(
		taskIDValidator,
		taskTitleValidator,
		subtaskTitleValidator,
		taskSelector,
		columnSelector,
		userBoardSelector,
		taskUpdater,
		log,
	)

	for _, c := range []struct {
		name                     string
		taskIDValidatorErr       error
		taskTitleValidatorErr    error
		subtaskTitleValidatorErr error
		taskSelectorErr          error
		columnSelectorErr        error
		userIsAdmin              bool
		userBoardSelectorErr     error
		taskUpdaterErr           error
		wantStatusCode           int
		assertFunc               func(*testing.T, *http.Response, string)
	}{
		{
			name:                     "TaskIDEmpty",
			taskIDValidatorErr:       api.ErrStrEmpty,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Task ID cannot be empty.",
			),
		},
		{
			name:                     "TaskIDNotInt",
			taskIDValidatorErr:       api.ErrStrNotInt,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Task ID must be an integer.",
			),
		},
		{
			name:                     "TaskIDUnexpectedErr",
			taskIDValidatorErr:       api.ErrStrTooLong,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				api.ErrStrTooLong.Error(),
			),
		},
		{
			name:                     "TaskTitleEmpty",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    api.ErrStrEmpty,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Task title cannot be empty.",
			),
		},
		{
			name:                     "TaskTitleTooLong",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    api.ErrStrTooLong,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Task title cannot be longer than 50 characters.",
			),
		},
		{
			name:                     "TaskTitleUnexpectedErr",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    api.ErrStrNotInt,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				api.ErrStrNotInt.Error(),
			),
		},
		{
			name:                     "SubtaskTitleEmpty",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: api.ErrStrEmpty,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Subtask title cannot be empty.",
			),
		},
		{
			name:                     "SubtaskTitleTooLong",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: api.ErrStrTooLong,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusBadRequest,
			assertFunc: assert.OnResErr(
				"Subtask title cannot be longer than 50 characters.",
			),
		},
		{
			name:                     "SubtaskTitleUnexpectedErr",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: api.ErrStrNotInt,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				api.ErrStrNotInt.Error(),
			),
		},
		{
			name:                     "TaskNotFound",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          sql.ErrNoRows,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusNotFound,
			assertFunc:               assert.OnResErr("Task not found."),
		},
		{
			name:                     "TaskSelectorErr",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          sql.ErrConnDone,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				sql.ErrConnDone.Error(),
			),
		},
		{
			name:                     "ColumnSelectorErr",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        sql.ErrNoRows,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusInternalServerError,
			assertFunc:               assert.OnLoggedErr(sql.ErrNoRows.Error()),
		},
		{
			name:                     "UserBoardSelectorErr",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     sql.ErrConnDone,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				sql.ErrConnDone.Error(),
			),
		},
		{
			name:                     "NoAccess",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     sql.ErrNoRows,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusForbidden,
			assertFunc: assert.OnResErr(
				"You do not have access to this board.",
			),
		},
		{
			name:                     "NotAdmin",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              false,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusForbidden,
			assertFunc: assert.OnResErr(
				"Only board admins can edit tasks.",
			),
		},
		{
			name:                     "TaskUpdaterErr",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              true,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           sql.ErrConnDone,
			wantStatusCode:           http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				sql.ErrConnDone.Error(),
			),
		},
		{
			name:                     "Success",
			taskIDValidatorErr:       nil,
			taskTitleValidatorErr:    nil,
			subtaskTitleValidatorErr: nil,
			taskSelectorErr:          nil,
			columnSelectorErr:        nil,
			userIsAdmin:              true,
			userBoardSelectorErr:     nil,
			taskUpdaterErr:           nil,
			wantStatusCode:           http.StatusOK,
			assertFunc:               func(*testing.T, *http.Response, string) {},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			taskIDValidator.Err = c.taskIDValidatorErr
			taskTitleValidator.Err = c.taskTitleValidatorErr
			subtaskTitleValidator.Err = c.subtaskTitleValidatorErr
			taskSelector.Err = c.taskSelectorErr
			columnSelector.Err = c.columnSelectorErr
			userBoardSelector.IsAdmin = c.userIsAdmin
			userBoardSelector.Err = c.userBoardSelectorErr
			taskUpdater.Err = c.taskUpdaterErr

			reqBody, err := json.Marshal(map[string]any{
				"column":      0,
				"title":       "",
				"description": "",
				"subtasks":    []map[string]any{{"title": ""}},
			})
			if err != nil {
				t.Fatal(err)
			}
			r, err := http.NewRequest("", "", bytes.NewReader(reqBody))
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()
			sut.Handle(w, r, "")
			res := w.Result()

			if err = assert.Equal(
				c.wantStatusCode, res.StatusCode,
			); err != nil {
				t.Error(err)
			}

			c.assertFunc(t, res, log.InMessage)
		})
	}
}