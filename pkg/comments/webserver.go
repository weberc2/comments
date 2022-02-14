package comments

import (
	"errors"
	"fmt"
	html "html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/weberc2/comments/pkg/comments/types"
	pz "github.com/weberc2/httpeasy"
)

type logging struct {
	Post   types.PostID    `json:"post"`
	Parent types.CommentID `json:"parent"`
	User   types.UserID    `json:"user,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type WebServer struct {
	LoginURL         string
	RegisterURL      string
	LogoutPath       string
	BaseURL          string
	Comments         CommentsModel
	AuthCallbackPath string
}

var repliesTemplate = html.Must(html.New("").Parse(`
{{- define "comment"}}
	<div id="{{.ID}}">
		<a id="{{.ID}}"></a>
		<div class="comment">
			{{ if not .Deleted }}
			<span class="author">{{.Author}}</p>
			{{ else }}
			<span class="author">DELETED</span>
			{{ end }}
			<span class="date">{{.Created}}</p>
			{{if eq .Author .User}}
			<a href="{{.BaseURL}}/posts/{{.Post}}/comments/{{.ID}}/delete-confirm">
				delete
			</a>
			<a href="{{.BaseURL}}/posts/{{.Post}}/comments/{{.ID}}/edit">
				edit
			</a>
			{{end}}
			{{/* if the user is logged in they can reply */}}
			{{if and .User (not .Deleted) }}
			<a href="{{.BaseURL}}/posts/{{.Post}}/comments/{{.ID}}/reply">
				reply
			</a>
			{{end}}
			{{ if not .Deleted }}
			<p class="body">{{.Body}}</p>
			{{ else }}
			<p class="body">DELETED</p>
			{{end}}
			<div class="comment-children">
			{{- range .Children}}
				{{template "comment" .}}
			{{- end}}
			</div>
		</div>
	</div>
{{end}}

<html>
<head>
<style>
.comment {
	border: 1px solid black;
	margin: 1em 0em 1em 1em;
	padding: 1em 0em 1em 1em;
}
.comment-children {
	padding-left: 1em;
}
</style>
</head>
<body>
<a href="{{.BaseURL}}/posts/{{.Post}}/comments/toplevel/reply">
	Reply To Post
</a>
<h1>Replies</h1>
<div id=replies>
{{if .User}}
    {{.User}} - <a href="{{.LogoutURL}}">logout</a>
{{else}}
    <a href="{{.LoginURL}}">login</a>
	<a href="{{.RegisterURL}}">register</a>
{{end}}

{{range .Replies}}
	{{template "comment" .}}
{{end}}
</div>
</body>
</html>`))

func (ws *WebServer) Replies(r pz.Request) pz.Response {
	post := types.PostID(r.Vars["post-id"])
	parent := types.CommentID(r.Vars["parent-id"])
	user := types.UserID(r.Headers.Get("User"))
	if parent == "toplevel" {
		parent = "" // this tells the CommentStore to fetch toplevel replies.
	}
	comments, err := ws.Comments.Replies(post, parent)
	if err != nil {
		if errors.Is(err, types.ErrCommentNotFound) {
			return pz.NotFound(nil, &logging{
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
			LoginURL    string          `json:"loginURL"`
			LogoutURL   string          `json:"logoutURL"`
			RegisterURL string          `json:"registerURL"`
			BaseURL     string          `json:"baseURL"`
			Post        types.PostID    `json:"post"`
			Parent      types.CommentID `json:"parent"`
			Replies     []*reply        `json:"replies"`
			User        types.UserID    `json:"user"`
		}{
			LoginURL: fmt.Sprintf(
				"%s?%s",
				ws.LoginURL,
				url.Values{
					"callback": []string{join(
						ws.BaseURL,
						ws.AuthCallbackPath,
					)},
					"redirect": []string{fmt.Sprintf(
						"/posts/%s/comments/%s/replies",
						post,
						func() types.CommentID {
							if parent == "" {
								return "toplevel"
							}
							return parent
						}(),
					)},
				}.Encode(),
			),
			LogoutURL:   join(ws.BaseURL, ws.LogoutPath),
			RegisterURL: ws.RegisterURL,
			BaseURL:     ws.BaseURL,
			Post:        post,
			Parent:      parent,
			User:        user,
			Replies: replies(
				comments,
				&globals{BaseURL: ws.BaseURL, User: user},
			),
		}),
		&logging{Post: post, Parent: parent, User: user},
	)
}

func join(lhs, rhs string) string {
	return fmt.Sprintf(
		"%s/%s",
		strings.TrimRight(lhs, "/"),
		strings.TrimLeft(rhs, "/"),
	)
}

type globals struct {
	BaseURL string
	User    types.UserID
}

type reply struct {
	*globals
	*types.Comment
	Children []*reply
}

func replies(comments []*types.Comment, globals *globals) []*reply {
	// values is just a buffer so we don't have to allocate O(n) replies.
	values := make([]reply, len(comments)+1)

	// repliesByID allows us to look up a reply by the id of its comment. We'll
	// put one "root" reply (whose comment is nil) in the map for toplevel
	// comments (comments whose `Parent` field is `nil`).
	repliesByID := map[types.CommentID]*reply{"": &values[0]}

	// insert a reply into `repliesByID` for each input comment
	for i, c := range comments {
		r := &values[i+1]
		r.Comment = c
		r.globals = globals
		repliesByID[c.ID] = r
	}

	// now that there is a reply for each comment in `repliesByID`, loop over
	// the comments again, fetch the reply corresponding to the current comment
	// and the reply corresponding to the comment's parent and append the
	// current comment's reply to the parent comment's reply's list of
	// children.
	for _, c := range comments {
		p := repliesByID[c.Parent]
		p.Children = append(p.Children, repliesByID[c.ID])
	}

	// return the root reply's children
	return values[0].Children
}

var deleteConfirmationTemplate = html.Must(html.New("").Parse(`<html>
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

func (ws *WebServer) DeleteConfirm(r pz.Request) pz.Response {
	context := struct {
		BaseURL string         `json:"baseURL"`
		User    types.UserID   `json:"user"`
		Post    types.PostID   `json:"post"`
		Comment *types.Comment `json:"comment"`
		Error   string         `json:"error,omitempty"`
	}{
		BaseURL: ws.BaseURL,
		Post:    types.PostID(r.Vars["post-id"]),
		Comment: &types.Comment{ID: types.CommentID(r.Vars["comment-id"])},
		User:    types.UserID(r.Headers.Get("User")), // empty if unauthorized
	}

	comment, err := ws.Comments.Comment(context.Post, context.Comment.ID)
	if err != nil {
		if errors.Is(err, types.ErrCommentNotFound) {
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
		Message  string          `json:"message,omitempty"`
		Post     types.PostID    `json:"post"`
		Comment  types.CommentID `json:"comment"`
		User     types.UserID    `json:"user"`
		Redirect string          `json:"redirect"`
		Error    string          `json:"error,omitempty"`
	}{
		Post:     types.PostID(r.Vars["post-id"]),
		Comment:  types.CommentID(r.Vars["comment-id"]),
		User:     types.UserID(r.Headers.Get("User")),
		Redirect: ws.BaseURL + "/" + r.URL.Query().Get("redirect"),
	}

	comment, err := ws.Comments.Comment(context.Post, context.Comment)
	if err != nil {
		return pz.HandleError("fetching comment", err, &context)
	}

	if comment.Author != context.User {
		context.Message = "authorizing user"
		context.Error = "user is not comment author"
		return pz.Unauthorized(nil, &context)
	}

	if err := ws.Comments.Delete(context.Post, context.Comment); err != nil {
		return pz.HandleError("deleting comment", err, &context)
	}

	if _, err := url.Parse(context.Redirect); err != nil {
		context.Message = "error parsing redirect; redirecting to `BaseURL`"
		context.Error = err.Error()
		return pz.TemporaryRedirect(context.Redirect, &context)
	}

	context.Message = "successfully deleted comment"
	return pz.TemporaryRedirect(context.Redirect, &context)
}

var replyTemplate = html.Must(html.New("").Parse(`<html>
<head></head>
<body>
<div id="comment">
	{{if .Comment.Body}}{{.Comment.Body}}{{else}}&lt;toplevel&gt;{{end}}
</div>
<div id="form">
<form action="{{.BaseURL}}/posts/{{.Comment.Post}}/comments/{{.Comment.ID}}/reply" method="POST">
	<textarea name="body"></textarea>
	<input type="submit" value="Submit">
</form>
</div>
</body>
</html>`))

func (ws *WebServer) ReplyForm(r pz.Request) pz.Response {
	context := struct {
		Message string        `json:"message"`
		BaseURL string        `json:"baseURL"`
		Comment types.Comment `json:"comment"`
		Error   string        `json:"error,omitempty"`
	}{
		BaseURL: ws.BaseURL,
		Comment: types.Comment{
			Post: types.PostID(r.Vars["post-id"]),
			ID:   types.CommentID(r.Vars["comment-id"]),
		},
	}

	if context.Comment.ID != "toplevel" {
		comment, err := ws.Comments.Comment(
			context.Comment.Post,
			context.Comment.ID,
		)
		if err != nil {
			context.Message = "fetching comment"
			context.Error = err.Error()
			return pz.HandleError("fetching comment", err, &context)
		}
		context.Comment = *comment
	}

	return pz.Ok(pz.HTMLTemplate(replyTemplate, &context), &context)
}

func (ws *WebServer) Reply(r pz.Request) pz.Response {
	context := struct {
		Message  string          `json:"message,omitempty"`
		Post     types.PostID    `json:"post"`
		Comment  types.CommentID `json:"comment"`
		Author   types.UserID    `json:"author,omitempty"`
		Redirect string          `json:"redirect,omitempty"`
		Error    string          `json:"error,omitempty"`
	}{
		Post:    types.PostID(r.Vars["post-id"]),
		Comment: types.CommentID(r.Vars["comment-id"]),
		Author:  types.UserID(r.Headers.Get("User")),
	}

	if context.Comment == "toplevel" {
		context.Comment = ""
	}

	// limitreader = mitigate dos attack
	data, err := ioutil.ReadAll(io.LimitReader(r.Body, 2056))
	if err != nil {
		context.Message = "reading request body"
		context.Error = err.Error()
		return pz.InternalServerError(&context)
	}

	values, err := url.ParseQuery(string(data))
	if err != nil {
		context.Message = "parsing form values"
		context.Error = err.Error()
		return pz.BadRequest(nil, &context)
	}

	c, err := ws.Comments.Put(&types.Comment{
		Post:   context.Post,
		Parent: context.Comment,
		Author: context.Author,
		Body:   values.Get("body"),
	})
	if err != nil {
		return pz.HandleError("creating comment", err, &context)
	}

	context.Redirect = fmt.Sprintf(
		"%s/posts/%s/comments/toplevel/replies#%s",
		ws.BaseURL,
		context.Post,
		c.ID,
	)
	context.Message = "successfully created comment"
	return pz.SeeOther(context.Redirect, &context)
}

var editTemplate = html.Must(html.New("").Parse(`<html>
<head></head>
<body>
	<p>{{.Comment.Body}}</p>
	<form action="{{.BaseURL}}/posts/{{.Comment.Post}}/comments/{{.Comment.ID}}/edit" method="POST">
		<textarea name="body">{{.Comment.Body}}</textarea>
		<input type="submit" value="Submit">
	</form>
</body>
</html>`))

func (ws *WebServer) EditForm(r pz.Request) pz.Response {
	context := struct {
		Message string        `json:"message"`
		BaseURL string        `json:"baseURL"`
		Comment types.Comment `json:"comment"`
		Error   string        `json:"error,omitempty"`
	}{
		BaseURL: ws.BaseURL,
		Comment: types.Comment{
			Post: types.PostID(r.Vars["post-id"]),
			ID:   types.CommentID(r.Vars["comment-id"]),
		},
	}

	comment, err := ws.Comments.Comment(
		context.Comment.Post,
		context.Comment.ID,
	)
	if err != nil {
		context.Message = "fetching comment"
		context.Error = err.Error()
		return pz.HandleError("fetching comment", err, &context)
	}
	context.Comment = *comment

	return pz.Ok(pz.HTMLTemplate(editTemplate, &context), &context)
}

func (ws *WebServer) Edit(r pz.Request) pz.Response {
	var context = struct {
		Message       string `json:"message"`
		CommentUpdate `json:",inline"`
		Error         string `json:"error,omitempty"`
	}{
		CommentUpdate: CommentUpdate{
			Post: types.PostID(r.Vars["post-id"]),
			ID:   types.CommentID(r.Vars["comment-id"]),
		},
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		context.Message = "reading request body"
		context.Error = err.Error()
		return pz.InternalServerError(&context)
	}

	values, err := url.ParseQuery(string(data))
	if err != nil {
		context.Message = "parsing form values"
		context.Error = err.Error()
		return pz.BadRequest(nil, &context)
	}

	if !values.Has("body") {
		context.Message = "parsing form values"
		context.Error = "missing required field `body`"
		return pz.BadRequest(
			pz.JSON(&pz.HTTPError{
				Status:  http.StatusBadRequest,
				Message: "parsing form values: missing required field `body`",
			}),
			&context,
		)
	}

	context.Body = values.Get("body")
	if err := ws.Comments.Update(&context.CommentUpdate); err != nil {
		context.Error = err.Error()
		return pz.HandleError("updating comment", err, &context)
	}

	context.Message = "successfully updated comment"
	return pz.SeeOther(
		fmt.Sprintf(
			"%s/posts/%s/comments/toplevel/replies#%s",
			ws.BaseURL,
			context.Post,
			context.ID,
		),
		&context,
	)
}

func (ws *WebServer) RepliesRoute() pz.Route {
	return pz.Route{
		Method:  "GET",
		Path:    "/posts/{post-id}/comments/{comment-id}/replies",
		Handler: ws.Replies,
	}
}

func (ws *WebServer) DeleteConfirmRoute() pz.Route {
	return pz.Route{
		Method:  "GET",
		Path:    "/posts/{post-id}/comments/{comment-id}/delete-confirm",
		Handler: ws.DeleteConfirm,
	}
}

func (ws *WebServer) DeleteRoute() pz.Route {
	return pz.Route{
		Method:  "GET",
		Path:    "/posts/{post-id}/comments/{comment-id}/delete",
		Handler: ws.Delete,
	}
}

func (ws *WebServer) ReplyFormRoute() pz.Route {
	return pz.Route{
		Method:  "GET",
		Path:    "/posts/{post-id}/comments/{comment-id}/reply",
		Handler: ws.ReplyForm,
	}
}

func (ws *WebServer) ReplyRoute() pz.Route {
	return pz.Route{
		Method:  "POST",
		Path:    "/posts/{post-id}/comments/{comment-id}/reply",
		Handler: ws.Reply,
	}
}

func (ws *WebServer) EditFormRoute() pz.Route {
	return pz.Route{
		Method:  "GET",
		Path:    "/posts/{post-id}/comments/{comment-id}/edit",
		Handler: ws.EditForm,
	}
}

func (ws *WebServer) EditRoute() pz.Route {
	return pz.Route{
		Method:  "POST",
		Path:    "/posts/{post-id}/comments/{comment-id}/edit",
		Handler: ws.Edit,
	}
}

func (ws *WebServer) Routes() []pz.Route {
	return []pz.Route{
		ws.RepliesRoute(),
		ws.DeleteConfirmRoute(),
		ws.DeleteRoute(),
		ws.ReplyFormRoute(),
		ws.ReplyRoute(),
		ws.EditFormRoute(),
		ws.EditRoute(),
	}
}
