//go:build itest

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/kxplxn/goteam/internal/api"
	boardAPI "github.com/kxplxn/goteam/internal/api/board"
	"github.com/kxplxn/goteam/pkg/assert"
	"github.com/kxplxn/goteam/pkg/auth"
	teamTable "github.com/kxplxn/goteam/pkg/db/team"
	lgcBoardTable "github.com/kxplxn/goteam/pkg/legacydb/board"
	lgcUserTable "github.com/kxplxn/goteam/pkg/legacydb/user"
	pkgLog "github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/pkg/token"
)

func TestBoardAPI(t *testing.T) {
	userSelector := lgcUserTable.NewSelector(db)
	boardInserter := lgcBoardTable.NewInserter(db)
	nameValidator := boardAPI.NewNameValidator()
	idValidator := boardAPI.NewIDValidator()
	boardSelector := lgcBoardTable.NewSelector(db)
	log := pkgLog.New()
	sut := api.NewHandler(
		auth.NewJWTValidator(jwtKey),
		map[string]api.MethodHandler{
			http.MethodPost: boardAPI.NewPOSTHandler(
				userSelector,
				nameValidator,
				lgcBoardTable.NewCounter(db),
				boardInserter,
				log,
			),
			http.MethodDelete: boardAPI.NewDeleteHandler(
				token.DecodeAuth,
				token.DecodeState,
				teamTable.NewBoardDeleter(svcDynamo, svcDynamo),
				log,
			),
			http.MethodPatch: boardAPI.NewPATCHHandler(
				userSelector,
				idValidator,
				nameValidator,
				boardSelector,
				lgcBoardTable.NewUpdater(db),
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
				for _, method := range []string{
					http.MethodPost, http.MethodPatch,
				} {
					t.Run(method, func(t *testing.T) {
						req := httptest.NewRequest(method, "/board", nil)
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
			authFunc       func(*http.Request)
			boardName      string
			wantStatusCode int
			assertFunc     func(*testing.T, *http.Response, string)
		}{
			{
				name:           "NotAdmin",
				authFunc:       addCookieAuth(jwtTeam2Member),
				boardName:      "Team 2 Board 2",
				wantStatusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"Only team admins can create boards.",
				),
			},
			{
				name:           "EmptyBoardName",
				authFunc:       addCookieAuth(jwtTeam1Admin),
				boardName:      "",
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     assert.OnResErr("Board name cannot be empty."),
			},
			{
				name:           "TooLongBoardName",
				authFunc:       addCookieAuth(jwtTeam1Admin),
				boardName:      "A Board Whose Name Is Just Too Long!",
				wantStatusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"Board name cannot be longer than 35 characters.",
				),
			},
			{
				name:           "TooManyBoards",
				authFunc:       addCookieAuth(jwtTeam1Admin),
				boardName:      "bob123's new board",
				wantStatusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"You have already created the maximum amount of boards " +
						"allowed per user. Please delete one of your boards " +
						"to create a new one.",
				),
			},
			{
				name:           "Success",
				authFunc:       addCookieAuth(jwtTeam4Admin),
				boardName:      "Team 4 Board 1",
				wantStatusCode: http.StatusOK,
				assertFunc: func(t *testing.T, _ *http.Response, _ string) {
					// Assert that bob124 is assigned to the board as admin.
					var count int
					err := db.QueryRow(
						"SELECT COUNT(*) boardID FROM app.board " +
							"WHERE teamID = 4",
					).Scan(&count)
					if err != nil {
						t.Error(err)
					}
					assert.Equal(t.Error, count, 1)

					var boardID int
					err = db.QueryRow(
						"SELECT id FROM app.board WHERE teamID = 4",
					).Scan(&boardID)
					if err != nil {
						t.Error(err)
					}

					// Assert that 4 columns are created for this board.
					for order := 1; order < 5; order++ {
						var columnCount int
						err = db.QueryRow(
							`SELECT COUNT(*) FROM app."column" `+
								`WHERE boardID = $1 AND "order" = $2`,
							boardID,
							order,
						).Scan(&columnCount)
						if err != nil {
							t.Error(err)
						}
						assert.Equal(t.Error, columnCount, 1)
					}
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				reqBody, err := json.Marshal(map[string]string{
					"name": c.boardName,
				})
				if err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest(
					http.MethodPost, "/", bytes.NewReader(reqBody),
				)
				c.authFunc(req)
				w := httptest.NewRecorder()

				sut.ServeHTTP(w, req)
				res := w.Result()

				assert.Equal(t.Error, res.StatusCode, c.wantStatusCode)

				// Run case-specific assertions.
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
			assertFunc     func(*testing.T)
		}{
			{
				name:           "NotAdmin",
				id:             "f0c5d521-ccb5-47cc-ba40-313ddb901165",
				authFunc:       addCookieAuth(tkTeam1Member),
				wantStatusCode: http.StatusForbidden,
				assertFunc:     func(*testing.T) {},
			},
			{
				name:           "NoState",
				id:             "f0c5d521-ccb5-47cc-ba40-313ddb901165",
				authFunc:       addCookieAuth(tkTeam3Admin),
				wantStatusCode: http.StatusForbidden,
				assertFunc:     func(*testing.T) {},
			},
			{
				name: "NoAccess",
				id:   "f0c5d521-ccb5-47cc-ba40-313ddb901165",
				authFunc: func(r *http.Request) {
					addCookieAuth(tkTeam3Admin)(r)
					addCookieState(tkTeam1State)(r)
				},
				wantStatusCode: http.StatusForbidden,
				assertFunc:     func(*testing.T) {},
			},
			{
				name: "EmptyID",
				id:   "",
				authFunc: func(r *http.Request) {
					addCookieAuth(tkTeam3Admin)(r)
					addCookieState(tkTeam3State)(r)
				},
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     func(*testing.T) {},
			},
			{
				name: "InvalidID",
				id:   "qwerty",
				authFunc: func(r *http.Request) {
					addCookieAuth(tkTeam3Admin)(r)
					addCookieState(tkTeam3State)(r)
				},
				wantStatusCode: http.StatusBadRequest,
				assertFunc:     func(*testing.T) {},
			},
			{
				name: "Success",
				id:   "f0c5d521-ccb5-47cc-ba40-313ddb901165",
				authFunc: func(r *http.Request) {
					addCookieAuth(tkTeam3Admin)(r)
					addCookieState(tkTeam3State)(r)
				},
				wantStatusCode: http.StatusOK,
				assertFunc: func(t *testing.T) {
					out, err := svcDynamo.GetItem(context.Background(),
						&dynamodb.GetItemInput{
							TableName: &teamTableName,
							Key: map[string]types.AttributeValue{
								"ID": &types.AttributeValueMemberS{
									Value: "74c80ae5-64f3-4298-a8ff-48f8f920c" +
										"7d4",
								},
							},
						},
					)
					assert.Nil(t.Fatal, err)

					var team teamTable.Team
					err = attributevalue.UnmarshalMap(out.Item, &team)
					assert.Nil(t.Fatal, err)

					assert.Equal(t.Error, len(team.Boards), 0)
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				r := httptest.NewRequest(
					http.MethodDelete, "/board?id="+c.id, nil,
				)
				c.authFunc(r)
				w := httptest.NewRecorder()

				sut.ServeHTTP(w, r)
				res := w.Result()

				assert.Equal(t.Error, res.StatusCode, c.wantStatusCode)

				// Run case-specific assertions.
				c.assertFunc(t)
			})
		}
	})

	t.Run("PATCH", func(t *testing.T) {
		for _, c := range []struct {
			name       string
			id         string
			boardName  string
			authFunc   func(*http.Request)
			statusCode int
			assertFunc func(*testing.T, *http.Response, string)
		}{
			{
				name:       "NotAdmin",
				id:         "1",
				boardName:  "New Board Name",
				authFunc:   addCookieAuth(jwtTeam1Member),
				statusCode: http.StatusForbidden,
				assertFunc: assert.OnResErr(
					"Only team admins can edit the board.",
				),
			},
			{
				name:       "IDEmpty",
				id:         "",
				boardName:  "",
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr("Board ID cannot be empty."),
			},
			{
				name:       "IDNotInt",
				id:         "A",
				boardName:  "",
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr("Board ID must be an integer."),
			},
			{
				name:       "BoardNameEmpty",
				id:         "2",
				boardName:  "",
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr("Board name cannot be empty."),
			},
			{
				name:       "BoardNameTooLong",
				id:         "2",
				boardName:  "A Board Whose Name Is Just Too Long!",
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusBadRequest,
				assertFunc: assert.OnResErr(
					"Board name cannot be longer than 35 characters.",
				),
			},
			{
				name:       "BoardNotFound",
				id:         "1001",
				boardName:  "New Board Name",
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusNotFound,
				assertFunc: assert.OnResErr("Board not found."),
			},
			{
				name:       "Success",
				id:         "2",
				boardName:  "New Board Name",
				authFunc:   addCookieAuth(jwtTeam1Admin),
				statusCode: http.StatusOK,
				assertFunc: func(t *testing.T, _ *http.Response, _ string) {
					var boardName string
					err := db.QueryRow(
						"SELECT name FROM app.board WHERE id = 2",
					).Scan(&boardName)
					if err != nil {
						t.Fatal(err)
					}
					if boardName != "New Board Name" {
						t.Error("Board name was not updated.")
					}
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				reqBody, err := json.Marshal(map[string]string{
					"name": c.boardName,
				})
				if err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest(
					http.MethodPatch, "/?id="+c.id, bytes.NewReader(reqBody),
				)
				c.authFunc(req)
				w := httptest.NewRecorder()

				sut.ServeHTTP(w, req)
				res := w.Result()

				assert.Equal(t.Error, res.StatusCode, c.statusCode)

				c.assertFunc(t, res, "")
			})
		}
	})
}
