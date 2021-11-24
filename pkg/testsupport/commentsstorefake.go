package testsupport

import (
	"fmt"

	"github.com/weberc2/comments/pkg/types"
)

type CommentsStoreFake map[types.PostID]map[types.CommentID]*types.Comment

func (csf CommentsStoreFake) Put(
	c *types.Comment,
) (*types.Comment, error) {
	if postComments := csf[c.Post]; postComments == nil {
		csf[c.Post] = map[types.CommentID]*types.Comment{c.ID: c}
		return c, nil
	}
	csf[c.Post][c.ID] = c
	return c, nil
}

func (csf CommentsStoreFake) Comment(
	post types.PostID,
	comment types.CommentID,
) (*types.Comment, error) {
	postComments, found := csf[post]
	if !found {
		return nil, &types.CommentNotFoundErr{
			Post:    post,
			Comment: comment,
		}
	}
	c, found := postComments[comment]
	if !found {
		return nil, &types.CommentNotFoundErr{
			Post:    post,
			Comment: comment,
		}
	}
	return c, nil
}

func (csf CommentsStoreFake) Replies(
	post types.PostID,
	comment types.CommentID,
) ([]*types.Comment, error) {
	postComments, found := csf[post]
	if !found {
		return nil, &types.CommentNotFoundErr{
			Post:    post,
			Comment: comment,
		}
	}
	var replies []*types.Comment
	for _, c := range postComments {
		if c.Parent == comment {
			replies = append(replies, c)
		}
	}
	return replies, nil
}

func (csf CommentsStoreFake) Delete(
	post types.PostID,
	comment types.CommentID,
) error {
	postComments, found := csf[post]
	if !found {
		return &types.CommentNotFoundErr{
			Post:    post,
			Comment: comment,
		}
	}
	if _, found := postComments[comment]; !found {
		return &types.CommentNotFoundErr{
			Post:    post,
			Comment: comment,
		}
	}
	delete(postComments, comment)
	return nil
}

func (csf CommentsStoreFake) Contains(comments ...*types.Comment) error {
	for i, comment := range comments {
		found, err := csf.Comment(comment.Post, comment.ID)
		if err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}
		if err := comment.Compare(found); err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}

	}
	return nil
}

func (csf CommentsStoreFake) Comments() []*types.Comment {
	var out []*types.Comment
	for _, comments := range csf {
		for _, comment := range comments {
			out = append(out, comment)
		}
	}
	return out
}
