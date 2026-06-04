package users

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenIssuer creates auth tokens for a user. Abstracting it keeps the service
// independent from the concrete signing strategy.
type TokenIssuer interface {
	Issue(u *User) (string, error)
}

// Claims is the JWT payload carried by issued tokens.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// JWTIssuer issues and parses HS256-signed JSON Web Tokens.
type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTIssuer builds an issuer with the given signing secret and token TTL.
func NewJWTIssuer(secret string, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), ttl: ttl}
}

// Issue returns a signed token whose subject is the user ID.
func (j *JWTIssuer) Issue(u *User) (string, error) {
	now := time.Now()
	claims := Claims{
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}

// Parse validates a signed token and returns its claims.
func (j *JWTIssuer) Parse(tokenString string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return &claims, nil
}
