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

	repliesTemplate, err := html.New("").Parse(repliesTemplate)
	if err != nil {
		log.Fatalf("parsing replies template: %v", err)
	}

	webServer := WebServer{
		LoginURL:        loginURL,
		BaseURL:         baseURL,
		Comments:        commentsService.Comments,
		RepliesTemplate: repliesTemplate,
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
			Handler: auth(key, commentsService.PutComment),
		},
		pz.Route{
			Method:  "GET",
			Path:    "/api/posts/{post-id}/comments/{comment-id}",
			Handler: commentsService.GetComment,
		},
		pz.Route{
			Method:  "GET",
			Path:    "/posts/{post-id}/comments/{parent-id}/replies",
			Handler: authenticate(key, webServer.Replies),
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

func authenticateHelper(key *ecdsa.PublicKey, r pz.Request) *authErr {
	authorization := r.Headers.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		return &authErr{
			Message: "invalid 'Authorization' header",
			Error:   "missing 'Bearer' prefix",
		}
	}

	var claims jwt.StandardClaims
	if _, err := jwt.ParseWithClaims(
		authorization[len("Bearer "):],
		&claims,
		func(*jwt.Token) (interface{}, error) {
			return key, nil
		},
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

func authenticate(key *ecdsa.PublicKey, handler pz.Handler) pz.Handler {
	return func(r pz.Request) pz.Response {
		if err := authenticateHelper(key, r); err != nil {
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

func auth(key *ecdsa.PublicKey, handler pz.Handler) pz.Handler {
	return authenticate(key, func(r pz.Request) pz.Response {
		if r.Headers.Get("User") == "" {
			return pz.Unauthorized(nil)
		}
		return handler(r)
	})
}

const repliesTemplate = `<html>
<head></head>
<body>
<h1>Replies</h1>
<div id=replies>
{{if .User}}{{.User}}{{else}}<a href="{{.LoginURL}}">login</a>{{end}}
{{$baseURL := .BaseURL}}
{{$post := .Post}}
{{$user := .User}}
{{range .Replies}}
	<div id="{{.ID}}">
		<div class="comment-header">
			<span class="author">{{.Author}}</p>
			<span class="date">{{.Created}}</p>
			{{if eq .Author $user}}
			<a href="{{$baseURL}}/posts/{{$post}}/comments/{{.ID}}/delete-confirm">
				delete
			</a>
			<a href="{{$baseURL}}/posts/{{$post}}/comments/{{.ID}}/edit">
				edit
			</a>
			{{end}}
			{{/* if the user is logged in they can reply */}}
			{{if $user}}
			<a href="{{$baseURL}}/posts/{{$post}}/comments/{{.ID}}/reply">
				reply
			</a>
			{{end}}
			<p class="body">{{.Body}}</p>
		</div>
	</div>
{{end}}
</div>
</body>
</html>`
