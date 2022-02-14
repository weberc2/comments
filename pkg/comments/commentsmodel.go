package comments

import (
	"fmt"
	"time"

	"html"

	"github.com/weberc2/comments/pkg/comments/types"
	pz "github.com/weberc2/httpeasy"
)

const (
	bodySizeMin = 8
	bodySizeMax = 2056
)

var (
	ErrInvalidPost  = &pz.HTTPError{Status: 400, Message: "invalid post"}
	ErrBodyTooShort = &pz.HTTPError{Status: 400, Message: "body too short"}
	ErrBodyTooLong  = &pz.HTTPError{Status: 400, Message: "body too long"}
)

type CommentsModel struct {
	types.CommentsStore
	IDFunc   func() types.CommentID
	TimeFunc func() time.Time
}

func validateCommentBody(body string) error {
	if len(body) < bodySizeMin {
		return ErrBodyTooShort
	}
	if len(body) > bodySizeMax {
		return ErrBodyTooLong
	}
	return nil
}

func (cm *CommentsModel) Put(c *types.Comment) (*types.Comment, error) {
	if c.Post == "" {
		return nil, ErrInvalidPost
	}
	if err := validateCommentBody(c.Body); err != nil {
		return nil, err
	}

	if c.Parent != "" {
		parent, err := cm.CommentsStore.Comment(
			c.Post,
			c.Parent,
		)
		if err != nil {
			return nil, fmt.Errorf("fetching parent comment: %w", err)
		}
		if parent.Deleted {
			return nil, fmt.Errorf(
				"fetching parent comment: %w",
				types.ErrCommentNotFound,
			)
		}
	}
	now := cm.TimeFunc()
	cp := *c
	cp.ID = cm.IDFunc()
	cp.Created = now
	cp.Modified = now
	cp.Deleted = false
	cp.Body = html.EscapeString(c.Body)
	if err := cm.CommentsStore.Put(&cp); err != nil {
		return nil, err
	}
	return &cp, nil
}

func (cm *CommentsModel) Delete(p types.PostID, c types.CommentID) error {
	if err := cm.CommentsStore.Update(
		types.NewCommentPatch(c, p).SetDeleted(true).
			SetModified(cm.TimeFunc()),
	); err != nil {
		return fmt.Errorf("soft-deleting comment: %w", err)
	}
	return nil
}

func (cm *CommentsModel) Replies(
	post types.PostID,
	parent types.CommentID,
) ([]*types.Comment, error) {
	comments, err := cm.CommentsStore.Replies(post, parent)
	if err != nil {
		return nil, fmt.Errorf("fetching comment replies: %w", err)
	}

	for _, comment := range comments {
		if comment.Deleted {
			// Redact author and body fields from deleted comments
			comment.Author = ""
			comment.Body = ""
		}
	}

	return comments, nil
}

type CommentUpdate struct {
	ID   types.CommentID `json:"comment"`
	Post types.PostID    `json:"post"`
	Body string          `json:"body"`
}

func (cm *CommentsModel) Update(update *CommentUpdate) error {
	if err := validateCommentBody(update.Body); err != nil {
		return fmt.Errorf("updating comment: %w", err)
	}

	c, err := cm.CommentsStore.Comment(update.Post, update.ID)
	if err != nil {
		return fmt.Errorf("updating comment: %w", err)
	}
	if c.Deleted {
		return fmt.Errorf("updating comment: %w", types.ErrCommentNotFound)
	}
	return cm.CommentsStore.Update(
		types.NewCommentPatch(update.ID, update.Post).
			SetBody(update.Body).
			SetModified(cm.TimeFunc()),
	)
}
