package comments

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/weberc2/comments/pkg/testsupport"
	"github.com/weberc2/comments/pkg/types"
)

const (
	goodBody = "sufficiently long body"
)

func TestCommentsModel_Put(t *testing.T) {
	now := time.Date(1988, 9, 3, 0, 0, 0, 0, time.UTC)
	for _, testCase := range []struct {
		name          string
		input         types.Comment
		wantedComment *types.Comment
		wantedErr     WantedError
	}{
		{
			name: "base case",
			input: types.Comment{
				Post:   "post",
				Author: "user",
				Body:   goodBody,
			},
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
			input:     types.Comment{Author: "user", Body: goodBody},
			wantedErr: ErrInvalidPost,
		},
		{
			name: "ignores `ID`, `Created`, and `Modified` fields",
			input: types.Comment{
				Post:     "post",
				ID:       "evil-id",
				Author:   "user",
				Created:  time.Now(),
				Modified: time.Now(),
				Body:     goodBody,
			},
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
			name:      "body too short",
			input:     types.Comment{Post: "post", Author: "author", Body: ""},
			wantedErr: ErrBodyTooShort,
		},
		{
			name: "body too long",
			input: types.Comment{
				Post:   "post",
				Author: "author",
				Body:   strings.Repeat("x", bodySizeMax+1),
			},
			wantedErr: ErrBodyTooLong,
		},
		{
			name: "html escape body",
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
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := testsupport.CommentsStoreFake{}
			model := CommentsModel{
				CommentsStore: store,
				IDFunc:        func() types.CommentID { return "comment" },
				TimeFunc:      func() time.Time { return now },
			}

			c, err := model.Put(&testCase.input)

			if testCase.wantedErr == nil || testCase.wantedErr == (NilError{}) {
				if err := (NilError{}).Compare(err); err != nil {
					t.Fatal(err)
				}
				if err := testCase.wantedComment.Compare(c); err != nil {
					t.Fatal(err)
				}
				if err := store.Contains(testCase.wantedComment); err != nil {
					t.Fatal(err)
				}
			} else if err := testCase.wantedErr.Compare(err); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type WantedError interface {
	Compare(other error) error
}

type NilError struct{}

func (NilError) Compare(other error) error {
	if other != nil {
		return fmt.Errorf("wanted `nil`, found `%T`: %v", other, other)
	}
	return nil
}
