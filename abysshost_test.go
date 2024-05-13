package abyssgo

import (
	"testing"
)

func TestAbyssHostCreation(t *testing.T) {
	credential, err := NewCredential("anon")
	if err != nil {
		t.Fatalf("NewCredential returned error: %s", err.Error())
	}
	_, err = MakeAbysshost(credential)
	if err != nil {
		t.Fatalf("MakeAbysshost returned error: %s", err.Error())
	}
}
