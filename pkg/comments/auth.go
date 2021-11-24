package comments

import (
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/weberc2/auth/pkg/client"
	"github.com/weberc2/comments/pkg/types"
	pz "github.com/weberc2/httpeasy"
)

type Authenticator struct {
	Auth client.Client
	Key  *ecdsa.PublicKey
}

func (a *Authenticator) AuthN(authType AuthType, h pz.Handler) pz.Handler {
	return func(r pz.Request) pz.Response {
		result := authType.validate(a.Key, r)
		r.Headers.Add("User", string(result.User))
		return h(r).WithLogging(result)
	}
}

func (a *Authenticator) AuthZ(authType AuthType, h pz.Handler) pz.Handler {
	return func(r pz.Request) pz.Response {
		result := authType.validate(a.Key, r)
		if result.User == "" {
			return pz.Unauthorized(nil, result)
		}
		return h(r).WithLogging(result)
	}
}

type AuthType interface {
	validate(key *ecdsa.PublicKey, r pz.Request) *result
}

type AuthTypeClientProgram struct{}

func (atcp AuthTypeClientProgram) validate(
	key *ecdsa.PublicKey,
	r pz.Request,
) *result {
	authorization := r.Headers.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		return resultErr(
			"invalid access token",
			fmt.Errorf("missing `Bearer` prefix"),
		)
	}

	user, err := validateAccessToken(authorization[len("Bearer "):], key)
	if err != nil {
		return resultErr("invalid access token", err)
	}

	return resultOK("successfully validated access token", user)
}

type result struct {
	Message string       `json:"message"`
	Error   string       `json:"error,omitempty"`
	User    types.UserID `json:"user,omitempty"`
}

func resultErr(message string, err error) *result {
	return &result{Message: message, Error: err.Error()}
}

func resultOK(message string, user types.UserID) *result {
	return &result{Message: message, User: user}
}

type AuthTypeWebServer struct {
	Auth client.Client
}

func (atws *AuthTypeWebServer) validate(
	key *ecdsa.PublicKey,
	r pz.Request,
) *result {
	accessCookie, err := r.Cookie("Access-Token")
	if err != nil {
		return resultErr("missing `Access-Token` cookie", err)
	}

	refreshCookie, err := r.Cookie("Refresh-Token")
	if err != nil {
		return resultErr("missing `Refresh-Token` cookie", err)
	}

	user, err := validateAccessToken(accessCookie.Value, key)
	if err != nil {
		if err, ok := err.(*jwt.ValidationError); ok {
			masked := err.Errors & jwt.ValidationErrorExpired
			if masked == jwt.ValidationErrorExpired {
				tokens, err := atws.Auth.Refresh(refreshCookie.Value)
				if err != nil {
					return resultErr("refreshing access token", err)
				}

				// We can probably trust that the token itself is good since
				// it's coming directly from the auth service, but we need its
				// user. If we got here, the previous access token's user
				// failed to parse because the token was expired.
				user, err := validateAccessToken(tokens.AccessToken, key)
				if err != nil {
					return resultErr("parsing `sub` (user) claim", err)
				}
				accessCookie.Value = tokens.AccessToken
				return resultOK("successfully refreshed access token", user)
			}
			return resultErr("validating access token", err)
		}
		return resultErr("parsing access token", err)
	}

	return resultOK("successfully validated access token", user)
}

func validateAccessToken(
	token string,
	key *ecdsa.PublicKey,
) (types.UserID, error) {
	var claims jwt.StandardClaims
	if _, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(*jwt.Token) (interface{}, error) {
			return key, nil
		},
	); err != nil {
		return "", err
	}
	return types.UserID(claims.Subject), nil
}
