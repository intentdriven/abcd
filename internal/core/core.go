// Package core is abcd's transport-agnostic engine. Every capability is a
// function taking a structured request and returning a structured result;
// nothing here writes to stdout or knows about a CLI, MCP, or prompt surface.
// The front doors under internal/surface/* marshal these results for their
// transport. This separation is the single constraint that lets the CLI, the
// markdown plugin surface, and a future MCP server share one core.
package core

import (
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Version is abcd's version, stamped at build time via -ldflags -X (see the
// Makefile). It defaults to "dev" for un-stamped local builds.
var Version = "dev"

// VersionInfo is the result of NewVersion.
type VersionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewVersion reports abcd's identity and build version.
func NewVersion() VersionInfo {
	return VersionInfo{Name: "abcd", Version: Version}
}

// StatusInfo is the result of Status: a read-only "where am I" snapshot of a
// directory, mirroring abcd's bare-invocation status convention (never mutates).
type StatusInfo struct {
	// Dir is the inspected directory with the home redacted to "~": it travels
	// into --json, and machine output never carries an absolute
	// developer-identity path (iss-81, iss-2609261950066257). It is display
	// only; the inspection reads the directory itself.
	Dir       string   `json:"dir"`
	IsGitRepo bool     `json:"is_git_repo"`
	HasRecord bool     `json:"has_record"` // .abcd/development present
	WorkTiers []string `json:"work_tiers"` // which .abcd/ tiers exist
}

// Status inspects dir without mutating it and reports whether it is a git repo
// and which abcd surfaces are present.
func Status(dir string) (StatusInfo, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return StatusInfo{}, err
	}
	s := StatusInfo{
		Dir: fsutil.RedactHome(abs),
		// .git is a directory in a normal clone but a regular gitfile in a linked
		// worktree or submodule — both are genuine checkouts, so test existence, not
		// dir-ness. HasRecord/WorkTiers stay dir-only (those must be directories).
		IsGitRepo: exists(filepath.Join(abs, ".git")),
		HasRecord: isDir(filepath.Join(abs, ".abcd", "development")),
	}
	for _, tier := range []struct{ path, name string }{
		{filepath.Join(".abcd", "development"), "development"},
		{filepath.Join(".abcd", "work"), "work"},
		{filepath.Join(".abcd", ".work.local"), "work.local"},
	} {
		if isDir(filepath.Join(abs, tier.path)) {
			s.WorkTiers = append(s.WorkTiers, tier.name)
		}
	}
	return s, nil
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// exists reports whether p exists (a file or a directory). A .git gitfile in a
// worktree/submodule is a regular file, so a plain existence check is the correct
// "is this a git checkout" test.
func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
