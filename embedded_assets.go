package prumo

import (
	"embed"
	"io/fs"
)

// allAssets is the tree the resource readers resolve paths against, with the
// same layout a checkout has. DefaultFS is this rather than one of the
// sub-trees below, because OpenSchema looks up "schemas/<name>" and
// OpenResourceFile looks up "src/prumo/resources/<parts>" — a filesystem
// rooted at the catalog directory satisfies the first and cannot satisfy the
// second, and neither satisfies both.
var allAssets fs.FS

//go:embed schemas src/prumo/resources
var combinedAssets embed.FS

//go:embed schemas/*.json
var schemaAssets embed.FS

//go:embed src/prumo/resources/adapters/*.md
var adapterAssets embed.FS

//go:embed src/prumo/resources/catalog/*.json
var catalogAssets embed.FS

//go:embed src/prumo/resources/workforce
var workforceAssets embed.FS

// EmbeddedAll returns the whole embedded tree, laid out as a checkout is.
func EmbeddedAll() fs.FS { return combinedAssets }

func EmbeddedSchemas() fs.FS  { return subFS(schemaAssets, "schemas") }
func EmbeddedAdapters() fs.FS { return subFS(adapterAssets, "src/prumo/resources/adapters") }
func EmbeddedCatalog() fs.FS  { return subFS(catalogAssets, "src/prumo/resources/catalog") }
func EmbeddedWorkforce() fs.FS {
	return subFS(workforceAssets, "src/prumo/resources/workforce")
}

func subFS(source embed.FS, path string) fs.FS {
	sub, err := fs.Sub(source, path)
	if err != nil {
		return source
	}
	return sub
}
