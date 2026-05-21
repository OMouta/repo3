package github

import "testing"

func TestMetadataKey(t *testing.T) {
	got := metadataKey("images/cat.jpeg")
	want := ".repo3/meta/images/cat.jpeg.json"
	if got != want {
		t.Fatalf("metadataKey = %q, want %q", got, want)
	}
}
