package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"repo3/internal/storage"
)

func TestMetadataKey(t *testing.T) {
	got := metadataKey("images/cat.jpeg")
	want := ".repo3/meta/images/cat.jpeg.json"
	if got != want {
		t.Fatalf("metadataKey = %q, want %q", got, want)
	}
}

func TestListBucketsFollowsGitHubPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/orgs/acme/repos?per_page=100&type=all":
			w.Header().Set("Link", fmt.Sprintf(`<%s/orgs/acme/repos?page=2>; rel="next"`, externalURL(r)))
			_, _ = w.Write([]byte(`[{"name":"first","created_at":"2026-05-21T10:00:00Z"}]`))
		case "/orgs/acme/repos?page=2":
			_, _ = w.Write([]byte(`[{"name":"second","created_at":"2026-05-21T10:01:00Z"}]`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	store := newTestStore(server)
	buckets, err := store.ListBuckets(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	got := []string{}
	for _, bucket := range buckets {
		got = append(got, bucket.Name)
	}
	want := []string{"first", "second"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buckets = %#v, want %#v", got, want)
	}
}

func TestListObjectsFollowsTreePagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/repos/acme/memes/git/trees/main?recursive=1":
			w.Header().Set("Link", fmt.Sprintf(`<%s/repos/acme/memes/git/trees/main?page=2>; rel="next"`, externalURL(r)))
			_, _ = w.Write([]byte(`{"tree":[{"path":"images/cat.jpeg","type":"blob","sha":"catsha","size":12}]}`))
		case "/repos/acme/memes/git/trees/main?page=2":
			_, _ = w.Write([]byte(`{"tree":[{"path":"images/dog.jpeg","type":"blob","sha":"dogsha","size":34},{"path":".repo3/meta/images/dog.jpeg.json","type":"blob","sha":"metasha","size":56}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	store := newTestStore(server)
	objects, err := store.ListObjects(context.Background(), "memes", "images/", 0)
	if err != nil {
		t.Fatal(err)
	}

	got := []string{}
	for _, obj := range objects {
		got = append(got, obj.Key)
	}
	want := []string{"images/cat.jpeg", "images/dog.jpeg"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("objects = %#v, want %#v", got, want)
	}
}

func TestMapStatusRichErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		target error
	}{
		{
			name:   "conflict",
			status: http.StatusConflict,
			body:   `{"message":"sha does not match"}`,
			target: storage.ErrOperationAborted,
		},
		{
			name:   "rate limit",
			status: http.StatusTooManyRequests,
			body:   `{"message":"API rate limit exceeded"}`,
			target: storage.ErrSlowDown,
		},
		{
			name:   "validation path",
			status: http.StatusUnprocessableEntity,
			body:   `{"message":"Validation Failed","errors":[{"resource":"Contents","field":"path","code":"invalid"}]}`,
			target: storage.ErrInvalidKey,
		},
		{
			name:   "validation generic",
			status: http.StatusUnprocessableEntity,
			body:   `{"message":"Validation Failed","errors":[{"resource":"Repository","field":"private","code":"invalid"}]}`,
			target: storage.ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapStatus(tt.status, []byte(tt.body))
			if !errors.Is(err, tt.target) {
				t.Fatalf("mapStatus error = %v, want %v", err, tt.target)
			}
		})
	}
}

func newTestStore(server *httptest.Server) *Store {
	return &Store{
		token:   "test-token",
		owner:   "acme",
		branch:  "main",
		apiBase: server.URL,
		client:  server.Client(),
	}
}

func externalURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
