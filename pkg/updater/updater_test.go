package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		remote   string
		local    string
		expected bool
	}{
		{"v1.0.1", "v1.0.0", true},
		{"v1.1.0", "v1.0.9", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.0.0", "v1.0.0", false},
		{"1.0.0", "v1.0.0", false},
		{"v1.0.0-hotfix1", "v1.0.0", true},
	}

	for _, tt := range tests {
		got := isNewerVersion(tt.remote, tt.local)
		if got != tt.expected {
			t.Errorf("isNewerVersion(%q, %q) = %v; expected %v", tt.remote, tt.local, got, tt.expected)
		}
	}
}

func TestExtractExecutableFromArchive(t *testing.T) {
	// Create sample .tar.gz in memory containing pngtuber-lite
	dummyContent := []byte("ELF_SAMPLE_BINARY")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name: "pngtuber-lite",
		Size: int64(len(dummyContent)),
		Mode: 0755,
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tw.Write(dummyContent); err != nil {
		t.Fatalf("failed to write tar data: %v", err)
	}
	_ = tw.Close()
	_ = gw.Close()

	extracted, err := extractExecutableFromArchive(buf.Bytes(), "pngtuber-lite-linux-amd64.tar.gz")
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if string(extracted) != string(dummyContent) {
		t.Errorf("extracted content mismatch: got %q, expected %q", string(extracted), string(dummyContent))
	}
}
