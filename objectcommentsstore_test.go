package main

import (
	"errors"
	"testing"
)

type postStoreFake []PostID

func (psf postStoreFake) Exists(post PostID) error {
	for i := range psf {
		if psf[i] == post {
			return nil
		}
	}
	return &PostNotFoundErr{post}
}

func TestCommentStore_Put_ParentNotFound(t *testing.T) {
	// Given a comment store with a post called "my-post"
	commentStore := ObjectCommentsStore{
		ObjectStore: objectStoreFake{},
		PostStore:   postStoreFake{"my-post"},
		IDFunc:      func() CommentID { return "" },
	}

	// When a comment is added to "my-post" with a `Parent` that doesn't exist
	_, err := commentStore.Put(&Comment{
		Post:   "my-post",
		Parent: "doesnt-exist",
	})

	// Then expect a `CommentNotFoundErr` is returned
	var cnfe *CommentNotFoundErr
	if errors.As(err, &cnfe) {
		if cnfe.Post == "my-post" && cnfe.Comment == "doesnt-exist" {
			return
		}
		err = cnfe
	}
	t.Fatalf(
		`Wanted CommentNotFoundErr{Post: "my-post", Comment: "doesnt-exist"}; found %# v`,
		err,
	)
}

func TestCommentStore_Put_PostNotFound(t *testing.T) {
	// Given a comment store with no posts
	commentStore := ObjectCommentsStore{
		ObjectStore: objectStoreFake{},
		PostStore:   postStoreFake{},
		IDFunc:      func() CommentID { return "" },
	}

	// When a comment is added on an unknown post
	_, err := commentStore.Put(&Comment{Post: "my-post"})

	// Then expect a `PostNotFoundErr` is returned
	var pnfe *PostNotFoundErr
	if errors.As(err, &pnfe) {
		if pnfe.Post == "my-post" {
			return
		}
		err = pnfe
	}
	t.Fatalf(`Wanted PostNotFoundErr{Post: "my-post"}; found %# v`, err)
}

func TestCOmmentStore_ListComments_PostNotFound(t *testing.T) {
	// Given a comment store with no posts
	commentStore := ObjectCommentsStore{
		ObjectStore: objectStoreFake{},
		PostStore:   postStoreFake{},
		IDFunc:      func() CommentID { return "" },
	}

	// When someone tries to list comments on a post that doesn't exist
	posts, err := commentStore.Replies("my-post", "")

	// Then expect an empty list is returned (given the S3 schema at the time
	// of this writing, we have no mechanism to distinguish between "post has
	// no comments" and "post doesn't exist" and it's unclear what utility
	// there is in making the distinction in the first place)
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}

	if len(posts) > 0 {
		t.Fatalf(
			"len(CommentStore.PostComments()): wanted `0`; found `%d`",
			len(posts),
		)
	}
}
