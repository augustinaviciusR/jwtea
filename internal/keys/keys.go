package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"log"
	"math/big"
)

type JwkRSA struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func MustGenerateRSA() (*rsa.PrivateKey, string, JwkRSA) {
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("generate RSA key: %v", err)
	}
	pub := &pk.PublicKey

	spki, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		log.Fatalf("marshal public key: %v", err)
	}
	sum := sha256.Sum256(spki)
	kid := base64.RawURLEncoding.EncodeToString(sum[:])

	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eInt := big.NewInt(int64(pub.E))
	e := base64.RawURLEncoding.EncodeToString(eInt.Bytes())

	jwk := JwkRSA{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   n,
		E:   e,
	}
	return pk, kid, jwk
}
