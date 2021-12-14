package comments

import (
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

	c.Post = types.PostID(r.Vars["post-id"])
	c.Author = types.UserID(r.Headers.Get("User"))
	c.Created = cs.TimeFunc().UTC()
	c.Modified = c.Created
	comment, err := cs.Comments.Put(&c)
	if err != nil {
		return pz.HandleError("putting comment", err)
	}
	return pz.Created(pz.JSON(comment), struct {
		Message string
		Comment *types.Comment
	}{
		Message: "Created comment",
		Comment: comment,
	})
}

func (cs *CommentsService) Replies(r pz.Request) pz.Response {
	var parent types.CommentID
	if commentID := r.Vars["comment-id"]; commentID != "toplevel" {
		parent = types.CommentID(commentID)
	}
	comments, err := cs.Comments.Replies(
		types.PostID(r.Vars["post-id"]),
		parent,
	)
	if err != nil {
		return pz.HandleError("retrieving comment replies", err)
	}
	return pz.Ok(pz.JSON(comments))
}

func (cs *CommentsService) GetComment(r pz.Request) pz.Response {
	comment, err := cs.Comments.Comment(
		types.PostID(r.Vars["post-id"]),
		types.CommentID(r.Vars["comment-id"]),
	)
	if err != nil {
		return pz.HandleError("retrieving comment", err)
	}
	return pz.Ok(pz.JSON(comment))
}
