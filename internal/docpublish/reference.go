package docpublish

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReferencePage is documentation generated from an authoritative machine
// contract rather than written by hand (W18.13). It is a projection: the
// contract is the source of truth, the page is a rendering of it.
type ReferencePage struct {
	Route    string `json:"route"`
	Title    string `json:"title"`
	Source   string `json:"source"`
	Markdown string `json:"markdown"`
}

// referenceContracts is the machine-readable contract registry. It is optional:
// a repository without it simply has no generated reference pages.
const referenceContracts = "docs/contracts/builtin.json"

type referenceContract struct {
	ID                   string   `json:"id"`
	Version              int      `json:"version"`
	Role                 string   `json:"role"`
	Description          string   `json:"description"`
	RequiredKnowledge    []string `json:"required_knowledge"`
	BlockingQuestions    []string `json:"blocking_questions"`
	EvidenceRequirements []string `json:"evidence_requirements"`
	Staleness            struct {
		Mode string `json:"mode"`
	} `json:"staleness"`
}

// buildReferencePages derives the API reference projection from the machine
// contracts and schemas. Nothing here is authored by a human: the pages exist
// so the contract surface is publishable and retrievable like any other
// document, without a second copy to maintain.
func buildReferencePages(root string) []ReferencePage {
	pages := []ReferencePage{}
	if contracts, ok := readReferenceContracts(root); ok && len(contracts) > 0 {
		pages = append(pages, ReferencePage{
			Route: "/reference/contracts", Title: "Documentation contracts",
			Source: referenceContracts, Markdown: renderContractsReference(contracts),
		})
	}
	if schemas, ok := readReferenceSchemas(root); ok && len(schemas) > 0 {
		pages = append(pages, ReferencePage{
			Route: "/reference/schemas", Title: "Machine schemas",
			Source: "schemas/", Markdown: renderSchemasReference(schemas),
		})
	}
	if len(pages) == 0 {
		// A renderer must still say something rather than emit nothing, so an
		// absent contract surface is reported instead of silently skipped.
		pages = append(pages, ReferencePage{
			Route: "/reference/index", Title: "API reference",
			Source:   "machine contracts",
			Markdown: "# API reference\n\nNo machine contracts or schemas were found in this repository.\n",
		})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Route < pages[j].Route })
	return pages
}

func readReferenceContracts(root string) ([]referenceContract, bool) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(referenceContracts)))
	if err != nil {
		return nil, false
	}
	var contracts []referenceContract
	if err := json.Unmarshal(data, &contracts); err != nil {
		return nil, false
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].ID < contracts[j].ID })
	return contracts, true
}

func renderContractsReference(contracts []referenceContract) string {
	var b strings.Builder
	b.WriteString("# Documentation contracts\n\n")
	b.WriteString("Generated from `" + referenceContracts + "`. Each contract states what a change owes to documentation; ")
	b.WriteString("the registry is the source of truth, this page is a projection of it.\n\n")
	b.WriteString(fmt.Sprintf("Total: %d contracts.\n\n", len(contracts)))
	b.WriteString("| Contract | Role | Required knowledge | Evidence |\n|---|---|---|---|\n")
	for _, c := range contracts {
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n",
			c.ID, fallback(c.Role, "—"),
			fallback(strings.Join(c.RequiredKnowledge, ", "), "—"),
			fallback(strings.Join(c.EvidenceRequirements, ", "), "—")))
	}
	b.WriteString("\n")
	for _, c := range contracts {
		b.WriteString("## `" + c.ID + "`\n\n")
		if c.Description != "" {
			b.WriteString(c.Description + "\n\n")
		}
		if len(c.RequiredKnowledge) > 0 {
			b.WriteString("Required knowledge:\n\n")
			for _, k := range c.RequiredKnowledge {
				b.WriteString("- " + k + "\n")
			}
			b.WriteString("\n")
		}
		if len(c.BlockingQuestions) > 0 {
			b.WriteString("Blocking questions:\n\n")
			for _, q := range c.BlockingQuestions {
				b.WriteString("- " + q + "\n")
			}
			b.WriteString("\n")
		}
		if c.Staleness.Mode != "" {
			b.WriteString("Freshness: `" + c.Staleness.Mode + "`\n\n")
		}
	}
	return b.String()
}

type referenceSchema struct {
	Name     string
	ID       string
	Required []string
	Version  string
}

func readReferenceSchemas(root string) ([]referenceSchema, bool) {
	files, err := filepath.Glob(filepath.Join(root, "schemas", "*.schema.json"))
	if err != nil || len(files) == 0 {
		return nil, false
	}
	out := make([]referenceSchema, 0, len(files))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var doc struct {
			ID       string   `json:"$id"`
			Schema   string   `json:"$schema"`
			Required []string `json:"required"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			continue
		}
		out = append(out, referenceSchema{
			Name: filepath.Base(file), ID: doc.ID,
			Required: doc.Required, Version: draftName(doc.Schema),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, true
}

func draftName(uri string) string {
	parts := strings.Split(strings.TrimSuffix(uri, "/schema"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func renderSchemasReference(schemas []referenceSchema) string {
	var b strings.Builder
	b.WriteString("# Machine schemas\n\n")
	b.WriteString("Generated from `schemas/*.schema.json`. These files are the machine contracts; ")
	b.WriteString("this page indexes them so the contract surface is retrievable without a hand-written duplicate.\n\n")
	b.WriteString(fmt.Sprintf("Total: %d schemas.\n\n", len(schemas)))
	b.WriteString("| Schema | Draft | Required fields |\n|---|---|---|\n")
	for _, s := range schemas {
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n",
			s.Name, fallback(s.Version, "—"), fallback(strings.Join(s.Required, ", "), "—")))
	}
	return b.String()
}

func fallback(value, when string) string {
	if strings.TrimSpace(value) == "" {
		return when
	}
	return value
}
