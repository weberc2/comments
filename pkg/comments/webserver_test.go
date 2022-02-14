package comments

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/weberc2/comments/pkg/comments/testsupport"
	"github.com/weberc2/comments/pkg/comments/types"
	pz "github.com/weberc2/httpeasy"
	pztest "github.com/weberc2/httpeasy/testsupport"
)

func TestWebServer_Reply(t *testing.T) {
	now := time.Date(1988, 9, 3, 0, 0, 0, 0, time.UTC)

	for _, testCase := range []struct {
		name           string
		post           types.PostID
		parent         types.CommentID
		user           types.UserID
		body           string
		store          testsupport.CommentsStoreFake
		wantedStatus   int
		wantedComments []*types.Comment
		wantedLocation string
	}{
		{
			name:         "toplevel reply",
			post:         "post",
			parent:       "toplevel",
			user:         "adam",
			body:         "hello, world",
			store:        testsupport.CommentsStoreFake{},
			wantedStatus: http.StatusSeeOther,
			wantedComments: []*types.Comment{{
				ID:       "comment",
				Post:     "post",
				Parent:   "",
				Author:   "adam",
				Body:     "hello, world",
				Created:  now,
				Modified: now,
			}},
			wantedLocation: "https://comments.example.org/posts/post/" +
				"comments/toplevel/replies#comment",
		},
		{
			name:   "nested reply",
			post:   "post",
			parent: "parent",
			user:   "david",
			body:   "hello, jesse",
			store: testsupport.CommentsStoreFake{
				"post": {
					"parent": {
						ID:       "parent",
						Post:     "post",
						Parent:   "",
						Author:   "jesse",
						Body:     "hello, world",
						Created:  now.Add(-24 * time.Hour),
						Modified: now.Add(-24 * time.Hour),
					},
				},
			},
			wantedStatus: http.StatusSeeOther,
			wantedComments: []*types.Comment{
				{
					ID:       "parent",
					Post:     "post",
					Parent:   "",
					Author:   "jesse",
					Body:     "hello, world",
					Created:  now.Add(-24 * time.Hour),
					Modified: now.Add(-24 * time.Hour),
				},
				{
					ID:       "comment",
					Post:     "post",
					Parent:   "parent",
					Author:   "david",
					Body:     "hello, jesse",
					Created:  now,
					Modified: now,
				},
			},
			wantedLocation: "https://comments.example.org/posts/post/" +
				"comments/toplevel/replies#comment",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			webServer := WebServer{
				Comments: CommentsModel{
					CommentsStore: testCase.store,
					TimeFunc:      func() time.Time { return now },
					IDFunc:        func() types.CommentID { return "comment" },
				},
				LoginURL:   "https://auth.example.org/login",
				LogoutPath: "logout",
				BaseURL:    "https://comments.example.org",
			}

			rsp := webServer.Reply(pz.Request{
				Vars: map[string]string{
					"post-id":    string(testCase.post),
					"comment-id": string(testCase.parent),
				},
				Headers: http.Header{"User": []string{string(testCase.user)}},
				Body: strings.NewReader(
					url.Values{"body": []string{testCase.body}}.Encode(),
				),
			})

			if rsp.Status != testCase.wantedStatus {
				t.Fatalf(
					"Response.Status: wanted `%d`; found `%d`",
					testCase.wantedStatus,
					rsp.Status,
				)
			}

			location := rsp.Headers.Get("Location")
			if location != testCase.wantedLocation {
				t.Fatalf(
					"Response.Headers[\"Location\"]: wanted `%s`; found `%s`",
					testCase.wantedLocation,
					location,
				)
			}

			if err := types.CompareComments(
				testCase.wantedComments,
				testCase.store.List(),
			); err != nil {
				t.Fatalf("CommentsStore: %v", err)
			}
		})
	}
}

func TestWebServer_Delete(t *testing.T) {
	now := time.Date(1988, 9, 3, 0, 0, 0, 0, time.UTC)
	for _, testCase := range []struct {
		name           string
		post           types.PostID
		comment        types.CommentID
		redirect       string
		user           types.UserID
		store          testsupport.CommentsStoreFake
		wantedStatus   int
		wantedComments []*types.Comment
		wantedLocation string
	}{
		{
			name:     "delete",
			post:     "post",
			comment:  "comment",
			redirect: "foo",
			user:     "adam",
			store: testsupport.CommentsStoreFake{
				"post": {
					"comment": &types.Comment{
						Post:     "post",
						ID:       "comment",
						Author:   "adam",
						Modified: now,
						Body:     "hello, world",
					},
				},
			},
			wantedStatus: http.StatusTemporaryRedirect,
			wantedComments: []*types.Comment{{
				Post:     "post",
				ID:       "comment",
				Author:   "adam",
				Modified: now,
				Deleted:  true,
				Body:     "hello, world",
			}},
			wantedLocation: "https://comments.example.org/foo",
		},
		{
			name:     "redirects to baseurl on redirect parse err",
			post:     "post",
			comment:  "comment",
			redirect: "!@#$%^&*()",
			user:     "adam",
			store: testsupport.CommentsStoreFake{
				"post": {
					"comment": &types.Comment{
						Post:   "post",
						ID:     "comment",
						Author: "adam",
						Body:   "hello, world",
					},
				},
			},
			wantedStatus: http.StatusTemporaryRedirect,
			wantedComments: []*types.Comment{{
				Post:     "post",
				ID:       "comment",
				Author:   "adam",
				Modified: now,
				Deleted:  true,
				Body:     "hello, world",
			}},
			wantedLocation: "https://comments.example.org/",
		},
		{
			name:    "user must be author",
			post:    "post",
			comment: "comment",
			user:    "eve",
			store: testsupport.CommentsStoreFake{
				"post": {
					"comment": &types.Comment{
						Post:   "post",
						ID:     "comment",
						Author: "adam",
						Body:   "hello, world",
					},
				},
			},
			// expect that the comment wasn't changed
			wantedComments: []*types.Comment{{
				Post:   "post",
				ID:     "comment",
				Author: "adam",
				Body:   "hello, world",
			}},
			wantedStatus: http.StatusUnauthorized,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			webServer := WebServer{
				Comments: CommentsModel{
					CommentsStore: testCase.store,
					TimeFunc:      func() time.Time { return now },
					IDFunc:        func() types.CommentID { return "comment" },
				},
				LoginURL:   "https://auth.example.org/login",
				LogoutPath: "logout",
				BaseURL:    "https://comments.example.org",
			}

			rsp := webServer.Delete(pz.Request{
				URL: &url.URL{RawQuery: "redirect=" + testCase.redirect},
				Vars: map[string]string{
					"post-id":    "post",
					"comment-id": "comment",
				},
				Headers: http.Header{"User": []string{string(testCase.user)}},
			})

			if rsp.Status != testCase.wantedStatus {
				t.Fatalf(
					"Response.Status: wanted `%d`; found `%d`",
					testCase.wantedStatus,
					rsp.Status,
				)
			}

			location := rsp.Headers.Get("Location")
			if location != testCase.wantedLocation {
				t.Fatalf(
					"Response.Headers[\"Location\"]: wanted `%s`; found `%s`",
					testCase.wantedLocation,
					location,
				)
			}

			if err := types.CompareComments(
				testCase.wantedComments,
				testCase.store.List(),
			); err != nil {
				t.Fatalf("CommentsStore: %v", err)
			}
		})
	}
}

func TestWebServer_Edit(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		state        testsupport.CommentsStoreFake
		post         types.PostID
		comment      types.CommentID
		requestBody  io.Reader
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
						Body:     "greetings and salutations",
					},
				},
			},
			post:    "post",
			comment: "id",
			requestBody: strings.NewReader(url.Values{
				"body": []string{"salutations"},
			}.Encode()),
			wantedStatus: http.StatusSeeOther,
			wantedBody:   Literal("303 See Other"),
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
			name:         "malformed request body",
			requestBody:  strings.NewReader("this should be form data"),
			wantedStatus: http.StatusBadRequest,
			wantedBody: &pz.HTTPError{
				Status:  http.StatusBadRequest,
				Message: "parsing form values: missing required field `body`",
			},
		},
	} {
		webServer := WebServer{
			Comments: CommentsModel{
				CommentsStore: testCase.state,
				TimeFunc:      func() time.Time { return now },
			},
			BaseURL: "https://example.org",
		}
		rsp := webServer.Edit(pz.Request{
			Vars: map[string]string{
				"post-id":    string(testCase.post),
				"comment-id": string(testCase.comment),
			},
			Body: testCase.requestBody,
		})

		if rsp.Status != testCase.wantedStatus {
			t.Fatalf(
				"HTTP Status: wanted `%d`; found `%d`",
				testCase.wantedStatus,
				rsp.Status,
			)
		}

		if testCase.wantedBody == nil {
			testCase.wantedBody = Literal("")
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
	}
}

type Literal string

func (wanted Literal) CompareData(found []byte) error {
	if wanted != Literal(found) {
		return fmt.Errorf("wanted `%s`; found `%s`", wanted, found)
	}
	return nil
}
