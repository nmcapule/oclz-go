package oauth2

import (
	"time"

	"github.com/tidwall/gjson"
)

type CredentialsManager interface {
	GenerateAuthorizationURL() string
	GenerateCredentials(data gjson.Result) (*Credentials, error)
	RefreshCredentials() (*Credentials, error)
	CredentialsExpiry() time.Time
}
