package ahoy

import (
	"os"
	"path/filepath"
)

// UpdateTargetKind classifies the PATH entry `abcd update` would act on. The
// update verb's dispatch (spc-32) keys on this: only a regular file whose
// provenance the verb can establish is ever swapped; every other shape is a
// named refusal, because another mechanism already owns it.
type UpdateTargetKind string

const (
	// UpdateTargetAbsent: no `abcd` anywhere on PATH.
	UpdateTargetAbsent UpdateTargetKind = "absent"
	// UpdateTargetPluginRoot: an owned symlink resolving into the current
	// plugin root — the plugin update owns that binary (itd-108 one-cut
	// coherence), never the update verb.
	UpdateTargetPluginRoot UpdateTargetKind = "plugin-root"
	// UpdateTargetDevShim: the track-latest dev shim — a chosen install mode.
	UpdateTargetDevShim UpdateTargetKind = "dev-shim"
	// UpdateTargetDangling: an abcd-owned entry whose binary is gone (the
	// iss-345 stranded shape included) — `ahoy install` is the heal.
	UpdateTargetDangling UpdateTargetKind = "owned-dangling"
	// UpdateTargetSuperseded: an owned symlink into a sibling plugin-cache
	// vintage the harness has moved past — the binary still exists, so it is
	// not stranded, but it answers an older release than the plugin now holds
	// (iss-2609161805447092). `ahoy install` is the heal.
	UpdateTargetSuperseded UpdateTargetKind = "owned-superseded"
	// UpdateTargetFile: a regular file. Ownership is decided by the update
	// verb's provenance check (digest against a published release), not here.
	UpdateTargetFile UpdateTargetKind = "file"
	// UpdateTargetForeign: a symlink abcd does not own, or an entry that
	// cannot be classified. Never touched.
	UpdateTargetForeign UpdateTargetKind = "foreign"
)

// UpdateTarget names the first `abcd` PATH occupant — the binary that actually
// answers when the user types `abcd` — classified by the same ownership
// predicate detection and install use (one canonical predicate, iss-345).
type UpdateTarget struct {
	Path         string           // first `abcd` on PATH ("" when absent)
	ResolvedPath string           // Path with symlinked ancestors/leaf resolved
	Kind         UpdateTargetKind //
	LaterOwned   string           // an abcd-owned entry shadowed behind Path ("" when none)
}

// ResolveUpdateTarget classifies the entry `abcd update` would act on. It
// walks PATH in order — the dispatch is keyed on what actually runs, not on
// one blessed location — and, when PATH holds nothing, reports the plugin
// root as the target when the running executable lives there (the
// plugin-invocation case still deserves the plugin-root refusal, not a
// "nothing installed" one).
func ResolveUpdateTarget() UpdateTarget {
	pluginRoot, rootOK := resolvePluginRoot()
	entries := scanPathEntries(pluginRoot)
	if len(entries) == 0 {
		if rootOK {
			if exe, err := osExecutable(); err == nil {
				if filepath.Dir(resolvePath(exe)) == resolvePath(pluginRoot) {
					return UpdateTarget{
						Path:         pluginBinaryPath(pluginRoot),
						ResolvedPath: resolvePath(pluginBinaryPath(pluginRoot)),
						Kind:         UpdateTargetPluginRoot,
					}
				}
			}
		}
		return UpdateTarget{Kind: UpdateTargetAbsent}
	}

	first := entries[0]
	tgt := UpdateTarget{Path: first.path, ResolvedPath: resolvePath(first.path)}
	for _, e := range entries[1:] {
		if e.owned() && !e.dangling {
			tgt.LaterOwned = e.path
			break
		}
	}

	switch {
	case first.kind == binTargetDevShim:
		tgt.Kind = UpdateTargetDevShim
	case first.owned() && first.dangling:
		tgt.Kind = UpdateTargetDangling
	case first.kind == binTargetOwnedSymlink && first.superseded:
		tgt.Kind = UpdateTargetSuperseded
	case first.kind == binTargetOwnedSymlink:
		tgt.Kind = UpdateTargetPluginRoot
	default:
		if fi, err := os.Lstat(first.path); err == nil && fi.Mode().IsRegular() {
			tgt.Kind = UpdateTargetFile
		} else {
			tgt.Kind = UpdateTargetForeign
		}
	}
	return tgt
}
