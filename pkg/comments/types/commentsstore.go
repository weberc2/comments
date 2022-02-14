package types

import (
	"net/http"

	pz "github.com/weberc2/httpeasy"
)

var (
	ErrCommentExists = &pz.HTTPError{
		Status:  http.StatusConflict,
		Message: "comment exists",
	}
	ErrCommentNotFound = &pz.HTTPError{
		Status:  http.StatusNotFound,
		Message: "comment not found",
	}
)

type CommentsStore interface {
	Put(*Comment) error
	Comment(PostID, CommentID) (*Comment, error)
	Replies(PostID, CommentID) ([]*Comment, error)
	Delete(PostID, CommentID) error
	Update(*CommentPatch) error
}
