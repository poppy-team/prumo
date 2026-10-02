package prumo

import (
	"embed"
	"io/fs"
)

//go:embed schemas/*.json
var schemaAssets embed.FS

//go:embed docs/contracts/builtin.json
var docContractsAsset embed.FS

//go:embed docs/profiles/builtin.json
var docProfilesAsset embed.FS

//go:embed src/prumo/resources/adapters/*.md
var adapterAssets embed.FS

//go:embed src/prumo/resources/catalog/*.json
var catalogAssets embed.FS

//go:embed src/prumo/resources/workforce
var workforceAssets embed.FS

func EmbeddedSchemas() fs.FS  { return subFS(schemaAssets, "schemas") }
func EmbeddedAdapters() fs.FS { return subFS(adapterAssets, "src/prumo/resources/adapters") }
func EmbeddedCatalog() fs.FS  { return subFS(catalogAssets, "src/prumo/resources/catalog") }
func EmbeddedWorkforce() fs.FS {
	return subFS(workforceAssets, "src/prumo/resources/workforce")
}

// EmbeddedDocContracts is the framework's documentation contract registry. It
// is embedded so `prumo init` can seed a project with a working documentation
// control plane instead of leaving `docs audit`, `docs readiness` and
// `docs authority` unable to resolve a registry.
func EmbeddedDocContracts() fs.FS { return subFS(docContractsAsset, "docs/contracts") }

// EmbeddedDocProfiles is the framework's documentation profile registry, the
// companion to EmbeddedDocContracts.
func EmbeddedDocProfiles() fs.FS { return subFS(docProfilesAsset, "docs/profiles") }

func subFS(source embed.FS, path string) fs.FS {
	sub, err := fs.Sub(source, path)
	if err != nil {
		return source
	}
	return sub
}
