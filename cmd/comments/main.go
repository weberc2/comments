package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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

	registerURL := os.Getenv("REGISTER_URL")
	if registerURL == "" {
		log.Fatal("missing required env var: REGISTER_URL")
	}

	authBaseURL := os.Getenv("AUTH_BASE_URL")
	if authBaseURL == "" {
		log.Fatal("missing required env var: AUTH_BASE_URL")
	}

	baseURLString := os.Getenv("BASE_URL")
	if baseURLString == "" {
		log.Fatal("missing required env var: BASE_URL")
	}
	baseURL, err := url.Parse(baseURLString)
	if err != nil {
		log.Fatalf("error parsing `BASE_URL` env var: %v", err)
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

	webServerAuth := client.AuthTypeWebServer{
		WebServerApp: client.WebServerApp{
			Client:          client.DefaultClient(authBaseURL),
			BaseURL:         baseURL,
			DefaultRedirect: "/",
			Key:             cookieEncryptionKey,
		},
	}

	a := client.Authenticator{Key: key}
	webServer := comments.AuthWebServer{
		WebServer: comments.WebServer{
			LoginURL:         loginURL,
			RegisterURL:      registerURL,
			LogoutPath:       "/auth/logout",
			BaseURL:          baseURLString,
			Comments:         commentsService.Comments,
			AuthCallbackPath: "/auth/callback",
		},
		AuthType:      &webServerAuth,
		Authenticator: a,
	}

	apiAuth := client.AuthTypeClientProgram{}

	if err := http.ListenAndServe(addr, pz.Register(
		pz.JSONLog(os.Stderr),
		append(
			webServer.Routes(),
			webServerAuth.AuthCodeCallbackRoute(webServer.AuthCallbackPath),
			webServerAuth.LogoutRoute(webServer.LogoutPath),
			pz.Route{
				Method:  "GET",
				Path:    "/api/posts/{post-id}/comments/{comment-id}/replies",
				Handler: commentsService.Replies,
			},
			pz.Route{
				Method:  "POST",
				Path:    "/api/posts/{post-id}/comments",
				Handler: a.Auth(apiAuth, commentsService.Put),
			},
			pz.Route{
				Method:  "GET",
				Path:    "/api/posts/{post-id}/comments/{comment-id}",
				Handler: commentsService.Get,
			},
			pz.Route{
				Method:  "PATCH",
				Path:    "/api/posts/{post-id}/comments/{comment-id}",
				Handler: a.Auth(apiAuth, commentsService.Update),
			},
		)...,
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
