package types

import (
	"encoding/json"
	"fmt"
	"sort"
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

func (wanted *Comment) Compare(found *Comment) error {
	if wanted == nil && found == nil {
		return nil
	}

	if wanted != nil && found == nil {
		return fmt.Errorf("Comment: unexpected `nil`")
	}

	if wanted == nil && found != nil {
		return fmt.Errorf("Comment: wanted `nil`; found not-nil")
	}

	if wanted.ID != found.ID {
		return fmt.Errorf(
			"Comment.ID: wanted `%s`; found `%s`",
			wanted.ID,
			found.ID,
		)
	}

	if wanted.Author != found.Author {
		return fmt.Errorf(
			"Comment.Author: wanted `%s`; found `%s`",
			wanted.Author,
			found.Author,
		)
	}

	if wanted.Parent != found.Parent {
		return fmt.Errorf(
			"Comment.Parent: wanted `%s`; found `%s`",
			wanted.Parent,
			found.Parent,
		)
	}

	if wanted.Body != found.Body {
		return fmt.Errorf(
			"Comment.Body: wanted `%s`; found `%s`",
			wanted.Body,
			found.Body,
		)
	}

	if wanted.Created != found.Created {
		return fmt.Errorf(
			"Comment.Created: wanted `%s`; found `%s`",
			wanted.Created,
			found.Created,
		)
	}

	if wanted.Modified != found.Modified {
		return fmt.Errorf(
			"Comment.Modified: wanted `%s`; found `%s`",
			wanted.Modified,
			found.Modified,
		)
	}

	return nil
}

func (wanted *Comment) CompareData(data []byte) error {
	var other Comment
	if err := json.Unmarshal(data, &other); err != nil {
		return fmt.Errorf("unmarshaling `Comment`: %w", err)
	}
	return wanted.Compare(&other)
}

func CompareComments(wanted, found []*Comment) error {
	if len(wanted) < len(found) {
		return fmt.Errorf(
			"stored comments: len `%d`; found len `%d`",
			len(wanted),
			len(found),
		)
	}

	sortComments(wanted)
	sortComments(found)

	for i := range wanted {
		if err := wanted[i].Compare(wanted[i]); err != nil {
			return fmt.Errorf("index %d: %w", i, err)
		}
	}

	return nil
}

func sortComments(comments []*Comment) {
	sort.Slice(comments, func(i, j int) bool {
		if comments[i].Post < comments[j].Post {
			return true
		}
		if comments[i].Post == comments[j].Post {
			return comments[i].ID < comments[j].ID
		}
		return false
	})
}
