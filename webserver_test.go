package main

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"net/url"
// 	"testing"
//
// 	pz "github.com/weberc2/httpeasy"
// )
//
// func TestDelete(t *testing.T) {
// 	type postComment struct {
// 		Post PostID
// 		Comment
// 	}
// 	for _, testCase := range []struct {
// 		name             string
// 		post             PostID
// 		comment          CommentID
// 		existingComments []postComment
// 		wantedStatus     int
// 		wantedComments   []postComment
// 	}{
// 		{
// 			name:    "delete",
// 			post:    "post",
// 			comment: "comment",
// 			existingComments: []postComment{
// 				{
// 					"post",
// 					Comment{
// 						ID:     "comment",
// 						Author: "adam",
// 						Body:   "hello, world",
// 					},
// 				},
// 			},
// 			wantedStatus:   http.StatusTemporaryRedirect,
// 			wantedComments: nil,
// 		},
// 	} {
// 		t.Run(testCase.name, func(t *testing.T) {
// 			counter := 0
// 			posts := postStoreFake{}
// 			webServer := WebServer{
// 				Comments: ObjectCommentStore{
// 					ObjectStore: objectStoreFake{},
// 					PostStore:   &posts,
// 					Bucket:      "bucket",
// 					Prefix:      "prefix",
// 					IDFunc: func() CommentID {
// 						id := testCase.existingComments[counter].ID
// 						counter++
// 						return id
// 					},
// 				},
// 				LoginURL:  "https://auth.example.org/login",
// 				LogoutURL: "https://auth.example.org/logout",
// 				BaseURL:   "https://comments.example.org",
// 			}
//
// 			seen := map[PostID]struct{}{}
//
// 			for i, c := range testCase.existingComments {
// 				if c.Parent != "" {
// 					t.Fatal(
// 						"ERROR: comment parents aren't supported by this " +
// 							"test harness",
// 					)
// 				}
// 				if _, found := seen[c.Post]; !found {
// 					posts = append(posts, c.Post)
// 					seen[c.Post] = struct{}{}
// 				}
// 				if _, err := webServer.Comments.Put(
// 					testCase.existingComments[i].Post,
// 					&testCase.existingComments[i].Comment,
// 				); err != nil {
// 					t.Fatalf("Unexpected err: %v", err)
// 				}
// 			}
//
// 			rsp := webServer.Delete(pz.Request{
// 				Vars: map[string]string{
// 					"post-id":    "post",
// 					"comment-id": "comment",
// 				},
// 				URL: &url.URL{
// 					RawQuery: "redirect=posts/post/comments/toplevel/replies",
// 				},
// 			})
//
// 			if rsp.Status != testCase.wantedStatus {
// 				data, err := readAll(rsp.Data)
// 				if err != nil {
// 					t.Logf("reading response body: %v", err)
// 				}
// 				t.Logf("response body: %s", data)
//
// 				data, err = json.Marshal(rsp.Logging)
// 				if err != nil {
// 					t.Logf("marshaling response logging: %v", err)
// 				}
// 				t.Logf("response logging: %s", data)
// 				t.Fatalf(
// 					"Response.Status: wanted `%d`; found `%d`",
// 					testCase.wantedStatus,
// 					rsp.Status,
// 				)
// 			}
//
// 			wantedComments := map[PostID][]Comment{}
// 			for _, comment := range testCase.wantedComments {
// 				wantedComments[comment.Post] = append(
// 					wantedComments[comment.Post],
// 					comment.Comment,
// 				)
// 			}
//
// 			for post, wantedComments := range wantedComments {
// 				comments, err := webServer.Comments.Replies(post, "toplevel")
// 				if err != nil {
// 					t.Fatalf("Unexpected err: %v", err)
// 				}
//
// 				if err := compareComments(
// 					wantedComments,
// 					comments,
// 				); err != nil {
// 					t.Fatal(err)
// 				}
// 			}
// 		})
// 	}
// }
//
// func compareComments(wanted, found []Comment) error {
// 	if len(wanted) != len(found) {
// 		return fmt.Errorf(
// 			"wanted %d comments; found %d",
// 			len(wanted),
// 			len(found),
// 		)
// 	}
//
// 	for i := range wanted {
// 		if err := wanted[i].compare(&found[i]); err != nil {
// 			return fmt.Errorf("index %d: %w", i, err)
// 		}
// 	}
//
// 	return nil
// }
