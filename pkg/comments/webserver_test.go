package comments

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/weberc2/comments/pkg/testsupport"
	"github.com/weberc2/comments/pkg/types"
	pz "github.com/weberc2/httpeasy"
)

func TestDelete(t *testing.T) {
	now := time.Date(1988, 9, 3, 0, 0, 0, 0, time.UTC)
	for _, testCase := range []struct {
		name           string
		post           types.PostID
		comment        types.CommentID
		store          testsupport.CommentsStoreFake
		wantedStatus   int
		wantedComments []*types.Comment
		wantedLocation string
	}{
		{
			name:    "delete",
			post:    "post",
			comment: "comment",
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
				URL: &url.URL{RawQuery: "redirect=foo"},
				Vars: map[string]string{
					"post-id":    "post",
					"comment-id": "comment",
				},
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
