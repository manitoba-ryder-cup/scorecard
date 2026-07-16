// Package jwt mints access tokens for the integration suite. Scorecard is a resource
// server — it never issues tokens — so tests sign their own with the same key pair
// the server validates against, standing in for a heimdall-issued token.
package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
	util "github.com/manitoba-ryder-cup/scorecard/test/_util"
	kjwt "github.com/travisbale/knowhere/jwt"
)

// MintAccessToken issues a signed access token carrying the given tenant, user, and
// scopes, accepted by the scorecard server's JWT middleware.
func MintAccessToken(t *testing.T, tenantID, userID uuid.UUID, scopes ...string) string {
	t.Helper()

	cfg := util.LoadConfig()
	issuer, err := kjwt.NewIssuer(&kjwt.Config{
		Issuer:                "scorecard-integration-tests",
		PrivateKeyPath:        cfg.JWTPrivateKeyPath,
		AccessTokenExpiration: time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create JWT issuer: %v", err)
	}

	scopeVals := make([]kjwt.Scope, len(scopes))
	for i, s := range scopes {
		scopeVals[i] = kjwt.Scope(s)
	}

	token, _, err := issuer.IssueAccessToken(tenantID, userID, scopeVals)
	if err != nil {
		t.Fatalf("failed to issue access token: %v", err)
	}
	return token
}
