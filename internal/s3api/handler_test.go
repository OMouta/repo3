package s3api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"repo3/internal/storage"
)

func TestObjectLifecycle(t *testing.T) {
	handler := NewHandler(storage.NewMemoryStore())

	req := httptest.NewRequest(http.MethodPut, "/memes", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create bucket status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/memes/cat.txt", strings.NewReader("meow"))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put object status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatal("put object did not return ETag")
	}

	req = httptest.NewRequest(http.MethodGet, "/memes/cat.txt", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get object status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got := rec.Body.String(); got != "meow" {
		t.Fatalf("get object body = %q, want %q", got, "meow")
	}

	req = httptest.NewRequest(http.MethodHead, "/memes/cat.txt", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("head object status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("head object body length = %d, want 0", rec.Body.Len())
	}

	req = httptest.NewRequest(http.MethodDelete, "/memes/cat.txt", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete object status = %d, want %d; body %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestPutObjectMetadataHeadersRoundTrip(t *testing.T) {
	handler := NewHandler(storage.NewMemoryStore())
	mustRequest(t, handler, http.MethodPut, "/memes", nil, http.StatusOK)

	req := httptest.NewRequest(http.MethodPut, "/memes/cat.txt", strings.NewReader("meow"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("x-amz-meta-owner", "idk")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put object status = %d, want %d; body %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodHead, "/memes/cat.txt", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("head object status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("x-amz-meta-owner"); got != "idk" {
		t.Fatalf("x-amz-meta-owner = %q", got)
	}
}

func TestListBucketsAndObjects(t *testing.T) {
	handler := NewHandler(storage.NewMemoryStore())

	mustRequest(t, handler, http.MethodPut, "/memes", nil, http.StatusOK)
	mustRequest(t, handler, http.MethodPut, "/memes/images/cat.txt", strings.NewReader("cat"), http.StatusOK)
	mustRequest(t, handler, http.MethodPut, "/memes/docs/readme.txt", strings.NewReader("readme"), http.StatusOK)

	rec := mustRequest(t, handler, http.MethodGet, "/", nil, http.StatusOK)
	if body := rec.Body.String(); !strings.Contains(body, "<Name>memes</Name>") {
		t.Fatalf("list buckets body missing bucket: %s", body)
	}

	rec = mustRequest(t, handler, http.MethodGet, "/memes?list-type=2&prefix=images/", nil, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, "<Key>images/cat.txt</Key>") {
		t.Fatalf("list objects body missing matching key: %s", body)
	}
	if strings.Contains(body, "<Key>docs/readme.txt</Key>") {
		t.Fatalf("list objects body included non-matching key: %s", body)
	}
}

func TestNoSuchKeyXML(t *testing.T) {
	handler := NewHandler(storage.NewMemoryStore())
	mustRequest(t, handler, http.MethodPut, "/memes", nil, http.StatusOK)

	rec := mustRequest(t, handler, http.MethodGet, "/memes/missing.txt", nil, http.StatusNotFound)
	body := rec.Body.String()
	if !strings.Contains(body, "<Code>NoSuchKey</Code>") || !strings.Contains(body, "<Key>missing.txt</Key>") {
		t.Fatalf("unexpected error body: %s", body)
	}
}

func mustRequest(t *testing.T, handler http.Handler, method, target string, body io.Reader, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body %s", method, target, rec.Code, wantStatus, rec.Body.String())
	}
	return rec
}
