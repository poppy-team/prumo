package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/raillen/prumo/internal/protocol"
)

// CommandInfo holds documentation metadata for a CLI command.
type CommandInfo struct {
	Name        string
	Category    string
	Summary     string
	Usage       string
	Description string
	Flags       []string
	Subcommands []string
	Examples    []string
}

var commandRegistry = map[string]CommandInfo{
	"init": {
		Name:     "init",
		Category: "Project Lifecycle",
		Summary:  "Initialize a new Prumo project workspace",
		Usage:    "prumo init [path] --profile <profile.json> [--non-interactive]",
		Description: "Scaffolds a new Prumo project at the given path (default: current directory).\n" +
			"Resolves the workforce from the specified project profile and generates prumo.json,\n" +
			".ai/ directory, docs/PRUMO.md, PROJECT_STATE.md, and initial derived state.",
		Flags: []string{
			"--profile <path>       Path to project-profile.json declaring preferred models and stack (required)",
			"--non-interactive      Execute without interactive prompts (fails if profile missing)",
			"--home <path>          Custom Prumo home directory",
			"--json                 Output structured JSON response",
		},
		Examples: []string{
			"prumo init ./my-project --profile examples/brasa/project-profile.json",
			"prumo init . --profile ./profile.json --non-interactive",
		},
	},
	"status": {
		Name:     "status",
		Category: "Project Lifecycle",
		Summary:  "Display current Prumo workspace root path",
		Usage:    "prumo status [--path <dir>]",
		Description: "Finds and prints the canonical root directory of the Prumo project.\n" +
			"Returns exit code 0 if found, exit code 6 (PRUMO_PROJECT_NOT_FOUND) if outside a project.",
		Flags: []string{
			"--path <dir>           Target directory to inspect (default: current directory)",
			"--json                 Output structured JSON envelope with root path",
		},
		Examples: []string{
			"prumo status",
			"prumo --json status --path ./subfolder",
		},
	},
	"validate": {
		Name:     "validate",
		Category: "Project Lifecycle",
		Summary:  "Validate project structure and schema conformance",
		Usage:    "prumo validate [path]",
		Description: "Performs strict schema validation against canonical JSON schemas.\n" +
			"Verifies prumo.json, goals, policies, and directory structure.",
		Flags: []string{
			"--json                 Output structured validation results",
		},
		Examples: []string{
			"prumo validate",
			"prumo validate ./my-project",
		},
	},
	"doctor": {
		Name:     "doctor",
		Category: "Project Lifecycle",
		Summary:  "Run comprehensive diagnostic health checks",
		Usage:    "prumo doctor [path | gui]",
		Description: "Performs deep health checks across all framework subsystems:\n" +
			"- Versioning and lockfile consistency\n" +
			"- Dependencies and DAG cycles\n" +
			"- Workforce resolution and catalog integrity\n" +
			"- Repository governance policies and gates\n" +
			"- Evidence models and test coverage\n" +
			"- Graphical session and accessibility bus when invoked with 'gui' (Chapter 26)",
		Flags: []string{
			"--json                 Output diagnostics as structured JSON envelope",
		},
		Examples: []string{
			"prumo doctor",
			"prumo doctor gui",
			"prumo --json doctor ./my-project",
		},
	},
	"framework-check": {
		Name:     "framework-check",
		Category: "Project Lifecycle",
		Summary:  "Validate framework internal integrity and catalog schemas",
		Usage:    "prumo framework-check",
		Description: "Validates the framework's own catalog, workforce skills, recipes,\n" +
			"JSON schemas, and embedded resources.",
		Flags: []string{
			"--json                 Output structured JSON response",
		},
		Examples: []string{
			"prumo framework-check",
			"prumo --json framework-check",
		},
	},
	"version": {
		Name:        "version",
		Category:    "General",
		Summary:     "Display Prumo CLI and protocol version",
		Usage:       "prumo version",
		Description: "Prints the semantic version of the Prumo CLI and the protocol contract version.",
		Flags: []string{
			"--json                 Output version information as JSON envelope",
		},
		Examples: []string{
			"prumo version",
			"prumo --json version",
		},
	},
	"goal": {
		Name:     "goal",
		Category: "Goals & Execution",
		Summary:  "Manage goal lifecycle, states, and amendments",
		Usage:    "prumo goal <subcommand> [args] [--path <dir>]",
		Description: "Manages verifiable Goals through the protocol state machine:\n" +
			"DRAFT -> PLANNED -> LOCKED -> EXECUTING -> VERIFYING -> REVIEWING -> DONE.",
		Subcommands: []string{
			"new <id> <title> --phase <phase> --objective <text>   Create a new goal",
			"list                                                  List all project goals and states",
			"state <id> <state>                                    Transition goal state (requires evidence for DONE)",
			"amend <id> --file <amendment.json>                    Apply formal amendment to locked goal",
		},
		Flags: []string{
			"--path <dir>           Project directory (default: .)",
			"--phase <phase>        Phase identifier (e.g., P00, P01)",
			"--objective <text>     Objective description",
			"--file <path>          Path to amendment file",
			"--json                 Output structured JSON envelope",
		},
		Examples: []string{
			"prumo goal new P00-G01 \"Foundation\" --phase P00 --objective \"Set up base CI and tests\" --path ./my-project",
			"prumo goal list --path ./my-project",
			"prumo goal state P00-G01 LOCKED --path ./my-project",
		},
	},
	"plan": {
		Name:     "plan",
		Category: "Goals & Execution",
		Summary:  "Inspect and update the Living Plan",
		Usage:    "prumo plan <subcommand> [args] [--path <dir>]",
		Description: "Maintains the continuous Living Plan across development goals,\n" +
			"supporting delta feedback, question resolution, and decision previews.",
		Subcommands: []string{
			"init                                                  Initialize living plan in project",
			"show                                                  Display current living plan summary",
			"update --delta <delta.json>                           Apply delta feedback to plan",
		},
		Flags: []string{
			"--path <dir>           Project directory (default: .)",
			"--delta <path>         Path to delta feedback JSON",
			"--json                 Output structured JSON envelope",
		},
		Examples: []string{
			"prumo plan show --path ./my-project",
			"prumo plan update --delta feedback.json --path ./my-project",
		},
	},
	"context": {
		Name:     "context",
		Category: "Goals & Execution",
		Summary:  "Plan, compile and explain context compilations (LPC)",
		Usage:    "prumo context plan <description> | compile --goal <text> | explain <CTX-id>",
		Description: "Applies Lean Progressive Context (LPC) to determine the minimal sufficient\n" +
			"context capsule, token budget, and retrieval strategy for an AI coding goal.\n" +
			"compile emits a replayable ContextManifest at an explicit L0-L4 disclosure\n" +
			"level; explain answers what a compilation included, why, what it left out and\n" +
			"whether it was sufficient.",
		Flags: []string{
			"--path <dir>           Project directory (default: .)",
			"--goal <text>          Goal the compilation must serve (compile)",
			"--budget <n>           Token budget (compile, default 8000)",
			"--level <L0..L4>       Progressive disclosure level (compile)",
			"--id <id>              Compilation id (compile; default derived from --goal)",
			"--required <refs>      Comma-separated refs that must be present (compile, explain)",
			"--json                 Output the JSON envelope",
		},
		Examples: []string{
			"prumo context plan \"debug database connection leak\" --path ./my-project --json",
			"prumo context compile --goal \"add a transition template\" --budget 3000 --level L2",
			"prumo context explain CTX-add-a-transition-template --budget 3000",
		},
	},
	"knowledge": {
		Name:     "knowledge",
		Category: "Goals & Execution",
		Summary:  "Derived knowledge manifest and stable-identity migration report",
		Usage:    "prumo knowledge manifest | ids",
		Description: "Builds the derived KnowledgeManifest from every persisted run store.\n" +
			"The manifest is a projection written under .prumo/runtime/ and is never a\n" +
			"canonical artifact; ids reports records that still carry a pre-migration\n" +
			"identifier (they keep resolving to their stable id).",
		Flags: []string{
			"--path <dir>           Project directory (default: .)",
			"--out <file>           Write the derived manifest to a path",
			"--json                 Output the JSON envelope",
		},
		Examples: []string{
			"prumo knowledge manifest",
			"prumo knowledge manifest --json",
			"prumo knowledge ids",
		},
	},
	"resolve": {
		Name:     "resolve",
		Category: "Workforce & Compilation",
		Summary:  "Resolve workforce skills, agents, and recipes for a profile",
		Usage:    "prumo resolve <profile.json>",
		Description: "Deterministically matches stack tokens, features, and risk profiles from\n" +
			"project-profile.json against the canonical workforce registry (151 skills, agents, recipes).",
		Flags: []string{
			"--json                 Output resolved workforce as JSON envelope",
		},
		Examples: []string{
			"prumo resolve examples/brasa/project-profile.json",
			"prumo --json resolve ./profile.json",
		},
	},
	"explain": {
		Name:     "explain",
		Category: "Workforce & Compilation",
		Summary:  "Inspect detailed contracts of workforce entities",
		Usage:    "prumo explain <type> <id>",
		Description: "Displays inputs, outputs, capabilities, required evidence, and instructions for:\n" +
			"- workforce <profile.json>: explain resolution rationale\n" +
			"- agent <agent-id>: explain agent persona and rules\n" +
			"- skill <skill-id>: explain implementation contract and checks\n" +
			"- recipe <recipe-id>: explain multi-step coordination flow",
		Flags: []string{
			"--json                 Output explanation as JSON envelope",
		},
		Examples: []string{
			"prumo explain skill lang-cpp",
			"prumo explain skill lang-rust --json",
			"prumo explain agent architect",
			"prumo explain recipe web-feature",
		},
	},
	"compile": {
		Name:     "compile",
		Category: "Workforce & Compilation",
		Summary:  "Compile and emit target harness adapters and skills",
		Usage:    "prumo compile --target <target> [--path <dir>]",
		Description: "Compiles canonical project state, workforce skills, instructions, and\n" +
			"rules into target-specific AI coding harness configurations.",
		Flags: []string{
			"--target <name>        Harness target: antigravity, opencode, codex, claudecode, claude, gemini, generic, traycer, chatgpt, kimi",
			"--path <dir>           Project directory (default: .)",
			"--json                 Output compilation summary as JSON",
		},
		Examples: []string{
			"prumo compile --target antigravity --path ./my-project",
			"prumo compile --target opencode --path ./my-project",
			"prumo compile --target codex --path ./my-project",
		},
	},
	"tool": {
		Name:     "tool",
		Category: "Tooling & Verification",
		Summary:  "Execute embedded developer and safety verification tools",
		Usage:    "prumo tool <tool-name> [args]",
		Description: "Executes pure Go native tools integrated with the Prumo Tool Gateway:\n" +
			"- check-escape-hatches: Scan source code for unregistered escape hatches\n" +
			"- verify-language-contract: Validate language safety profiles\n" +
			"- scan-sanitizers: Audit compiler sanitizer matrix\n" +
			"- scan-secrets: Scan for exposed API keys and credentials\n" +
			"- check-permissions: Audit file and directory permissions\n" +
			"- analyze-complexity: Static cognitive and cyclomatic complexity analysis\n" +
			"- check-bare-errors: Detect swallowed/unhandled error returns\n" +
			"- check-subprocesses: Audit external process spawning for command injection",
		Flags: []string{
			"--json                 Output tool results as JSON envelope",
		},
		Examples: []string{
			"prumo tool check-escape-hatches .",
			"prumo tool check-escape-hatches ./src/my-service",
			"prumo tool scan-secrets .",
			"prumo tool analyze-complexity ./internal",
		},
	},
	"adopt": {
		Name:     "adopt",
		Category: "Adoption & Migration",
		Summary:  "Brownfield project scanner and adoption engine",
		Usage:    "prumo adopt <subcommand> [path]",
		Description: "Scans non-Prumo codebases, discovers technical facts, classifies tech stacks,\n" +
			"and generates a non-destructive migration ledger and candidate Prumo configuration.",
		Subcommands: []string{
			"scan [path]        Scan directory tree and index project artifacts",
			"facts [path]       Extract observed facts (languages, frameworks, build systems)",
			"classify [path]    Classify project architecture and capability profile",
			"scaffold [path]    Generate candidate prumo.json without overwriting user files",
		},
		Flags: []string{
			"--json             Output scan results as JSON envelope",
		},
		Examples: []string{
			"prumo adopt scan ./legacy-app",
			"prumo adopt facts ./legacy-app --json",
			"prumo adopt classify ./legacy-app",
		},
	},
	"report": {
		Name:     "report",
		Category: "Governance & Traceability",
		Summary:  "Ingest execution reports and query project intelligence",
		Usage:    "prumo report <subcommand> [args] [--path <dir>]",
		Description: "Collects verifiable task execution evidence and updates derived\n" +
			"project intelligence metrics (.prumo/history/project-intelligence.json).",
		Subcommands: []string{
			"add <report.json>  Ingest a task execution evidence report",
			"summary            Print summary of project metrics, goals, and test history",
		},
		Flags: []string{
			"--path <dir>       Project directory (default: .)",
			"--json             Output summary as JSON envelope",
		},
		Examples: []string{
			"prumo report add ./task-report.json --path ./my-project",
			"prumo report summary --path ./my-project",
		},
	},
	"snapshot": {
		Name:     "snapshot",
		Category: "Project Lifecycle",
		Summary:  "Create an archive backup snapshot of project state",
		Usage:    "prumo snapshot [path] --output <archive.zip>",
		Description: "Generates a complete, reproducible zip archive of project canonical state,\n" +
			"governance policies, and goals for backup or safe migration rollback.",
		Flags: []string{
			"--output <path>    Target zip archive path (required)",
			"--json             Output snapshot metadata as JSON",
		},
		Examples: []string{
			"prumo snapshot ./my-project --output ./backup-v1.zip",
		},
	},
	"migrate": {
		Name:     "migrate",
		Category: "Project Lifecycle",
		Summary:  "Migrate project metadata and schemas across versions",
		Usage:    "prumo migrate [path] [--dry-run]",
		Description: "Upgrades project schemas and configurations to the current Prumo version.\n" +
			"Always run with --dry-run first to preview changes.",
		Flags: []string{
			"--dry-run          Preview migration changes without modifying files",
			"--json             Output migration plan as JSON",
		},
		Examples: []string{
			"prumo migrate ./my-project --dry-run",
			"prumo migrate ./my-project",
		},
	},
	"repo": {
		Name:     "repo",
		Category: "Governance & Traceability",
		Summary:  "Repository governance policy and risk gate enforcement",
		Usage:    "prumo repo <subcommand> [args]",
		Description: "Enforces repository policy (.prumo/repository/policy.json) for branches,\n" +
			"commit message conventions, PR risk evaluation, and emergency bypass audits.",
		Subcommands: []string{
			"policy             Display effective repository governance policy",
			"branch-check <name> Validate branch naming pattern",
			"commit-check <msg>  Validate conventional commit message format",
			"pr-check [args]    Evaluate PR risk level and required verification gates",
		},
		Flags: []string{
			"--json             Output policy evaluation as JSON",
		},
		Examples: []string{
			"prumo repo policy",
			"prumo repo branch-check feat/my-feature",
			"prumo repo commit-check \"feat(core): implement feature\"",
		},
	},
	"docs": {
		Name:     "docs",
		Category: "Governance & Traceability",
		Summary:  "Documentation architecture and semantic delta checks",
		Usage:    "prumo docs <subcommand> [args]",
		Description: "Validates local canonical engineering documentation against schema\n" +
			"contracts and tracks semantic documentation deltas.",
		Subcommands: []string{
			"contracts          List documentation contracts (show <id> for detail)",
			"profiles           List documentation profiles (show <id> for detail)",
			"audit              Report contract coverage for this repository",
			"readiness          Report Goal implementation readiness",
			"impact <paths...>  Analyze documentation impact of changed paths",
			"delta <action>     Manage semantic documentation deltas",
			"contradictions     Detect contradictions across bound documents",
			"authority          Validate authority map, version drift and routing links",
			"plan --goal <id>   Preflight documentation obligations (--changed … --reconcile for postflight)",
			"explain <id>       Explain a planned obligation, contract or bound document",
			"agents <build|verify|explain>  Compile and verify agent instruction surfaces",
			"verify [--strict]  Run deterministic documentation checks (--strict also requires semantic readiness)",
			"gauntlet           Run bounded deterministic documentation rounds",
			"glossary           Validate the terminology registry and its use in current documents",
			"version            Check the documentation version policy against the shipped release",
			"metrics            Report documentation intelligence computed from repository state",
			"resources <read|list>  Read documentation resources over the MCP surface",
			"mutations          Report which documentation mutations are allowed (denied by default)",
			"adopt <inspect|propose>  Discover and propose contracts in a brownfield repository",
		},
		Flags: []string{
			"--json             Output results as JSON",
			"--path <dir>       Repository root (default .)",
			"--goal <id>        Goal identifier for readiness",
			"--out <dir>        Target dir for compiled vendor surfaces",
		},
		Examples: []string{
			"prumo docs audit",
			"prumo docs readiness --goal M5",
			"prumo docs authority --json",
			"prumo docs plan --goal M6 --changed schemas/goal.schema.json",
			"prumo docs plan --goal M6 --changed cmd/prumo/main.go --reconcile",
			"prumo docs explain cli.reference --goal M6",
			"prumo docs agents build",
			"prumo docs agents verify --json",
			"prumo docs verify --strict",
			"prumo docs glossary",
			"prumo docs version",
		},
	},
	"trace": {
		Name:     "trace",
		Category: "Governance & Traceability",
		Summary:  "Query traceability graph, events, and evidence",
		Usage:    "prumo trace <subcommand> [args]",
		Description: "Inspects provenance, causality DAGs, execution events, and\n" +
			"cryptographic evidence links across the lifecycle.",
		Subcommands: []string{
			"graph              Display traceability graph connections",
			"events             List recorded execution events",
			"evidence           Inspect cryptographic evidence entries",
		},
		Flags: []string{
			"--json             Output traceability data as JSON",
		},
		Examples: []string{
			"prumo trace graph",
			"prumo trace events --json",
		},
	},
	"journal": {
		Name:     "journal",
		Category: "Governance & Traceability",
		Summary:  "Inspect operational event journal",
		Usage:    "prumo journal [subcommand] [args]",
		Description: "Provides append-only operational audit trail inspection for agent\n" +
			"actions, tool executions, and state modifications.",
		Flags: []string{
			"--json             Output journal entries as JSON",
		},
		Examples: []string{
			"prumo journal",
		},
	},
	"experience": {
		Name:     "experience",
		Category: "Governance & Traceability",
		Summary:  "Developer experience heuristics and next-action guidance",
		Usage:    "prumo experience [subcommand] [args]",
		Description: "Evaluates developer workflow ergonomics, active bottlenecks, and\n" +
			"provides proactive next-action recommendations.",
		Flags: []string{
			"--json             Output experience metrics as JSON",
		},
		Examples: []string{
			"prumo experience",
		},
	},
	"setup": {
		Name:        "setup",
		Category:    "Environment & Connectors",
		Summary:     "Initialize global Prumo directories and machine state",
		Usage:       "prumo setup",
		Description: "Sets up ~/.prumo directories, catalogs, and local machine configuration.",
		Flags: []string{
			"--home <path>      Custom Prumo home directory (default: ~/.prumo)",
			"--json             Output setup status as JSON",
		},
		Examples: []string{
			"prumo setup",
		},
	},
	"install": {
		Name:     "install",
		Category: "Environment & Connectors",
		Summary:  "Install IDE and agent connectors or CLI binaries",
		Usage:    "prumo install <target> [--dir <path>]",
		Description: "Installs target harness connectors (e.g., opencode, antigravity, codex)\n" +
			"into host environments.",
		Flags: []string{
			"--dir <path>       Target installation directory",
			"--home <path>      Custom Prumo home directory",
			"--json             Output installation report as JSON",
		},
		Examples: []string{
			"prumo install opencode",
			"prumo install antigravity",
		},
	},
	"uninstall": {
		Name:        "uninstall",
		Category:    "Environment & Connectors",
		Summary:     "Uninstall specified connector",
		Usage:       "prumo uninstall <target>",
		Description: "Removes an installed harness connector from the host environment.",
		Flags: []string{
			"--home <path>      Custom Prumo home directory",
			"--json             Output uninstallation status as JSON",
		},
		Examples: []string{
			"prumo uninstall opencode",
		},
	},
	"connector": {
		Name:        "connector",
		Category:    "Environment & Connectors",
		Summary:     "Manage and inspect harness connectors",
		Usage:       "prumo connector <subcommand> [args]",
		Description: "Inspects status, capabilities, and health of installed AI harness connectors.",
		Subcommands: []string{
			"list               List all supported and installed connectors",
			"status <name>      Check status of a specific connector",
		},
		Flags: []string{
			"--json             Output connector data as JSON",
		},
		Examples: []string{
			"prumo connector list",
			"prumo connector status opencode",
		},
	},
	"run": {
		Name:     "run",
		Category: "Control Plane Runtime",
		Summary:  "Execute the autonomous goal implementation loop",
		Usage:    "prumo run [args]",
		Description: "Starts the deterministic execution loop for the active project goal,\n" +
			"enforcing Red-Green-Refactor cycles and quality gates.",
		Flags: []string{
			"--json             Output execution events as JSON",
		},
		Examples: []string{
			"prumo run",
		},
	},
	"continue": {
		Name:        "continue",
		Category:    "Control Plane Runtime",
		Summary:     "Resume execution from a checkpointed state",
		Usage:       "prumo continue [args]",
		Description: "Resumes execution from the most recent safe checkpoint in .prumo/runtime/.",
		Flags: []string{
			"--json             Output resume status as JSON",
		},
		Examples: []string{
			"prumo continue",
		},
	},
	"budget": {
		Name:        "budget",
		Category:    "Control Plane Runtime",
		Summary:     "Inspect token, cost, and execution budgets",
		Usage:       "prumo budget [args]",
		Description: "Displays consumed tokens, model call costs, and remaining budget limits.",
		Flags: []string{
			"--json             Output budget metrics as JSON",
		},
		Examples: []string{
			"prumo budget",
		},
	},
	"debug": {
		Name:        "debug",
		Category:    "Control Plane Runtime",
		Summary:     "Debug runtime state and Working Context Capsules",
		Usage:       "prumo debug [args]",
		Description: "Inspects memory capsules, active goal context, and runtime journal entries.",
		Flags: []string{
			"--json             Output debug state as JSON",
		},
		Examples: []string{
			"prumo debug",
		},
	},
	"model": {
		Name:     "model",
		Category: "Control Plane Runtime",
		Summary:  "Model registry, capabilities, and provider routing",
		Usage:    "prumo model [subcommand] [args]",
		Description: "Inspects configured AI models, capabilities (context window, tool calling),\n" +
			"and cost parameters across providers.",
		Flags: []string{
			"--json             Output model registry as JSON",
		},
		Examples: []string{
			"prumo model list",
		},
	},
	"env": {
		Name:        "env",
		Category:    "Environment & Connectors",
		Summary:     "Environment diagnostics and PATH configuration",
		Usage:       "prumo env [args]",
		Description: "Displays active environment variables, toolchain availability, and PATH state.",
		Flags: []string{
			"--json             Output environment status as JSON",
		},
		Examples: []string{
			"prumo env",
		},
	},
	"runtime": {
		Name:        "runtime",
		Category:    "Control Plane Runtime",
		Summary:     "Control plane runtime management",
		Usage:       "prumo runtime [subcommand] [args]",
		Description: "Inspects or resets runtime control plane state.",
		Flags: []string{
			"--json             Output runtime status as JSON",
		},
		Examples: []string{
			"prumo runtime status",
		},
	},
	"package": {
		Name:        "package",
		Category:    "Workforce & Compilation",
		Summary:     "Package management and distribution",
		Usage:       "prumo package [subcommand] [args]",
		Description: "Inspects or bundles skill packages and distribution archives.",
		Flags: []string{
			"--json             Output package info as JSON",
		},
		Examples: []string{
			"prumo package list",
			"prumo package sync",
		},
	},
	"workforce": {
		Name:        "workforce",
		Category:    "Workforce & Compilation",
		Summary:     "Workforce package synchronization and catalog management",
		Usage:       "prumo workforce <sync|list> [flags]",
		Description: "Synchronizes skills, agents, and recipes from remote or cached catalog, ensuring core skills (clean-code, cognitive-clarity) and updating prumo.lock.",
		Flags: []string{
			"--path <dir>       Target project directory (default: .)",
			"--offline          Use local cached skills only (no network requests)",
			"--force-remote     Bypass cache and force re-download from upstream",
			"--remote <url>     Custom upstream raw base URL",
			"--skills <list>    Comma-separated list of specific skills to sync",
			"--json             Output result as JSON envelope",
		},
		Examples: []string{
			"prumo workforce sync",
			"prumo workforce sync --offline",
			"prumo workforce list",
		},
	},
	"ask": {
		Name:     "ask",
		Category: "Harness",
		Summary:  "One-shot headless query interface for questions, analysis, and shell pipes",
		Usage:    "prumo ask [flags] [prompt]",
		Description: "One-shot, read-only by default interface for quick analysis, code explanation, and shell scripting without launching the interactive TUI. " +
			"Accepts prompt, stdin pipe, explicit context files, and returns clean response on stdout and diagnostics on stderr.",
		Flags: []string{
			"--file <path>        Add file content as context (repeatable)",
			"--context <type>     Context mode (e.g. repo)",
			"--model <id>         Model id to use",
			"--provider <name>    fake|openai-compat|anthropic|opencode (default: fake)",
			"--system <text>      Custom system instruction",
			"--raw                Raw output without newline decoration",
			"--json               Output envelope with model metadata as JSON",
		},
		Examples: []string{
			"prumo ask \"explain this function\"",
			"cat test.log | prumo ask \"analyze why tests failed\"",
			"prumo ask --file internal/model.go \"summarize this file\"",
			"prumo ask --json \"quick status check\"",
		},
	},
	"serve": {
		Name:     "serve",
		Category: "Harness",
		Summary:  "Start the Prumo Harness daemon over local Unix socket or remote TCP+TLS",
		Usage:    "prumo serve [--path <dir>] [--socket <path>]",
		Description: "Starts the persistent Harness daemon providing the versioned JSONL Agent Protocol (v0.4.0) " +
			"for desktop clients (Native Workspace Viewer), TUI, and external tools.",
		Flags: []string{
			"--path <dir>         Workspace root (default: .)",
			"--socket <path>      Daemon Unix socket path",
			"--listen <addr>      Expose serve over TCP+TLS",
			"--tls-cert <path>    TLS certificate path",
			"--tls-key <path>     TLS key path",
		},
		Examples: []string{
			"prumo serve",
			"prumo serve --path /path/to/project",
			"prumo serve --socket /tmp/prumo.sock",
		},
	},
	"agent": {
		Name:     "agent",
		Category: "Harness",
		Summary:  "Interactive coding agent TUI or headless harness (run/serve/ps/logs/...)",
		Usage:    "prumo agent [subcommand|flags]",
		Description: "Without subcommands, launches the interactive terminal coding agent (TUI). " +
			"With subcommands, runs the NativeAgent state machine headlessly: context, model, tools, permissions, checkpoints. " +
			"Provider-neutral (fake|fake-tools|openai-compat|anthropic|opencode); external agents via AgentProvider adapters. " +
			"`opencode` delegates the whole turn to the opencode CLI, which runs it with its own tools and its own authentication — including the models it serves for free. " +
			"A run that needs approval stops with status awaiting_approval and is answered with approve/deny.",
		Flags: []string{
			"--goal <text>        Goal for agent run (default: headless run)",
			"--path <dir>         Workspace root (default: .)",
			"--provider <name>    fake|fake-tools|openai-compat|anthropic|opencode (default: fake)",
			"--model <id>         Model id for real providers",
			"--base-url <url>     Base URL for real providers",
			"--api-key <key>      API key (else PRUMO_MODEL_API_KEY)",
			"--permission <mode>  allow|ask|deny default action (default: allow)",
			"--ask-kind <kinds>   Comma-separated tool kinds that require approval",
			"--max-turns <n>      Max turns (default: 5)",
			"--run <id>           Run id",
			"--request <id>       Permission request id to approve or deny",
			"--reason <text>      Why a permission was denied",
			"--from <agent>       Handoff sender (default: native)",
			"--to <agent>         Handoff recipient",
			"--client <ver>       Client protocol version to negotiate",
			"--manifest            Print the full protocol IDL manifest",
			"--socket <path>      Daemon socket (serve/ps/logs/steer/approve)",
			"--remote <addr>      Daemon TCP endpoint (ps/logs/steer/approve/schedule/...)",
			"--token <t>          Remote token (or --token-file / PRUMO_DAEMON_TOKEN)",
			"--token-file <p>     File holding the remote token",
			"--remote-tls-cert <p> CA cert pinning the remote server",
			"--listen <addr>      Expose serve over TCP+TLS (requires token)",
			"--tls-cert <p>       Server TLS certificate",
			"--tls-key <p>        Server TLS key",
			"--message <text>     Steering input for a live run",
			"--every <secs>       Schedule interval (min 5)",
			"--job <id>           Job id (schedule/unschedule)",
			"--session <id>       Planning session id (promote)",
			"--session-file <p>   Planning session file (promote)",
			"--keep <n>           Checkpoints kept per run by gc",
			"--max-age-days <n>   Orphan age collected by gc",
			"--start              Start the promoted run immediately",
			"--opencode-url <u>   OpenCode server URL to probe",
			"--sandbox <kind>     local|container (default: local)",
			"--sandbox-image <i>  Container image (required with container)",
			"--sandbox-runtime <r> Docker runtime (e.g. runsc for gVisor)",
			"--context-budget <n> Context token budget (default: 8000)",
			"--context-level <l>  Disclosure level L0-L4 (default: L1)",
			"--budget-tokens <n>  Hard token budget (0 = track only)",
			"--budget-usd <n>     Hard USD budget (0 = track only)",
			"--budget-tools <n>   Hard tool-call budget (0 = track only)",
			"--strict             Require a passing test.run for completion",
			"--gates <file>       JSON gate policies evaluated at completion",
			"--compact-keep <n>   Cap conversation (0 = off)",
			"--compact-budget <n> Tokens before auto-compact (default: context budget)",
			"--egress-deny        Fail-closed network egress for local exec",
			"--egress-allow <h>   Comma-separated allowed hosts",
			"--mcp <cmd>          MCP server command (stdio, untrusted)",
			"--json               Output structured JSON envelope",
		},
		Examples: []string{
			"prumo agent run --goal 'fix typo' --path . ",
			"prumo agent resume --run R-agent-1 --path .",
			"prumo agent handoff --run R-agent-1 --to codex",
			"prumo agent events --run R-agent-1 --path .",
			"prumo agent protocol --client 0.1.0",
			"prumo agent serve --path .",
			"prumo agent ps",
			"prumo agent approve --run R-agent-1 --request perm-c1",
			"prumo agent deny --run R-agent-1 --request perm-c1 --reason 'outside the workspace'",
			"prumo agent logs --run R-agent-1",
			"prumo agent providers",
			"prumo agent models --provider openai-compat --model gpt-4o-mini",
		},
	},
	"tui": {
		Name:     "tui",
		Category: "Harness",
		Summary:  "Terminal client for the Agent Protocol (palette → goal → run → stream → evidence)",
		Usage:    "prumo tui [flags]",
		Description: "Supervises a `prumo agent serve` daemon and drives it over the public Agent Protocol. " +
			"Palette-first navigation, live AgentEvent timeline, and a terminal evidence panel. " +
			"Imports only the public SDK — never prumo/internal. Interactive: no --json output.",
		Flags: []string{
			"--path <dir>         Workspace root (default: .)",
			"--socket <path>      Attach to an existing daemon instead of starting one",
			"--remote <addr>      Attach over TCP+TLS instead (requires a token)",
			"--token <t>          Remote token (or --token-file / PRUMO_DAEMON_TOKEN)",
			"--token-file <p>     File holding the remote token",
			"--remote-tls-cert <p> CA cert pinning the remote server",
			"--provider <name>    fake|fake-tools|openai-compat|anthropic|opencode (default: fake)",
			"--model <id>         Model id for real providers",
			"--max-turns <n>      Max turns (default: 5)",
			"--theme <id>         theme.default|theme.high-contrast|theme.no-color|theme.reduced-motion",
		},
		Examples: []string{
			"prumo tui --path .",
			"prumo tui --theme theme.no-color",
			"prumo tui --socket .prumo/runtime/harness/agentd.sock",
			"prumo tui --remote 127.0.0.1:7777 --token-file .prumo/agentd.token",
		},
	},
	"native": {
		Name:     "native",
		Category: "Harness",
		Summary:  "Native desktop agent-aware Workspace Viewer and light editor (Rust + Freya)",
		Usage:    "prumo native [--workspace <dir>] [--socket <path>]",
		Description: "Launches the Prumo Native Workspace Viewer built with Rust and Freya. " +
			"Provides an IDE-style Explorer, multi-tab editing, Quick Open, working-copy diff, daemon timeline, " +
			"run start, and permission approval over the local Agent Protocol. " +
			"Interactive: no --json output.",
		Flags: []string{
			"--workspace <dir>    Workspace root directory (default: .)",
			"--socket <path>      Daemon Unix socket (default: <workspace>/.prumo/runtime/harness/agentd.sock)",
		},
		Examples: []string{
			"prumo native",
			"prumo native --workspace ./my-project",
			"prumo native --workspace . --socket .prumo/runtime/harness/agentd.sock",
		},
	},
	"viewer": {
		Name:        "viewer",
		Category:    "Harness",
		Summary:     "Alias for `prumo native`: native desktop Workspace Viewer",
		Usage:       "prumo viewer [flags]",
		Description: "Alias for `prumo native`.",
		Examples: []string{
			"prumo viewer",
			"prumo viewer --workspace .",
		},
	},
	"ui": {
		Name:     "ui",
		Category: "Harness",
		Summary:  "Interface map: composition tree, element positions and interconnections",
		Usage:    "prumo ui <map|verify|impact|config> [flags]",
		Description: "Compiles the complete map of a user interface — components, subcomponents, text, inputs, " +
			"buttons, menus — with where each element sits and how the elements interconnect, then projects it " +
			"for the developer, the documentation site and the code agent. Structure, labels and placement are " +
			"declared; symbols and coverage are derived from the implementation, and a declared element always wins. " +
			"Applies to any project Prumo builds that has an interface, and each audience can be switched off " +
			"individually under `ui.interface_map` in prumo.json.",
		Flags: []string{
			"--path <dir>         Project root (default: .)",
			"--file <map>         Canonical map to compile (default: docs/ui-ux/interface-map.json)",
			"--target <names>     Narrow the projections: developer,site,agent (comma separated)",
			"--write              Emit the projections under .prumo/runtime/interface-map/",
			"--json               Output structured JSON envelope",
		},
		Examples: []string{
			"prumo ui config",
			"prumo ui verify --path .",
			"prumo ui map --write --target agent",
			"prumo ui impact node:palette",
		},
	},
	"automation": {
		Name:        "automation",
		Category:    "Platform & Automation",
		Summary:     "Automation workflow engine",
		Usage:       "prumo automation [subcommand] [args]",
		Description: "Manages scheduled and event-driven automation recipes.",
		Flags: []string{
			"--json             Output automation status as JSON",
		},
		Examples: []string{
			"prumo automation list",
		},
	},
}

// helpCategoryOrder is the order the top-level help prints command categories
// in. It is a package variable rather than a local slice so a test can assert
// that every category in the registry appears here — the defect that hid
// `agent` was a category missing from this list.
var helpCategoryOrder = []string{
	"Project Lifecycle",
	"Goals & Execution",
	"Workforce & Compilation",
	"Tooling & Verification",
	"Adoption & Migration",
	"Governance & Traceability",
	"Control Plane Runtime",
	"Harness",
	"Platform & Automation",
	"Environment & Connectors",
	"General",
}

// PrintGeneralHelp prints top-level CLI help to stdout.
func PrintGeneralHelp(asJSON bool) int {
	if asJSON {
		categories := map[string][]map[string]string{}
		for _, info := range commandRegistry {
			categories[info.Category] = append(categories[info.Category], map[string]string{
				"name":    info.Name,
				"summary": info.Summary,
				"usage":   info.Usage,
			})
		}
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"version":    protocol.CLIVersion,
			"categories": categories,
		}))
	}

	fmt.Printf("Prumo CLI v%s\n\n", protocol.CLIVersion)
	fmt.Println("Usage:")
	fmt.Println("  prumo [--json] [--home <path>] <command> [subcommand] [flags]")
	fmt.Println()

	catMap := map[string][]CommandInfo{}
	for _, info := range commandRegistry {
		catMap[info.Category] = append(catMap[info.Category], info)
	}

	for _, cat := range helpCategoryOrder {
		cmds, ok := catMap[cat]
		if !ok || len(cmds) == 0 {
			continue
		}
		sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })

		fmt.Printf("%s:\n", cat)
		for _, cmd := range cmds {
			fmt.Printf("  %-18s %s\n", cmd.Name, cmd.Summary)
		}
		fmt.Println()
	}

	fmt.Println("Global Flags:")
	fmt.Println("  --json              Format command output as structured JSON envelope")
	fmt.Println("  --home <path>       Specify custom Prumo home directory (default: ~/.prumo)")
	fmt.Println("  --help, -h          Display help information for Prumo or any command")
	fmt.Println()
	fmt.Println("Quick Examples:")
	fmt.Println("  prumo init ./my-project --profile examples/brasa/project-profile.json")
	fmt.Println("  prumo tool check-escape-hatches .")
	fmt.Println("  prumo compile --target antigravity --path ./my-project")
	fmt.Println("  prumo doctor ./my-project")
	fmt.Println("  prumo agent                     # launch interactive agent TUI")
	fmt.Println()
	fmt.Println("Run 'prumo <command> --help' or 'prumo help <command>' for detailed help on any command.")
	return exitOK
}

// PrintCommandHelp prints detailed help for a specific command to stdout.
func PrintCommandHelp(command string, asJSON bool) int {
	cmd := strings.ToLower(command)
	info, found := commandRegistry[cmd]
	if !found {
		fmt.Fprintf(os.Stderr, "error: unknown command '%s'\n\n", command)
		fmt.Fprintf(os.Stderr, "Run 'prumo --help' to view all available commands.\n")
		return exitUsage
	}

	if asJSON {
		return printEnvelope(protocol.OkEnvelope(map[string]any{
			"command":     info.Name,
			"category":    info.Category,
			"summary":     info.Summary,
			"usage":       info.Usage,
			"description": info.Description,
			"flags":       info.Flags,
			"subcommands": info.Subcommands,
			"examples":    info.Examples,
		}))
	}

	fmt.Printf("COMMAND: prumo %s\n\n", info.Name)
	fmt.Printf("Summary:\n  %s\n\n", info.Summary)
	fmt.Printf("Usage:\n  %s\n\n", info.Usage)

	if info.Description != "" {
		fmt.Printf("Description:\n")
		lines := strings.Split(info.Description, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()
	}

	if len(info.Subcommands) > 0 {
		fmt.Println("Subcommands:")
		for _, sub := range info.Subcommands {
			fmt.Printf("  %s\n", sub)
		}
		fmt.Println()
	}

	if len(info.Flags) > 0 {
		fmt.Println("Flags & Options:")
		for _, flg := range info.Flags {
			fmt.Printf("  %s\n", flg)
		}
		fmt.Println()
	}

	if len(info.Examples) > 0 {
		fmt.Println("Examples:")
		for _, ex := range info.Examples {
			fmt.Printf("  $ %s\n", ex)
		}
		fmt.Println()
	}

	return exitOK
}
