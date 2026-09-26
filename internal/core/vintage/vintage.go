// Package vintage is abcd's staleness comparator: it establishes whether the
// running binary is the one that should be running, by comparing the binary's
// own build vintage against a reference supplied by a provider. It is
// detection-only — it never writes, never heals, and (with the one network
// provider excepted) never touches the network. The comparator's outcome is
// consumed by every surface that reports vintage; the surfaces render, this
// package decides.
//
// Unknown is a first-class outcome. Go's VCS stamping has known holes (go run /
// go test binaries, -buildvcs=false, .git-less builds, `go install
// module@version` history, and dirty vcs.modified rebuilds), so a binary whose
// vintage cannot be determined is reported unknown — never silently treated as
// fresh.
package vintage

import (
	"debug/buildinfo"
	"io"
	"runtime/debug"
)

// Outcome is the comparator's verdict.
type Outcome string

const (
	// Fresh means the running binary matches the reference.
	Fresh Outcome = "fresh"
	// Stale means it does not — an older (or otherwise differing) vintage.
	Stale Outcome = "stale"
	// Unknown means the vintage could not be determined on either side. It is
	// never mapped to Fresh: an undeterminable binary is exactly the state that
	// hides a stale one.
	Unknown Outcome = "unknown"
)

// Current is the running binary's vintage, read from its build metadata.
type Current struct {
	// Revision is the identifier the comparison keys on: the embedded VCS
	// revision for a source-built binary, or the stamped release version for a
	// pinned one. Empty when nothing could be read.
	Revision string
	// Known is false when the vintage cannot be determined and so must never be
	// reported as fresh: an unstamped build (go run, go test, -buildvcs=false, a
	// .git-less build, or `go install module@version` history all leave no
	// vcs.revision) or a dirty rebuild (vcs.modified == true).
	Known bool
}

// Expected is what a provider says should be running.
type Expected struct {
	// Revision is the reference the current vintage is compared against.
	Revision string
	// Known is false when the reference itself could not be resolved (an
	// unreadable checkout tip, an absent manifest pin). A comparison against an
	// unresolved reference is Unknown, never Fresh.
	Known bool
}

// Provider supplies the expected vintage. The disk providers in this cut read
// the source checkout tip and the plugin-cache manifest pin; a network provider
// is a later drop-in behind this same one-method interface. The interface is
// the seam the fit-challenge required: real swappability, not a fixed-arity
// comparator hard-wired to three disk sources.
type Provider interface {
	Expected() Expected
}

// Report is one comparison's result. It carries the values the surfaces render
// (the notice names the tip a binary trails; the version render shows both).
type Report struct {
	Outcome  Outcome
	Current  string
	Expected string
}

// Compare is the single comparator every surface consumes. Unknown is
// first-class: an undeterminable current OR an unresolved expected yields
// Unknown, never Fresh.
func Compare(cur Current, p Provider) Report {
	if !cur.Known {
		return Report{Outcome: Unknown, Current: cur.Revision}
	}
	exp := p.Expected()
	if !exp.Known {
		return Report{Outcome: Unknown, Current: cur.Revision}
	}
	if cur.Revision == exp.Revision {
		return Report{Outcome: Fresh, Current: cur.Revision, Expected: exp.Revision}
	}
	return Report{Outcome: Stale, Current: cur.Revision, Expected: exp.Revision}
}

// CurrentBuildVintage reads the running binary's embedded build metadata.
func CurrentBuildVintage() Current {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return currentFromSettings(false, nil)
	}
	return currentFromSettings(true, bi.Settings)
}

// OfReader reads the vintage a binary on disk was built at from its embedded
// build metadata, without executing it. The same unknown triggers apply as for
// the running binary: no vcs.revision, or a dirty rebuild, is not Known. An
// error means r is not a Go binary whose build metadata can be read at all.
func OfReader(r io.ReaderAt) (Current, error) {
	bi, err := buildinfo.Read(r)
	if err != nil {
		return Current{}, err
	}
	return currentFromSettings(true, bi.Settings), nil
}

// currentFromSettings is the pure core of CurrentBuildVintage, split out so the
// four unknown triggers can be exercised without rebuilding the test binary.
func currentFromSettings(ok bool, settings []debug.BuildSetting) Current {
	if !ok {
		return Current{Known: false}
	}
	var rev string
	var modified, haveModified bool
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			haveModified = true
			modified = s.Value == "true"
		}
	}
	// No revision at all — go run, go test, -buildvcs=false, a .git-less build,
	// or `go install module@version` history — cannot be located against a tip.
	if rev == "" {
		return Current{Known: false}
	}
	// A dirty rebuild carries the last commit's revision but working-tree changes
	// on top, so that revision no longer identifies what is actually running.
	if haveModified && modified {
		return Current{Revision: rev, Known: false}
	}
	return Current{Revision: rev, Known: true}
}
