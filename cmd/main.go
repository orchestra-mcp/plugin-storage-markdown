package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/orchestra-mcp/sdk-go/plugin"
	"github.com/orchestra-mcp/plugin-storage-markdown/internal"
)

func main() {
	workspace := flag.String("workspace", ".", "Root workspace directory")

	storage := internal.NewStoragePlugin(*workspace)

	p := plugin.New("storage.markdown").
		Version("0.1.0").
		Description("Markdown file storage with Protobuf metadata").
		Author("Orchestra").
		Binary("storage-markdown").
		ProvidesStorage("markdown").
		SetStorageHandler(storage).
		BuildWithTools()

	// Parse standard plugin flags (--orchestrator-addr, --listen-addr, --certs-dir).
	// This must be called after flag.String above so all flags are registered.
	p.ParseFlags()

	// Re-read workspace after flag.Parse has been called.
	storage = internal.NewStoragePlugin(*workspace)
	p.Server().SetStorageHandler(storage)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := p.Run(ctx); err != nil {
		log.Fatalf("storage.markdown: %v", err)
	}
}
