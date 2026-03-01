package storagemarkdown

import (
	"github.com/orchestra-mcp/plugin-storage-markdown/internal"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// NewStorage creates the markdown storage handler for the given workspace directory.
func NewStorage(workspace string) plugin.StorageHandler {
	return internal.NewStoragePlugin(workspace)
}
