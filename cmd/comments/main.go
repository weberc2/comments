package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/weberc2/auth/pkg/client"
	"github.com/weberc2/comments/pkg/comments"
	"github.com/weberc2/comments/pkg/pgcommentsstore"
	"github.com/weberc2/comments/pkg/types"
	pz "github.com/weberc2/httpeasy"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	loginURL := os.Getenv("LOGIN_URL")
	if loginURL == "" {
		log.Fatal("missing required env var: LOGIN_URL")
	}

	logoutURL := os.Getenv("LOGOUT_URL")
	if logoutURL == "" {
		log.Fatal("missing required env var: LOGOUT_URL")
	}

	authBaseURL := os.Getenv("AUTH_BASE_URL")
	if authBaseURL == "" {
		log.Fatal("missing required env var: AUTH_BASE_URL")
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		log.Fatal("missing required env var: BASE_URL")
	}

	cookieEncryptionKey := os.Getenv("COOKIE_ENCRYPTION_KEY")
	if cookieEncryptionKey == "" {
		log.Fatal("missing required env var: COOKIE_ENCRYPTION_KEY")
	}

	accessTokenPublicKey := os.Getenv("ACCESS_KEY")
	if accessTokenPublicKey == "" {
		log.Fatal("missing required env var: ACCESS_KEY")
	}
	key, err := decodeKey(accessTokenPublicKey)
	if err != nil {
		log.Fatalf("decoding ACCESS_KEY: %v", err)
	}

	commentsStore, err := pgcommentsstore.OpenEnv()
	if err != nil {
		log.Fatalf("creating postgres comments store client: %v", err)
	}

	if err := commentsStore.EnsureTable(); err != nil {
		log.Fatalf("ensuring comments table exists: %v", err)
	}

	commentsService := comments.CommentsService{
		Comments: comments.CommentsModel{
			CommentsStore: commentsStore,
			IDFunc: func() types.CommentID {
				return types.CommentID(uuid.NewString())
			},
			TimeFunc: time.Now,
		},
	}

	webServer := comments.WebServer{
		LoginURL:         loginURL,
		LogoutURL:        logoutURL,
		BaseURL:          baseURL,
		Comments:         commentsService.Comments,
		AuthCallbackPath: "/auth/callback",
	}

	webServerAuth := client.AuthTypeWebServer{
		WebServerApp: client.WebServerApp{
			Client:          client.DefaultClient(authBaseURL),
			BaseURL:         baseURL,
			DefaultRedirect: baseURL,
			Key:             cookieEncryptionKey,
		},
	}
	apiAuth := client.AuthTypeClientProgram{}

	a := client.Authenticator{Key: key}

	if err := http.ListenAndServe(addr, pz.Register(
		pz.JSONLog(os.Stderr),
		webServerAuth.AuthCodeCallbackRoute(webServer.AuthCallbackPath),
		pz.Route{
			Method:  "GET",
			Path:    "/api/posts/{post-id}/comments/{comment-id}/replies",
			Handler: commentsService.Replies,
		},
		pz.Route{
			Method:  "POST",
			Path:    "/api/posts/{post-id}/comments",
			Handler: a.AuthN(apiAuth, commentsService.Put),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/api/posts/{post-id}/comments/{comment-id}",
			Handler: commentsService.Get,
		},
		pz.Route{
			Method:  "PATCH",
			Path:    "/api/posts/{post-id}/comments/{comment-id}",
			Handler: a.AuthN(apiAuth, commentsService.Update),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{parent-id}/replies",
			Handler: a.AuthN(&webServerAuth, webServer.Replies),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{comment-id}/delete-confirm",
			Handler: a.AuthN(&webServerAuth, webServer.DeleteConfirm),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{comment-id}/delete",
			Handler: a.AuthN(&webServerAuth, webServer.Delete),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{comment-id}/reply",
			Handler: a.AuthN(&webServerAuth, webServer.ReplyForm),
		},
		pz.Route{
			Method:  "POST",
			Path:    "/posts/{post-id}/comments/{comment-id}/reply",
			Handler: a.AuthN(&webServerAuth, webServer.Reply),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{comment-id}/edit",
			Handler: a.AuthN(&webServerAuth, webServer.EditForm),
		},
		pz.Route{
			Method:  "POST",
			Path:    "/posts/{post-id}/comments/{comment-id}/edit",
			Handler: a.AuthN(&webServerAuth, webServer.Edit),
		},
	)); err != nil {
		log.Fatal(err)
	}
}

func decodeKey(encoded string) (*ecdsa.PublicKey, error) {
	data := []byte(encoded)
	for {
		block, rest := pem.Decode(data)
		if block.Type == "PUBLIC KEY" {
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parsing x509 PKIX public key: %v", err)
			}
			if key, ok := pub.(*ecdsa.PublicKey); ok {
				return key, nil
			}
			return nil, fmt.Errorf(
				"invalid key type; wanted *ecdsa.PublicKey; found %T",
				pub,
			)
		}
		if len(rest) < 1 {
			return nil, io.EOF
		}
		data = rest
	}
}
