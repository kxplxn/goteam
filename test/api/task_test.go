//go:build itest

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kxplxn/goteam/internal/api"
	taskAPI "github.com/kxplxn/goteam/internal/api/task"
	"github.com/kxplxn/goteam/pkg/assert"
	"github.com/kxplxn/goteam/pkg/auth"
	boardTable "github.com/kxplxn/goteam/pkg/dbaccess/board"
	columnTable "github.com/kxplxn/goteam/pkg/dbaccess/column"
	taskTable "github.com/kxplxn/goteam/pkg/dbaccess/task"
	userTable "github.com/kxplxn/goteam/pkg/dbaccess/user"
	pkgLog "github.com/kxplxn/goteam/pkg/log"
)

// TestTaskHandler tests the http.Handler for the task API route and asserts
// that it behaves correctly during various execution paths.
func TestTaskHandler(t *testing.T) {
	idValidator := taskAPI.NewIDValidator()
	titleValidator := taskAPI.NewTitleValidator()
	taskSelector := taskTable.NewSelector(db)
	columnSelector := columnTable.NewSelector(db)
	boardSelector := boardTable.NewSelector(db)
	userSelector := userTable.NewSelector(db)
	log := pkgLog.New()

	sut := api.NewHandler(
		auth.NewBearerTokenReader(),
		auth.NewJWTValidator(jwtKey),
		map[string]api.MethodHandler{
			http.MethodPost: taskAPI.NewPOSTHandler(
				userSelector,
				titleValidator,
				titleValidator,
				columnSelector,
				boardSelector,
				taskTable.NewInserter(db),
				log,
			),
			http.MethodPatch: taskAPI.NewPATCHHandler(
				userSelector,
				idValidator,
				titleValidator,
				titleValidator,
				taskSelector,
				columnSelector,
				boardSelector,
				taskTable.NewUpdater(db),
				log,
			),
			http.MethodDelete: taskAPI.NewDELETEHandler(
				userSelector,
				idValidator,
				taskSelector,
				columnSelector,
				boardSelector,
				taskTable.NewDeleter(db),
				log,
			),
		},
	)

	t.Run("Auth", func(t *testing.T) {
		for _, c := range []struct {
			name     string
			authFunc func(*http.Request)
		}{
			// Auth Cases
			{name: "HeaderEmpty", authFunc: func(*http.Request) {}},
			{name: "HeaderInvalid", authFunc: addCookieAuth("asdfasldfkjasd")},
		} {
			t.Run(c.name, func(t *testing.T) {
				for _, httpMethod := range []string{
					http.MethodPost, http.MethodPatch, http.MethodDelete,
				} {
					t.Run(httpMethod, func(t *testing.T) {
						req, err := http.NewRequest(httpMethod, "", nil)
						if err != nil {
							t.Fatal(err)
						}
						c.authFunc(req)
						w := httptest.NewRecorder()

						sut.ServeHTTP(w, req)
						res := w.Result()

						assert.Equal(t.Error,
							res.StatusCode, http.StatusUnauthorized,
						)

						assert.Equal(t.Error,
							res.Header.Values("WWW-Authenticate")[0], "Bearer",
						)
					})
				}
			})
		}
	})

	t.Run("POST", func(t *testing.T) {
		for _, c := range []struct {
			name           string
			reqBody        map[string]any
			authFunc       func(*http.Request)
			wantStatusCode int
			assertFunc     func(*testing.T, *http.Response, string)
		}{
			{
				name: "TaskTitleEmpty",
				reqBody: map[string]any{
					"title":       "",
					"description": "",
					"column":      0,
					"subtasks":    []string{},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     assert.OnResErr("Task title cannot be empty."),
			},
			{
				name: "TaskTitleTooLong",
				reqBody: map[string]any{
					"title": "asdqweasdqweasdqweasdqweasdqweasdqweasdqweasd" +
						"qweasd",
					"description": "",
					"column":      0,
					"subtasks":    []string{},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"Task title cannot be longer than 50 characters.",
				),
			},
			{
				name: "SubtaskTitleEmpty",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"column":      0,
					"subtasks":    []string{""},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"Subtask title cannot be empty.",
				),
			},
			{
				name: "SubtaskTitleTooLong",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"column":      0,
					"subtasks": []string{
						"asdqweasdqweasdqweasdqweasdqweasdqweasdqweasdqweasd",
					},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"Subtask title cannot be longer than 50 characters.",
				),
			},
			{
				name: "ColumnNotFound",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"column":      1001,
					"subtasks":    []string{"Some Subtask"},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusNotFound,
				assertFunc:     assert.OnResErr("Column not found."),
			},
			{
				name: "NoAccess",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"column":      5,
					"subtasks":    []string{"Some Subtask"},
				},
				authFunc:       addCookieAuth(jwtTeam2Admin),
				wantStatusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"You do not have access to this board.",
				),
			},
			{
				name: "NotAdmin",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"column":      5,
					"subtasks":    []string{"Some Subtask"},
				},
				authFunc:       addCookieAuth(jwtTeam1Member),
				wantStatusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"Only team admins can create tasks.",
				),
			},
			{
				name: "Success",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "Do something. Then, do something else.",
					"column":      7,
					"subtasks": []string{
						"Some Subtask", "Some Other Subtask",
					},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusOK,
				assertFunc: func(t *testing.T, _ *http.Response, _ string) {
					// A task with the order of 1 and 2 already exists in the given
					// column. Therefore, the order of the newly created task must
					// be 3.
					var taskID, taskOrder int
					if err := db.QueryRow(
						`SELECT id, "order" FROM app.task `+
							`WHERE columnID = $1 AND title = 'Some Task'`,
						7,
					).Scan(&taskID, &taskOrder); err != nil {
						t.Error(err)
					}
					assert.Equal(t.Error, taskOrder, 2)

					// The order of the subtasks must be set respective to their
					// sequential order.
					for i, subtaskTitle := range []string{
						"Some Subtask", "Some Other Subtask",
					} {
						wantOrder := i
						var subtaskOrder int
						if err := db.QueryRow(
							`SELECT "order" FROM app.subtask `+
								`WHERE taskID = $1 AND title = $2`,
							taskID,
							subtaskTitle,
						).Scan(&subtaskOrder); err != nil {
							t.Error(err)
						}
						assert.Equal(t.Error, subtaskOrder, wantOrder)
					}
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				reqBodyBytes, err := json.Marshal(c.reqBody)
				if err != nil {
					t.Fatal(err)
				}
				req, err := http.NewRequest(
					http.MethodPost, "", bytes.NewReader(reqBodyBytes),
				)
				if err != nil {
					t.Fatal(err)
				}

				c.authFunc(req)

				w := httptest.NewRecorder()

				sut.ServeHTTP(w, req)
				res := w.Result()

				assert.Equal(t.Error, res.StatusCode, c.wantStatusCode)

				c.assertFunc(t, res, "")
			})
		}
	})

	t.Run("PATCH", func(t *testing.T) {
		for _, c := range []struct {
			name           string
			taskID         string
			reqBody        map[string]any
			authFunc       func(*http.Request)
			wantStatusCode int
			assertFunc     func(*testing.T, *http.Response, string)
		}{
			{
				name:   "TaskIDEmpty",
				taskID: "",
				reqBody: map[string]any{
					"title":       "",
					"description": "",
					"subtasks":    []map[string]any{},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     assert.OnResErr("Task ID cannot be empty."),
			},
			{
				name:   "TaskIDNotInt",
				taskID: "A",
				reqBody: map[string]any{
					"title":       "",
					"description": "",
					"subtasks":    []map[string]any{},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     assert.OnResErr("Task ID must be an integer."),
			},
			{
				name:   "TaskTitleEmpty",
				taskID: "0",
				reqBody: map[string]any{
					"title":       "",
					"description": "",
					"subtasks":    []map[string]any{},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     assert.OnResErr("Task title cannot be empty."),
			},
			{
				name:   "TaskTitleTooLong",
				taskID: "0",
				reqBody: map[string]any{
					"title": "asdqweasdqweasdqweasdqweasdqweasdqweasdqweasd" +
						"qweasd",
					"description": "",
					"subtasks":    []map[string]any{},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"Task title cannot be longer than 50 characters.",
				),
			},
			{
				name:   "SubtaskTitleEmpty",
				taskID: "0",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"subtasks": []map[string]any{
						{"title": ""},
					},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"Subtask title cannot be empty.",
				),
			},
			{
				name:   "SubtaskTitleTooLong",
				taskID: "0",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"subtasks": []map[string]any{{
						"title": "asdqweasdqweasdqweasdqweasdqweasdqweasdqwea" +
							"sdqweasd",
					}},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"Subtask title cannot be longer than 50 characters.",
				),
			},
			{
				name:   "TaskNotFound",
				taskID: "1001",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"subtasks":    []map[string]any{{"title": "Some Subtask"}},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusNotFound,
				assertFunc:     assert.OnResErr("Task not found."),
			},
			{
				name:   "WrongTeam",
				taskID: "7",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"subtasks": []map[string]any{{
						"title": "Some Subtask",
					}},
				},
				authFunc:       addCookieAuth(jwtTeam2Admin),
				wantStatusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"You do not have access to this board.",
				),
			},
			{
				name:   "NotAdmin",
				taskID: "8",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "",
					"subtasks": []map[string]any{{
						"title": "Some Subtask",
					}},
				},
				authFunc:       addCookieAuth(jwtTeam1Member),
				wantStatusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"Only team admins can edit tasks.",
				),
			},
			{
				name:   "OK",
				taskID: "8",
				reqBody: map[string]any{
					"title":       "Some Task",
					"description": "Some Description",
					"subtasks": []map[string]any{
						{
							"title": "Some Subtask",
							"order": 1,
							"done":  false,
						},
						{
							"title": "Some Other Subtask",
							"order": 2,
							"done":  true,
						},
					},
				},
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusOK,
				assertFunc: func(t *testing.T, _ *http.Response, _ string) {
					var (
						title       string
						description *string
						columnID    int
						order       int
					)
					if err := db.QueryRow(
						`SELECT title, description, columnID, "order" `+
							`FROM app.task WHERE id = 8`,
					).Scan(&title, &description, &columnID, &order); err != nil {
						t.Error(err)
					}

					assert.Equal(t.Error, title, "Some Task")
					assert.Equal(t.Error, *description, "Some Description")

					rows, err := db.Query(
						`SELECT title, "order", isDone FROM app.subtask WHERE taskID = 8`,
					)
					if err != nil {
						t.Fatal(err)
					}
					var subtasks []taskTable.Subtask
					for rows.Next() {
						var subtask taskTable.Subtask
						if err := rows.Scan(
							&subtask.Title, &subtask.Order, &subtask.IsDone,
						); err != nil {
							t.Fatal(err)
						}
						subtasks = append(subtasks, subtask)
					}
					assert.Equal(t.Error, len(subtasks), 2)
					assert.Equal(t.Error, subtasks[0].Title, "Some Subtask")
					assert.Equal(t.Error, subtasks[0].Order, 1)
					assert.True(t.Error, !subtasks[0].IsDone)
					assert.Equal(t.Error, "Some Other Subtask", subtasks[1].Title)
					assert.Equal(t.Error, 2, subtasks[1].Order)
					assert.True(t.Error, subtasks[1].IsDone)
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				reqBodyBytes, err := json.Marshal(c.reqBody)
				if err != nil {
					t.Fatal(err)
				}
				req, err := http.NewRequest(
					http.MethodPatch,
					"?id="+c.taskID,
					bytes.NewReader(reqBodyBytes),
				)
				if err != nil {
					t.Fatal(err)
				}

				c.authFunc(req)

				w := httptest.NewRecorder()

				sut.ServeHTTP(w, req)
				res := w.Result()

				assert.Equal(t.Error, res.StatusCode, c.wantStatusCode)

				c.assertFunc(t, res, "")
			})
		}
	})

	t.Run("DELETE", func(t *testing.T) {
		for _, c := range []struct {
			name           string
			id             string
			authFunc       func(*http.Request)
			wantStatusCode int
			assertFunc     func(*testing.T, *http.Response, string)
		}{
			{
				name:           "IDEmpty",
				id:             "",
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     assert.OnResErr("Task ID cannot be empty."),
			},
			{
				name:           "IDNotInt",
				id:             "A",
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     assert.OnResErr("Task ID must be an integer."),
			},
			{
				name:           "TaskNotFound",
				id:             "1001",
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusNotFound,
				assertFunc:     assert.OnResErr("Task not found."),
			},
			{
				name:           "WrongTeam",
				id:             "8",
				authFunc:       addCookieAuth(jwtTeam2Admin),
				wantStatusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"You do not have access to this board.",
				),
			},
			{
				name:           "NotAdmin",
				id:             "8",
				authFunc:       addCookieAuth(jwtTeam1Member),
				wantStatusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"Only board admins can delete tasks.",
				),
			},
			{
				name:           "OK",
				id:             "9",
				authFunc:       addCookieAuth(jwtTeam1Admin),
				wantStatusCode: http.StatusOK,
				assertFunc: func(t *testing.T, _ *http.Response, _ string) {
					var taskCount int
					err := db.QueryRow(
						"SELECT COUNT(*) FROM app.task WHERE id = 9",
					).Scan(&taskCount)
					if err != nil {
						t.Fatal(err)
					}
					assert.Equal(t.Error, taskCount, 0)
					var subtaskCount int
					err = db.QueryRow(
						"SELECT COUNT(*) FROM app.subtask WHERE taskID = 9",
					).Scan(&taskCount)
					assert.Equal(t.Error, subtaskCount, 0)
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				r, err := http.NewRequest(http.MethodDelete, "?id="+c.id, nil)
				if err != nil {
					t.Fatal(err)
				}
				c.authFunc(r)
				w := httptest.NewRecorder()

				sut.ServeHTTP(w, r)
				res := w.Result()

				assert.Equal(t.Error, res.StatusCode, c.wantStatusCode)

				c.assertFunc(t, res, "")
			})
		}
	})
}