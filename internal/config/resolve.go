package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ---------------------------------------------------------------------------
// DocsMode — the two supported documentation storage modes
// ---------------------------------------------------------------------------

// DocsMode represents the documentation storage strategy.
type DocsMode string

const (
	// ModeVault stores all documentation under a single configurable base path:
	// <base_path>/<system>/...
	ModeVault DocsMode = "vault"

	// ModeInProject stores documentation relative to the project root under
	// docs/doc-agent/ with no <system> subfolder.
	ModeInProject DocsMode = "in-project"
)

// ---------------------------------------------------------------------------
// ProjectMarker — per-project override stored in .doc-agent.json
// ---------------------------------------------------------------------------

// ProjectMarker represents the contents of a .doc-agent.json file placed at a
// project root to override the global doc mode for that project.
// The schema is a subset of AppConfig so a single JSON vocabulary covers both.
type ProjectMarker struct {
	// Mode overrides the global mode for this project directory.
	Mode DocsMode `json:"mode"`
	// Path is an optional vault base path for per-repo vault edge cases.
	Path string `json:"path,omitempty"`
}

// markerFileName is the name of the per-project override file.
const markerFileName = ".doc-agent.json"

// readMarker reads the .doc-agent.json marker from the given directory.
// Returns:
//   - marker: parsed contents (zero value if absent)
//   - found: true if the file existed and was parsed successfully
//   - err: non-nil on I/O error or malformed JSON; missing file is NOT an error
func ReadMarker(dir string) (ProjectMarker, bool, error) {
	path := filepath.Join(dir, markerFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProjectMarker{}, false, nil
		}
		return ProjectMarker{}, false, err
	}

	var marker ProjectMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return ProjectMarker{}, false, err
	}

	return marker, true, nil
}

// ---------------------------------------------------------------------------
// resolveMode — deterministic precedence rule (pure function)
// ---------------------------------------------------------------------------

// isValidMode reports whether m is one of the supported documentation modes.
// Marker and config files are hand-edited JSON, so unknown values are expected
// input, not programming errors.
func isValidMode(m DocsMode) bool {
	return m == ModeVault || m == ModeInProject
}

// resolveMode applies the resolution precedence rule and returns the effective
// DocsMode for the current context:
//
//	marker.mode (cwd) > global config.mode > built-in default "vault"
//
// A level with an unknown mode value is skipped as if it were absent, so a
// hand-edited marker or config can never propagate an invalid mode downstream.
//
// Parameters:
//   - markerMode: the mode declared in the per-project marker (empty if absent)
//   - markerFound: true if a marker file was actually found in cwd
//   - globalMode: the mode from the global config (empty = use built-in default)
//
// This is a pure function with no I/O; testable without filesystem fixtures.
func ResolveMode(markerMode DocsMode, markerFound bool, globalMode DocsMode) DocsMode {
	// Highest priority: per-project marker (when present and valid)
	if markerFound && isValidMode(markerMode) {
		return markerMode
	}

	// Mid priority: global config mode (when set and valid)
	if isValidMode(globalMode) {
		return globalMode
	}

	// Built-in default: vault (preserves pre-v4 behaviour)
	return ModeVault
}

// ExpandUserPath expands a leading "~" to the user's home directory and trims
// surrounding whitespace.
//
// Without this a vault path typed as "~/vault" was stored verbatim, resolved
// without error, and created a directory literally named "~" beside the current
// working directory. The real vault stayed untouched while the user hunted for
// where their documentation went — a silent failure, which is the worst kind.
//
// A "~user/..." form is deliberately left alone: resolving another account's
// home is not portable, and quietly treating it as the current user's home would
// be a different wrong answer rather than none.
func ExpandUserPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "~" && !strings.HasPrefix(trimmed, "~/") && !strings.HasPrefix(trimmed, `~\`) {
		return trimmed
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Nothing sensible to expand to; hand back what was typed rather than
		// inventing a path.
		return trimmed
	}
	if trimmed == "~" {
		return home
	}
	return filepath.Join(home, filepath.FromSlash(strings.ReplaceAll(trimmed[2:], `\`, "/")))
}
