package comments

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"

	"github.com/weberc2/comments/pkg/types"
)

type ObjectCommentsStore struct {
	ObjectStore types.ObjectStore
	Bucket      string
	Prefix      string
}

func (ocs *ObjectCommentsStore) putObject(path string, data []byte) error {
	if err := ocs.ObjectStore.PutObject(
		ocs.Bucket,
		filepath.Join(ocs.Prefix, path),
		bytes.NewReader(data),
	); err != nil {
		return fmt.Errorf("putting object: %w", err)
	}
	return nil
}

func (ocs *ObjectCommentsStore) getObject(key string) ([]byte, error) {
	body, err := ocs.ObjectStore.GetObject(
		ocs.Bucket,
		filepath.Join(ocs.Prefix, key),
	)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("reading object body: %w", err)
	}
	return data, nil
}

func (ocs *ObjectCommentsStore) putComment(c *types.Comment) error {
	data, err := json.Marshal(&c)
	if err != nil {
		return fmt.Errorf("marshaling comment: %w", err)
	}

	// If a `parent` was provided, then make sure it exists
	if c.Parent != "" {
		if _, err := ocs.Comment(c.Post, c.Parent); err != nil {
			return fmt.Errorf("getting parent comment: %w", err)
		}
	}
	if err := ocs.putObject(
		fmt.Sprintf("posts/%s/comments/%s/__comment__", c.Post, c.ID),
		data,
	); err != nil {
		return fmt.Errorf("putting comment object: %w", err)
	}
	return nil
}

func (ocs *ObjectCommentsStore) putParentLink(c *types.Comment) error {
	parent := c.Parent
	if c.Parent == "" {
		parent = "__toplevel__"
	}
	return ocs.putObject(
		fmt.Sprintf("posts/%s/comments/%s/comments/%s", c.Post, parent, c.ID),
		nil,
	)
}

func (ocs *ObjectCommentsStore) Put(c *types.Comment) (*types.Comment, error) {
	cp := *c
	if err := ocs.putComment(&cp); err != nil {
		return nil, fmt.Errorf("putting comment: %w", err)
	}
	if err := ocs.putParentLink(&cp); err != nil {
		return nil, fmt.Errorf("putting parent link: %w", err)
	}
	return &cp, nil
}

func (ocs *ObjectCommentsStore) listObjects(prefix string) ([]string, error) {
	entries, err := ocs.ObjectStore.ListObjects(
		ocs.Bucket,
		filepath.Join(ocs.Prefix, prefix),
	)
	if err != nil {
		return nil, fmt.Errorf("listing objects: %w", err)
	}
	return entries, nil
}

func (ocs *ObjectCommentsStore) getComment(key string) (types.Comment, error) {
	data, err := ocs.getObject(key)
	if err != nil {
		return types.Comment{}, fmt.Errorf("getting object: %w", err)
	}
	var c types.Comment
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("marshaling comment: %w", err)
	}
	return c, nil
}

func (ocs *ObjectCommentsStore) Comment(
	post types.PostID,
	comment types.CommentID,
) (*types.Comment, error) {
	key := fmt.Sprintf("posts/%s/comments/%s/__comment__", post, comment)
	c, err := ocs.getComment(key)
	if err != nil {
		var e *types.ObjectNotFoundErr
		if errors.As(err, &e) {
			return nil, fmt.Errorf(
				"getting comment: %w",
				&types.CommentNotFoundErr{Post: post, Comment: comment},
			)
		}
		return nil, fmt.Errorf("getting comment: %w", err)
	}
	return &c, nil
}

func (ocs *ObjectCommentsStore) Replies(
	post types.PostID,
	comment types.CommentID,
) ([]*types.Comment, error) {
	if comment == "" {
		comment = "__toplevel__"
	}

	prefix := fmt.Sprintf("posts/%s/comments/%s/comments/", post, comment)
	keys, err := ocs.listObjects(prefix)
	if err != nil {
		return nil, fmt.Errorf(
			"listing objects with prefix '%s': %w",
			prefix,
			err,
		)
	}

	comments := make([]*types.Comment, len(keys))
	for i, key := range keys {
		comment, err := ocs.Comment(post, types.CommentID(filepath.Base(key)))
		if err != nil {
			return nil, fmt.Errorf("getting comment: %w", err)
		}
		comments[i] = comment
	}

	sort.Slice(comments, func(i, j int) bool {
		return comments[i].Created.Before(comments[j].Created)
	})

	return comments, nil
}

func (ocs *ObjectCommentsStore) Delete(
	post types.PostID,
	comment types.CommentID,
) error {
	// To avoid dangling pointers, delete the pointer first and then the
	// comment object itself.

	c, err := ocs.Comment(post, comment)
	if err != nil {
		var e *types.ObjectNotFoundErr
		if errors.As(err, &e) {
			return fmt.Errorf(
				"getting comment: %w",
				&types.CommentNotFoundErr{Post: post, Comment: comment},
			)
		}
		return fmt.Errorf("deleting comment: %w", err)
	}

	parent := c.Parent
	if c.Parent == "" {
		parent = "__toplevel__"
	}
	if err := ocs.ObjectStore.DeleteObject(
		ocs.Bucket,
		fmt.Sprintf("posts/%s/comments/%s/comments/%s", post, parent, c.ID),
	); err != nil {
		log.Printf(
			`{"message": "parent link not found", "post": "%s", "parent": "%s", "comment": "%s", "error": "%s"}`,
			post,
			parent,
			comment,
			err.Error(),
		)
	}

	if err := ocs.ObjectStore.DeleteObject(
		ocs.Bucket,
		fmt.Sprintf("posts/%s/comments/%s/__comment__", post, comment),
	); err != nil {
		var e *types.ObjectNotFoundErr
		if errors.As(err, &e) {
			return fmt.Errorf(
				"getting comment: %w",
				&types.CommentNotFoundErr{Post: post, Comment: comment},
			)
		}
		return fmt.Errorf("deleting comment: %w", err)
	}
	return nil
}
