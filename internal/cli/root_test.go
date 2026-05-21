package cli

import (
	"reflect"
	"testing"
)

func TestReorderFlags(t *testing.T) {
	got := reorderFlags([]string{"file.txt", "s3://memes/file.txt", "--create-bucket", "--endpoint", "http://example.test"})
	want := []string{"--create-bucket", "--endpoint", "http://example.test", "file.txt", "s3://memes/file.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reorderFlags = %#v, want %#v", got, want)
	}
}
