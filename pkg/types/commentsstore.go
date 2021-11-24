package types

import "fmt"

type CommentNotFoundErr struct {
	Post    PostID
	Comment CommentID
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
