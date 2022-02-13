package comments

import (
	"github.com/weberc2/auth/pkg/client"
	pz "github.com/weberc2/httpeasy"
)

type AuthWebServer struct {
	WebServer
	client.AuthType
	client.Authenticator
}

func (aws *AuthWebServer) RepliesRoute() pz.Route {
	return aws.optional(aws.WebServer.RepliesRoute())
}

func (aws *AuthWebServer) DeleteConfirmRoute() pz.Route {
	return aws.auth(aws.WebServer.DeleteConfirmRoute())
}

func (aws *AuthWebServer) DeleteRoute() pz.Route {
	return aws.auth(aws.WebServer.DeleteRoute())
}

func (aws *AuthWebServer) ReplyFormRoute() pz.Route {
	return aws.auth(aws.WebServer.ReplyFormRoute())
}

func (aws *AuthWebServer) ReplyRoute() pz.Route {
	return aws.auth(aws.WebServer.ReplyRoute())
}

func (aws *AuthWebServer) EditFormRoute() pz.Route {
	return aws.auth(aws.WebServer.EditFormRoute())
}

func (aws *AuthWebServer) EditRoute() pz.Route {
	return aws.auth(aws.WebServer.EditRoute())
}

func (aws *AuthWebServer) Routes() []pz.Route {
	return []pz.Route{
		aws.RepliesRoute(),
		aws.DeleteConfirmRoute(),
		aws.DeleteRoute(),
		aws.ReplyFormRoute(),
		aws.ReplyRoute(),
		aws.EditFormRoute(),
		aws.EditRoute(),
	}
}

func (aws *AuthWebServer) auth(r pz.Route) pz.Route {
	return pz.Route{
		Method:  r.Method,
		Path:    r.Path,
		Handler: aws.Authenticator.Auth(aws.AuthType, r.Handler),
	}
}

func (aws *AuthWebServer) optional(r pz.Route) pz.Route {
	return pz.Route{
		Method:  r.Method,
		Path:    r.Path,
		Handler: aws.Authenticator.Optional(aws.AuthType, r.Handler),
	}
}
