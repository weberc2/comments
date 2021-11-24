package comments

import (
	"errors"
	"time"

	"github.com/weberc2/comments/pkg/types"
	pz "github.com/weberc2/httpeasy"
)

type CommentsService struct {
	Comments CommentsModel
	TimeFunc func() time.Time
}

func (cs *CommentsService) PutComment(r pz.Request) pz.Response {
	var c types.Comment
	if err := r.JSON(&c); err != nil {
		return pz.BadRequest(
			pz.String("Malformed `Comment` JSON"),
			struct {
				Error string `json:"error"`
			}{
				Error: err.Error(),
			},
		)
	}

	type msg struct {
		Message string
	}
	c.Post = types.PostID(r.Vars["post-id"])
	c.Author = types.UserID(r.Headers.Get("User"))
	c.Created = cs.TimeFunc().UTC()
	c.Modified = c.Created
	comment, err := cs.Comments.Put(&c)
	if err != nil {
		return handle("putting comment", err)
	}
	return pz.Created(pz.JSON(comment), struct {
		Message string
		Comment *types.Comment
	}{
		Message: "Created comment",
		Comment: comment,
	})
}

func (cs *CommentsService) PostComments(r pz.Request) pz.Response {
	var parent types.CommentID
	if commentID := r.Vars["comment-id"]; commentID != "toplevel" {
		parent = types.CommentID(commentID)
	}
	comments, err := cs.Comments.Replies(
		types.PostID(r.Vars["post-id"]),
		parent,
	)
	if err != nil {
		return handle("retrieving comment replies", err)
	}
	return pz.Ok(pz.JSON(comments))
}

func (cs *CommentsService) GetComment(r pz.Request) pz.Response {
	comment, err := cs.Comments.Comment(
		types.PostID(r.Vars["post-id"]),
		types.CommentID(r.Vars["comment-id"]),
	)
	if err != nil {
		return handle("retrieving comment", err)
	}
	return pz.Ok(pz.JSON(comment))
}

func handle(message string, err error, logging ...interface{}) pz.Response {
	logging = append(logging, struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}{
		Message: message,
		Error:   err.Error(),
	})

	cause := err
	for {
		if unwrapped := errors.Unwrap(cause); unwrapped != nil {
			cause = unwrapped
			continue
		}
		break
	}

	if e, ok := cause.(types.Error); ok {
		httpErr := e.HTTPError()
		return pz.Response{
			Status:  httpErr.Status,
			Data:    pz.JSON(httpErr),
			Logging: logging,
		}
	}

	return pz.InternalServerError(logging)
}
