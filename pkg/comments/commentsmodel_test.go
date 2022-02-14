package comments

import (
	"strings"
	"testing"
	"time"

	"github.com/weberc2/comments/pkg/testsupport"
	"github.com/weberc2/comments/pkg/types"
)

const (
	goodBody = "sufficiently long body"
)

func TestCommentsModel_Update(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		state       testsupport.CommentsStoreFake
		input       *CommentUpdate
		wantedState []*types.Comment
		wantedErr   types.WantedError
	}{
		{
			name: "simple",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  false,
						Body:     "hello, world",
					},
				},
			},
			input: &CommentUpdate{ID: "id", Post: "post", Body: "greetings"},
			wantedState: []*types.Comment{{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someTime,
				Modified: now,
				Deleted:  false,
				Body:     "greetings",
			}},
		},
		{
			name: "can't edit deleted",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  true,
						Body:     "hello, world",
					},
				},
			},
			input: &CommentUpdate{ID: "id", Post: "post", Body: "greetings"},
			wantedState: []*types.Comment{{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someTime,
				Modified: someTime,
				Deleted:  true,
				Body:     "hello, world",
			}},
			wantedErr: types.ErrCommentNotFound,
		},
		{
			name: "validates body",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  false,
						Body:     "hello, world",
					},
				},
			},
			input: &CommentUpdate{ID: "id", Post: "post", Body: ""},
			wantedState: []*types.Comment{{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someTime,
				Modified: someTime,
				Deleted:  false,
				Body:     "hello, world",
			}},
			wantedErr: ErrBodyTooShort,
		},
		{
			name: "errors propagate",
			input: &CommentUpdate{
				ID:   "not-found",
				Post: "not-found",
				Body: "salutations",
			},
			wantedErr: types.ErrCommentNotFound,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			model := CommentsModel{
				CommentsStore: testCase.state,
				TimeFunc:      func() time.Time { return now },
			}

			if testCase.wantedErr == nil {
				testCase.wantedErr = types.NilError{}
			}

			if err := testCase.wantedErr.CompareErr(
				model.Update(testCase.input),
			); err != nil {
				t.Fatal(err)
			}

			if err := types.CompareComments(
				testCase.wantedState,
				testCase.state.List(),
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCommentsModel_Replies(t *testing.T) {
	for _, testCase := range []struct {
		name           string
		state          testsupport.CommentsStoreFake
		post           types.PostID
		parent         types.CommentID
		wantedComments []*types.Comment
		wantedErr      types.WantedError
	}{
		{
			name: "simple",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  false,
						Body:     "body",
					},
				},
			},
			post:   "post",
			parent: "",
			wantedComments: []*types.Comment{{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "author",
				Created:  someTime,
				Modified: someTime,
				Deleted:  false,
				Body:     "body",
			}},
		},
		{
			name: "deleted posts are redacted",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  true,
						Body:     "body",
					},
				},
			},
			post:   "post",
			parent: "",
			wantedComments: []*types.Comment{{
				ID:       "id",
				Post:     "post",
				Parent:   "",
				Author:   "",
				Created:  someTime,
				Modified: someTime,
				Deleted:  true,
				Body:     "",
			}},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.wantedErr == nil {
				testCase.wantedErr = types.NilError{}
			}

			model := CommentsModel{
				CommentsStore: testCase.state,
				TimeFunc:      func() time.Time { return now },
			}

			comments, err := model.Replies(testCase.post, testCase.parent)

			if err := types.CompareComments(
				testCase.wantedComments,
				comments,
			); err != nil {
				t.Fatal(err)
			}

			if err := testCase.wantedErr.CompareErr(err); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCommentsModel_Delete(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		state       testsupport.CommentsStoreFake
		post        types.PostID
		comment     types.CommentID
		wantedState testsupport.CommentsStoreFake
		wantedErr   types.WantedError
	}{
		{
			name: "simple",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "parent",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  false,
					},
				},
			},
			post:    "post",
			comment: "id",
			wantedState: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "parent",
						Author:   "author",
						Created:  someTime,
						Modified: now,
						Deleted:  true,
					},
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			model := CommentsModel{
				CommentsStore: testCase.state,
				IDFunc:        func() types.CommentID { return "comment" },
				TimeFunc:      func() time.Time { return now },
			}

			if testCase.wantedErr == nil {
				testCase.wantedErr = types.NilError{}
			}
			if err := testCase.wantedErr.CompareErr(
				model.Delete(testCase.post, testCase.comment),
			); err != nil {
				t.Fatal(err)
			}

			if err := testCase.wantedState.Compare(
				testCase.state,
			); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCommentsModel_Put(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		state         testsupport.CommentsStoreFake
		input         types.Comment
		wantedComment *types.Comment
		wantedErr     types.WantedError
	}{
		{
			name: "base case",
			input: types.Comment{
				Post:   "post",
				Author: "user",
				Body:   goodBody,
			},
			state: testsupport.CommentsStoreFake{},
			wantedComment: &types.Comment{
				Post:     "post",
				ID:       "comment",
				Author:   "user",
				Created:  now,
				Modified: now,
				Body:     goodBody,
			},
		},
		{
			name:      "missing `post` field",
			state:     testsupport.CommentsStoreFake{},
			input:     types.Comment{Author: "user", Body: goodBody},
			wantedErr: ErrInvalidPost,
		},
		{
			name:  "ignores `ID`, `Created`, `Modified`, and `Deleted` fields",
			state: testsupport.CommentsStoreFake{},
			input: types.Comment{
				Post:     "post",
				ID:       "evil-id",
				Author:   "user",
				Created:  someTime,
				Modified: someTime,
				Deleted:  true,
				Body:     goodBody,
			},
			wantedComment: &types.Comment{
				Post:     "post",
				ID:       "comment",
				Author:   "user",
				Created:  now,
				Modified: now,
				Deleted:  false,
				Body:     goodBody,
			},
		},
		{
			name:      "body too short",
			state:     testsupport.CommentsStoreFake{},
			input:     types.Comment{Post: "post", Author: "author", Body: ""},
			wantedErr: ErrBodyTooShort,
		},
		{
			name:  "body too long",
			state: testsupport.CommentsStoreFake{},
			input: types.Comment{
				Post:   "post",
				Author: "author",
				Body:   strings.Repeat("x", bodySizeMax+1),
			},
			wantedErr: ErrBodyTooLong,
		},
		{
			name:  "html escape body",
			state: testsupport.CommentsStoreFake{},
			input: types.Comment{
				Post:   "post",
				Author: "user",
				Body:   "<script></script>",
			},
			wantedComment: &types.Comment{
				Post:     "post",
				ID:       "comment",
				Author:   "user",
				Created:  now,
				Modified: now,
				Body:     "&lt;script&gt;&lt;/script&gt;",
			},
		},
		{
			name: "reply to deleted comment",
			state: testsupport.CommentsStoreFake{
				"post": {
					"id": {
						ID:       "id",
						Post:     "post",
						Parent:   "",
						Author:   "author",
						Created:  someTime,
						Modified: someTime,
						Deleted:  true,
					},
				},
			},
			input: types.Comment{
				ID:     "reply",
				Post:   "post",
				Parent: "id",
				Author: "author",
				Body:   goodBody,
			},
			wantedErr: types.ErrCommentNotFound,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			model := CommentsModel{
				CommentsStore: testCase.state,
				IDFunc:        func() types.CommentID { return "comment" },
				TimeFunc:      func() time.Time { return now },
			}

			c, err := model.Put(&testCase.input)

			if testCase.wantedErr == nil {
				testCase.wantedErr = types.NilError{}
			}

			if err := testCase.wantedErr.CompareErr(err); err != nil {
				t.Fatal(err)
			}
			if err := testCase.wantedComment.Compare(c); err != nil {
				t.Fatal(err)
			}
			if testCase.wantedComment != nil {
				if err := testCase.state.Contains(
					testCase.wantedComment,
				); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

var (
	someTime = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	now      = time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)
)
