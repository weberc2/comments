package types

import (
	"encoding/json"
	"errors"
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
	Deleted  bool      `json:"deleted"`
	Body     string    `json:"body"`
}

type Error string

func (err Error) Error() string { return string(err) }

func (wanted Error) Compare(found Error) error {
	if wanted != found {
		return fmt.Errorf("Error: wanted `%s`; found `%s`", wanted, found)
	}
	return nil
}

func (wanted Error) CompareErr(found error) error {
	if errors.Is(found, wanted) {
		return nil
	}

	return fmt.Errorf(
		"Error: wanted error `%T`; found `%T`: %v",
		wanted,
		found,
		found,
	)
}

var (
	ErrWantedNotNil Error = "wanted not-nil; found `nil`"
	ErrWantedNil    Error = "wanted `nil`; found not-nil"
)

type FieldMismatchErr struct {
	Field  Field
	Wanted interface{}
	Found  interface{}
}

func (err *FieldMismatchErr) Error() string {
	return fmt.Sprintf(
		"Comment.%s: wanted `%v`; found `%v`",
		err.Field.GoString(),
		err.Wanted,
		err.Found,
	)
}

func (wanted *FieldMismatchErr) Compare(found *FieldMismatchErr) error {
	if wanted == found {
		return nil
	}

	if wanted != nil && found == nil {
		return ErrWantedNotNil
	}

	if wanted == nil && found != nil {
		return ErrWantedNil
	}

	if wanted.Field != found.Field {
		return fmt.Errorf(
			"FieldMismatchErr.Field: wanted `%s`; found `%s`",
			wanted.Field,
			found.Field,
		)
	}

	if wanted.Wanted != found.Wanted {
		return fmt.Errorf(
			"FieldMismatchErr.Wanted: wanted `%v` (`%T`); found `%v` (`%T`)",
			wanted.Wanted,
			wanted.Wanted,
			found.Wanted,
			found.Wanted,
		)
	}

	if wanted.Found != found.Found {
		return fmt.Errorf(
			"FieldMismatchErr.Found: wanted `%v`; found `%v`",
			wanted.Found,
			found.Found,
		)
	}

	return nil
}

func (wanted *FieldMismatchErr) CompareErr(found error) error {
	var other *FieldMismatchErr
	if !errors.As(found, &other) {
		return fmt.Errorf(
			"wanted `*types.FieldMismatchErr`; found `%T`: %v",
			found,
			found,
		)
	}
	return wanted.Compare(other)
}

func (wanted *Comment) Compare(found *Comment) error {
	if wanted == nil && found == nil {
		return nil
	}

	if wanted != nil && found == nil {
		return ErrWantedNotNil
	}

	if wanted == nil && found != nil {
		return ErrWantedNil
	}

	if wanted.ID != found.ID {
		return &FieldMismatchErr{
			Field:  FieldID,
			Wanted: wanted.ID,
			Found:  found.ID,
		}
	}

	if wanted.Post != found.Post {
		return &FieldMismatchErr{
			Field:  FieldPost,
			Wanted: wanted.Post,
			Found:  found.Post,
		}
	}

	if wanted.Author != found.Author {
		return &FieldMismatchErr{
			Field:  FieldAuthor,
			Wanted: wanted.Author,
			Found:  found.Author,
		}
	}

	if wanted.Parent != found.Parent {
		return &FieldMismatchErr{
			Field:  FieldParent,
			Wanted: wanted.Parent,
			Found:  found.Parent,
		}
	}

	if !wanted.Created.Equal(found.Created) {
		return &FieldMismatchErr{
			Field:  FieldCreated,
			Wanted: wanted.Created,
			Found:  found.Created,
		}
	}

	if !wanted.Modified.Equal(found.Modified) {
		return &FieldMismatchErr{
			Field:  FieldModified,
			Wanted: wanted.Modified,
			Found:  found.Modified,
		}
	}

	if wanted.Deleted != found.Deleted {
		return &FieldMismatchErr{
			Field:  FieldDeleted,
			Wanted: wanted.Deleted,
			Found:  found.Deleted,
		}
	}

	if wanted.Body != found.Body {
		return &FieldMismatchErr{
			Field:  FieldBody,
			Wanted: wanted.Body,
			Found:  found.Body,
		}
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

type SliceLengthMismatchErr struct {
	Wanted int
	Found  int
}

func (err *SliceLengthMismatchErr) Error() string {
	return fmt.Sprintf("wanted len() `%d`; found `%d`", err.Wanted, err.Found)
}

func (wanted *SliceLengthMismatchErr) Compare(
	found *SliceLengthMismatchErr,
) error {
	if wanted == found {
		return nil
	}

	if wanted != nil && found == nil {
		return fmt.Errorf("SliceLengthMismatchErr: %s", ErrWantedNotNil)
	}

	if wanted == nil && found != nil {
		return fmt.Errorf("SliceLengthMismatchErr: %s", ErrWantedNil)
	}

	if wanted.Wanted != found.Wanted {
		return fmt.Errorf(
			"SliceLengthMismatchErr.Wanted: wanted `%d`; found `%d`",
			wanted.Wanted,
			found.Wanted,
		)
	}
	if wanted.Found != found.Found {
		return fmt.Errorf(
			"SliceLengthMismatchErr.Found: wanted `%d`; found `%d`",
			wanted.Found,
			found.Found,
		)
	}
	return nil
}

func (wanted *SliceLengthMismatchErr) CompareErr(found error) error {
	var other *SliceLengthMismatchErr
	if errors.As(found, &other) {
		return wanted.Compare(other)
	}
	return fmt.Errorf(
		"SliceLengthMismatchError: wanted type `%T`; found `%T`: %v",
		wanted,
		found,
		found,
	)
}

func CompareComments(wanted, found []*Comment) error {
	if len(wanted) != len(found) {
		return &SliceLengthMismatchErr{Wanted: len(wanted), Found: len(found)}
	}

	sortComments(wanted)
	sortComments(found)

	for i := range wanted {
		if err := wanted[i].Compare(found[i]); err != nil {
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
