package comments

import (
	"time"

	"github.com/weberc2/comments/pkg/types"
	"html"
)

const (
	bodySizeMin = 8
	bodySizeMax = 2056
)

var (
	ErrInvalidPost  = &types.HTTPError{Status: 400, Message: "invalid post"}
	ErrBodyTooShort = &types.HTTPError{Status: 400, Message: "body too short"}
	ErrBodyTooLong  = &types.HTTPError{Status: 400, Message: "body too long"}
)

type CommentsModel struct {
	types.CommentsStore
	IDFunc   func() types.CommentID
	TimeFunc func() time.Time
}

func (cm *CommentsModel) Put(c *types.Comment) (*types.Comment, error) {
	if c.Post == "" {
		return nil, ErrInvalidPost
	}
	if len(c.Body) < bodySizeMin {
		return nil, ErrBodyTooShort
	}
	if len(c.Body) > bodySizeMax {
		return nil, ErrBodyTooLong
	}
	now := cm.TimeFunc()
	cp := *c
	cp.ID = cm.IDFunc()
	cp.Created = now
	cp.Modified = now
	cp.Body = html.EscapeString(c.Body)
	return cm.CommentsStore.Put(&cp)
}