// Package protocol is the versioned Harness protocol baseline: version
// negotiation between engine and clients (CLI/TUI/Desktop/ACP).
// Full IDL/SDK generation remains future work; this is the compat kernel.
package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is the current Harness protocol version.
const Version = "0.1.0"

// MinCompatible is the oldest client protocol the engine still speaks.
const MinCompatible = "0.1.0"

// Schemas lists the protocol schemas shipped with this version.
var Schemas = []string{
	"schemas/harness-checkpoint.schema.json",
	"schemas/harness-handoff.schema.json",
	"schemas/agent-event.schema.json",
}

// Negotiate checks a client version against the engine. Returns the engine
// version and whether interop is supported.
func Negotiate(client string) (server string, compatible bool, err error) {
	if strings.TrimSpace(client) == "" {
		return "", false, fmt.Errorf("client version required")
	}
	cMaj, cMin, err := split(client)
	if err != nil {
		return "", false, err
	}
	sMaj, sMin, err := split(Version)
	if err != nil {
		return "", false, err
	}
	mMaj, mMin, err := split(MinCompatible)
	if err != nil {
		return "", false, err
	}
	if cMaj != sMaj {
		return Version, false, nil
	}
	if cMaj < mMaj || (cMaj == mMaj && cMin < mMin) {
		return Version, false, nil
	}
	_ = sMin
	return Version, true, nil
}

func split(v string) (maj, min int, err error) {
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("invalid version %q", v)
	}
	maj, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid version %q", v)
	}
	min, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid version %q", v)
	}
	return maj, min, nil
}
