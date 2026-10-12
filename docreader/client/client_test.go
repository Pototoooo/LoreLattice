package client

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/Pototoooo/lorelattice/docreader/proto"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
	log.Println("INFO: Initializing DocReader client tests")
}

func TestReadURL(t *testing.T) {
	addr := os.Getenv("DOCREADER_TEST_ADDR")
	if addr == "" {
		t.Skip("set DOCREADER_TEST_ADDR for live parser integration tests")
	}
	client, err := NewClient(addr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	client.SetDebug(true)

	startTime := time.Now()
	resp, err := client.Read(
		context.Background(),
		&proto.ReadRequest{
			Url:   "https://example.com",
			Title: "test",
		},
	)
	log.Printf("INFO: Read(URL) completed in %v", time.Since(startTime))

	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("Read returned error: %s", resp.Error)
	}
	if resp.MarkdownContent == "" {
		t.Error("Expected non-empty markdown content")
	}
	log.Printf("INFO: content_len=%d, images=%d", len(resp.MarkdownContent), len(resp.ImageRefs))
}

func TestReadFile(t *testing.T) {
	addr := os.Getenv("DOCREADER_TEST_ADDR")
	if addr == "" {
		t.Skip("set DOCREADER_TEST_ADDR for live parser integration tests")
	}
	client, err := NewClient(addr)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()
	client.SetDebug(true)

	fileContent, err := os.ReadFile("../testdata/test.md")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	startTime := time.Now()
	resp, err := client.Read(
		context.Background(),
		&proto.ReadRequest{
			FileContent: fileContent,
			FileName:    "test.md",
			FileType:    "md",
		},
	)
	log.Printf("INFO: Read(file) completed in %v", time.Since(startTime))

	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if resp.Error != "" {
		t.Fatalf("Read returned error: %s", resp.Error)
	}
	if resp.MarkdownContent == "" {
		t.Error("Expected non-empty markdown content")
	}

	imageRefs := GetImageRefsFromResponse(resp)
	log.Printf("INFO: content_len=%d, images=%d", len(resp.MarkdownContent), len(imageRefs))
}

// Unit coverage of the real gRPC client without an external parser or website.
// Live parser coverage remains in the opt-in tests above and Python parser suite.
type fixtureDocReader struct {
	proto.UnimplementedDocReaderServer
}

func (fixtureDocReader) Read(_ context.Context, req *proto.ReadRequest) (*proto.ReadResponse, error) {
	if req.Url != "" {
		return &proto.ReadResponse{MarkdownContent: "# fixture URL"}, nil
	}
	return &proto.ReadResponse{MarkdownContent: string(req.FileContent)}, nil
}
func TestReadRPCFixture(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	proto.RegisterDocReaderServer(server, fixtureDocReader{})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)
	client, err := NewClientWithAuth(listener.Addr().String(), &AuthConfig{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	for _, req := range []*proto.ReadRequest{{Url: "https://fixture.invalid"}, {FileContent: []byte("# local fixture"), FileName: "fixture.md", FileType: "md"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := client.Read(ctx, req)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		want := "# fixture URL"
		if req.Url == "" {
			want = string(req.FileContent)
		}
		if resp.MarkdownContent != want {
			t.Fatalf("content=%q want=%q", resp.MarkdownContent, want)
		}
	}
}
