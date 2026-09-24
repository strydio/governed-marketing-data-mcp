package authn

import (
	"crypto/subtle"
	"errors"
	"strings"
)

var (
	ErrMissingCredentials = errors.New("missing credentials")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Principal struct {
	Ref string
}

type Authenticator struct {
	demoToken string
}

func NewAuthenticator(demoToken string) (*Authenticator, error) {
	if demoToken == "" {
		return nil, ErrMissingCredentials
	}
	return &Authenticator{demoToken: demoToken}, nil
}

func (a *Authenticator) Authenticate(authorisationHeader string) (Principal, error) {
	if authorisationHeader == "" {
		return Principal{}, ErrMissingCredentials
	}
	parts := strings.SplitN(authorisationHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return Principal{}, ErrInvalidCredentials
	}
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(a.demoToken)) != 1 {
		return Principal{}, ErrInvalidCredentials
	}
	return Principal{Ref: "uk_marketing_demo"}, nil
}
