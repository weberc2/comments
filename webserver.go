package main

import (
	"errors"
	html "html/template"

	pz "github.com/weberc2/httpeasy"
)

type WebServer struct {
	LoginURL        string
	BaseURL         string
	Comments        CommentStore
	RepliesTemplate *html.Template
}

func (ws *WebServer) Replies(r pz.Request) pz.Response {
	post := PostID(r.Vars["post-id"])
	parent := CommentID(r.Vars["parent-id"])
	if parent == "toplevel" {
		parent = "__toplevel__"
	}
	replies, err := ws.Comments.Replies(post, parent)
	if err != nil {
		var c *CommentNotFoundErr
		if errors.As(err, &c) {
			pz.NotFound(nil, struct {
				Post   PostID
				Parent CommentID
				Error  string
			}{
				Post:   post,
				Parent: parent,
				Error:  err.Error(),
			})
		}

		return pz.InternalServerError(struct {
			Post   PostID
			Parent CommentID
			Error  string
		}{
			Post:   post,
			Parent: parent,
			Error:  err.Error(),
		})
	}

	return pz.Ok(
		pz.HTMLTemplate(ws.RepliesTemplate, struct {
			LoginURL string
			BaseURL  string
			Post     PostID
			Parent   CommentID
			Replies  []Comment
			User     UserID
		}{
			LoginURL: ws.LoginURL,
			BaseURL:  ws.BaseURL,
			Post:     post,
			Parent:   parent,
			Replies:  replies,
			User:     UserID(r.Headers.Get("User")), // empty if unauthorized
		}),
		struct {
			Post   PostID
			Parent CommentID
		}{
			Post:   post,
			Parent: parent,
		},
	)
}
