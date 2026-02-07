package core

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"time"

	"jwtea/internal/keys"

	"github.com/golang-jwt/jwt/v5"
)

func generateJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

type TokenGenerator struct {
	PrivKey *rsa.PrivateKey
	Kid     string
	Issuer  string
}

type TokenRequest struct {
	Subject               string
	Audience              string
	Scope                 string
	ExpiresIn             time.Duration
	CustomClaims          map[string]any
	ChaosExpired          bool
	ChaosInvalidSignature bool
}

type TokenResult struct {
	AccessToken string
	IDToken     string
	ExpiresIn   int64
}

func NewTokenGenerator(privKey *rsa.PrivateKey, kid, issuer string) *TokenGenerator {
	return &TokenGenerator{
		PrivKey: privKey,
		Kid:     kid,
		Issuer:  issuer,
	}
}

func (g *TokenGenerator) Generate(req TokenRequest) (*TokenResult, error) {
	now := time.Now()

	atExp := now.Add(req.ExpiresIn)
	if req.ChaosExpired {
		atExp = now.Add(-1 * time.Hour)
	}

	accessClaims := jwt.MapClaims{
		"iss": g.Issuer,
		"sub": req.Subject,
		"aud": req.Audience,
		"iat": now.Unix(),
		"exp": atExp.Unix(),
		"jti": generateJTI(),
	}

	if req.Scope != "" {
		accessClaims["scope"] = req.Scope
	}

	for k, v := range req.CustomClaims {
		accessClaims[k] = v
	}

	at := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	at.Header["kid"] = g.Kid

	signingKey := g.PrivKey
	if req.ChaosInvalidSignature {
		k, _, _ := keys.MustGenerateRSA()
		signingKey = k
	}

	signedAT, err := at.SignedString(signingKey)
	if err != nil {
		return nil, err
	}

	idExp := now.Add(req.ExpiresIn)
	if req.ChaosExpired {
		idExp = now.Add(-1 * time.Hour)
	}

	idClaims := jwt.MapClaims{
		"iss": g.Issuer,
		"sub": req.Subject,
		"aud": req.Audience,
		"iat": now.Unix(),
		"exp": idExp.Unix(),
	}

	idt := jwt.NewWithClaims(jwt.SigningMethodRS256, idClaims)
	idt.Header["kid"] = g.Kid
	signedIDT, err := idt.SignedString(signingKey)
	if err != nil {
		return nil, err
	}

	return &TokenResult{
		AccessToken: signedAT,
		IDToken:     signedIDT,
		ExpiresIn:   int64(atExp.Sub(now).Seconds()),
	}, nil
}

func ParseAndValidateToken(tokenStr string, pubKey *rsa.PublicKey) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
