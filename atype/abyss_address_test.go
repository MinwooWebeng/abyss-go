package atype

import (
	"testing"
)

func TestAbyssAddress(t *testing.T) {
	address, ok := MakeAbyssAddress("fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i", "192.168.0.1", 1605, "/home")
	if !ok {
		t.Fatal("failed to make address")
	}
	if address.Text != "abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168.0.1:1605/home" {
		t.Fatal("text result not match")
	}

	addr2, ok := ParseAbyssAddress(address.Text)
	if !ok {
		t.Fatal("failed to parse address")
	}
	if address != addr2 {
		t.Fatal("address not match")
	}
}
func TestAbyssAddress_err1(t *testing.T) { //too short public key hash (less than 32 bytes)
	_, ok := MakeAbyssAddress("1234567890123456789012345678901", "192.168.0.1", 1605, "/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}

	_, ok = ParseAbyssAddress("abyss:1234567890123456789012345678901:192.168.0.1:1605/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
}

// func TestAbyssAddress_err2(t *testing.T) { //invalid ipv4 address
// 	_, ok := MakeAbyssAddress("fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i", "192.168", 1605, "/home")
// 	if ok {
// 		t.Fatal("failed to detect faulty address")
// 	}
// 	_, ok = ParseAbyssAddress("abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168:1605/home")
// 	if ok {
// 		t.Fatal("failed to detect faulty address")
// 	}
// }

func TestAbyssAddress_err3(t *testing.T) { //invalid port number
	_, ok := MakeAbyssAddress("fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i", "192.168.0.1", 0, "/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
	_, ok = ParseAbyssAddress("abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168.0.1:0/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
	_, ok = ParseAbyssAddress("abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168.0.1:65536/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
	_, ok = ParseAbyssAddress("abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168.0.1:-2327/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
	_, ok = ParseAbyssAddress("abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168.0.1:65d53a6/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
	_, ok = ParseAbyssAddress("abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168.0.1:/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
	_, ok = ParseAbyssAddress("abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168.0.1:1605a/home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
}

func TestAbyssAddress_err4(t *testing.T) { //invalid path
	_, ok := MakeAbyssAddress("fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i", "192.168.0.1", 1605, "home")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
	_, ok = ParseAbyssAddress("abyss:fJa6v71dbjNvQfoBNMdLqtflUIZNLvd4Q3XUl67Ypi0i:192.168.0.1:1605")
	if ok {
		t.Fatal("failed to detect faulty address")
	}
}
