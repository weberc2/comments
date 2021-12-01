package comments

import (
	"time"

	"html"

	"github.com/weberc2/comments/pkg/types"
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
