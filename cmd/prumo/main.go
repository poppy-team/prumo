package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/raillen/prumo/internal/cliops"
	"github.com/raillen/prumo/internal/project"
	"github.com/raillen/prumo/internal/protocol"
)

const (
	exitOK           = 0
	exitValidation   = 1
	exitUsage        = 2
	exitConfig       = 3
	exitUnavailable  = 4
	exitInternal     = 5
	exitProjectError = 6
)

func repoRoot() string {
	if root := os.Getenv("PRUMO_REPO_ROOT"); root != "" {
		return root
	}
	if executable, err := os.Executable(); err == nil {
		candidate := executable
		for range 6 {
			candidate = dirName(candidate)
			if exists(candidate, "go.mod") && exists(candidate, "schemas") {
				return candidate
			}
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidate := cwd
		for range 6 {
			if exists(candidate, "go.mod") && exists(candidate, "schemas") {
				return candidate
			}
			parent := dirName(candidate)
			if parent == candidate {
				break
			}
			candidate = parent
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		devPath := filepath.Join(home, "Documentos", "Projetos", "prumo")
		if exists(devPath, "go.mod") && exists(devPath, "schemas") {
			return devPath
		}
	}
	return "."
}

func dirName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "."
}

func exists(parts ...string) bool {
	path := parts[0]
	for _, part := range parts[1:] {
		path += "/" + part
	}
	_, err := os.Stat(path)
	return err == nil
}

func printEnvelope(env protocol.Envelope) int {
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to encode JSON: %s\n", err)
		return exitInternal
	}
	fmt.Println(string(data))
	if env.Ok {
		return exitOK
	}
	return exitValidation
}

func envelopeError(code, message string) int {
	return printEnvelope(protocol.ErrEnvelope(protocol.Diagnostic{Code: code, Message: message}))
}

func serviceError(asJSON bool, err error) int {
	if err == nil {
		return exitOK
	}
	message := err.Error()
	if asJSON {
		return envelopeError(protocol.CodeValidationFailed, message)
	}
	fmt.Fprintf(os.Stderr, "error: %s\n", message)
	if message == "not a recognized Prumo project: missing prumo.json" {
		return exitProjectError
	}
	return exitValidation
}

func usage() int {
	fmt.Fprintf(os.Stderr, "Usage: prumo [--json] [--home <path>] <command> [subcommand] [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Run 'prumo --help' or 'prumo help' to view all available commands and options.\n")
	return exitUsage
}

func flag(args []string, name string) (string, []string, bool) {
	for i, arg := range args {
		if arg == name {
			if i+1 >= len(args) {
				return "", args, false
			}
			value := args[i+1]
			rest := append(append([]string{}, args[:i]...), args[i+2:]...)
			return value, rest, true
		}
	}
	return "", args, true
}

func hasFlag(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	asJSON := false
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			asJSON = true
			continue
		}
		rest = append(rest, arg)
	}
	if len(rest) == 0 {
		return usage()
	}
	if rest[0] == "--help" || rest[0] == "-h" {
		return PrintGeneralHelp(asJSON)
	}
	if rest[0] == "help" {
		if len(rest) > 1 {
			return PrintCommandHelp(rest[1], asJSON)
		}
		return PrintGeneralHelp(asJSON)
	}
	if hasFlag(rest[1:], "--help") || hasFlag(rest[1:], "-h") {
		return PrintCommandHelp(rest[0], asJSON)
	}
	home := ""
	cleaned := make([]string, 0, len(rest))
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--home" {
			if i+1 >= len(rest) {
				fmt.Fprintf(os.Stderr, "error: --home requires a value\n")
				return exitUsage
			}
			home = rest[i+1]
			i++
			continue
		}
		cleaned = append(cleaned, rest[i])
	}
	rest = cleaned
	svc := cliops.New(repoRoot())
	switch rest[0] {
	case "repo":
		return runRepositoryPolicy(asJSON, rest)
	case "docs":
		return runDocumentation(asJSON, rest)
	case "run", "continue", "budget", "debug", "tool", "model", "env", "runtime":
		return runRuntime(asJSON, rest)
	case "agent":
		return runAgent(asJSON, rest[1:])
	case "package", "automation":
		return runPlatform(asJSON, rest)
	case "workforce":
		return runWorkforce(asJSON, home, rest[1:])
	case "setup":
		return runSetup(asJSON, home)
	case "install":
		return runInstall(asJSON, home, rest[1:])
	case "uninstall":
		return runUninstall(asJSON, home, rest[1:])
	case "status":
		path := "."
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--path" {
				if i+1 >= len(rest) {
					fmt.Fprintf(os.Stderr, "error: --path requires a value\n")
					return exitUsage
				}
				path = rest[i+1]
				i++
			}
		}
		root, err := project.FindRoot(path)
		if err != nil {
			if asJSON {
				return envelopeError(protocol.CodeProjectNotFound, err.Error())
			}
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			return exitProjectError
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{"root": root}))
		}
		fmt.Println(root)
		return exitOK
	case "version":
		if len(rest) > 1 {
			fmt.Fprintf(os.Stderr, "error: version takes no arguments\n")
			return exitUsage
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]string{"version": protocol.CLIVersion, "protocol_version": protocol.ProtocolVersion}))
		}
		fmt.Println(protocol.CLIVersion)
		return exitOK
	case "init":
		profile, restArgs, ok := flag(rest[1:], "--profile")
		if !ok {
			fmt.Fprintf(os.Stderr, "error: --profile requires a value\n")
			return exitUsage
		}
		nonInteractive := hasFlag(restArgs, "--non-interactive")
		positional := []string{}
		for _, arg := range restArgs {
			if arg != "--non-interactive" {
				positional = append(positional, arg)
			}
		}
		path := "."
		if len(positional) > 0 {
			path = positional[0]
		}
		if profile == "" {
			if nonInteractive {
				fmt.Fprintf(os.Stderr, "error: --profile is required with --non-interactive\n")
				return exitUsage
			}
			fmt.Fprintf(os.Stderr, "error: --profile is required\n")
			return exitUsage
		}
		resolution, err := svc.Init(path, profile)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{"agents": resolution.Agents, "skills": resolution.Skills, "recipes": resolution.Recipes}))
		}
		fmt.Printf("Initialized Prumo v0.5 in %s\n", path)
		fmt.Printf("Agents: %d | Skills: %d | Recipes: %d\n", len(resolution.Agents), len(resolution.Skills), len(resolution.Recipes))
		return exitOK
	case "resolve":
		if len(rest) != 2 {
			fmt.Fprintf(os.Stderr, "error: resolve requires a profile path\n")
			return exitUsage
		}
		resolution, err := svc.Resolve(rest[1])
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(resolution))
		}
		fmt.Println("Agents:")
		for _, value := range resolution.Agents {
			fmt.Printf("  - %s\n", value)
		}
		fmt.Println("Skills:")
		for _, value := range resolution.Skills {
			fmt.Printf("  - %s\n", value)
		}
		fmt.Println("Recipes:")
		for _, value := range resolution.Recipes {
			fmt.Printf("  - %s\n", value)
		}
		return exitOK
	case "validate":
		path := "."
		if len(rest) > 1 {
			path = rest[1]
		}
		errors := svc.Validate(path)
		if len(errors) > 0 {
			if asJSON {
				return envelopeError(protocol.CodeValidationFailed, errors[0])
			}
			fmt.Println("Validation failed:")
			for _, message := range errors {
				fmt.Printf("- %s\n", message)
			}
			return exitValidation
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{"path": path, "valid": true}))
		}
		fmt.Println("Prumo validation passed.")
		return exitOK
	case "goal":
		return runGoal(svc, asJSON, rest[1:])
	case "plan":
		return runPlan(asJSON, rest[1:])
	case "adopt":
		return runAdopt(asJSON, rest[1:])
	case "trace":
		return runTrace(asJSON, rest[1:])
	case "journal":
		return runJournal(asJSON, rest[1:])
	case "experience":
		return runExperience(asJSON, rest[1:])
	case "connector":
		return runConnector(asJSON, rest[1:])
	case "context":
		return runContext(svc, asJSON, rest[1:])
	case "report":
		return runReport(svc, asJSON, rest[1:])
	case "migrate":
		path := "."
		if len(rest) > 1 && rest[1] != "--dry-run" {
			path = rest[1]
		}
		report, err := svc.Migrate(path, hasFlag(rest, "--dry-run"))
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(report))
		}
		fmt.Printf("Project migration report (v%v -> v%v):\n", report["from_version"], report["to_version"])
		for _, change := range report["changes"].([]string) {
			fmt.Printf("  - %s\n", change)
		}
		return exitOK
	case "compile":
		target := ""
		path := "."
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--target" {
				if i+1 >= len(rest) {
					fmt.Fprintf(os.Stderr, "error: --target requires a value\n")
					return exitUsage
				}
				target = rest[i+1]
				i++
			} else if rest[i] == "--path" {
				if i+1 >= len(rest) {
					fmt.Fprintf(os.Stderr, "error: --path requires a value\n")
					return exitUsage
				}
				path = rest[i+1]
				i++
			}
		}
		if target == "" {
			fmt.Fprintf(os.Stderr, "error: --target is required\n")
			return exitUsage
		}
		created, err := svc.Compile(path, target)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{"target": target, "created": created}))
		}
		fmt.Printf("Compiled %s: %d files\n", target, len(created))
		return exitOK
	case "snapshot":
		path := "."
		output := ""
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--output" {
				if i+1 >= len(rest) {
					fmt.Fprintf(os.Stderr, "error: --output requires a value\n")
					return exitUsage
				}
				output = rest[i+1]
				i++
			} else if len(path) == 1 && path == "." {
				path = rest[i]
			}
		}
		if err := svc.Snapshot(path, output); err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			target := output
			if target == "" {
				target = path + "/.prumo/" + baseName(path) + "-prumo-snapshot.zip"
			}
			return printEnvelope(protocol.OkEnvelope(map[string]any{"output": target}))
		}
		if output == "" {
			fmt.Println(path + "/.prumo/" + baseName(path) + "-prumo-snapshot.zip")
		} else {
			fmt.Println(output)
		}
		return exitOK
	case "doctor":
		path := "."
		if len(rest) > 1 && rest[1] != "--json" {
			path = rest[1]
		}
		findings, err := svc.Doctor(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		hasErrors := false
		for _, finding := range findings {
			if finding["severity"] == "ERROR" {
				hasErrors = true
			}
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(findings))
		}
		if len(findings) == 0 {
			fmt.Println("Prumo Doctor: all checks passed cleanly.")
			return exitOK
		}
		fmt.Printf("Prumo Doctor found %d issue(s):\n", len(findings))
		for _, finding := range findings {
			fmt.Printf("[%s] (%s) %s\n", finding["severity"], finding["category"], finding["message"])
		}
		if hasErrors {
			return exitValidation
		}
		return exitOK
	case "explain":
		return runExplain(svc, asJSON, rest[1:])
	case "framework-check":
		errors := svc.FrameworkCheck()
		if len(errors) > 0 {
			if asJSON {
				return envelopeError(protocol.CodeValidationFailed, errors[0])
			}
			fmt.Println("Framework validation failed:")
			for _, message := range errors {
				fmt.Printf("- %s\n", message)
			}
			return exitValidation
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{"valid": true}))
		}
		fmt.Println("Prumo framework validation passed.")
		return exitOK
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command '%s'\n\n", rest[0])
		fmt.Fprintf(os.Stderr, "Run 'prumo --help' or 'prumo help' to view all available commands.\n")
		return exitUsage
	}
}

func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func pathFlag(args []string) string {
	for i, arg := range args {
		if arg == "--path" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return "."
}

func runGoal(svc *cliops.Service, asJSON bool, args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "error: goal requires a subcommand\n")
		return exitUsage
	}
	switch args[0] {
	case "new":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "error: goal new requires id and title\n")
			return exitUsage
		}
		phase, _, ok := flag(args[3:], "--phase")
		if !ok || phase == "" {
			fmt.Fprintf(os.Stderr, "error: goal new requires --phase\n")
			return exitUsage
		}
		objective, _, _ := flag(args[3:], "--objective")
		path := pathFlag(args)
		if err := svc.NewGoal(path, args[1], args[2], phase, objective); err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(map[string]any{"id": args[1], "state": "DRAFT"}))
		}
		fmt.Printf("Created Goal %s\n", args[1])
		return exitOK
	case "state":
		if len(args) < 3 {
			fmt.Fprintf(os.Stderr, "error: goal state requires id and state\n")
			return exitUsage
		}
		reason, _, _ := flag(args[3:], "--reason")
		path := pathFlag(args)
		goal, err := svc.GoalState(path, args[1], args[2], reason)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(goal))
		}
		fmt.Printf("Goal %s transitioned to %s\n", args[1], goal["state"])
		return exitOK
	case "amend":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "error: goal amend requires id\n")
			return exitUsage
		}
		file, _, _ := flag(args[2:], "--file")
		reason, _, _ := flag(args[2:], "--reason")
		approvedBy, _, _ := flag(args[2:], "--approved-by")
		path := pathFlag(args)
		goal, err := svc.GoalAmend(path, args[1], file, reason, approvedBy)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(goal))
		}
		fmt.Printf("Goal %s amended to revision %v\n", args[1], goal["revision"])
		return exitOK
	case "list":
		path := pathFlag(args)
		goals, err := svc.GoalList(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(goals))
		}
		fmt.Printf("%-12s %-8s %-12s %-5s %s\n", "ID", "PHASE", "STATE", "REV", "TITLE")
		fmt.Println("-----------------------------------------------------------------")
		for _, goal := range goals {
			fmt.Printf("%-12v %-8v %-12v %-5v %v\n", goal["id"], goal["phase"], goal["state"], goal["revision"], goal["title"])
		}
		return exitOK
	default:
		fmt.Fprintf(os.Stderr, "error: unknown goal subcommand: %s\n", args[0])
		return exitUsage
	}
}

func runContext(svc *cliops.Service, asJSON bool, args []string) int {
	if len(args) < 2 || args[0] != "plan" {
		fmt.Fprintf(os.Stderr, "error: context requires: plan <task>\n")
		return exitUsage
	}
	path := pathFlag(args)
	plan := svc.ContextPlan(path, args[1])
	if asJSON {
		return printEnvelope(protocol.OkEnvelope(plan))
	}
	fmt.Printf("Profile: %v\nStrategy: %v\n", plan["profile"], plan["strategy"])
	return exitOK
}

func runReport(svc *cliops.Service, asJSON bool, args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "error: report requires a subcommand\n")
		return exitUsage
	}
	switch args[0] {
	case "add":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "error: report add requires a file\n")
			return exitUsage
		}
		path := pathFlag(args)
		data, err := svc.ReportAdd(path, args[1])
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(data["summary"]))
		}
		summary, _ := data["summary"].(map[string]any)
		fmt.Printf("Recorded task report; total tasks: %v\n", summary["tasks"])
		return exitOK
	case "summary":
		path := pathFlag(args)
		summary, err := svc.ReportSummary(path)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(summary))
		}
		data, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(data))
		return exitOK
	default:
		fmt.Fprintf(os.Stderr, "error: unknown report subcommand: %s\n", args[0])
		return exitUsage
	}
}

func runExplain(svc *cliops.Service, asJSON bool, args []string) int {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "error: explain requires a topic\n")
		return exitUsage
	}
	topic := args[0]
	target := ""
	positional := []string{}
	for i := 1; i < len(args); i++ {
		if args[i] == "--path" {
			i++
			continue
		}
		positional = append(positional, args[i])
	}
	if len(positional) > 0 {
		target = positional[0]
	}
	path := pathFlag(args)
	switch topic {
	case "workforce":
		if target == "" {
			fmt.Fprintf(os.Stderr, "error: explain workforce requires a profile path\n")
			return exitUsage
		}
		result, err := svc.ExplainWorkforce(target)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}
		fmt.Printf("Agents: %d | Skills: %d | Recipes: %d\n", len(result["agents"].(map[string]any)), len(result["skills"].(map[string]any)), len(result["recipes"].(map[string]any)))
		return exitOK
	case "agent", "skill", "recipe":
		if target == "" {
			fmt.Fprintf(os.Stderr, "error: explain %s requires an id\n", topic)
			return exitUsage
		}
		kind := topic + "s"
		result, found, err := svc.ExplainCatalog(kind, target)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if !found {
			fmt.Fprintf(os.Stderr, "error: not found: %s %s\n", topic, target)
			return exitValidation
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}
		fmt.Printf("%s: %s\n", topic, target)
		return exitOK
	case "context":
		result, err := svc.ExplainContext(path, target)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}
		fmt.Printf("Task: %v\nStrategy: %v\n", result["task_id"], result["strategy"])
		return exitOK
	case "model":
		result, err := svc.ExplainModel(path, target)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}
		fmt.Printf("Role: %v\nTarget: %v\n", result["role"], result["target"])
		return exitOK
	case "execution":
		result, err := svc.ExplainExecution(path, target)
		if err != nil {
			return serviceError(asJSON, err)
		}
		if asJSON {
			return printEnvelope(protocol.OkEnvelope(result))
		}
		fmt.Printf("Profile: %v\n", result["profile_id"])
		return exitOK
	default:
		fmt.Fprintf(os.Stderr, "error: unknown explain topic: %s\n", topic)
		return exitUsage
	}
}
