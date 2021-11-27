package types

import "fmt"

type CommentNotFoundErr struct {
	Post    PostID
	Comment CommentID
}

func (err *CommentNotFoundErr) HTTPError() *HTTPError {
	return &HTTPError{Status: 404, Message: "comment not found"}
}

func (err *CommentNotFoundErr) Error() string {
	return fmt.Sprintf(
		"comment not found: post=%s comment=%s",
		err.Post,
		err.Comment,
	)
}

type CommentsStore interface {
	Put(*Comment) (*Comment, error)
	Comment(PostID, CommentID) (*Comment, error)
	Replies(PostID, CommentID) ([]*Comment, error)
	Delete(PostID, CommentID) error
}

// fail compilation if `CommentNotFoundErr` doesn't satisfy the `Error`
// interface.
var _ Error = &CommentNotFoundErr{}
