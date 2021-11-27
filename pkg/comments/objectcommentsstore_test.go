package comments

import (
	"errors"
	"testing"

	"github.com/weberc2/comments/pkg/testsupport"
	"github.com/weberc2/comments/pkg/types"
)

func TestObjectCommentsStore_Put_ParentNotFound(t *testing.T) {
	commentStore := ObjectCommentsStore{
		ObjectStore: testsupport.ObjectStoreFake{},
	}

	// When a comment is added to "my-post" with a `Parent` that doesn't exist
	_, err := commentStore.Put(&types.Comment{
		Post:   "my-post",
		Parent: "doesnt-exist",
	})

	// Then expect a `types.CommentNotFoundErr` is returned
	var cnfe *types.CommentNotFoundErr
	if errors.As(err, &cnfe) {
		if cnfe.Post == "my-post" && cnfe.Comment == "doesnt-exist" {
			return
		}
		err = cnfe
	}
	t.Fatalf(
		`Wanted types.CommentNotFoundErr{Post: "my-post", Comment: "doesnt-exist"}; found %# v`,
		err,
	)
}
