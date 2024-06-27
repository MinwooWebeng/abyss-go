package atype

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

func TestAbyssIdentity(t *testing.T) {
	priv_key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		t.Fatal("failed to generate rsa key pair")
	}
	pubkey := x509.MarshalPKCS1PublicKey(&priv_key.PublicKey)
	pemfile := pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Headers: nil, Bytes: pubkey})
	identity, err := MakeAbyssIdentity(pemfile, "mallang")
	if err != nil {
		t.Fatal("failed to encode public key")
	}

	hashfunc := sha3.New256()
	hashfunc.Write(pubkey)
	hashfunc.Write([]byte("mallang"))
	hashval := base58.Encode(hashfunc.Sum(nil))

	if hashval != identity.Hash {
		t.Fatal("hash value mismatch")
	}
}
