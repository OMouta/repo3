package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassifyAction(t *testing.T) {
	tests := []struct {
		method     string
		target     string
		wantAction string
		wantBucket string
		wantKey    string
	}{
		{method: http.MethodGet, target: "/", wantAction: "list_buckets"},
		{method: http.MethodPut, target: "/memes", wantAction: "create_bucket", wantBucket: "memes"},
		{method: http.MethodGet, target: "/memes?list-type=2&prefix=images/", wantAction: "list_objects", wantBucket: "memes"},
		{method: http.MethodPut, target: "/memes/images/cat.jpeg", wantAction: "put_object", wantBucket: "memes", wantKey: "images/cat.jpeg"},
		{method: http.MethodGet, target: "/memes/images/cat.jpeg", wantAction: "get_object", wantBucket: "memes", wantKey: "images/cat.jpeg"},
		{method: http.MethodHead, target: "/memes/images/cat.jpeg", wantAction: "head_object", wantBucket: "memes", wantKey: "images/cat.jpeg"},
		{method: http.MethodDelete, target: "/memes/images/cat.jpeg", wantAction: "delete_object", wantBucket: "memes", wantKey: "images/cat.jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.target, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			action, bucket, key := classifyAction(req)
			if action != tt.wantAction || bucket != tt.wantBucket || key != tt.wantKey {
				t.Fatalf("classifyAction = (%q, %q, %q), want (%q, %q, %q)", action, bucket, key, tt.wantAction, tt.wantBucket, tt.wantKey)
			}
		})
	}
}
