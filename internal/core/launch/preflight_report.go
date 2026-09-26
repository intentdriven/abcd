package launch

// preflight_report.go — the pre-flight report a preview and a cut both leave
// (itd-65 AC7, spc-2609201955279019 piece 5): one writer both callers share.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Report modes: which run wrote the report.
const (
	ModePreview = "preview"
	ModeCut     = "cut"
)

// Report verdicts.
const (
	VerdictClear   = "clear"
	VerdictRefused = "refused"
)

// preflightReportRelDir is where every report lands: the gitignored local
// tier's logs, one directory per run. It is never committed, and the dirty-tree
// gate never reads it as dirt.
const preflightReportRelDir = localTierPrefix + "logs/launch"

// PreflightReport is one run's pre-flight record: every gate, every refusal,
// every warning, and the dirty-tree override when one was used.
type PreflightReport struct {
	Mode     string        `json:"mode"`
	At       time.Time     `json:"at"`
	Version  string        `json:"version,omitempty"`
	Verdict  string        `json:"verdict"`
	Refusals []string      `json:"refusals"`
	Warnings []string      `json:"warnings"`
	Gates    []GateSummary `json:"gates"`
	// AllowDirty records that --allow-dirty waived the dirty-tree gate, and
	// Dirty every uncommitted path the gate saw.
	AllowDirty bool     `json:"allow_dirty"`
	Dirty      []string `json:"dirty,omitempty"`
	// Parity is the file-level diff against the previous release's payload,
	// and DeepSmoke the deep installability tier, when the run made them.
	Parity    *ParityReport    `json:"parity,omitempty"`
	DeepSmoke *DeepSmokeReport `json:"deep_smoke,omitempty"`
}

// PreflightReport is the preview's pre-flight record. The clock is the
// caller's: a report is a durable artefact, and a core that read the clock
// itself would put an unpinnable input inside it.
func (r DryRunReport) PreflightReport(at time.Time) PreflightReport {
	rep := newPreflightReport(ModePreview, at, r.Version, r.WouldRefuseOn, r.Warnings, r.Gates, false, nil)
	rep.Parity, rep.DeepSmoke = r.Parity, r.DeepSmoke
	return rep
}

// PreflightReport is the cut's pre-flight record, for the version the cut will
// carry (empty when the cut has not derived one yet).
func (p PayloadPrecheck) PreflightReport(at time.Time, version string) PreflightReport {
	rep := newPreflightReport(ModeCut, at, version, p.Refusals, p.Warnings, p.Gates, p.AllowDirty, p.Dirty)
	rep.Parity, rep.DeepSmoke = p.Parity, p.DeepSmoke
	return rep
}

func newPreflightReport(mode string, at time.Time, version string, refusals, warnings []string,
	gates []GateSummary, allowDirty bool, dirty []string) PreflightReport {
	rep := PreflightReport{
		Mode: mode, At: at.UTC(), Version: version, Verdict: VerdictClear,
		Refusals: append([]string{}, refusals...), Warnings: append([]string{}, warnings...),
		Gates: gates, AllowDirty: allowDirty, Dirty: dirty,
	}
	if len(rep.Refusals) > 0 {
		rep.Verdict = VerdictRefused
	}
	return rep
}

// WritePreflightReport writes rep as preflight.json and preflight.md into a
// directory of its own under .abcd/.work.local/logs/launch/, named for the
// report's instant, and returns that directory repo-relative. Two runs in one
// second get two directories: the directory is created exclusively, never
// reused.
func WritePreflightReport(repoRoot string, rep PreflightReport) (string, error) {
	if err := fsutil.EnsureRealDirAll(repoRoot, filepath.FromSlash(preflightReportRelDir), 0o755); err != nil {
		return "", fmt.Errorf("the pre-flight report directory: %w", err)
	}
	stamp := rep.At.UTC().Format("20060102T150405Z")
	rel := ""
	for n := 0; ; n++ {
		name := stamp
		if n > 0 {
			name = fmt.Sprintf("%s-%03d", stamp, n)
		}
		candidate := path.Join(preflightReportRelDir, name)
		err := os.Mkdir(filepath.Join(repoRoot, filepath.FromSlash(candidate)), 0o755)
		if err == nil {
			rel = candidate
			break
		}
		if !errors.Is(err, os.ErrExist) || n >= 999 {
			return "", fmt.Errorf("the pre-flight report directory: %w", err)
		}
	}
	dir := filepath.Join(repoRoot, filepath.FromSlash(rel))
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", err
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, "preflight.json"), append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, "preflight.md"), []byte(rep.Markdown()), 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

// Markdown renders the report for a person: the verdict, every refusal, every
// warning, the override when one was used, and the gate table.
func (rep PreflightReport) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Launch pre-flight report (%s)\n\n", rep.Mode)
	fmt.Fprintf(&b, "- at: %s\n", rep.At.UTC().Format(time.RFC3339))
	if rep.Version != "" {
		fmt.Fprintf(&b, "- version: %s\n", rep.Version)
	}
	fmt.Fprintf(&b, "- verdict: **%s**\n", rep.Verdict)
	if rep.AllowDirty {
		fmt.Fprintf(&b, "- --allow-dirty: the dirty-tree gate was waived for %d uncommitted path(s)\n", len(rep.Dirty))
	}
	section := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		fmt.Fprintf(&b, "\n## %s\n\n", title)
		for _, l := range lines {
			fmt.Fprintf(&b, "- %s\n", l)
		}
	}
	section("Refusals", rep.Refusals)
	section("Warnings", rep.Warnings)
	if rep.AllowDirty {
		section("Carried by --allow-dirty", rep.Dirty)
	}
	if rep.Parity != nil {
		b.WriteString(rep.Parity.Markdown())
	}
	b.WriteString("\n## Gates\n\n| Gate | Tier | Status | Detail |\n|---|---|---|---|\n")
	for _, g := range rep.Gates {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", g.Name, orDash(g.Tier), g.Status, strings.ReplaceAll(g.Detail, "|", "\\|"))
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
