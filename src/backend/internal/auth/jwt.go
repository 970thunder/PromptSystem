package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	Issued    int64  `json:"iat"`
	Expiry    int64  `json:"exp"`
	NotBefore int64  `json:"nbf"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	JTI       string `json:"jti"`
}

const (
	// TokenIssuer identifies this service as the JWT issuer.
	TokenIssuer = "promptos-backend"
	// TokenAudience identifies the intended API audience.
	TokenAudience = "promptos-frontend"
)

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (tm *TokenManager) Generate(userID int, email string) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	now := time.Now().UTC()
	jti, err := randomJTI()
	if err != nil {
		return "", err
	}
	claims := Claims{
		Subject:   fmt.Sprintf("%d", userID),
		Email:     email,
		Issued:    now.Unix(),
		Expiry:    now.Add(tm.ttl).Unix(),
		NotBefore: now.Unix(),
		Issuer:    TokenIssuer,
		Audience:  TokenAudience,
		JTI:       jti,
	}

	headerPayload, err := encodeSegment(header)
	if err != nil {
		return "", err
	}

	claimPayload, err := encodeSegment(claims)
	if err != nil {
		return "", err
	}

	unsigned := headerPayload + "." + claimPayload
	signature := tm.sign(unsigned)

	return unsigned + "." + signature, nil
}

func (tm *TokenManager) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	// Reject tokens whose header does not declare HS256. This prevents
	// algorithm-confusion attacks where an attacker swaps alg to none or
	// another scheme.
	headerPayload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerPayload, &header); err != nil || header.Alg != "HS256" {
		return Claims{}, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(tm.sign(unsigned)), []byte(parts[2])) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}

	if claims.Expiry <= time.Now().UTC().Unix() {
		return Claims{}, ErrExpiredToken
	}
	if claims.NotBefore > time.Now().UTC().Unix() {
		return Claims{}, ErrInvalidToken
	}
	if claims.Issuer != TokenIssuer || claims.Audience != TokenAudience {
		return Claims{}, ErrInvalidToken
	}
	if claims.JTI == "" {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}

func (tm *TokenManager) sign(unsigned string) string {
	mac := hmac.New(sha256.New, tm.secret)
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeSegment(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func randomJTI() (string, error) {
	var buffer [16]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer[:]), nil
}
