package cli

import (
	"testing"
)

func TestServeAddrDefault(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "default", env: map[string]string{}, want: ":9000"},
		{name: "addr", env: map[string]string{"REPO3_ADDR": ":9100"}, want: ":9100"},
		{name: "port", env: map[string]string{"REPO3_PORT": "9100"}, want: ":9100"},
		{name: "colon port", env: map[string]string{"REPO3_PORT": ":9200"}, want: ":9200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serveAddrDefault(tt.env); got != tt.want {
				t.Fatalf("serveAddrDefault = %q, want %q", got, tt.want)
			}
		})
	}
}
