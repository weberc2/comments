package comments

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"time"

	pz "github.com/weberc2/httpeasy"
)

const (
	bodySizeMin = 8
	bodySizeMax = 2056
)

type HTTPError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

var (
	ErrBodyTooShort = HTTPError{400, "Comment body too short"}
	ErrBodyTooLong  = HTTPError{400, "Comment body too long"}
)

type CommentsService struct {
	Comments CommentsStore
	TimeFunc func() time.Time
}

func (cs *CommentsService) PutComment(r pz.Request) pz.Response {
	var c Comment
	if err := r.JSON(&c); err != nil {
		return pz.BadRequest(pz.String("Malformed `Comment` JSON"), e{err})
	}
	const bodySizeMin = 8

	type msg struct {
		Message string
	}
	if len(c.Body) < bodySizeMin {
		return pz.BadRequest(
			pz.JSON(ErrBodyTooShort),
			msg{fmt.Sprintf(
				"wanted len(body) >= %d; found %d",
				bodySizeMin,
				len(c.Body),
			)},
		)
	}
	if len(c.Body) > bodySizeMax {
		return pz.BadRequest(
			pz.JSON(ErrBodyTooLong),
			msg{fmt.Sprintf(
				"wanted len(body) <= %d; found %d",
				bodySizeMax,
				len(c.Body),
			)},
		)
	}
	c.Post = PostID(r.Vars["post-id"])
	c.Body = html.EscapeString(c.Body)
	c.Author = UserID(r.Headers.Get("User"))
	c.Created = cs.TimeFunc().UTC()
	c.Modified = c.Created
	comment, err := cs.Comments.Put(&c)
	if err != nil {
		return pz.InternalServerError(e{err})
	}
	return pz.Created(pz.JSON(comment), struct {
		Message string
		Comment *Comment
	}{
		Message: "Created comment",
		Comment: comment,
	})
}

func (cs *CommentsService) PostComments(r pz.Request) pz.Response {
	var parent CommentID
	if commentID := r.Vars["comment-id"]; commentID != "toplevel" {
		parent = CommentID(commentID)
	}
	comments, err := cs.Comments.Replies(PostID(r.Vars["post-id"]), parent)
	if err != nil {
		var c *CommentNotFoundErr
		if errors.As(err, &c) {
			return pz.NotFound(
				pz.Stringf(
					"comment '%s' not found with post '%s'",
					r.Vars["comment-id"],
					r.Vars["post-id"],
				),
				struct {
					Message string
					Err     error
				}{
					Message: "comment not found",
					Err:     err,
				},
			)
		}
		return pz.InternalServerError(e{err})
	}
	return pz.Ok(pz.JSON(comments))
}

func (cs *CommentsService) GetComment(r pz.Request) pz.Response {
	comment, err := cs.Comments.Comment(
		PostID(r.Vars["post-id"]),
		CommentID(r.Vars["comment-id"]),
	)
	if err != nil {
		var c *CommentNotFoundErr
		if errors.As(err, &c) {
			return pz.NotFound(
				pz.Stringf(
					"comment '%s' not found with post '%s'",
					r.Vars["comment-id"],
					r.Vars["post-id"],
				),
				struct {
					Message string
					Err     error
				}{
					Message: "comment not found",
					Err:     err,
				},
			)
		}
		return pz.InternalServerError(e{err})
	}
	return pz.Ok(pz.JSON(comment))
}

type e struct {
	err error
}

func (e e) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct{ Err string }{e.err.Error()})
}
