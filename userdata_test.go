package dochaincore

import "testing"

func TestBuildUserData(t *testing.T) {
	s, err := buildUserData(&options{})
	if err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Error("got empty string user data")
	}
}
