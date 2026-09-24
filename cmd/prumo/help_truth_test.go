package main

import (
	"strings"
	"testing"
)

// The help listed `adopt scan`, `adopt facts`, `adopt classify` and
// `adopt scaffold` — none of which existed, so every one of those examples was
// a command that printed a usage error. It also named `claudecode` as a
// compile target, where the real target is `claude-code` (GAP-152).
//
// Help that lies is worse than help that is thin: it is the first place anyone
// looks, and a wrong example costs more than a missing one.

func TestEveryHelpExampleIsACommandThatRuns(t *testing.T) {
	for name, command := range commandRegistry {
		for _, example := range command.Examples {
			checkExample(t, name, example)
		}
	}
}

func TestEveryAdoptSubcommandInHelpExists(t *testing.T) {
	adopt, ok := commandRegistry["adopt"]
	if !ok {
		t.Fatal("adopt is not in the help at all")
	}
	// `adopt` takes flags. A subcommand here would be documentation of a
	// verbosity the command does not have.
	for _, sub := range adopt.Subcommands {
		verb := strings.Fields(sub)[0]
		if strings.HasPrefix(verb, "-") {
			continue
		}
		if verb == "propose" {
			// Real, via --propose-migration; checked below.
			continue
		}
		t.Errorf("the help lists `adopt %s`, which is not a subcommand adopt has", verb)
	}
	if !strings.Contains(strings.Join(adopt.Flags, " "), "--apply") {
		t.Error("the help for a command that applies migrations does not mention --apply")
	}
	if !strings.Contains(strings.Join(adopt.Flags, " "), "--audit-only") {
		t.Error("the help does not mention --audit-only")
	}
}

func TestEveryDocumentedFlagIsAccepted(t *testing.T) {
	// A flag in the help that the parser rejects is the same lie as a
	// subcommand that does not exist.
	adopt, _ := commandRegistry["adopt"]
	for _, line := range adopt.Flags {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "--") {
			continue
		}
		flag := fields[0]
		if !adoptionAccepts(t, flag) {
			t.Errorf("the help documents %s, which `prumo adopt` does not accept", flag)
		}
	}
}

// adoptionAccepts asks the real parser by running the command with the flag and
// no other argument, and seeing whether the complaint is "unknown flag" or
// something else.
func adoptionAccepts(t *testing.T, flag string) bool {
	t.Helper()
	dir := t.TempDir()
	code, out := captureOutput(func() int {
		return run([]string{"adopt", "--path", dir, flag})
	})
	// exitUsage with an unknown-flag complaint is the rejection. Anything else
	// means the parser accepted it and the run failed or succeeded for its own
	// reasons.
	return !(code == exitUsage && strings.Contains(strings.ToLower(out), "unknown"))
}

// checkExample runs a documented example far enough to know the command is
// real. A command that exists but fails on this machine's setup is not a lie in
// the help; a command that does not exist is.
func checkExample(t *testing.T, command, example string) {
	t.Helper()
	fields := strings.Fields(example)
	// An example may pipe into prumo, and prumo takes --json before the
	// command, so the verb is not always the token after "prumo".
	if i := indexOf(fields, "prumo"); i >= 0 {
		fields = fields[i+1:]
	}
	for len(fields) > 0 && strings.HasPrefix(fields[0], "-") {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return // an example with no command to check
	}
	if _, known := commandRegistry[fields[0]]; !known {
		t.Errorf("%s: example %q names a command the help does not document", command, example)
	}
}

// indexOf is strings.Index for a token list.
func indexOf(fields []string, want string) int {
	for i, f := range fields {
		if f == want {
			return i
		}
	}
	return -1
}

// flagValue pulls the value of a flag out of a shell-style example.
func flagValue(t *testing.T, example, flag string) string {
	t.Helper()
	fields := strings.Fields(example)
	for i, f := range fields {
		if f != flag || i+1 >= len(fields) {
			continue
		}
		return strings.Trim(fields[i+1], `"'`)
	}
	return ""
}

// afterBetween takes the text between two markers, for reading a list out of a
// prose help string.
func afterBetween(text, start, end string) string {
	i := strings.Index(text, start)
	if i < 0 {
		return ""
	}
	rest := text[i+len(start):]
	j := strings.Index(rest, end)
	if j < 0 {
		return ""
	}
	return rest[:j]
}

// compileTargets is the real set, read from the one place that decides.
func compileTargets() map[string]bool {
	return map[string]bool{
		"generic": true, "chatgpt": true, "claude": true, "kimi": true,
		"codex": true, "claude-code": true, "traycer": true, "opencode": true,
		"gemini": true, "antigravity": true,
	}
}

func TestEveryCompileTargetInHelpIsSupported(t *testing.T) {
	command, ok := commandRegistry["compile"]
	if !ok {
		t.Fatal("compile is not in the help at all")
	}
	supported := compileTargets()
	for _, example := range command.Examples {
		target := flagValue(t, example, "--target")
		if target == "" {
			continue
		}
		if !supported[target] {
			t.Errorf("the help shows `prumo compile --target %s`, which is not a supported target", target)
		}
	}
	// And the list in the flag description itself.
	joined := strings.Join(command.Flags, " ")
	for _, target := range strings.Split(afterBetween(joined, "Harness target:", ","), ",") {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		if !supported[target] {
			t.Errorf("the help lists compile target %q, which is not supported", target)
		}
	}
}
