package comments

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/weberc2/comments/pkg/testsupport"
	"github.com/weberc2/comments/pkg/types"
	pz "github.com/weberc2/httpeasy"
)

func TestReply(t *testing.T) {
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
				Post:     "post",
				ID:       "comment",
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
						Post:     "post",
						ID:       "parent",
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
					Post:     "post",
					ID:       "parent",
					Author:   "jesse",
					Body:     "hello, world",
					Created:  now.Add(-24 * time.Hour),
					Modified: now.Add(-24 * time.Hour),
				},
				{
					Post:     "post",
					ID:       "comment",
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
				LoginURL:  "https://auth.example.org/login",
				LogoutURL: "https://auth.example.org/logout",
				BaseURL:   "https://comments.example.org",
			}

			rsp := webServer.Reply(pz.Request{
				Vars: map[string]string{
					"post-id":    "post",
					"comment-id": "parent",
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
				testCase.store.Comments(),
			); err != nil {
				t.Fatalf("CommentsStore: %v", err)
			}
		})
	}
}

func TestDelete(t *testing.T) {
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
						Post:   "post",
						ID:     "comment",
						Author: "adam",
						Body:   "hello, world",
					},
				},
			},
			wantedStatus:   http.StatusTemporaryRedirect,
			wantedComments: nil,
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
			wantedStatus:   http.StatusTemporaryRedirect,
			wantedComments: nil,
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
			// expect that the comment wasn't deleted
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
				LoginURL:  "https://auth.example.org/login",
				LogoutURL: "https://auth.example.org/logout",
				BaseURL:   "https://comments.example.org",
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
				testCase.store.Comments(),
			); err != nil {
				t.Fatalf("CommentsStore: %v", err)
			}
		})
	}
}
