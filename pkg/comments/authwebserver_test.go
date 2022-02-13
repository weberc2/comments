package comments

import (
	"errors"
	"net/http"
	"testing"

	"github.com/weberc2/auth/pkg/client"
	"github.com/weberc2/comments/pkg/testsupport"
	pz "github.com/weberc2/httpeasy"
)

func TestAuthWebServer(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		method   func(*AuthWebServer) pz.Route
		optional bool
	}{
		{
			name:     "replies",
			method:   (*AuthWebServer).RepliesRoute,
			optional: true,
		},
		{
			name:     "delete-confirm",
			method:   (*AuthWebServer).DeleteConfirmRoute,
			optional: false,
		},
		{
			name:     "delete",
			method:   (*AuthWebServer).DeleteRoute,
			optional: false,
		},
		{
			name:     "reply-form",
			method:   (*AuthWebServer).ReplyFormRoute,
			optional: false,
		},
		{
			name:     "reply",
			method:   (*AuthWebServer).ReplyRoute,
			optional: false,
		},
		{
			name:     "edit-form",
			method:   (*AuthWebServer).EditFormRoute,
			optional: false,
		},
		{
			name:     "edit",
			method:   (*AuthWebServer).EditRoute,
			optional: false,
		},
	} {
		rsp := testCase.method(&AuthWebServer{
			WebServer: WebServer{
				Comments: CommentsModel{
					CommentsStore: testsupport.CommentsStoreFake{},
				},
			},
			AuthType: client.ConstantAuthType(client.ResultErr(
				"ERR",
				errors.New("ERR"),
			)),
		}).Handler(pz.Request{Headers: make(http.Header)})

		if testCase.optional && rsp.Status == 401 {
			t.Fatalf("expected optional authentication, but got `401`")
		}
		if !testCase.optional && rsp.Status != 401 {
			t.Fatalf(
				"expected required authentication--wanted `401`; found `%d`",
				rsp.Status,
			)
		}
	}
}
