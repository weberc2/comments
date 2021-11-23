package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
)

type PostNotFoundErr struct{ Post PostID }

func (err *PostNotFoundErr) Error() string {
	return fmt.Sprintf("post not found: %s", err.Post)
}

type PostStore interface {
	Exists(PostID) error
}

type CommentStore struct {
	ObjectStore ObjectStore
	PostStore   PostStore
	Bucket      string
	Prefix      string
	IDFunc      func() CommentID
}

func (cs *CommentStore) putObject(path string, data []byte) error {
	if err := cs.ObjectStore.PutObject(
		cs.Bucket,
		filepath.Join(cs.Prefix, path),
		bytes.NewReader(data),
	); err != nil {
		return fmt.Errorf("putting object: %w", err)
	}
	return nil
}

func (cs *CommentStore) getObject(key string) ([]byte, error) {
	body, err := cs.ObjectStore.GetObject(
		cs.Bucket,
		filepath.Join(cs.Prefix, key),
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

func (cs *CommentStore) putComment(c *Comment) error {
	data, err := json.Marshal(&c)
	if err != nil {
		return fmt.Errorf("marshaling comment: %w", err)
	}
	if err := cs.PostStore.Exists(c.Post); err != nil {
		return fmt.Errorf("checking post existence: %w", err)
	}

	// If a `parent` was provided, then make sure it exists
	if c.Parent != "" {
		if _, err := cs.Comment(c.Post, c.Parent); err != nil {
			return fmt.Errorf("getting parent comment: %w", err)
		}
	}
	if err := cs.putObject(
		fmt.Sprintf("posts/%s/comments/%s/__comment__", c.Post, c.ID),
		data,
	); err != nil {
		return fmt.Errorf("putting comment object: %w", err)
	}
	return nil
}

func (cs *CommentStore) putParentLink(c *Comment) error {
	parent := c.Parent
	if c.Parent == "" {
		parent = "__toplevel__"
	}
	return cs.putObject(
		fmt.Sprintf("posts/%s/comments/%s/comments/%s", c.Post, parent, c.ID),
		nil,
	)
}

func (cs *CommentStore) PutComment(c *Comment) (CommentID, error) {
	cp := *c
	cp.ID = cs.IDFunc()
	if err := cs.putComment(&cp); err != nil {
		return "", fmt.Errorf("putting comment: %w", err)
	}
	if err := cs.putParentLink(&cp); err != nil {
		return "", fmt.Errorf("putting parent link: %w", err)
	}
	return cp.ID, nil
}

func (cs *CommentStore) listObjects(prefix string) ([]string, error) {
	entries, err := cs.ObjectStore.ListObjects(
		cs.Bucket,
		filepath.Join(cs.Prefix, prefix),
	)
	if err != nil {
		return nil, fmt.Errorf("listing objects: %w", err)
	}
	return entries, nil
}

func (cs *CommentStore) getComment(key string) (Comment, error) {
	data, err := cs.getObject(key)
	if err != nil {
		return Comment{}, fmt.Errorf("getting object: %w", err)
	}
	var c Comment
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("marshaling comment: %w", err)
	}
	return c, nil
}

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

func (cs *CommentStore) Comment(post PostID, comment CommentID) (Comment, error) {
	key := fmt.Sprintf("posts/%s/comments/%s/__comment__", post, comment)
	c, err := cs.getComment(key)
	if err != nil {
		var e *ObjectNotFoundErr
		if errors.As(err, &e) {
			return Comment{}, fmt.Errorf(
				"getting comment: %w",
				&CommentNotFoundErr{Post: post, Comment: comment},
			)
		}
		return Comment{}, fmt.Errorf("getting comment: %w", err)
	}
	return c, nil
}

func (cs *CommentStore) Replies(post PostID, parent CommentID) ([]Comment, error) {
	if parent == "" {
		parent = "__toplevel__"
	}

	prefix := fmt.Sprintf("posts/%s/comments/%s/comments/", post, parent)
	keys, err := cs.listObjects(prefix)
	if err != nil {
		return nil, fmt.Errorf(
			"listing objects with prefix '%s': %w",
			prefix,
			err,
		)
	}

	comments := make([]Comment, len(keys))
	for i, key := range keys {
		comment, err := cs.Comment(post, CommentID(filepath.Base(key)))
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

func (cs *CommentStore) Delete(post PostID, comment CommentID) error {
	// To avoid dangling pointers, delete the pointer first and then the
	// comment object itself.

	c, err := cs.Comment(post, comment)
	if err != nil {
		var e *ObjectNotFoundErr
		if errors.As(err, &e) {
			return fmt.Errorf(
				"getting comment: %w",
				&CommentNotFoundErr{Post: post, Comment: comment},
			)
		}
		return fmt.Errorf("deleting comment: %w", err)
	}

	parent := c.Parent
	if c.Parent == "" {
		parent = "__toplevel__"
	}
	if err := cs.ObjectStore.DeleteObject(
		cs.Bucket,
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

	if err := cs.ObjectStore.DeleteObject(
		cs.Bucket,
		fmt.Sprintf("posts/%s/comments/%s/__comment__", post, comment),
	); err != nil {
		var e *ObjectNotFoundErr
		if errors.As(err, &e) {
			return fmt.Errorf(
				"getting comment: %w",
				&CommentNotFoundErr{Post: post, Comment: comment},
			)
		}
		return fmt.Errorf("deleting comment: %w", err)
	}
	return nil
}
