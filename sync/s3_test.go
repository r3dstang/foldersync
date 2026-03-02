package sync

import (
	"testing"
)

func TestS3Destination_fullKey(t *testing.T) {
	tests := []struct {
		prefix string
		rel    string
		want   string
	}{
		{"", "foo.txt", "foo.txt"},
		{"backups", "foo.txt", "backups/foo.txt"},
		{"backups/", "foo.txt", "backups/foo.txt"},
		{"backups", "a/b/c.txt", "backups/a/b/c.txt"},
		{"", "/foo.txt", "foo.txt"}, // leading slash stripped
	}

	for _, tt := range tests {
		d := &S3Destination{prefix: tt.prefix}
		if got := d.fullKey(tt.rel); got != tt.want {
			t.Errorf("fullKey(prefix=%q, rel=%q) = %q, want %q", tt.prefix, tt.rel, got, tt.want)
		}
	}
}

func TestS3Destination_relKey(t *testing.T) {
	tests := []struct {
		prefix string
		full   string
		want   string
	}{
		{"", "foo.txt", "foo.txt"},
		{"backups", "backups/foo.txt", "foo.txt"},
		{"backups/", "backups/foo.txt", "foo.txt"},
		{"backups", "backups/a/b/c.txt", "a/b/c.txt"},
	}

	for _, tt := range tests {
		d := &S3Destination{prefix: tt.prefix}
		if got := d.relKey(tt.full); got != tt.want {
			t.Errorf("relKey(prefix=%q, full=%q) = %q, want %q", tt.prefix, tt.full, got, tt.want)
		}
	}
}

// TestS3Destination_keyRoundTrip verifies that relKey(fullKey(rel)) == rel.
func TestS3Destination_keyRoundTrip(t *testing.T) {
	cases := []struct {
		prefix string
		keys   []string
	}{
		{"", []string{"foo.txt", "a/b/c.txt"}},
		{"backups", []string{"foo.txt", "a/b/c.txt"}},
		{"backups/", []string{"foo.txt", "a/b/c.txt"}},
	}

	for _, tc := range cases {
		d := &S3Destination{prefix: tc.prefix}
		for _, key := range tc.keys {
			if got := d.relKey(d.fullKey(key)); got != key {
				t.Errorf("prefix=%q: relKey(fullKey(%q)) = %q, want %q", tc.prefix, key, got, key)
			}
		}
	}
}
