package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	html "html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	pz "github.com/weberc2/httpeasy"
)

type PostID string
type CommentID string
type UserID string

type Comment struct {
	ID       CommentID `json:"id"`
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

	repliesTemplateString := os.Getenv("REPLIES_TEMPLATE")
	if err != nil {
		log.Fatal("missing required env var: REPLIES_TEMPLATE")
	}
	repliesTemplate, err := html.New("").Parse(repliesTemplateString)
	if err != nil {
		log.Fatalf("parsing REPLIES_TEMPLATE: %v", err)
	}

	deleteConfirmationTemplateString := os.Getenv(
		"DELETE_CONFIRMATION_TEMPLATE",
	)
	if err != nil {
		log.Fatal("missing required env var: DELETE_CONFIRMATION_TEMPLATE")
	}
	deleteConfirmationTemplate, err := html.New("").Parse(
		deleteConfirmationTemplateString,
	)
	if err != nil {
		log.Fatalf("parsing DELETE_CONFIRMATION_TEMPLATE: %v", err)
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
		LoginURL:                   loginURL,
		LogoutURL:                  logoutURL,
		BaseURL:                    baseURL,
		Comments:                   commentsService.Comments,
		RepliesTemplate:            repliesTemplate,
		DeleteConfirmationTemplate: deleteConfirmationTemplate,
	}

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
			Handler: auth(key, authHeaderToken, commentsService.PutComment),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/api/posts/{post-id}/comments/{comment-id}",
			Handler: commentsService.GetComment,
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{parent-id}/replies",
			Handler: authenticate(key, cookieToken, webServer.Replies),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{comment-id}/delete-confirm",
			Handler: authenticate(key, cookieToken, webServer.DeleteConfirm),
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

type authErr struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

type tokenLocation func(pz.Request) (string, *authErr)

func authHeaderToken(r pz.Request) (string, *authErr) {
	authorization := r.Headers.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		return "", &authErr{
			Message: "invalid 'Authorization' header",
			Error:   "missing 'Bearer' prefix",
		}
	}
	return authorization[len("Bearer "):], nil
}

func cookieToken(r pz.Request) (string, *authErr) {
	c, err := r.Cookie("Access-Token")
	if err != nil {
		return "", &authErr{
			Message: "missing cookie `Access-Token`",
			Error:   err.Error(),
		}
	}

	return c.Value, nil
}

func authenticateHelper(
	key *ecdsa.PublicKey,
	tl tokenLocation,
	r pz.Request,
) *authErr {
	tok, err := tl(r)
	if err != nil {
		return err
	}

	var claims jwt.StandardClaims
	if _, err := jwt.ParseWithClaims(
		tok,
		&claims,
		func(*jwt.Token) (interface{}, error) { return key, nil },
	); err != nil {
		return &authErr{
			Message: "invalid 'Authorization' header",
			Error:   err.Error(),
		}
	}

	if err := claims.Valid(); err != nil {
		return &authErr{
			Message: "invalid access token claim(s)",
			Error:   err.Error(),
		}
	}

	r.Headers.Set("User", claims.Subject)
	return nil
}

func authenticate(
	key *ecdsa.PublicKey,
	tl tokenLocation,
	handler pz.Handler,
) pz.Handler {
	return func(r pz.Request) pz.Response {
		if err := authenticateHelper(key, tl, r); err != nil {
			// TODO: Add httpeasy.Response.WithLogging() method
			return handler(r).WithLogging(err)
		}
		return handler(r).WithLogging(struct {
			Message string `json:"message"`
			User    string `json:"user"`
		}{
			Message: "authenticated successfully",
			User:    r.Headers.Get("User"),
		})
	}
}

func auth(
	key *ecdsa.PublicKey,
	tl tokenLocation,
	handler pz.Handler,
) pz.Handler {
	return authenticate(key, tl, func(r pz.Request) pz.Response {
		if r.Headers.Get("User") == "" {
			return pz.Unauthorized(nil)
		}
		return handler(r)
	})
}
