package mcp

import "github.com/raillen/prumo/internal/toolgateway"

func serverDescriptorForTest() toolgateway.MCPServerDescriptor {
	return toolgateway.MCPServerDescriptor{ID: "test-srv", Trust: "untrusted"}
}
