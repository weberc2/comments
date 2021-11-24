package comments

import (
	"errors"
	html "html/template"
	"net/url"

	pz "github.com/weberc2/httpeasy"
)

type logging struct {
	Post   PostID    `json:"post"`
	Parent CommentID `json:"parent"`
	User   UserID    `json:"user,omitempty"`
	Error  string    `json:"error,omitempty"`
}

type WebServer struct {
	LoginURL  string
	LogoutURL string
	BaseURL   string
	Comments  CommentsStore
}

func (ws *WebServer) Replies(r pz.Request) pz.Response {
	post := PostID(r.Vars["post-id"])
	parent := CommentID(r.Vars["parent-id"])
	user := UserID(r.Headers.Get("User"))
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
				User:   user,
				Error:  err.Error(),
			})
		}

		return pz.InternalServerError(&logging{
			Post:   post,
			Parent: parent,
			User:   user,
			Error:  err.Error(),
		})
	}

	return pz.Ok(
		pz.HTMLTemplate(repliesTemplate, struct {
			LoginURL  string     `json:"loginURL"`
			LogoutURL string     `json:"logoutURL"`
			BaseURL   string     `json:"baseURL"`
			Post      PostID     `json:"post"`
			Parent    CommentID  `json:"parent"`
			Replies   []*Comment `json:"replies"`
			User      UserID     `json:"user"`
		}{
			LoginURL:  ws.LoginURL,
			LogoutURL: ws.LogoutURL,
			BaseURL:   ws.BaseURL,
			Post:      post,
			Parent:    parent,
			Replies:   replies,
			User:      user, // empty if unauthorized
		}),
		&logging{Post: post, Parent: parent, User: user},
	)
}

func (ws *WebServer) DeleteConfirm(r pz.Request) pz.Response {
	context := struct {
		BaseURL string   `json:"baseURL"`
		User    UserID   `json:"user"`
		Post    PostID   `json:"post"`
		Comment *Comment `json:"comment"`
		Error   string   `json:"error,omitempty"`
	}{
		BaseURL: ws.BaseURL,
		Post:    PostID(r.Vars["post-id"]),
		Comment: &Comment{ID: CommentID(r.Vars["comment-id"])},
		User:    UserID(r.Headers.Get("User")), // empty if unauthorized
	}

	comment, err := ws.Comments.Comment(context.Post, context.Comment.ID)
	if err != nil {
		var e *CommentNotFoundErr
		if errors.As(err, &e) {
			context.Error = err.Error()
			return pz.NotFound(nil, context)
		}
		return pz.InternalServerError(context)
	}

	context.Comment = comment
	return pz.Ok(pz.HTMLTemplate(deleteConfirmationTemplate, context), context)
}

func (ws *WebServer) Delete(r pz.Request) pz.Response {
	context := struct {
		Message  string    `json:"message"`
		Post     PostID    `json:"post"`
		Comment  CommentID `json:"comment"`
		Redirect string    `json:"redirect"`
		Error    string    `json:"error,omitempty"`
	}{
		Post:     PostID(r.Vars["post-id"]),
		Comment:  CommentID(r.Vars["comment-id"]),
		Redirect: ws.BaseURL + "/" + r.URL.Query().Get("redirect"),
	}

	if err := ws.Comments.Delete(context.Post, context.Comment); err != nil {
		var e *CommentNotFoundErr
		if errors.As(err, &e) {
			context.Message = "comment not found"
			context.Error = err.Error()
			return pz.NotFound(nil, context)
		}
		context.Message = "deleting comment"
		context.Error = err.Error()
		return pz.InternalServerError(context)
	}

	if _, err := url.Parse(context.Redirect); err != nil {
		context.Message = "error parsing redirect; redirecting to `BaseURL`"
		context.Error = err.Error()
		return pz.TemporaryRedirect(context.Redirect, context)
	}

	context.Message = "successfully deleted comment"
	return pz.TemporaryRedirect(context.Redirect, context)
}

var (
	repliesTemplate = html.Must(html.New("").Parse(`<html>
<head></head>
<body>
<h1>Replies</h1>
<div id=replies>
{{if .User}}
    {{.User}} - <a href="{{.LogoutURL}}">logout</a>
{{else}}
    <a href="{{.LoginURL}}">login</a>
{{end}}

{{$baseURL := .BaseURL}}
{{$post := .Post}}
{{$user := .User}}
{{range .Replies}}
	<div id="{{.ID}}">
		<div class="comment-header">
			<span class="author">{{.Author}}</p>
			<span class="date">{{.Created}}</p>
			{{if eq .Author $user}}
			<a href="{{$baseURL}}/posts/{{$post}}/comments/{{.ID}}/delete-confirm">
				delete
			</a>
			<a href="{{$baseURL}}/posts/{{$post}}/comments/{{.ID}}/edit">
				edit
			</a>
			{{end}}
			{{/* if the user is logged in they can reply */}}
			{{if $user}}
			<a href="{{$baseURL}}/posts/{{$post}}/comments/{{.ID}}/reply">
				reply
			</a>
			{{end}}
			<p class="body">{{.Body}}</p>
		</div>
	</div>
{{end}}
</div>
</body>
</html>`))

	deleteConfirmationTemplate = html.Must(html.New("").Parse(`<html>
<head></head>
<body>
<h1>Confirm Comment Deletion</h1>
<div id="comment">
    {{.Comment.Body}}
</div>
<div id="cancel">
    {{/*
       * For now, return to the comment itself. In the future we may pass a
       * return location through in case we have multiple delete comment
       * buttons.
    */}}
    <a href="{{.BaseURL}}/posts/{{.Post}}/comments/{{.Comment.ID}}">Cancel</a>
</div>
<div id="delete">
    <a href="{{.BaseURL}}/posts/{{.Post}}/comments/{{.Comment.ID}}/delete?redirect=posts/{{.Post}}/comments/toplevel/replies">
        Delete
    </a>
</div>
</div>
</body>
</html>`))
)
