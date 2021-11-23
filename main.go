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

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/weberc2/auth/pkg/client"
	pz "github.com/weberc2/httpeasy"
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

type noopPostStore struct{}

func (nps noopPostStore) Exists(PostID) error { return nil }

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

	bucket := os.Getenv("BUCKET")
	if bucket == "" {
		log.Fatal("missing required env var: BUCKET")
	}

	accessTokenPublicKey := os.Getenv("ACCESS_KEY")
	if accessTokenPublicKey == "" {
		log.Fatal("missing required env var: ACCESS_KEY")
	}
	key, err := decodeKey(accessTokenPublicKey)
	if err != nil {
		log.Fatalf("decoding ACCESS_KEY: %v", err)
	}

	sess, err := session.NewSession()
	if err != nil {
		log.Fatalf("creating AWS session: %v", err)
	}

	commentsService := CommentsService{
		Comments: CommentStore{
			Bucket:      bucket,
			Prefix:      "",
			ObjectStore: &S3ObjectStore{s3.New(sess)},
			PostStore:   noopPostStore{},
			IDFunc: func() CommentID {
				return CommentID(uuid.NewString())
			},
		},
		TimeFunc: time.Now,
	}

	webServer := WebServer{
		LoginURL:  loginURL,
		LogoutURL: logoutURL,
		BaseURL:   baseURL,
		Comments:  commentsService.Comments,
	}

	webServerAuth := AuthTypeWebServer{Auth: client.DefaultClient(authBaseURL)}
	apiAuth := AuthTypeClientProgram{}
	auth := Authenticator{Key: key}

	http.ListenAndServe(addr, pz.Register(
		pz.JSONLog(os.Stderr),
		pz.Route{
			Method:  "GET",
			Path:    "/api/posts/{post-id}/comments/{comment-id}/replies",
			Handler: commentsService.PostComments,
		},
		pz.Route{
			Method:  "POST",
			Path:    "/api/posts/{post-id}/comments",
			Handler: auth.AuthN(apiAuth, commentsService.PutComment),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/api/posts/{post-id}/comments/{comment-id}",
			Handler: commentsService.GetComment,
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{parent-id}/replies",
			Handler: auth.AuthN(&webServerAuth, webServer.Replies),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{comment-id}/delete-confirm",
			Handler: auth.AuthN(&webServerAuth, webServer.DeleteConfirm),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{comment-id}/delete",
			Handler: auth.AuthN(&webServerAuth, webServer.Delete),
		},
	))
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
