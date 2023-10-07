//go:build utest

package board

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"server/api"
	"server/assert"
	"server/dbaccess"
	userboardTable "server/dbaccess/userboard"
	pkgLog "server/log"
)

// TestDELETEHandler tests the Handle method of DELETEHandler to assert that it
// behaves correctly in all possible scenarios.
func TestDELETEHandler(t *testing.T) {
	validator := &api.FakeStringValidator{}
	userBoardSelector := &userboardTable.FakeSelector{}
	userBoardDeleter := &dbaccess.FakeDeleter{}
	log := &pkgLog.FakeErrorer{}
	sut := NewDELETEHandler(
		validator, userBoardSelector, userBoardDeleter, log,
	)

	// Used on cases where no case-specific assertions are required.
	emptyAssertFunc := func(*testing.T, *http.Response, string) {}

	for _, c := range []struct {
		name                        string
		validatorOutErr             error
		userBoardSelectorOutIsAdmin bool
		userBoardSelectorOutErr     error
		boardDeleterOutErr          error
		wantStatusCode              int
		assertFunc                  func(*testing.T, *http.Response, string)
	}{
		{
			name:                        "ValidatorErr",
			validatorOutErr:             errors.New("some validator err"),
			userBoardSelectorOutIsAdmin: true,
			userBoardSelectorOutErr:     nil,
			boardDeleterOutErr:          nil,
			wantStatusCode:              http.StatusBadRequest,
			assertFunc:                  emptyAssertFunc,
		},
		{
			name:                        "ConnDone",
			validatorOutErr:             nil,
			userBoardSelectorOutIsAdmin: false,
			userBoardSelectorOutErr:     sql.ErrConnDone,
			boardDeleterOutErr:          nil,
			wantStatusCode:              http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				sql.ErrConnDone.Error(),
			),
		},
		{
			name:                        "UserBoardNotFound",
			validatorOutErr:             nil,
			userBoardSelectorOutIsAdmin: false,
			userBoardSelectorOutErr:     sql.ErrNoRows,
			boardDeleterOutErr:          nil,
			wantStatusCode:              http.StatusForbidden,
			assertFunc:                  emptyAssertFunc,
		},
		{
			name:                        "NotAdmin",
			validatorOutErr:             nil,
			userBoardSelectorOutIsAdmin: false,
			userBoardSelectorOutErr:     nil,
			boardDeleterOutErr:          nil,
			wantStatusCode:              http.StatusForbidden,
			assertFunc:                  emptyAssertFunc,
		},
		{
			name:                        "DeleteErr",
			validatorOutErr:             nil,
			userBoardSelectorOutIsAdmin: true,
			userBoardSelectorOutErr:     nil,
			boardDeleterOutErr:          errors.New("delete board error"),
			wantStatusCode:              http.StatusInternalServerError,
			assertFunc: assert.OnLoggedErr(
				"delete board error",
			),
		},
		{
			name:                        "Success",
			validatorOutErr:             nil,
			userBoardSelectorOutIsAdmin: true,
			userBoardSelectorOutErr:     nil,
			boardDeleterOutErr:          nil,
			wantStatusCode:              http.StatusOK,
			assertFunc:                  emptyAssertFunc,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			// Set pre-determinate return values for sut's dependencies.
			validator.OutErr = c.validatorOutErr
			userBoardSelector.OutIsAdmin = c.userBoardSelectorOutIsAdmin
			userBoardSelector.OutErr = c.userBoardSelectorOutErr
			userBoardDeleter.OutErr = c.boardDeleterOutErr

			// Prepare request and response recorder.
			req, err := http.NewRequest(http.MethodPost, "", nil)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()

			// Handle request with sut and get the result.
			sut.Handle(w, req, "")
			res := w.Result()

			// Assert on the status code.
			if err := assert.Equal(c.wantStatusCode, res.StatusCode); err != nil {
				t.Error(err)
			}

			// Run case-specific assertions.
			c.assertFunc(t, res, log.InMessage)
		})
	}
}
