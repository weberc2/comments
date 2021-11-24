package testsupport

import (
	"github.com/weberc2/comments/pkg/types"
)

type CommentsStoreFake map[types.PostID]map[types.CommentID]*types.Comment

func (csf CommentsStoreFake) Put(
	c *types.Comment,
) (*types.Comment, error) {
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
