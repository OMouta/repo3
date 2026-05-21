package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPutCreateBucketRetriesUpload(t *testing.T) {
	var requests []string
	var contentType string
	var owner string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPut && r.URL.Path == "/memes/hello.txt" && len(requests) == 1 {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`<Error><Code>NoSuchBucket</Code><Message>The specified bucket does not exist.</Message></Error>`))
			return
		}
		if r.Method == http.MethodPut && r.URL.Path == "/memes/hello.txt" {
			contentType = r.Header.Get("Content-Type")
			owner = r.Header.Get("x-amz-meta-owner")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dir := t.TempDir()
	localPath := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(localPath, []byte("hello from repo3"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Run([]string{"put", localPath, "s3://memes/hello.txt", "--create-bucket", "--content-type", "text/plain", "--meta", "owner=idk", "--endpoint", server.URL}, bytes.NewReader(nil), &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run returned error: %v; stderr=%s", err, stderr.String())
	}

	want := []string{
		"PUT /memes/hello.txt",
		"PUT /memes",
		"PUT /memes/hello.txt",
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests = %#v, want %#v", requests, want)
	}
	if contentType != "text/plain" {
		t.Fatalf("Content-Type = %q", contentType)
	}
	if owner != "idk" {
		t.Fatalf("x-amz-meta-owner = %q", owner)
	}
}

func TestMetadataFlags(t *testing.T) {
	flags := metadataFlags{"owner=idk", "x-amz-meta-uploaded-by=repo3"}
	got, err := flags.Map()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"owner": "idk", "uploaded-by": "repo3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata = %#v, want %#v", got, want)
	}
}
