package types

import (
	"encoding/json"
	"fmt"
	"time"
)

type PostID string
type CommentID string
type UserID string

type Comment struct {
	ID       CommentID `json:"id"`
	Post     PostID    `json:"post"`
	Parent   CommentID `json:"parent"`
	Author   UserID    `json:"author"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Body     string    `json:"body"`
}

func (c *Comment) compare(other *Comment) error {
	if c == nil && other == nil {
		return nil
	}

	if c != nil && other == nil {
		return fmt.Errorf("Comment: unexpected `nil`")
	}

	if c == nil && other != nil {
		return fmt.Errorf("Comment: wanted `nil`; found not-nil")
	}

	if c.ID != other.ID {
		return fmt.Errorf(
			"Comment.ID: wanted `%s`; found `%s`",
			c.ID,
			other.ID,
		)
	}

	if c.Author != other.Author {
		return fmt.Errorf(
			"Comment.Author: wanted `%s`; found `%s`",
			c.Author,
			other.Author,
		)
	}

	if c.Parent != other.Parent {
		return fmt.Errorf(
			"Comment.Parent: wanted `%s`; found `%s`",
			c.Parent,
			other.Parent,
		)
	}

	if c.Body != other.Body {
		return fmt.Errorf(
			"Comment.Body: wanted `%s`; found `%s`",
			c.Body,
			other.Body,
		)
	}

	if c.Created != other.Created {
		return fmt.Errorf(
			"Comment.Created: wanted `%s`; found `%s`",
			c.Created,
			other.Created,
		)
	}

	if c.Modified != other.Modified {
		return fmt.Errorf(
			"Comment.Modified: wanted `%s`; found `%s`",
			c.Modified,
			other.Modified,
		)
	}

	return nil
}

func (wanted *Comment) Compare(data []byte) error {
	var other Comment
	if err := json.Unmarshal(data, &other); err != nil {
		return fmt.Errorf("unmarshaling `Comment`: %w", err)
	}
	return wanted.compare(&other)
}
