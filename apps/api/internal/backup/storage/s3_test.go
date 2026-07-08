package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"http://minio:9000/", "http://minio:9000"},
		{"http://minio:9000", "http://minio:9000"},
		{"https://s3.eu-central-1.amazonaws.com", "https://s3.eu-central-1.amazonaws.com"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := NormalizeEndpoint(tc.in); got != tc.want {
			t.Fatalf("NormalizeEndpoint(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEffectivePathStyle(t *testing.T) {
	if !EffectivePathStyle("http://minio:9000", false) {
		t.Fatal("expected path-style for MinIO endpoint")
	}
	if EffectivePathStyle("https://s3.eu-central-1.amazonaws.com", false) {
		t.Fatal("expected virtual-hosted style for AWS endpoint")
	}
	if !EffectivePathStyle("", true) {
		t.Fatal("expected configured path-style to win")
	}
}

func TestEndpointUsesHTTP(t *testing.T) {
	if !EndpointUsesHTTP("http://minio:9000") {
		t.Fatal("expected http endpoint")
	}
	if EndpointUsesHTTP("https://minio:9000") {
		t.Fatal("expected https endpoint to return false")
	}
	if EndpointUsesHTTP("") {
		t.Fatal("expected empty endpoint to return false")
	}
}

func TestS3ClientCustomEndpointUsesStableSignedHeaders(t *testing.T) {
	t.Parallel()

	objects := map[string][]byte{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if strings.Contains(auth, "accept-encoding") ||
			strings.Contains(auth, "amz-sdk-invocation-id") ||
			strings.Contains(auth, "amz-sdk-request") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `<Error><Code>SignatureDoesNotMatch</Code><Message>signed headers were mutated by proxy</Message></Error>`)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/bucket/")
		switch {
		case r.Method == http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read put body: %v", err)
			}
			objects[path] = body
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2":
			objectKey := ""
			objectSize := 0
			for key, data := range objects {
				objectKey = key
				objectSize = len(data)
				break
			}
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, fmt.Sprintf(`
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Contents>
    <Key>%s</Key>
    <LastModified>%s</LastModified>
    <Size>%d</Size>
  </Contents>
</ListBucketResult>`, objectKey, time.Date(2026, 7, 8, 6, 0, 0, 0, time.UTC).Format(time.RFC3339), objectSize))
		case r.Method == http.MethodGet:
			body, ok := objects[path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write(body)
		case r.Method == http.MethodDelete:
			delete(objects, path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewS3Client(context.Background(), S3Config{
		Endpoint:     server.URL,
		Region:       "us-east-1",
		Bucket:       "bucket",
		AccessKeyID:  "access-key",
		SecretKey:    "secret-key",
		UsePathStyle: true,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if err := client.Put(context.Background(), "backups/test.txt", strings.NewReader("hello"), "text/plain"); err != nil {
		t.Fatalf("put object: %v", err)
	}

	reader, err := client.Get(context.Background(), "backups/test.txt")
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected object body %q", string(data))
	}

	objectsList, err := client.List(context.Background(), "backups/")
	if err != nil {
		t.Fatalf("list objects: %v", err)
	}
	if len(objectsList) != 1 || objectsList[0].Key != "backups/test.txt" {
		t.Fatalf("unexpected objects: %+v", objectsList)
	}

	if err := client.Delete(context.Background(), "backups/test.txt"); err != nil {
		t.Fatalf("delete object: %v", err)
	}
}
