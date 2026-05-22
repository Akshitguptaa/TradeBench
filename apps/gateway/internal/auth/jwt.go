// Package auth provides HMAC-SHA256 JWT issuance and verification for the
// TradeBench gateway. In dev mode the secret is loaded from an env var and no
// external identity provider is used.
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when a token cannot be parsed or validated.
var ErrInvalidToken = errors.New("invalid or expired token")

// JWTAuth handles token issuance and verification using HS256.
type JWTAuth struct {
	secret []byte
	expiry time.Duration
}

// New creates a JWTAuth with the given HMAC secret and token lifetime.
func New(secret string, expiry time.Duration) *JWTAuth {
	return &JWTAuth{
		secret: []byte(secret),
		expiry: expiry,
	}
}

// IssueToken creates a signed HS256 JWT for the given contestant.
// The token contains a "sub" (contestant_id) claim and expires after the
// configured duration.
func (a *JWTAuth) IssueToken(contestantID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   contestantID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(a.expiry)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secret)
}

// VerifyToken parses and validates a JWT string, returning the contestant_id
// (subject claim) on success.
func (a *JWTAuth) VerifyToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return a.secret, nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}

	return claims.Subject, nil
}
