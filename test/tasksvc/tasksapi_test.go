//go:build itest

package tasksvc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/kxplxn/goteam/internal/tasksvc/tasksapi"
	"github.com/kxplxn/goteam/pkg/api"
	"github.com/kxplxn/goteam/pkg/assert"
	"github.com/kxplxn/goteam/pkg/cookie"
	"github.com/kxplxn/goteam/pkg/db/tasktbl"
	"github.com/kxplxn/goteam/pkg/log"
	"github.com/kxplxn/goteam/test"
)

func TestTasksAPI(t *testing.T) {
	authDecoder := cookie.NewAuthDecoder(test.JWTKey)
	log := log.New()
	sut := api.NewHandler(map[string]api.MethodHandler{
		http.MethodGet: tasksapi.NewGetHandler(
			tasksapi.NewBoardIDValidator(),
			tasktbl.NewRetrieverByBoard(test.DB()),
			authDecoder,
			tasktbl.NewRetrieverByTeam(test.DB()),
			log,
		),
		http.MethodPatch: tasksapi.NewPatchHandler(
			authDecoder,
			tasksapi.NewColNoValidator(),
			tasktbl.NewMultiUpdater(test.DB()),
			log,
		),
	})

	t.Run("GET", func(t *testing.T) {
		t.Run("WithBoardID", func(t *testing.T) {
			for _, c := range []struct {
				name       string
				boardID    string
				authFunc   func(*http.Request)
				statusCode int
				assertFunc func(*testing.T, *http.Response, string)
			}{
				{
					name:       "NoAuth",
					boardID:    "",
					authFunc:   func(*http.Request) {},
					statusCode: http.StatusUnauthorized,
				},
				{
					name:       "InvalidAuth",
					boardID:    "",
					authFunc:   test.AddAuthCookie("asdkjlfhass"),
					statusCode: http.StatusUnauthorized,
				},
				{
					name:       "BoardIDInvalid",
					boardID:    "askdfjhas",
					authFunc:   test.AddAuthCookie(test.T4MemberToken),
					statusCode: http.StatusBadRequest,
				},
				{
					name:       "TaskWrongTeam",
					boardID:    "ca47fbec-269e-4ef4-a74a-bcfbcd599fd5",
					authFunc:   test.AddAuthCookie(test.T1MemberToken),
					statusCode: http.StatusForbidden,
				},
				{
					name:       "OK",
					boardID:    "ca47fbec-269e-4ef4-a74a-bcfbcd599fd5",
					authFunc:   test.AddAuthCookie(test.T4MemberToken),
					statusCode: http.StatusOK,
					assertFunc: func(
						t *testing.T, resp *http.Response, _ string,
					) {
						wantResp := tasksapi.GetResp{
							{
								TeamID: "3c3ec4ea-a850-4fc5-aab0-24e9e7223bb" +
									"c",
								BoardID: "ca47fbec-269e-4ef4-a74a-bcfbcd599fd" +
									"5",
								ColNo: 0,
								ID: "55e275e4-de80-4241-b73b-88e784d5522" +
									"b",
								Title:       "team 4 task 1",
								Description: "team 4 task 1 description",
								Order:       1,
								Subtasks: []tasktbl.Subtask{
									{Title: "team 4 subtask 1", IsDone: false},
								},
							},
							{
								TeamID: "3c3ec4ea-a850-4fc5-aab0-24e9e7223bb" +
									"c",
								BoardID: "ca47fbec-269e-4ef4-a74a-bcfbcd599fd" +
									"5",
								ColNo: 0,
								ID: "5ccd750d-3783-4832-891d-025f24a4944" +
									"f",
								Title:       "team 4 task 2",
								Description: "team 4 task 2 description",
								Order:       0,
								Subtasks: []tasktbl.Subtask{
									{Title: "team 4 subtask 2", IsDone: true},
								},
							},
						}

						var respBody tasksapi.GetResp
						err := json.NewDecoder(resp.Body).Decode(&respBody)
						if err != nil {
							t.Fatal(err)
						}

						assert.Equal(t.Error, len(respBody), len(wantResp))
						for i, wt := range wantResp {
							task := respBody[i]
							assert.Equal(t.Error, task.TeamID, wt.TeamID)
							assert.Equal(t.Error, task.BoardID, wt.BoardID)
							assert.Equal(t.Error,
								task.ColNo, wt.ColNo,
							)
							assert.Equal(t.Error, task.ID, wt.ID)
							assert.Equal(t.Error, task.Title, wt.Title)
							assert.Equal(t.Error,
								task.Description, wt.Description,
							)
							assert.Equal(t.Error, task.Order, wt.Order)

							assert.Equal(t.Error,
								len(task.Subtasks), len(wt.Subtasks),
							)
							for j, wst := range wt.Subtasks {
								subtask := task.Subtasks[j]
								assert.Equal(t.Error, subtask.Title, wst.Title)
								assert.Equal(t.Error,
									subtask.IsDone, wst.IsDone,
								)
							}
						}
					},
				},
			} {
				t.Run(c.name, func(t *testing.T) {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(
						http.MethodGet, "/tasks?boardID="+c.boardID, nil,
					)
					c.authFunc(r)

					sut.ServeHTTP(w, r)
					resp := w.Result()

					assert.Equal(t.Error, resp.StatusCode, c.statusCode)
				})
			}
		})

		t.Run("WithoutBoardID", func(t *testing.T) {
			for _, c := range []struct {
				name       string
				authFunc   func(*http.Request)
				statusCode int
				assertFunc func(*testing.T, *http.Response, string)
			}{
				{
					name:       "NoAuth",
					authFunc:   func(*http.Request) {},
					statusCode: http.StatusUnauthorized,
				},
				{
					name:       "InvalidAuth",
					authFunc:   test.AddAuthCookie("asdkjlfhass"),
					statusCode: http.StatusUnauthorized,
				},
				{
					name:       "OK",
					authFunc:   test.AddAuthCookie(test.T1MemberToken),
					statusCode: http.StatusOK,
					assertFunc: func(
						t *testing.T, resp *http.Response, _ string,
					) {
						wantResp := tasksapi.GetResp{
							{
								TeamID:  "afeadc4a-68b0-4c33-9e83-4648d20ff26a",
								BoardID: "91536664-9749-4dbb-a470-6e52aa353ae4",
								ColNo:   0,
								ID: "55e275e4-de80-4241-b73b-88e784d5522" +
									"b",
								Title:       "task 5",
								Description: "",
								Order:       1,
								Subtasks:    []tasktbl.Subtask{},
							},
						}

						var respBody tasksapi.GetResp
						err := json.NewDecoder(resp.Body).Decode(&respBody)
						if err != nil {
							t.Fatal(err)
						}

						assert.Equal(t.Error, len(respBody), len(wantResp))
						for i, wt := range wantResp {
							task := respBody[i]
							assert.Equal(t.Error, task.TeamID, wt.TeamID)
							assert.Equal(t.Error, task.BoardID, wt.BoardID)
							assert.Equal(t.Error,
								task.ColNo, wt.ColNo,
							)
							assert.Equal(t.Error, task.ID, wt.ID)
							assert.Equal(t.Error, task.Title, wt.Title)
							assert.Equal(t.Error,
								task.Description, wt.Description,
							)
							assert.Equal(t.Error, task.Order, wt.Order)

							assert.Equal(t.Error,
								len(task.Subtasks), len(wt.Subtasks),
							)
						}
					},
				},
			} {
				t.Run(c.name, func(t *testing.T) {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(
						http.MethodGet, "/tasks", nil,
					)
					c.authFunc(r)

					sut.ServeHTTP(w, r)
					resp := w.Result()

					assert.Equal(t.Error, resp.StatusCode, c.statusCode)
				})
			}
		})
	})

	t.Run("PATCH", func(t *testing.T) {
		for _, c := range []struct {
			name       string
			reqBody    string
			authFunc   func(*http.Request)
			statusCode int
			assertFunc func(*testing.T, *http.Response, []any)
		}{
			{
				name:       "NoAuth",
				reqBody:    `[]`,
				authFunc:   func(*http.Request) {},
				statusCode: http.StatusUnauthorized,
				assertFunc: assert.OnRespErr("Auth token not found."),
			},
			{
				name:       "InvalidAuth",
				reqBody:    `[]`,
				authFunc:   test.AddAuthCookie("asdfjkahsd"),
				statusCode: http.StatusUnauthorized,
				assertFunc: assert.OnRespErr("Invalid auth token."),
			},
			{
				name:       "NotAdmin",
				reqBody:    `[]`,
				authFunc:   test.AddAuthCookie(test.T1MemberToken),
				statusCode: http.StatusForbidden,
				assertFunc: assert.OnRespErr(
					"Only team admins can edit tasks.",
				),
			},
			{
				name:       "NoTasks",
				reqBody:    `[]`,
				authFunc:   test.AddAuthCookie(test.T1AdminToken),
				statusCode: http.StatusBadRequest,
				assertFunc: assert.OnRespErr("No tasks provided."),
			},
			{
				name: "OK",
				reqBody: `[{
                    "id": "c684a6a0-404d-46fa-9fa5-1497f9874567", 
                    "title": "task 5",
                    "order": 2,
                    "subtasks": [],
                    "boardID": "f0c5d521-ccb5-47cc-ba40-313ddb901165",
                    "colNo": 2
                }]`,
				authFunc:   test.AddAuthCookie(test.T1AdminToken),
				statusCode: http.StatusOK,
				assertFunc: func(t *testing.T, _ *http.Response, _ []any) {
					out, err := test.DB().GetItem(
						context.Background(),
						&dynamodb.GetItemInput{
							TableName: &tableName,
							Key: map[string]types.AttributeValue{
								"TeamID": &types.AttributeValueMemberS{
									Value: "afeadc4a-68b0-4c33-9e83-4648d20ff" +
										"26a",
								},
								"ID": &types.AttributeValueMemberS{
									Value: "c684a6a0-404d-46fa-9fa5-1497f9874" +
										"567",
								},
							},
						},
					)
					assert.Nil(t.Fatal, err)

					var task tasktbl.Task
					assert.Nil(t.Fatal, attributevalue.UnmarshalMap(
						out.Item, &task,
					))

					assert.Equal(t.Error,
						task.ID, "c684a6a0-404d-46fa-9fa5-1497f9874567",
					)
					assert.Equal(t.Error, task.Title, "task 5")
					assert.Equal(t.Error, task.Order, 2)
					assert.Equal(t.Error, len(task.Subtasks), 0)
					assert.Equal(t.Error,
						task.BoardID, "f0c5d521-ccb5-47cc-ba40-313ddb901165",
					)
					assert.Equal(t.Error, task.ColNo, 2)
				},
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				r := httptest.NewRequest(
					http.MethodPatch, "/tasks", strings.NewReader(c.reqBody),
				)
				c.authFunc(r)

				sut.ServeHTTP(w, r)

				resp := w.Result()
				assert.Equal(t.Error, resp.StatusCode, c.statusCode)
				c.assertFunc(t, resp, []any{})
			})
		}
	})
}
