package testsupport

import (
	"fmt"

	"github.com/weberc2/comments/pkg/types"
)

type CommentsStoreFake map[types.PostID]map[types.CommentID]*types.Comment

func (csf CommentsStoreFake) Put(c *types.Comment) error {
	if postComments := csf[c.Post]; postComments != nil {
		if _, found := postComments[c.ID]; !found {
			csf[c.Post][c.ID] = c
			return nil
		}
		return types.ErrCommentExists
	}
	csf[c.Post] = map[types.CommentID]*types.Comment{c.ID: c}
	return nil
}

func (csf CommentsStoreFake) Comment(
	post types.PostID,
	comment types.CommentID,
) (*types.Comment, error) {
	postComments, found := csf[post]
	if !found {
		return nil, types.ErrCommentNotFound
	}
	c, found := postComments[comment]
	if !found {
		return nil, types.ErrCommentNotFound
	}
	return c, nil
}

func (csf CommentsStoreFake) Replies(
	post types.PostID,
	comment types.CommentID,
) ([]*types.Comment, error) {
	postComments, found := csf[post]
	if !found {
		return nil, types.ErrCommentNotFound
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
		return types.ErrCommentNotFound
	}
	if _, found := postComments[comment]; !found {
		return types.ErrCommentNotFound
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

func (csf CommentsStoreFake) List() []*types.Comment {
	var out []*types.Comment
	for _, comments := range csf {
		for _, comment := range comments {
			out = append(out, comment)
		}
	}
	return out
}

func (csf CommentsStoreFake) Update(patch *types.CommentPatch) error {
	if !patch.IsSet(types.FieldID) {
		return fmt.Errorf(
			"`CommentPatch` is missing required field `%s`",
			types.FieldID,
		)
	}
	if !patch.IsSet(types.FieldPost) {
		return fmt.Errorf(
			"`CommentPatch` is missing required field `%s`",
			types.FieldPost,
		)
	}

	if comments, found := csf[patch.Post()]; found {
		if comment, found := comments[patch.ID()]; found {
			patch.Apply(comment)
			return nil
		}
		// fallthrough to types.CommentNotFoundErr{} below
	}
	return types.ErrCommentNotFound
}

func (csf CommentsStoreFake) Compare(other CommentsStoreFake) error {
	if csf == nil && other == nil {
		return nil
	}

	if csf == nil && other != nil {
		return fmt.Errorf("wanted `nil`; found not-nil")
	}

	if csf != nil && other == nil {
		return fmt.Errorf("wanted not-nil; found `nil`")
	}

	if err := types.CompareComments(csf.List(), other.List()); err != nil {
		return fmt.Errorf("CommentsStoreFake.Compare(): %w", err)
	}

	return nil
}
