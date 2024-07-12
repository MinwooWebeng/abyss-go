package anet

import (
	"abyss/and"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func NewPemBytes() []byte {
	pub_key, _, _ := ed25519.GenerateKey(rand.Reader)
	pubkey_x509, _ := x509.MarshalPKIXPublicKey(pub_key)
	return pem.EncodeToMemory(&pem.Block{Type: "OPENSSH PUBLIC KEY", Bytes: pubkey_x509})
}

func CompareNDE(target and.NeighborDiscoveryEvent, correct and.NeighborDiscoveryEvent) (bool, string) {
	return target.EventType == correct.EventType &&
		target.Localpath == correct.Localpath &&
		target.Peer_hash == correct.Peer_hash &&
		target.Path == correct.Path &&
		target.World.GetUUID() == correct.World.GetUUID() &&
		target.Status == correct.Status &&
		target.Message == correct.Message, "expected: " + correct.Stringify() + "\ngot: " + target.Stringify()
}
func CompareNDEMultiple(subject and.NeighborDiscoveryEvent, correct []and.NeighborDiscoveryEvent) (bool, string, []and.NeighborDiscoveryEvent) {
	for i, ev := range correct {
		if ok, _ := CompareNDE(subject, ev); ok {
			return true, "", append(correct[:i], correct[i+1:]...)
		}
	}

	return false, "no matching nde: " + subject.Stringify(), nil
}

func TimeoutCheckNDE(networker1 *Networker, correct and.NeighborDiscoveryEvent) (bool, string) {
	var ok bool
	var msg string
	done := make(chan bool)
	go func() {
		ok, msg = CompareNDE(<-networker1.NdhEventCh, correct)
		done <- true
	}()

	select {
	case <-done:
		return ok, msg
	case <-time.After(time.Second):
		return false, "timeout"
	}
}
func TimeoutCheckNDEMultiple(networker1 *Networker, correct []and.NeighborDiscoveryEvent) (bool, string) {
	if len(correct) == 0 {
		return true, ""
	}

	var ok bool
	var msg string
	done := make(chan bool)
	go func() {
		for len(correct) != 0 {
			ok, msg, correct = CompareNDEMultiple(<-networker1.NdhEventCh, correct)
		}
		done <- true
	}()

	select {
	case <-done:
		return ok, msg
	case <-time.After(time.Second):
		return false, "timeout"
	}
}

func TestNetworkerCreation(t *testing.T) {
	networker1, err := NewNetworker(NewPemBytes(), "hostA")
	if err != nil {
		t.Fatal(err)
	}
	networker2, _ := NewNetworker(NewPemBytes(), "hostB")
	networker3, _ := NewNetworker(NewPemBytes(), "hostC")

	networker1.WaitClose()
	networker2.WaitClose()
	networker3.WaitClose()
}

func TestNetworkerJoin(t *testing.T) {
	networker1, _ := NewNetworker(NewPemBytes(), "hostA")
	w1 := NewWorld("https://www.abyssium.com/some_world.aml")
	networker1.OpenWorld("/home", w1)

	time.Sleep(time.Second)

	networker2, _ := NewNetworker(NewPemBytes(), "hostB")
	networker2.JoinAny("/host1_home", networker1.netcore.LocalAddr(), networker1.netcore.LocalAddr().Pubkey_hash, "/home")

	fmt.Println("waiting...")

	h1 := networker1.netcore.LocalIdentity().Hash
	h2 := networker2.netcore.LocalIdentity().Hash

	if ok, msg := TimeoutCheckNDE(networker1,
		and.NeighborDiscoveryEvent{
			EventType: and.PeerJoin, Localpath: "",
			Peer_hash: h2, Peer: nil, Path: "",
			World: w1, Status: 0, Message: ""}); !ok {
		t.Fatal(msg)
	}
	if ok, msg := TimeoutCheckNDE(networker2,
		and.NeighborDiscoveryEvent{
			EventType: and.JoinSuccess, Localpath: "/host1_home",
			Peer_hash: h1, Peer: nil, Path: "/home",
			World: w1, Status: 200, Message: "OK"}); !ok {
		t.Fatal(msg)
	}
	if ok, msg := TimeoutCheckNDE(networker2,
		and.NeighborDiscoveryEvent{
			EventType: and.PeerJoin, Localpath: "",
			Peer_hash: h1, Peer: nil, Path: "",
			World: w1, Status: 0, Message: ""}); !ok {
		t.Fatal(msg)
	}

	is_fin := false
	select {
	case <-networker1.NdhEventCh:
	case <-networker2.NdhEventCh:
	case <-time.After(time.Second):
		is_fin = true
	}
	if !is_fin {
		t.Fatal("why more events?")
	}

	networker1.WaitClose()
	networker2.WaitClose()
}

func TestNetworkerJoinDouble(t *testing.T) {
	networker1, _ := NewNetworker(NewPemBytes(), "hostA")
	w1 := NewWorld("https://www.abyssium.com/some_world.aml")
	networker1.OpenWorld("/home", w1)

	time.Sleep(time.Second)

	networker2, _ := NewNetworker(NewPemBytes(), "hostB")
	networker2.JoinAny("/B_host1_home", networker1.netcore.LocalAddr(), networker1.netcore.LocalAddr().Pubkey_hash, "/home")

	networker3, _ := NewNetworker(NewPemBytes(), "hostC")
	networker3.JoinAny("/C_host1_home", networker1.netcore.LocalAddr(), networker1.netcore.LocalAddr().Pubkey_hash, "/home")

	fmt.Println("waiting...")

	h1 := networker1.netcore.LocalIdentity().Hash
	h2 := networker2.netcore.LocalIdentity().Hash
	h3 := networker3.netcore.LocalIdentity().Hash

	if ok, msg := TimeoutCheckNDEMultiple(networker1,
		[]and.NeighborDiscoveryEvent{
			{
				EventType: and.PeerJoin, Localpath: "",
				Peer_hash: h2, Peer: nil, Path: "",
				World: w1, Status: 0, Message: ""},
			{
				EventType: and.PeerJoin, Localpath: "",
				Peer_hash: h3, Peer: nil, Path: "",
				World: w1, Status: 0, Message: ""},
		}); !ok {
		t.Fatal(msg)
	}
	if ok, msg := TimeoutCheckNDE(networker2,
		and.NeighborDiscoveryEvent{
			EventType: and.JoinSuccess, Localpath: "/B_host1_home",
			Peer_hash: h1, Peer: nil, Path: "/home",
			World: w1, Status: 200, Message: "OK"}); !ok {
		t.Fatal(msg)
	}
	if ok, msg := TimeoutCheckNDEMultiple(networker2,
		[]and.NeighborDiscoveryEvent{
			{
				EventType: and.PeerJoin, Localpath: "",
				Peer_hash: h1, Peer: nil, Path: "",
				World: w1, Status: 0, Message: ""},
			{
				EventType: and.PeerJoin, Localpath: "",
				Peer_hash: h3, Peer: nil, Path: "",
				World: w1, Status: 0, Message: ""},
		}); !ok {
		t.Fatal(msg)
	}
	if ok, msg := TimeoutCheckNDE(networker3,
		and.NeighborDiscoveryEvent{
			EventType: and.JoinSuccess, Localpath: "/C_host1_home",
			Peer_hash: h1, Peer: nil, Path: "/home",
			World: w1, Status: 200, Message: "OK"}); !ok {
		t.Fatal(msg)
	}
	if ok, msg := TimeoutCheckNDEMultiple(networker3,
		[]and.NeighborDiscoveryEvent{
			{
				EventType: and.PeerJoin, Localpath: "",
				Peer_hash: h1, Peer: nil, Path: "",
				World: w1, Status: 0, Message: ""},
			{
				EventType: and.PeerJoin, Localpath: "",
				Peer_hash: h2, Peer: nil, Path: "",
				World: w1, Status: 0, Message: ""},
		}); !ok {
		t.Fatal(msg)
	}

	is_fin := false
	var evn int
	var ev and.NeighborDiscoveryEvent
	select {
	case ev = <-networker1.NdhEventCh:
		evn = 1
	case ev = <-networker2.NdhEventCh:
		evn = 2
	case ev = <-networker3.NdhEventCh:
		evn = 3
	case <-time.After(time.Second):
		is_fin = true
	}
	if !is_fin {
		t.Fatal("why more events(" + strconv.Itoa(evn) + "): " + ev.Stringify())
	}

	networker1.WaitClose()
	networker2.WaitClose()
}
