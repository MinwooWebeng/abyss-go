package atype

import (
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
)

// utf-8
type AbyssIdentity struct {
	Publickey []byte //pem file content
	Name      string

	Parsed_publickey any
	Hash             string //never directly write this
}

func MakeAbyssIdentity(publickey []byte, name string) (AbyssIdentity, error) {
	var identity AbyssIdentity
	identity.Publickey = publickey
	identity.Name = name

	block_p, _ := pem.Decode(publickey)
	if block_p == nil {
		return identity, errors.New("invalid public key")
	}

	switch block_p.Type {
	case "RSA PUBLIC KEY":
		parsed_publickey, err := x509.ParsePKCS1PublicKey(block_p.Bytes)
		if err != nil {
			return identity, err
		}
		identity.Parsed_publickey = parsed_publickey
	case "OPENSSH PUBLIC KEY":
		parsed_publickey, err := x509.ParsePKIXPublicKey(block_p.Bytes)
		if err != nil {
			return identity, err
		}
		identity.Parsed_publickey = parsed_publickey
	default:
		return identity, errors.New("unsupported public key")
	}

	hashfunc := sha3.New256()
	hashfunc.Write(block_p.Bytes)
	hashfunc.Write([]byte(name))
	hash := hashfunc.Sum(nil)

	identity.Hash = base58.Encode(hash)
	return identity, nil
}
