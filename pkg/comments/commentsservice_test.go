package comments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/weberc2/comments/pkg/testsupport"
	"github.com/weberc2/comments/pkg/types"
	pz "github.com/weberc2/httpeasy"
	pztest "github.com/weberc2/httpeasy/testsupport"
)

func TestCommentsService_Delete(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		state        testsupport.CommentsStoreFake
		post         types.PostID
		comment      types.CommentID
		wantedStatus int
		wantedBody   WantedData
		wantedState  []*types.Comment
	}{
		{
			name: "delete works",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  false,
						Body:     "greetings",
					},
				},
			},
			post:         "post",
			comment:      "id",
			wantedStatus: http.StatusOK,
			wantedBody: &DeleteCommentResponse{
				Message: "deleted comment",
				Post:    "post",
				Comment: "id",
				Status:  http.StatusOK,
			},
			wantedState: []*types.Comment{{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someTime,
				Modified: now,
				Deleted:  true,
				Body:     "greetings",
			}},
		},
		{
			name:         "errors propagate",
			state:        testsupport.CommentsStoreFake{},
			post:         "post",
			comment:      "id",
			wantedStatus: http.StatusNotFound,
			wantedBody:   types.ErrCommentNotFound,
			wantedState:  nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			service := CommentsService{
				Comments: CommentsModel{
					CommentsStore: &testCase.state,
					TimeFunc:      func() time.Time { return now },
				},
				TimeFunc: func() time.Time { return now },
			}
			rsp := service.Delete(pz.Request{
				Vars: map[string]string{
					"post-id":    string(testCase.post),
					"comment-id": string(testCase.comment),
				},
			})
			if rsp.Status != testCase.wantedStatus {
				t.Fatalf(
					"status: wanted `%d`; found `%d`",
					testCase.wantedStatus,
					rsp.Status,
				)
			}
			if err := pztest.CompareSerializer(
				testCase.wantedBody,
				rsp.Data,
			); err != nil {
				t.Fatal(err)
			}
			if err := pztest.CompareSerializer(
				testCase.wantedBody,
				rsp.Data,
			); err != nil {
				t.Fatal(err)
			}
			if err := types.CompareComments(
				testCase.wantedState,
				testCase.state.List(),
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCommentsService_Update(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		state        testsupport.CommentsStoreFake
		post         types.PostID
		comment      types.CommentID
		body         string
		wantedStatus int
		wantedBody   pztest.WantedData
		wantedState  []*types.Comment
	}{
		{
			name: "simple",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  false,
						Body:     "hello, world",
					},
				},
			},
			post:         "post",
			comment:      "id",
			body:         `{"body": "salutations"}`,
			wantedStatus: http.StatusOK,
			wantedBody:   UpdateResponseSuccess,
			wantedState: []*types.Comment{{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someTime,
				Modified: now,
				Deleted:  false,
				Body:     "salutations",
			}},
		},
		{
			// test that unmarshal errors are handled correctly.
			name:         "unmarshal error",
			post:         "post",
			comment:      "id",
			body:         "",
			wantedStatus: http.StatusBadRequest,
			wantedBody: &pz.HTTPError{
				Message: "error unmarshaling json",
				Status:  http.StatusBadRequest,
			},
		},
		{
			// test that the errors are handled by trying to update a comment
			// that doesn't exist--the commentsstore should return a
			// `types.CommentNotFoundErr` which should be marshaled by our
			// handler.
			name:         "errors handled",
			post:         "post",
			comment:      "not-found",
			body:         `{"body": "this is a valid body"}`,
			wantedStatus: http.StatusNotFound,
			wantedBody:   types.ErrCommentNotFound,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			service := CommentsService{
				Comments: CommentsModel{
					CommentsStore: testCase.state,
					IDFunc:        func() types.CommentID { return "comment" },
					TimeFunc:      func() time.Time { return now },
				},
				TimeFunc: func() time.Time { return now },
			}
			rsp := service.Update(pz.Request{
				Vars: map[string]string{
					"post-id":    string(testCase.post),
					"comment-id": string(testCase.comment),
				},
				Body: strings.NewReader(testCase.body),
			})
			if rsp.Status != testCase.wantedStatus {
				t.Fatalf(
					"HTTP Status: wanted `%d`; found `%d`",
					testCase.wantedStatus,
					rsp.Status,
				)
			}
			if err := pztest.CompareSerializer(
				testCase.wantedBody,
				rsp.Data,
			); err != nil {
				t.Fatal(err)
			}
			if err := types.CompareComments(
				testCase.wantedState,
				testCase.state.List(),
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCommentsService_Put(t *testing.T) {
	now := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, testCase := range []struct {
		name         string
		input        string
		wantedStatus int
		wantedBody   WantedData
	}{
		{
			name:         "creates comment",
			input:        `{"body": "great comment"}`,
			wantedStatus: http.StatusCreated,
			wantedBody: &types.Comment{
				ID:       "comment",
				Post:     "post",
				Author:   "user",
				Parent:   "",
				Created:  now,
				Modified: now,
				Deleted:  false,
				Body:     "great comment",
			},
		},
		{
			// If the caller tries to pass explicit values for id, author,
			// created, or modified fields, these fields are ignored in favor
			// of authoritative sources (`id` is generated by the underlying
			// CommentStore, `author` defers to the `User` header, and
			// `created` and `modified` are set to the current time).
			name: "ignores id, author, created, modified",
			input: `{
	"id": "asdf",
	"post": "post",
	"parent": "",
	"author": "foo",
	"created": "1970-01-01T00:00:00.000000Z",
	"modified": "1970-01-01T00:00:00.000000Z",
	"body": "great comment"
}`,
			wantedStatus: http.StatusCreated,
			wantedBody: &types.Comment{
				ID:       "comment",
				Post:     "post",
				Parent:   "",
				Author:   "user",
				Created:  now,
				Modified: now,
				Body:     "great comment",
			},
		},
		{
			name:         "errors serialized correctly",
			input:        `{"body": ""}`,
			wantedStatus: 400,
			wantedBody:   ErrBodyTooShort,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			commentsService := CommentsService{
				Comments: CommentsModel{
					CommentsStore: testsupport.CommentsStoreFake{},
					TimeFunc:      func() time.Time { return now },
					IDFunc:        func() types.CommentID { return "comment" },
				},
				TimeFunc: func() time.Time { return now },
			}

			rsp := commentsService.Put(pz.Request{
				Vars:    map[string]string{"post-id": "post"},
				Headers: http.Header{"User": []string{"user"}},
				Body:    strings.NewReader(testCase.input),
			})

			if rsp.Status != testCase.wantedStatus {
				data, err := readAll(rsp.Data)
				if err != nil {
					t.Logf("error reading response body: %v", err)
				}
				for _, logging := range rsp.Logging {
					data, err := json.Marshal(logging)
					if err != nil {
						t.Logf("error marshalling logging as JSON: %v", err)
					}
					t.Logf("log: %s", data)
				}
				t.Fatalf(
					"Response.Status: wanted `%d`; found `%d`: %s",
					testCase.wantedStatus,
					rsp.Status,
					data,
				)
			}

			data, err := readAll(rsp.Data)
			if err != nil {
				t.Fatalf("Response.Data: reading serializer: %v", err)
			}

			if err := testCase.wantedBody.CompareData(data); err != nil {
				t.Errorf("Response.Data: %v", err)
				t.Logf("Actual body: %s", data)
			}
		})
	}
}

func readAll(s pz.Serializer) ([]byte, error) {
	writerTo, err := s()
	if err != nil {
		return nil, fmt.Errorf("serializing: %w", err)
	}

	var b bytes.Buffer
	if _, err := writerTo.WriteTo(&b); err != nil {
		return nil, fmt.Errorf("copying data to buffer: %w", err)
	}

	return b.Bytes(), nil
}

type WantedData interface {
	CompareData([]byte) error
}
