package main

import "testing"

func TestSameStorageProvider(t *testing.T) {
	tests := []struct {
		recorded, configured string
		want                 bool
	}{
		{"local", "local", true},
		{"", "local", true},
		{"rustfs", "s3", true},
		{"r2", "local", false},
	}
	for _, tt := range tests {
		if got := sameStorageProvider(tt.recorded, tt.configured); got != tt.want {
			t.Errorf("sameStorageProvider(%q, %q) = %v, want %v", tt.recorded, tt.configured, got, tt.want)
		}
	}
}
