package pgcommentsstore

import (
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/weberc2/comments/pkg/comments/types"
)

func TestPGCommentsStore_Put(t *testing.T) {
	store, err := testPGCommentsStore()
	if err != nil {
		t.Fatal(err)
	}

	for _, testCase := range []struct {
		name          string
		state         []*types.Comment
		input         types.Comment
		wantedComment *types.Comment
		wantedError   types.WantedError
	}{
		{
			name: "simple",
			input: types.Comment{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someDate,
				Modified: someDate,
				Body:     "body",
			},
			wantedError: nil,
		},
		{
			// GIVEN a comment with ID `id` exists
			// WHEN we try to Put() a comment with the same ID
			// THEN expect we get a `ErrCommentExists`
			name: "unique ids",
			state: []*types.Comment{
				{
					ID:       "id",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
			},
			input: types.Comment{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someDate,
				Modified: someDate,
				Body:     "body",
			},
			wantedError: types.ErrCommentExists,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := store.ClearTable(); err != nil {
				t.Fatalf("clearing `comments` table: %v", err)
			}

			for _, comment := range testCase.state {
				if err := store.Put(comment); err != nil {
					t.Fatalf(
						"unexpected error preparing test database state: %v",
						err,
					)
				}
			}

			if testCase.wantedError == nil {
				testCase.wantedError = types.NilError{}
			}
			if err := testCase.wantedError.CompareErr(
				store.Put(&testCase.input),
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPGCommentsStore_Comment(t *testing.T) {
	store, err := testPGCommentsStore()
	if err != nil {
		t.Fatal(err)
	}

	input := types.Comment{
		ID:       "id",
		Post:     "post",
		Parent:   "",
		Author:   "author",
		Created:  someDate,
		Modified: someDate,
		Body:     "body",
	}

	if err := store.Put(&input); err != nil {
		t.Fatalf("unexpected error putting comment: %v", err)
	}

	found, err := store.Comment(input.Post, input.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching comment: %v", err)
	}

	if err := input.Compare(found); err != nil {
		t.Fatal(err)
	}
}

func TestPGCommentsStore_Update(t *testing.T) {
	store, err := testPGCommentsStore()
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name        string
		state       []*types.Comment
		input       *types.CommentPatch
		wantedState []*types.Comment
		wantedError types.WantedError
	}{
		{
			name: "simple",
			state: []*types.Comment{
				{
					ID:       "id",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Deleted:  false,
					Body:     "body",
				},
			},
			input: types.NewCommentPatch("id", "post").SetDeleted(true),
			wantedState: []*types.Comment{{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someDate,
				Modified: someDate,
				Deleted:  true,
				Body:     "body",
			}},
			wantedError: nil,
		},
		{
			name:        "not found",
			input:       types.NewCommentPatch("id", "post").SetDeleted(true),
			wantedState: nil,
			wantedError: types.ErrCommentNotFound,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := store.ClearTable(); err != nil {
				t.Fatalf("clearing `comments` table: %v", err)
			}

			for _, comment := range testCase.state {
				if err := store.Put(comment); err != nil {
					t.Fatalf("unexpected error putting comment: %v", err)
				}
			}

			if testCase.wantedError == nil {
				testCase.wantedError = types.NilError{}
			}
			if err := testCase.wantedError.CompareErr(
				store.Update(testCase.input),
			); err != nil {
				t.Fatal(err)
			}

			comments, err := store.List()
			if err != nil {
				t.Fatal(err)
			}

			if err := types.CompareComments(
				testCase.wantedState,
				comments,
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPGCommentsStore_Delete(t *testing.T) {
	store, err := testPGCommentsStore()
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name        string
		state       []*types.Comment
		post        types.PostID
		comment     types.CommentID
		wantedState []*types.Comment
		wantedError types.WantedError
	}{
		{
			name: "simple",
			state: []*types.Comment{
				{
					ID:       "id",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
			},
			post:        "post",
			comment:     "id",
			wantedState: nil,
			wantedError: nil,
		},
		{
			name:        "not found",
			post:        "post",
			comment:     "id",
			wantedState: nil,
			wantedError: types.ErrCommentNotFound,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := store.ClearTable(); err != nil {
				t.Fatalf("clearing `comments` table: %v", err)
			}

			for _, comment := range testCase.state {
				if err := store.Put(comment); err != nil {
					t.Fatalf("unexpected error putting comment: %v", err)
				}
			}

			if testCase.wantedError == nil {
				testCase.wantedError = types.NilError{}
			}
			err := store.Delete(testCase.post, testCase.comment)
			if err := testCase.wantedError.CompareErr(err); err != nil {
				t.Fatal(err)
			}

			comments, err := store.List()
			if err != nil {
				t.Fatal(err)
			}

			if err := types.CompareComments(
				[]*types.Comment{},
				comments,
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPGCommentsStore_Replies(t *testing.T) {
	store, err := testPGCommentsStore()
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name          string
		state         []*types.Comment
		post          types.PostID
		parent        types.CommentID
		wantedReplies []*types.Comment
		wantedErr     types.WantedError
	}{
		{
			name:          "empty",
			state:         nil,
			post:          "post",
			parent:        "parent",
			wantedReplies: nil,
			wantedErr:     nil,
		},
		{
			name: "toplevel-simple",
			state: []*types.Comment{
				{
					ID:       "0",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "1",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "different-post-comment",
					Post:     "some-other-post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
			},
			post:   "post",
			parent: "",
			wantedReplies: []*types.Comment{
				{
					ID:       "0",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "1",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
			},
			wantedErr: nil,
		},
		{
			// GIVEN two posts in the database, one with the parent `parent`
			//   and the other with no parent (i.e., "toplevel")
			// WHEN replies are fetched for the comment `parent`
			// THEN expect only the comment whose parent is `parent` is
			//   returned.
			name: "with-parent",
			state: []*types.Comment{
				{
					ID:       "parent",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "child",
					Post:     "post",
					Parent:   "parent",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "unrelated",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "different-post-comment",
					Post:     "some-other-post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
			},
			post:   "post",
			parent: "parent",
			wantedReplies: []*types.Comment{
				{
					ID:       "child",
					Post:     "post",
					Parent:   "parent",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
			},
			wantedErr: nil,
		},
		{
			name: "recursive",
			state: []*types.Comment{
				{
					ID:       "parent",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "child",
					Post:     "post",
					Parent:   "parent",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "grandchild",
					Post:     "post",
					Parent:   "child",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "unrelated",
					Post:     "post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "different-post-comment",
					Post:     "some-other-post",
					Parent:   "",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
			},
			post:   "post",
			parent: "parent",
			wantedReplies: []*types.Comment{
				{
					ID:       "child",
					Post:     "post",
					Parent:   "parent",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
				{
					ID:       "grandchild",
					Post:     "post",
					Parent:   "child",
					Author:   "author",
					Created:  someDate,
					Modified: someDate,
					Body:     "body",
				},
			},
			wantedErr: nil,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if err := store.ClearTable(); err != nil {
				t.Fatalf("clearing `comments` table: %v", err)
			}

			for _, comment := range testCase.state {
				if err := store.Put(comment); err != nil {
					t.Fatalf(
						"unexpected error preparing test database state: %v",
						err,
					)
				}
			}

			found, err := store.Replies(testCase.post, testCase.parent)

			if testCase.wantedErr == nil {
				testCase.wantedErr = types.NilError{}
			}
			if err := testCase.wantedErr.CompareErr(err); err != nil {
				t.Fatalf("comparing errors: %v", err)
			}

			if err := types.CompareComments(
				testCase.wantedReplies,
				found,
			); err != nil {
				t.Fatalf("comparing replies: %v", err)
			}
		})
	}
}

func testPGCommentsStore() (*PGCommentsStore, error) {
	pgcs, err := OpenEnv()
	if err != nil {
		return nil, fmt.Errorf("opening test database: %w", err)
	}
	return pgcs, resetTable(pgcs)
}

func resetTable(store *PGCommentsStore) error {
	if err := store.DropTable(); err != nil {
		return err
	}
	return store.EnsureTable()
}

var someDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
