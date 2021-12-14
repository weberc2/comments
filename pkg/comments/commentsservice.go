package comments

import (
	"encoding/json"
	"fmt"
	"net/http"
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

type DeleteCommentResponse struct {
	Message string          `json:"message"`
	Post    types.PostID    `json:"post"`
	Comment types.CommentID `json:"comment"`
	Status  int             `json:"status"`
}

func (wanted *DeleteCommentResponse) Compare(
	found *DeleteCommentResponse,
) error {
	if wanted == found {
		return nil
	}
	if wanted == nil && found != nil {
		return fmt.Errorf("DeleteCommentResponse: %w", types.ErrWantedNil)
	}
	if wanted != nil && found == nil {
		return fmt.Errorf("DeleteCommentResponse: %w", types.ErrWantedNotNil)
	}

	if wanted.Message != found.Message {
		return fmt.Errorf(
			"DeleteCommentResponse.Message: wanted `%s`; found `%s`",
			wanted.Message,
			found.Message,
		)
	}

	if wanted.Post != found.Post {
		return fmt.Errorf(
			"DeleteCommentResponse.Post: wanted `%s`; found `%s`",
			wanted.Post,
			found.Post,
		)
	}

	if wanted.Comment != found.Comment {
		return fmt.Errorf(
			"DeleteCommentResponse.Comment: wanted `%s`; found `%s`",
			wanted.Comment,
			found.Comment,
		)
	}

	if wanted.Status != found.Status {
		return fmt.Errorf(
			"DeleteStatusResponse.Status: wanted `%d`; found `%d`",
			wanted.Status,
			found.Status,
		)
	}

	return nil
}

func (rsp *DeleteCommentResponse) CompareData(data []byte) error {
	var other DeleteCommentResponse
	if err := json.Unmarshal(data, &other); err != nil {
		return fmt.Errorf("unmarshaling `DeleteCommentResponse`: %w", err)
	}
	return nil
}

func (cs *CommentsService) DeleteComment(r pz.Request) pz.Response {
	post := types.PostID(r.Vars["post-id"])
	comment := types.CommentID(r.Vars["comment-id"])
	if err := cs.Comments.Delete(post, comment); err != nil {
		return pz.HandleError("deleting comment", err)
	}
	rsp := DeleteCommentResponse{
		Message: "comment deleted",
		Post:    post,
		Comment: comment,
		Status:  http.StatusOK,
	}
	return pz.Ok(pz.JSON(&rsp), &rsp)
}
