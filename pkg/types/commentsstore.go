package types

import (
	"errors"
	"fmt"

	pz "github.com/weberc2/httpeasy"
)

type CommentExistsErr struct {
	Post    PostID
	Comment CommentID
}

func (err *CommentExistsErr) HTTPError() *pz.HTTPError {
	return &pz.HTTPError{Status: 409, Message: "comment exists"}
}

func (err *CommentExistsErr) Error() string {
	return fmt.Sprintf(
		"comment exists: post=%s comment=%s",
		err.Post,
		err.Comment,
	)
}

func (wanted *CommentExistsErr) CompareErr(err error) error {
	var other *CommentExistsErr
	if errors.As(err, &other) {
		return wanted.Compare(other)
	}
	return fmt.Errorf(
		"Wanted `*types.CommentExistsErr`; found `%T`: %v",
		err,
		err,
	)
}

func (wanted *CommentExistsErr) Compare(other *CommentExistsErr) error {
	if wanted == other {
		return nil
	}

	if wanted == nil && other != nil {
		return fmt.Errorf("wanted `nil`; found not-nil")
	}

	if wanted != nil && other == nil {
		return fmt.Errorf("wanted not-nil; found `nil`")
	}

	if wanted.Post != other.Post {
		return fmt.Errorf(
			"CommentExistsErr.Post: wanted `%s`; found `%s`",
			wanted.Post,
			other.Post,
		)
	}

	if wanted.Comment != other.Comment {
		return fmt.Errorf(
			"CommentExistsErr.Comment: wanted `%s`; found `%s`",
			wanted.Comment,
			other.Comment,
		)
	}

	return nil
}

type CommentNotFoundErr struct {
	Post    PostID
	Comment CommentID
}

func (err *CommentNotFoundErr) HTTPError() *pz.HTTPError {
	return &pz.HTTPError{Status: 404, Message: "comment not found"}
}

func (err *CommentNotFoundErr) Error() string {
	return fmt.Sprintf(
		"comment not found: post=%s comment=%s",
		err.Post,
		err.Comment,
	)
}

func (wanted *CommentNotFoundErr) CompareErr(err error) error {
	var other *CommentNotFoundErr
	if errors.As(err, &other) {
		return wanted.Compare(other)
	}
	return fmt.Errorf(
		"Wanted `*types.CommentNotFoundErr`; found `%T`: %v",
		err,
		err,
	)
}

func (wanted *CommentNotFoundErr) Compare(other *CommentNotFoundErr) error {
	if wanted == other {
		return nil
	}

	if wanted == nil && other != nil {
		return fmt.Errorf("wanted `nil`; found not-nil")
	}

	if wanted != nil && other == nil {
		return fmt.Errorf("wanted not-nil; found `nil`")
	}

	if wanted.Post != other.Post {
		return fmt.Errorf(
			"CommentNotFoundErr.Post: wanted `%s`; found `%s`",
			wanted.Post,
			other.Post,
		)
	}

	if wanted.Comment != other.Comment {
		return fmt.Errorf(
			"CommentNotFoundErr.Comment: wanted `%s`; found `%s`",
			wanted.Comment,
			other.Comment,
		)
	}

	return nil
}

type CommentsStore interface {
	Put(*Comment) error
	Comment(PostID, CommentID) (*Comment, error)
	Replies(PostID, CommentID) ([]*Comment, error)
	Delete(PostID, CommentID) error
	Update(*CommentPatch) error
}

// fail compilation if `CommentNotFoundErr` doesn't satisfy the `Error` and
// `WantedError` interfaces.
var _ pz.Error = &CommentNotFoundErr{}
var _ WantedError = &CommentNotFoundErr{}
