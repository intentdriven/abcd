package ahoy

import (
	"path/filepath"
	"slices"

	"github.com/intentdriven/abcd/internal/core/shellpath"
	"github.com/intentdriven/abcd/internal/core/tools"
	"github.com/intentdriven/abcd/internal/fsutil"
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
	return slices.Contains(establishGapIDs, id) || id == UnrecordedGapID || id == LegacyPinGapID || id == ShellPathOfferGapID ||
		id == HooksArmGapID
}

// shellSettingsAsks is the line the plan lists for the shell settings
// question (itd-2610240919030303), or nil where stepShellPath will not ask it:
// the folder the run puts abcd in is already on the PATH, the person's
// settings already add it, or the file is one setup does not edit.
var shellSettingsAsks = func(cwd string, det DetectionResult, target string) []string {
	if target == "" {
		return nil
	}
	dir := filepath.Dir(target)
	if dirOnPath(dir) {
		return nil
	}
	plan := shellpath.PlanFor(shellEnv(), dir)
	if plan.State != shellpath.Offer {
		return nil
	}
	return []string{"the shell settings: one line in " + displayPath(plan.File)}
}

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
// answer to it, in the order the run asks them: the live brief session
// question, the abcd download, the shell settings, the commit checks, the status line, the commit identity, and one
// question for each tool abcd offers to install. Each is listed only where
// the run will put it.
func personOnlyAsks(cwd string, det DetectionResult, opts InstallOptions, target string) []string {
	var out []string
	present := gapIDSet(det.Gaps)
	// The live brief session question runs after the README pair and before
	// the download (stepSessions); --live-sessions answers it unasked.
	if present[SessionsOfferGapID] && opts.LiveSessions == "" {
		out = append(out, "the live brief session question: whether the team edits its brief live, and where sessions may run, kept in .abcd/config.json")
	}
	tag, dl := downloadOffer(cwd, det, opts, target)
	if dl {
		out = append(out, "the abcd download: abcd "+tag+" to "+displayPath(target))
	}
	// The shell question judges the entry the run leaves, so it is listed
	// only where there will be one.
	if entryWillExist(cwd, det, target, dl) {
		out = append(out, shellSettingsAsks(cwd, det, target)...)
	}
	if armOffered(cwd, det) {
		out = append(out, "the commit checks: core.hooksPath set to "+guardHooksDirRelPath+" in this clone's .git/config")
	}
	if present[StatusLineOfferGapID] && statusLineReachable(cwd, det, dl) {
		out = append(out, "the status line: one entry in your assistant's settings, "+displayPath(harnessSettingsPath()))
	}
	switch {
	case present[LegacyPinGapID]:
		out = append(out, "the commit identity: move the identity this repository names to its owner's own record on this machine")
	case present[UnrecordedGapID]:
		out = append(out, "the commit identity: record who you commit as, on this machine")
	}
	for _, id := range establishGapIDs {
		if present[id] {
			out = append(out, "the commit identity: who commits here, in this repository's .git/config")
			break
		}
	}
	// A tool is asked about only where abcd offers its own copy: a copy it
	// will use, or a tool it cannot install here, asks nothing.
	for _, g := range det.Gaps {
		if g.Category == Dependency && g.Tool != nil && g.Resolvable && g.Tool.Plan != nil && g.Tool.Plan.Action == tools.InstallOwn {
			out = append(out, "installing "+g.Tool.Tool+", a tool abcd uses")
		}
	}
	return out
}

// entryWillExist reports whether the run leaves abcd's PATH entry on disk:
// it is there now, the download question can put it there, or the plugin's
// verified cache supplies the copy.
func entryWillExist(cwd string, det DetectionResult, target string, downloadOffered bool) bool {
	if target == "" {
		return false
	}
	if present, err := fsutil.ExistsNoFollow(target); err == nil && present {
		return true
	}
	return downloadOffered || (det.pluginRoot != "" && cachePresent(pluginDataDir(det.pluginRoot).dir, cwd))
}
