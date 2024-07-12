package anet

import (
	"abyss/atype"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
)

func CreateRandomHost() (ed25519.PrivateKey, ed25519.PublicKey, INetCore, error) {
	pub_key1, priv_key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}
	pubkey1, err := x509.MarshalPKIXPublicKey(pub_key1)
	if err != nil {
		return nil, nil, nil, err
	}
	pemfile1 := pem.EncodeToMemory(&pem.Block{Type: "OPENSSH PUBLIC KEY", Bytes: pubkey1})
	id1, err := atype.MakeAbyssIdentity(pemfile1, "host1")
	if err != nil {
		return nil, nil, nil, err
	}
	nc1, err := NewGoQuicNetCore(id1)
	if err != nil {
		return nil, nil, nil, err
	}
	return priv_key, pub_key1, nc1, nil
}

func TestNetCoreSimple(t *testing.T) {
	_, _, nc1, err := CreateRandomHost()
	if err != nil {
		t.Fatal("failed to generate net core: " + err.Error())
	}
	_, _, nc2, err := CreateRandomHost()
	if err != nil {
		t.Fatal("failed to generate net core: " + err.Error())
	}

	nc2_perceived_hash := make(chan string, 1)
	nc2_remchan := make(chan *Transmission, 1)
	go_err := make(chan error, 1)
	go func() {
		nc2_remote, err := nc1.Accept()
		if err != nil {
			nc2_remchan <- nil
			go_err <- err
			return
		}
		nc2_perceived_hash <- nc2_remote.GetHash()
		nc2_remchan <- nc2_remote
		go_err <- nil
	}()

	nc1_remote, err := nc2.Connect(nc1.LocalAddr())
	if err != nil {
		t.Fatal("failed to connect: " + err.Error())
	}

	if ge := <-go_err; ge != nil {
		t.Fatal("failed to accept: " + ge.Error())
	}

	if nc2.LocalIdentity().Hash != <-nc2_perceived_hash {
		t.Fatal("failed to check nc2 hash")
	}
	if nc1.LocalIdentity().Hash != nc1_remote.GetHash() {
		t.Fatal("failed to check nc1 hash")
	}

	nc1_rem_id := nc1_remote.identity
	nc2_rem_id := (<-nc2_remchan).identity
	fmt.Println(string(nc1_rem_id.Name))
	fmt.Println(string(nc1_rem_id.Hash))
	fmt.Println(string(nc1_rem_id.Publickey))
	fmt.Println(string(nc2_rem_id.Name))
	fmt.Println(string(nc2_rem_id.Hash))
	fmt.Println(string(nc2_rem_id.Publickey))
}

// TODO: simultaneous dial test
func TestSimultaneousConnect(t *testing.T) {
	_, _, nc1, err := CreateRandomHost()
	if err != nil {
		t.Fatal("failed to generate net core: " + err.Error())
	}
	_, _, nc2, err := CreateRandomHost()
	if err != nil {
		t.Fatal("failed to generate net core: " + err.Error())
	}

	go1_done := make(chan error, 1)
	go2_done := make(chan error, 1)

	go func() {
		var err error
		defer func() {
			go1_done <- err
		}()
		_, err = nc1.Connect(nc2.LocalAddr())
		if err != nil {
			return
		}
		_, err = nc1.Connect(nc2.LocalAddr())
		if err != nil {
			return
		}
	}()
	go func() {
		var err error
		defer func() {
			go2_done <- err
		}()
		_, err = nc2.Accept()
		if err != nil {
			return
		}
		_, err = nc2.Accept()
		if err != nil {
			return
		}
	}()

	err1 := <-go1_done
	if err1 != nil {
		t.Error(err1.Error())
	}
	err2 := <-go2_done
	if err2 != nil {
		t.Error(err2.Error())
	}
}
