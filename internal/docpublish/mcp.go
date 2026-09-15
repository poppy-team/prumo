// Documentation MCP surface (W18.11, W18.12).
//
// Two objects, deliberately separate:
//
//   - Server exposes the documentation graph for reading: resources and two
//     read-only tools. It has no mutating method at all, so no policy can be
//     bypassed by calling one.
//   - MutationServer is the only writer, and it denies everything unless an
//     explicit allowlist and reviewer approval say otherwise. Constructing it
//     is an opt-in decision; the read-only surface never reaches it.
//
// The separation is the security property: a caller that was granted read
// access cannot mutate documentation, because there is nothing to call.
package docpublish

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ResourceURI is the scheme every documentation resource is addressed with.
const ResourceURI = "prumo://docs/"

// Resource is one read-only documentation resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	MIMEType    string `json:"mimeType"`
	Description string `json:"description,omitempty"`
	Audience    string `json:"audience,omitempty"`
	Historical  bool   `json:"historical,omitempty"`
}

// ResourceContent is the payload a read returns.
type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType"`
	Text     string `json:"text"`
}

// ToolSpec is a tool the documentation surface advertises.
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	// Kind is always "read-only" on the documentation surface; the field exists
	// so a caller can assert that rather than assume it.
	Kind string `json:"kind"`
}

// Server serves the documentation graph over MCP, read-only.
type Server struct {
	graph Graph
}

// NewServer loads the graph the resources are projected from.
func NewServer(root string, opts Options) (*Server, error) {
	graph, err := LoadWithOptions(root, opts)
	if err != nil {
		return nil, err
	}
	return &Server{graph: graph}, nil
}

// Graph returns the graph the server projects, so a caller can inspect it
// without a second load.
func (s *Server) Graph() Graph { return s.graph }

// List returns every readable resource. Agent-only pages are included: they are
// documentation too, and their audience is stated rather than hidden.
func (s *Server) List() []Resource {
	resources := make([]Resource, 0, len(s.graph.Pages))
	for _, page := range s.graph.Pages {
		resources = append(resources, Resource{
			URI: ResourceURI + strings.TrimPrefix(page.Route, "/"), Name: page.Title,
			Title: page.Title, MIMEType: "text/markdown",
			Description: "source: " + page.Source,
			Audience:    string(page.Audience), Historical: page.Historical,
		})
	}
	for _, page := range s.graph.Reference {
		resources = append(resources, Resource{
			URI: ResourceURI + strings.TrimPrefix(page.Route, "/"), Name: page.Title,
			Title: page.Title, MIMEType: "text/markdown",
			Description: "generated from " + page.Source,
		})
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].URI < resources[j].URI })
	return resources
}

// Read returns the Markdown of one resource.
func (s *Server) Read(uri string) (ResourceContent, error) {
	route := strings.TrimPrefix(strings.TrimSpace(uri), ResourceURI)
	route = "/" + strings.Trim(route, "/")
	if route == "/" {
		route = s.indexRoute()
	}
	for _, page := range s.graph.Pages {
		if page.Route == route {
			return ResourceContent{URI: uri, MIMEType: "text/markdown", Text: page.Markdown}, nil
		}
	}
	for _, page := range s.graph.Reference {
		if page.Route == route {
			return ResourceContent{URI: uri, MIMEType: "text/markdown", Text: page.Markdown}, nil
		}
	}
	return ResourceContent{}, fmt.Errorf("no documentation resource at %s", uri)
}

func (s *Server) indexRoute() string {
	for _, page := range s.graph.Pages {
		if page.IsIndex && page.Audience == AudienceHuman {
			return page.Route
		}
	}
	return "/"
}

// Tools lists the read-only tools. Their Kind is read-only by construction, not
// by configuration.
func (s *Server) Tools() []ToolSpec {
	return []ToolSpec{
		{
			Name: "docs.list", Description: "List documentation resources.", Kind: "read-only",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name: "docs.read", Description: "Read one documentation resource as Markdown.", Kind: "read-only",
			InputSchema: map[string]any{
				"type": "object", "required": []string{"uri"},
				"properties": map[string]any{"uri": map[string]any{"type": "string"}},
			},
		},
	}
}

// Dispatch answers a documentation MCP method. Only read-only methods are
// recognised; anything else is refused, which is what keeps the surface honest
// when a caller guesses at a mutation.
func (s *Server) Dispatch(method string, params map[string]any) (any, error) {
	switch method {
	case "resources/list":
		return map[string]any{"resources": s.List()}, nil
	case "resources/read":
		uri, _ := params["uri"].(string)
		content, err := s.Read(uri)
		if err != nil {
			return nil, err
		}
		return map[string]any{"contents": []ResourceContent{content}}, nil
	case "tools/list":
		return map[string]any{"tools": s.Tools()}, nil
	case "tools/call":
		name, _ := params["name"].(string)
		args, _ := params["arguments"].(map[string]any)
		return s.call(name, args)
	default:
		return nil, fmt.Errorf("documentation surface is read-only: %q is not a supported method", method)
	}
}

func (s *Server) call(name string, args map[string]any) (any, error) {
	switch name {
	case "docs.list":
		resources := s.List()
		out := make([]map[string]any, 0, len(resources))
		for _, r := range resources {
			out = append(out, map[string]any{"uri": r.URI, "name": r.Name, "historical": r.Historical})
		}
		return map[string]any{"content": out}, nil
	case "docs.read":
		uri, _ := args["uri"].(string)
		content, err := s.Read(uri)
		if err != nil {
			return nil, err
		}
		return map[string]any{"content": content.Text}, nil
	default:
		return nil, fmt.Errorf("unknown documentation tool %q", name)
	}
}

// mutationVerbs are the operation names the mutation surface understands. It is
// an allowlist: an operation outside it is denied even if a policy lists it.
var mutationVerbs = []string{"unit.write", "unit.deprecate", "translation.record", "media.record", "release.publish"}

// MutationVerbs lists every operation the mutation surface can perform.
func MutationVerbs() []string { return append([]string{}, mutationVerbs...) }

// MutationPolicy gates the mutation surface. The zero value denies everything:
// a caller must opt in twice (operation allowlist, and approval when demanded).
type MutationPolicy struct {
	Allowed         []string
	RequireApproval bool
}

// MutationRequest is one attempted documentation mutation.
type MutationRequest struct {
	Operation string         `json:"operation"`
	Arguments map[string]any `json:"arguments,omitempty"`
	// Approval names the reviewer who authorised the mutation. It is required
	// when the policy demands approval, and an empty approval is never
	// interpreted as consent.
	Approval string `json:"approval,omitempty"`
}

// MutationServer is the only writer on the documentation surface.
type MutationServer struct {
	Policy MutationPolicy
	Apply  func(operation string, args map[string]any) (any, error)
}

// Execute gates then applies a mutation. Every refusal states its reason.
func (m MutationServer) Execute(req MutationRequest) (any, error) {
	if !isMutationVerb(req.Operation) {
		return nil, fmt.Errorf("unknown documentation mutation %q", req.Operation)
	}
	if !containsValue(m.Policy.Allowed, req.Operation) {
		return nil, fmt.Errorf("mutation %q is not enabled by policy", req.Operation)
	}
	if m.Policy.RequireApproval && strings.TrimSpace(req.Approval) == "" {
		return nil, fmt.Errorf("mutation %q requires reviewer approval", req.Operation)
	}
	if m.Apply == nil {
		return nil, fmt.Errorf("mutation %q has no handler configured", req.Operation)
	}
	return m.Apply(req.Operation, req.Arguments)
}

// Describe reports the gate state without performing anything, so an operator
// can see what is enabled before enabling it.
func (m MutationServer) Describe() map[string]any {
	enabled := []string{}
	for _, verb := range mutationVerbs {
		if containsValue(m.Policy.Allowed, verb) {
			enabled = append(enabled, verb)
		}
	}
	sort.Strings(enabled)
	return map[string]any{
		"enabled":          enabled,
		"require_approval": m.Policy.RequireApproval,
		"all_verbs":        MutationVerbs(),
	}
}

func isMutationVerb(operation string) bool {
	return containsValue(mutationVerbs, operation)
}

func containsValue(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// MarshalDispatch is a convenience for embedding the read-only surface in a
// JSON-RPC loop: it renders the dispatch result as a JSON object.
func (s *Server) MarshalDispatch(method string, params map[string]any) (json.RawMessage, error) {
	result, err := s.Dispatch(method, params)
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}
