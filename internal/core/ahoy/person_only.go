package ahoy

import (
	"path/filepath"
	"slices"
)

// The steps only a person can approve (itd-2610240823055889, criterion 3):
// the commit identity, a download, the assistant's settings and the shell
// settings. Each is asked as its own question naming the value and where it
// is written, whatever the answer to the plan or to an approval of a kind of
// change, so none is ever listed under, or declined by, a generic approval.

// personOnlyGap reports whether the gap id is closed by a question of its own
// rather than by the approval of its category: the commit identity's gaps,
// which stepGitIdentity asks about at a terminal.
func personOnlyGap(id string) bool {
	return slices.Contains(establishGapIDs, id)
}

// shellSettingsAsks is the line the plan lists for the shell settings
// question, or nil where it is not asked. The shell-profile PATH line
// (itd-2610240919030303) fills it, with stepShellSettings; until then setup
// asks nothing about the shell.
var shellSettingsAsks = func(cwd string, det DetectionResult, target string) []string { return nil }

// stepShellSettings is the shell settings question's place in the run: after
// the PATH entry is written and before the status line is offered, asked as
// its own question naming the line and the profile it is written to. The
// shell-profile PATH line (itd-2610240919030303) fills it; until then the
// install says the directory is not on PATH and leaves the profile alone.
var stepShellSettingsHook = func(a *applyCtx) {}

func (a *applyCtx) stepShellSettings() { stepShellSettingsHook(a) }

// plannedBinTarget is the PATH entry the run will act on, as the plan lists
// it before the run resolves it: the --bin-dir entry, or the entry an owned
// install already holds, else the default. It probes nothing.
func plannedBinTarget(opts InstallOptions, det DetectionResult) string {
	if opts.BinDir != "" {
		return filepath.Join(opts.BinDir, binName)
	}
	return effectiveBinTarget(det.pluginRoot)
}

// statusLineReachable reports whether the status line can be wired in this
// run: abcd's PATH entry is on record and passes the checks the wiring makes,
// or the download question can put it there first.
func statusLineReachable(cwd string, det DetectionResult, downloadOffered bool) bool {
	if downloadOffered {
		return true
	}
	// The plugin's verified cache supplies the PATH copy earlier in this run.
	if det.pluginRoot != "" && cachePresent(pluginDataDir(det.pluginRoot).dir, cwd) {
		return true
	}
	entry, _ := (&applyCtx{det: det}).statusLineEntry()
	return entry != ""
}

// approvalGaps is the gaps the approvals are asked over: the status-line
// offer is left out where the status line cannot be wired, so it is never
// asked about only to be refused after the answers (iss-2610240823574047).
func approvalGaps(cwd string, det DetectionResult, opts InstallOptions, target string) []Gap {
	_, dl := downloadOffer(cwd, det, opts, target)
	if statusLineReachable(cwd, det, dl) {
		return det.Gaps
	}
	return slices.DeleteFunc(slices.Clone(det.Gaps), func(g Gap) bool { return g.ID == StatusLineOfferGapID })
}

// personOnlyAsks is the plan's list of the questions that follow whatever the
// answer to it, in the order the run asks them: the abcd download, the shell
// settings, the status line, the commit identity, and one question for each
// missing tool.
func personOnlyAsks(cwd string, det DetectionResult, opts InstallOptions, target string) []string {
	var out []string
	tag, dl := downloadOffer(cwd, det, opts, target)
	if dl {
		out = append(out, "the abcd download: abcd "+tag+" to "+displayPath(target))
	}
	out = append(out, shellSettingsAsks(cwd, det, target)...)
	present := gapIDSet(det.Gaps)
	if present[StatusLineOfferGapID] && statusLineReachable(cwd, det, dl) {
		out = append(out, "the status line: one entry in your assistant's settings, "+displayPath(harnessSettingsPath()))
	}
	for _, id := range establishGapIDs {
		if present[id] {
			out = append(out, "the commit identity: who commits here, in this repository's .git/config")
			break
		}
	}
	for _, g := range det.Gaps {
		if g.Category == Dependency && g.Tool != nil && !onPath(cwd, g.Tool.Tool) {
			out = append(out, "installing "+g.Tool.Tool+", a tool abcd uses")
		}
	}
	return out
}
