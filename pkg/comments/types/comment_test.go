package types

import (
	"testing"
	"time"
)

func TestCompareComments(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		wanted    []*Comment
		found     []*Comment
		wantedErr WantedError
	}{
		{
			name:   "empty list matches empty list",
			wanted: []*Comment{},
			found:  []*Comment{},
		},
		{
			name:   "empty list matches nil",
			wanted: []*Comment{},
			found:  []*Comment{},
		},
		{
			name:   "nil matches nil",
			wanted: nil,
			found:  nil,
		},
		{
			name:   "matching lists match",
			wanted: []*Comment{{}},
			found:  []*Comment{{}},
		},
		{
			name:   "order doesn't matter",
			wanted: []*Comment{{Post: "foo"}, {Post: "bar"}},
			found:  []*Comment{{Post: "bar"}, {Post: "foo"}},
		},
		{
			name:      "found longer than wanted",
			wanted:    []*Comment{},
			found:     []*Comment{{}},
			wantedErr: &SliceLengthMismatchErr{Wanted: 0, Found: 1},
		},
		{
			name:      "mismatched values",
			wanted:    []*Comment{nil},
			found:     []*Comment{{}},
			wantedErr: ErrWantedNil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.wantedErr == nil {
				testCase.wantedErr = NilError{}
			}
			if err := testCase.wantedErr.CompareErr(CompareComments(
				testCase.wanted,
				testCase.found,
			)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestComment_Compare(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		wanted    *Comment
		found     *Comment
		wantedErr WantedError
	}{
		{
			name:   "id",
			wanted: &Comment{ID: "0"},
			found:  &Comment{ID: "1"},
			wantedErr: &FieldMismatchErr{
				Field:  FieldID,
				Wanted: CommentID("0"),
				Found:  CommentID("1"),
			},
		},
		{
			name:   "post",
			wanted: &Comment{Post: "0"},
			found:  &Comment{Post: "1"},
			wantedErr: &FieldMismatchErr{
				Field:  FieldPost,
				Wanted: PostID("0"),
				Found:  PostID("1"),
			},
		},
		{
			name:   "parent",
			wanted: &Comment{Parent: "0"},
			found:  &Comment{Parent: "1"},
			wantedErr: &FieldMismatchErr{
				Field:  FieldParent,
				Wanted: CommentID("0"),
				Found:  CommentID("1"),
			},
		},
		{
			name:   "author",
			wanted: &Comment{Author: "0"},
			found:  &Comment{Author: "1"},
			wantedErr: &FieldMismatchErr{
				Field:  FieldAuthor,
				Wanted: UserID("0"),
				Found:  UserID("1"),
			},
		},
		{
			name:   "created",
			wanted: &Comment{Created: someTime},
			found:  &Comment{Created: someOtherTime},
			wantedErr: &FieldMismatchErr{
				Field:  FieldCreated,
				Wanted: someTime,
				Found:  someOtherTime,
			},
		},
		{
			name:   "modified",
			wanted: &Comment{Modified: someTime},
			found:  &Comment{Modified: someOtherTime},
			wantedErr: &FieldMismatchErr{
				Field:  FieldModified,
				Wanted: someTime,
				Found:  someOtherTime,
			},
		},
		{
			name:   "deleted",
			wanted: &Comment{Deleted: false},
			found:  &Comment{Deleted: true},
			wantedErr: &FieldMismatchErr{
				Field:  FieldDeleted,
				Wanted: false,
				Found:  true,
			},
		},
		{
			name:   "body",
			wanted: &Comment{Body: "hello"},
			found:  &Comment{Body: "goodbye"},
			wantedErr: &FieldMismatchErr{
				Field:  FieldBody,
				Wanted: "hello",
				Found:  "goodbye",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.wantedErr == nil {
				testCase.wantedErr = NilError{}
			}

			if err := testCase.wantedErr.CompareErr(
				testCase.wanted.Compare(testCase.found),
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

var (
	someTime      = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	someOtherTime = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
)
