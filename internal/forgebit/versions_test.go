package forgebit

import (
	"context"
	"net"
	"strings"
	"testing"
)

func TestUploadErrorDoesNotLeakPresignedURL(t *testing.T) {
	// Grab a free port, then close it so the upload's connection is refused
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	upload := VersionUpload{URL: "http://" + addr + "/ver_1.zip?X-Amz-Signature=supersecret"}
	err = UploadVersionArtifact(context.Background(), upload, strings.NewReader("hello"), 5)
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if strings.Contains(err.Error(), "supersecret") || strings.Contains(err.Error(), "X-Amz") {
		t.Fatalf("error leaks the presigned URL: %v", err)
	}
}
