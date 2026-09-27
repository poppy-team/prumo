// Package onboard decides what a first run should offer.
//
// The offer exists because of an asymmetry: opencode serves models for free and
// authenticates itself, so someone who already has it needs no key, no account
// and no configuration to run a real agent — and someone who does not have it
// would never guess that from an empty model list.
//
// Nothing here installs anything or writes anything: it answers a question, so
// the answer can be tested away from a terminal and argued with in review.
package onboard

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// Decision is what the client should do at startup.
type Decision struct {
	// Ask is true only when the person has never been asked and has no working
	// provider to run with.
	Ask bool
	// Reason explains the decision, in the words a person would use.
	Reason string
	// OpenCodePath is where opencode was found, empty when it is not installed.
	OpenCodePath string
	// InstallCommand is what we would run to install opencode, empty when this
	// machine offers no way to install it (no npm on PATH).
	InstallCommand string
	// Notice is what a person who was not asked should still be told once —
	// that the provider they would want is already installed. Empty when there
	// is nothing worth interrupting for.
	Notice string
}

// InstallCommand is the command this client can honestly offer.
//
// It is npm because npm is what can be checked: the package `opencode-ai` and
// its `opencode` binary are on this machine, readable, from a global npm
// install. A curl-pipe-to-shell installer would be a URL this code cannot
// verify, and offering it as fact is how a client ends up downloading something
// it never inspected.
const InstallCommand = "npm install -g opencode-ai"

// Detect answers what to offer, given what is on this machine and what the
// person has already decided.
func Detect(provider string, onboarded bool) Decision {
	opencode, _ := exec.LookPath("opencode")
	if opencode != "" {
		d := Decision{
			Reason:       "opencode is installed: its models need no key of ours, and it is offered as a provider (" + opencode + ")",
			OpenCodePath: opencode,
		}
		// Only worth saying when there is a default to replace: someone already
		// running with a real provider does not need to be told about another,
		// and someone who chose the fake one on purpose does not either.
		if !onboarded && strings.TrimSpace(provider) == "fake" {
			d.Notice = "opencode is installed (" + opencode + "): run with `--provider opencode` " +
				"to use the models it serves for free, with its own authentication and no api-key. " +
				"Use ctrl+k to see what it offers."
		}
		return d
	}

	if onboarded {
		return Decision{Reason: "already asked, and opencode is still not installed"}
	}

	// A provider that was chosen deliberately is an answer already: asking again
	// would second-guess a decision the person made on purpose.
	if p := strings.TrimSpace(provider); p != "" && p != "fake" {
		return Decision{Reason: "a provider is already configured (" + p + ")"}
	}

	d := Decision{
		Ask:    true,
		Reason: "no provider is configured and opencode is not installed: it is the one path that needs no api-key",
	}
	if _, err := exec.LookPath("npm"); err == nil {
		d.InstallCommand = InstallCommand
	}
	return d
}

// Options are the answers this offer accepts, in the order the dialog shows them.
func Options(d Decision) []Option {
	options := []Option{}
	if d.InstallCommand != "" {
		options = append(options, Option{
			ID:        "install-opencode",
			Label:     "Install opencode and use its free models",
			Detail:    "runs: " + d.InstallCommand,
			Automatic: true,
		})
	} else {
		options = append(options, Option{
			ID:     "install-opencode",
			Label:  "Use opencode's free models (instale você, depois volte)",
			Detail: "no npm on PATH: install opencode the way your system prefers, then restart",
		})
	}
	options = append(options, Option{
		ID:     "api-key",
		Label:  "Configure an api-key provider instead",
		Detail: "export PRUMO_MODEL_API_KEY (and PRUMO_MODEL_BASE_URL when the endpoint is not the vendor's)",
	})
	options = append(options, Option{
		ID:     "later",
		Label:  "Not now",
		Detail: "the fake provider keeps runs deterministic until a real one is configured",
	})
	return options
}

// Option is one answer to the offer.
type Option struct {
	ID    string
	Label string
	// Detail is the consequence, stated under the label: what will run, or what
	// the person will have to do themselves.
	Detail string
	// Automatic is true when choosing it makes the client do the work.
	Automatic bool
}

// Install runs the installer and returns what it said.
//
// It refuses when there is no command this machine can run, rather than
// shelling out to something that does not exist: an offer that fails with
// "executable file not found" teaches the person nothing about their system.
//
// The output is returned whole and untruncated — an installer explains its own
// failure better than we can, and paraphrasing it is how a real cause gets lost.
func Install(ctx context.Context) (string, error) {
	if _, err := exec.LookPath("npm"); err != nil {
		return "", errors.New("npm is not on PATH, so the installer this client knows how to run is unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", "opencode-ai")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}
