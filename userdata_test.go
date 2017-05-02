package dochaincore

import "testing"

func TestBuildUserData(t *testing.T) {
	keyPair, err := createSSHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	s, err := buildUserData(&options{}, keyPair)
	if err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Error("got empty string user data")
	}
}
