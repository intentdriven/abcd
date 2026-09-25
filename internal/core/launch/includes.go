package launch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// includeConfigRelPath is the committed include-source config.
const includeConfigRelPath = ".abcd/config/launch-payload.json"

// PreflightError is a payload-preflight config fault: a missing/malformed
// include config, or an absolute / ".." / denied-rooted include. The caller
// writes NO manifest and reports the diagnostic (dry-run returns it as its only
// error case).
type PreflightError struct {
	msg string
	err error
}

func (e *PreflightError) Error() string { return e.msg }

// Unwrap exposes the sentinel a preflight fault carries, when it has one.
func (e *PreflightError) Unwrap() error { return e.err }

// ErrNoLaunchPayload reports that the repository declares no launch payload at
// all: it carries no include config. That is not a misconfiguration — a
// repository that ships no plugin bundle legitimately has none — so the front
// door recognises it and names the release path such a repository does have,
// rather than relaying a missing-file error (iss-2608270559313719).
var ErrNoLaunchPayload = errors.New("this repository declares no launch payload")

func preflight(format string, a ...any) error {
	return &PreflightError{msg: fmt.Sprintf(format, a...)}
}

// launchPayloadConfig is the consumed shape of launch-payload.json.
type launchPayloadConfig struct {
	Includes []string `json:"includes"`
}

// windowsDriveRe matches a Windows drive-absolute prefix (C:/ or C:\).
var windowsDriveRe = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// LoadIncludes reads .abcd/config/launch-payload.json, hand-validates its shape
// (no schema dependency — see §1.1), and returns the normalised, de-duplicated
// include patterns. A missing/malformed config, or an absolute / ".." /
// denied-rooted include, is a PreflightError.
func LoadIncludes(repoRoot string) ([]string, error) {
	raw, err := readIncludeConfig(repoRoot)
	if err != nil {
		return nil, err
	}
	// Hand-validate the container shape: a top-level object with a non-empty
	// includes array of non-empty strings. Reject anything else (no schema lib).
	rawIncludes, ok := raw["includes"]
	if !ok {
		return nil, preflight("include config missing required 'includes' key: %s", includeConfigRelPath)
	}
	var includes []string
	if err := json.Unmarshal(rawIncludes, &includes); err != nil {
		return nil, preflight("include config 'includes' is not an array of strings: %s", includeConfigRelPath)
	}
	if len(includes) == 0 {
		return nil, preflight("include config 'includes' must be a non-empty array: %s", includeConfigRelPath)
	}

	var patterns []string
	seen := map[string]struct{}{}
	for _, raw := range includes {
		if raw == "" {
			return nil, preflight("include pattern is empty")
		}
		norm := normalizeInclude(raw)
		if err := rejectBadInclude(norm); err != nil {
			return nil, err
		}
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		patterns = append(patterns, norm)
	}
	if len(patterns) == 0 {
		return nil, preflight("include config yielded no usable patterns: %s", includeConfigRelPath)
	}
	return patterns, nil
}

// readIncludeConfig reads the launch-payload config as a JSON object, the one
// reader both the include list and the gate policy go through.
func readIncludeConfig(repoRoot string) (map[string]json.RawMessage, error) {
	path := filepath.Join(repoRoot, includeConfigRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &PreflightError{msg: "include config not found: " + includeConfigRelPath, err: ErrNoLaunchPayload}
		}
		return nil, preflight("include config unreadable: %s: %v", includeConfigRelPath, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, preflight("include config is not a JSON object: %s: %v", includeConfigRelPath, err)
	}
	return raw, nil
}

// normalizeInclude strips a leading ./ and a trailing / and collapses to POSIX.
func normalizeInclude(pattern string) string {
	norm := pattern
	for strings.HasPrefix(norm, "./") {
		norm = norm[2:]
	}
	if len(norm) > 1 {
		norm = strings.TrimRight(norm, "/")
	}
	return norm
}

// rejectBadInclude rejects an absolute / backslash / ".." / denied-rooted
// include (literal OR targeted glob).
func rejectBadInclude(pattern string) error {
	if pattern == "" {
		return preflight("include pattern is empty")
	}
	if strings.Contains(pattern, "\\") {
		return preflight("include pattern contains a backslash (POSIX paths only): %q", pattern)
	}
	if strings.HasPrefix(pattern, "/") || filepath.IsAbs(pattern) {
		return preflight("include pattern is absolute: %q", pattern)
	}
	if windowsDriveRe.MatchString(pattern) {
		return preflight("include pattern is a Windows-drive absolute path: %q", pattern)
	}
	segments := strings.Split(pattern, "/")
	for _, seg := range segments {
		if seg == ".." {
			return preflight("include pattern contains a '..' traversal: %q", pattern)
		}
		// The structural deny binds EVERY component, case-insensitively: an
		// include that names a denied namespace at any depth (or in a case
		// variant) is refused up front, not left for the walk to prune
		// (GHSA-g2v7-wfmv-v28r, #335).
		if segmentDenied(seg) {
			return preflight("include pattern names a denied namespace segment %q: %q", seg, pattern)
		}
	}
	if firstSegmentGlobsDenied(segments[0]) {
		return preflight("include pattern's first segment could match a denied namespace: %q", pattern)
	}
	return nil
}

// nonDeniedSentinels distinguish a TARGETED denied-rooted glob from a BROAD root
// glob. A pure * / ** matches these too (so it is broad — allowed, pruned during
// the walk); a targeted glob like .a* matches a denied name but NOT these.
var nonDeniedSentinels = []string{"commands", "scripts", "README.md"}

// firstSegmentGlobsDenied reports whether a glob first segment is a TARGETED
// attempt to reach a denied namespace (matches a denied name but is not a broad
// root wildcard).
func firstSegmentGlobsDenied(first string) bool {
	if !hasGlobMeta(first) {
		return false
	}
	matchesDenied := false
	for denied := range DenyNamespaces {
		if ok, _ := filepath.Match(first, denied); ok {
			matchesDenied = true
			break
		}
	}
	if !matchesDenied {
		return false
	}
	for _, sentinel := range nonDeniedSentinels {
		if ok, _ := filepath.Match(first, sentinel); ok {
			return false // also matches a non-denied name → broad, not targeted
		}
	}
	return true
}

// ClosureFn returns the runtime-closure set (repo-relative POSIX paths) for the
// scripts/ include — the AST-reachable set the shipped plugin ships, never the
// whole dev tree. Since the Go payload layout is not yet settled, the default
// reads a pinned closure list from config rather than re-deriving via Python AST
// (spec §1 step 10 — the single open dependency). A nil map means no closure
// scoping is applied (scripts/ then behaves like any other include).
type ClosureFn func(repoRoot string) (map[string]struct{}, error)

// scriptsClosureRelPath is the pinned closure list the default ClosureFn reads.
const scriptsClosureRelPath = ".abcd/config/scripts-closure.json"

// defaultClosureFn reads the pinned closure list from config. When the file is
// absent it returns a nil map (no scoping) so a repo without a settled payload
// layout is not spuriously blocked.
func defaultClosureFn(repoRoot string) (map[string]struct{}, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, scriptsClosureRelPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg struct {
		Closure []string `json:"closure"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(cfg.Closure))
	for _, p := range cfg.Closure {
		set[filepath.ToSlash(p)] = struct{}{}
	}
	return set, nil
}
