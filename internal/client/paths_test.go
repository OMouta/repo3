package client

import "testing"

func TestParseS3Path(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantBucket string
		wantKey    string
	}{
		{name: "object", raw: "s3://memes/cat.png", wantBucket: "memes", wantKey: "cat.png"},
		{name: "nested object", raw: "s3://memes/images/cat.png", wantBucket: "memes", wantKey: "images/cat.png"},
		{name: "prefix", raw: "s3://memes/images/", wantBucket: "memes", wantKey: "images/"},
		{name: "bucket only", raw: "s3://memes", wantBucket: "memes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseS3Path(tt.raw)
			if err != nil {
				t.Fatalf("ParseS3Path returned error: %v", err)
			}
			if got.Bucket != tt.wantBucket || got.Key != tt.wantKey {
				t.Fatalf("ParseS3Path = %#v, want bucket %q key %q", got, tt.wantBucket, tt.wantKey)
			}
		})
	}
}

func TestParseS3PathInvalid(t *testing.T) {
	for _, raw := range []string{"s3://", "s3:///cat.png", "memes/cat.png"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := ParseS3Path(raw); err == nil {
				t.Fatal("ParseS3Path returned nil error")
			}
		})
	}
}
