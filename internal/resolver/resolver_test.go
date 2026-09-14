package resolver

import "testing"

func TestResolveBrasa(t *testing.T) {
	catalog, err := LoadCatalog("../..")
	if err != nil {
		t.Skipf("catalog missing: %v", err)
	}
	profile, err := LoadProfile("../../examples/brasa/project-profile.json")
	if err != nil {
		t.Skipf("profile missing: %v", err)
	}
	resolution := catalog.Resolve(profile)
	if len(resolution.Skills) == 0 {
		t.Fatalf("expected skills to be resolved")
	}
	seen := map[string]bool{}
	for _, skill := range resolution.Skills {
		seen[skill] = true
	}
	for _, expected := range []string{"clean-code"} {
		if !seen[expected] {
			t.Fatalf("expected skill %s in %v", expected, resolution.Skills)
		}
	}
}

func TestResolveSystemsSkills(t *testing.T) {
	catalog, err := LoadCatalog("../..")
	if err != nil {
		t.Fatalf("catalog missing: %v", err)
	}

	tests := []struct {
		name          string
		stack         []string
		expectedSkill string
	}{
		{"C++ stack", []string{"cpp", "cmake"}, "lang-cpp"},
		{"C stack", []string{"c", "gcc"}, "lang-c"},
		{"Zig stack", []string{"zig"}, "lang-zig"},
		{"D stack", []string{"dlang"}, "lang-d"},
		{"Odin stack", []string{"odin"}, "lang-odin"},
		{"C3 stack", []string{"c3"}, "lang-c3"},
		{"Assembly stack", []string{"asm", "x86_64"}, "lang-asm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Profile{
				Raw: map[string]any{
					"stack": tt.stack,
				},
			}
			resolution := catalog.Resolve(profile)
			found := false
			for _, s := range resolution.Skills {
				if s == tt.expectedSkill {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected skill %s to resolve for stack %v, got %v", tt.expectedSkill, tt.stack, resolution.Skills)
			}
		})
	}
}

func TestResolveManagedSkills(t *testing.T) {
	catalog, err := LoadCatalog("../..")
	if err != nil {
		t.Fatalf("catalog missing: %v", err)
	}

	tests := []struct {
		name          string
		stack         []string
		expectedSkill string
	}{
		{"C# stack", []string{"csharp", "dotnet"}, "lang-csharp"},
		{"Java stack", []string{"java", "gradle"}, "lang-java"},
		{"Kotlin stack", []string{"kotlin"}, "lang-kotlin"},
		{"Dart stack", []string{"dart", "flutter"}, "lang-dart"},
		{"Elixir stack", []string{"elixir", "mix"}, "lang-elixir"},
		{"Rust stack", []string{"rust", "cargo"}, "lang-rust"},
		{"Go stack", []string{"go", "golang"}, "lang-go"},
		{"Swift stack", []string{"swift", "spm"}, "lang-swift"},
		{"TypeScript stack", []string{"typescript", "tsc"}, "lang-typescript"},
		{"JavaScript stack", []string{"javascript", "node"}, "lang-javascript"},
		{"Python stack", []string{"python", "pytest"}, "lang-python"},
		{"Ruby stack", []string{"ruby", "rails"}, "lang-ruby"},
		{"Lua stack", []string{"lua"}, "lang-lua"},
		{"PHP stack", []string{"php", "composer"}, "lang-php"},
		{"Bash stack", []string{"bash"}, "lang-bash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Profile{
				Raw: map[string]any{
					"stack": tt.stack,
				},
			}
			resolution := catalog.Resolve(profile)
			found := false
			for _, s := range resolution.Skills {
				if s == tt.expectedSkill {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected skill %s to resolve for stack %v, got %v", tt.expectedSkill, tt.stack, resolution.Skills)
			}
		})
	}
}

func TestResolveDomainSkills(t *testing.T) {
	catalog, err := LoadCatalog("../..")
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	tests := []struct {
		name          string
		stack         []string
		expectedSkill string
	}{
		{"HTML stack", []string{"html", "web"}, "lang-html"},
		{"CSS stack", []string{"css", "styles"}, "lang-css"},
		{"SQL stack", []string{"sql", "postgresql"}, "lang-sql"},
		{"GraphQL stack", []string{"graphql"}, "lang-graphql"},
		{"GLSL stack", []string{"glsl", "shaders"}, "lang-glsl"},
		{"HLSL stack", []string{"hlsl"}, "lang-hlsl"},
		{"WGSL stack", []string{"wgsl", "webgpu"}, "lang-wgsl"},
		{"Dockerfile stack", []string{"dockerfile", "containers"}, "lang-dockerfile"},
		{"Terraform stack", []string{"terraform", "iac"}, "lang-terraform"},
		{"YAML stack", []string{"yaml", "config"}, "lang-yaml"},
		{"JSON stack", []string{"json"}, "lang-json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := Profile{
				Raw: map[string]any{
					"stack": tt.stack,
				},
			}
			resolution := catalog.Resolve(profile)
			found := false
			for _, s := range resolution.Skills {
				if s == tt.expectedSkill {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected skill %s to resolve for stack %v, got %v", tt.expectedSkill, tt.stack, resolution.Skills)
			}
		})
	}
}

func TestResolveCoreSkills(t *testing.T) {
	catalog, err := LoadCatalog("../..")
	if err != nil {
		t.Fatalf("failed to load catalog: %v", err)
	}

	emptyProfile := Profile{Raw: map[string]any{}}
	resolution := catalog.Resolve(emptyProfile)

	expectedCore := []string{
		"clean-code",
		"cognitive-clarity",
		"testing-quality",
	}

	resolvedSet := map[string]bool{}
	for _, s := range resolution.Skills {
		resolvedSet[s] = true
	}

	for _, exp := range expectedCore {
		if !resolvedSet[exp] {
			t.Fatalf("expected core skill %q to be resolved for empty profile, resolved skills: %v", exp, resolution.Skills)
		}
	}
}
