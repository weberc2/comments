package main

import (
	"errors"
	html "html/template"

	pz "github.com/weberc2/httpeasy"
)

type logging struct {
	Post   PostID    `json:"post"`
	Parent CommentID `json:"parent"`
	Error  string    `json:"error"`
}

type WebServer struct {
	LoginURL                   string
	LogoutURL                  string
	BaseURL                    string
	Comments                   CommentStore
	RepliesTemplate            *html.Template
	DeleteConfirmationTemplate *html.Template
}

func (ws *WebServer) Replies(r pz.Request) pz.Response {
	post := PostID(r.Vars["post-id"])
	parent := CommentID(r.Vars["parent-id"])
	if parent == "toplevel" {
		parent = "" // this tells the CommentStore to fetch toplevel replies.
	}
	replies, err := ws.Comments.Replies(post, parent)
	if err != nil {
		var c *CommentNotFoundErr
		if errors.As(err, &c) {
			pz.NotFound(nil, &logging{
				Post:   post,
				Parent: parent,
				Error:  err.Error(),
			})
		}

		return pz.InternalServerError(&logging{
			Post:   post,
			Parent: parent,
			Error:  err.Error(),
		})
	}

	return pz.Ok(
		pz.HTMLTemplate(ws.RepliesTemplate, struct {
			LoginURL  string
			LogoutURL string
			BaseURL   string
			Post      PostID
			Parent    CommentID
			Replies   []Comment
			User      UserID
		}{
			LoginURL:  ws.LoginURL,
			LogoutURL: ws.LogoutURL,
			BaseURL:   ws.BaseURL,
			Post:      post,
			Parent:    parent,
			Replies:   replies,
			User:      UserID(r.Headers.Get("User")), // empty if unauthorized
		}),
		&logging{Post: post, Parent: parent},
	)
}

func (ws *WebServer) DeleteConfirmation(r pz.Request) pz.Response {
	params := struct {
		Post    PostID
		Comment CommentID
		User    UserID
	}{
		Post:    PostID(r.Vars["post-id"]),
		Comment: CommentID(r.Vars["comment-id"]),
		User:    UserID(r.Headers.Get("User")), // empty if unauthorized
	}

	return pz.Ok(
		pz.HTMLTemplate(ws.DeleteConfirmationTemplate, params),
		params,
	)
}
