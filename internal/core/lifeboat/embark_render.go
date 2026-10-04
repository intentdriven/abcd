package lifeboat

// embark_render.go carries the transport-agnostic human renders for the two
// embark results (mirroring PackResult.Render). Both build a string and never
// print — the surface writes it, or emits the struct as --json. Every string
// drawn from lifeboat content (source name, coverage questions, paths, marker
// notes) passes through sanitize so a hostile or archived lifeboat cannot spoof
// or corrupt the terminal report; the identity-path scrub is the surface's job
// (scrubPaths, applied to errors).

import (
	"fmt"
	"strings"
)

// Render is the human-readable `embark probe` report. The coverage BLANKS and
// their questions come FIRST — that is what a lifeboat hands a product thinker —
// then the write plan, the marker action, any conflicts as ONE bulk block, and
// the ignored/record-hash footer.
func (p EmbarkPlan) Render() string {
	var b strings.Builder
	verified := ""
	if p.ManifestVerified {
		verified = "  (lifeboat verified)"
	}
	fmt.Fprintf(&b, "embark probe: %s%s\n", sanitize(p.SourceName), verified)
	renderCoverageBlanks(&b, p.Coverage)
	renderPlanSummary(&b, p.TargetDir, p.Planned, p.Marker)
	renderConflictList(&b, p.Conflicts, "embark from would refuse and write nothing")
	renderIgnoredSummary(&b, p.Ignored)
	if p.RecordManifestSHA256 != "" {
		// Verbatim from the untrusted provenance (which manifest verification
		// deliberately excludes) — sanitise like every lifeboat-derived string.
		fmt.Fprintf(&b, "record manifest sha256: %s\n", sanitize(p.RecordManifestSHA256))
	}
	return b.String()
}

// Render is the human-readable `embark from` outcome. On the conflict-refusal
// path (Conflicts populated, nothing written) it renders the coverage blanks, the
// bulk conflict block, and the "nothing was written" line. On success it renders
// the coverage blanks first, then the write summary and the marker verb.
func (r EmbarkResult) Render() string {
	var b strings.Builder
	if len(r.Conflicts) > 0 {
		renderCoverageBlanks(&b, r.Coverage)
		renderConflictList(&b, r.Conflicts, "nothing was written")
		gap(&b)
		b.WriteString("nothing was written — resolve the conflicts and re-run\n")
		return b.String()
	}
	renderCoverageBlanks(&b, r.Coverage)
	gap(&b)
	fmt.Fprintf(&b, "embarked %s into %s\n", sanitize(r.SourceName), sanitize(r.TargetDir))
	fmt.Fprintf(&b, "  written:   %d%s\n", r.Written, familiesSuffix(r.Families))
	fmt.Fprintf(&b, "  unchanged: %d\n", r.Unchanged)
	if r.Marker.Target == "" {
		fmt.Fprintf(&b, "  marker:    %s\n", markerVerb(r.Marker.Action))
	} else {
		fmt.Fprintf(&b, "  marker:    %s %s\n", sanitize(r.Marker.Target), markerVerb(r.Marker.Action))
	}
	if r.Marker.Note != "" {
		fmt.Fprintf(&b, "  marker note: %s\n", sanitize(r.Marker.Note))
	}
	return b.String()
}

// gap writes a single blank-line separator, but only once content has begun — so
// the first block of a render never leads with a stray newline.
func gap(b *strings.Builder) {
	if b.Len() > 0 {
		b.WriteString("\n")
	}
}

// renderCoverageBlanks writes the blanks-first handoff: any pass the lifeboat
// declares exempt, then the unanswered brief sections and the questions a human
// must answer. A declared exemption prints whatever the coverage says, since it
// is a fact about the package rather than about coverage.json; past it, an
// absent coverage prints nothing, a degraded one prints a one-line note, and a
// present one with no blanks prints nothing (there is nothing to answer).
func renderCoverageBlanks(b *strings.Builder, cov *CoverageHandoff) {
	if cov == nil {
		return
	}
	// A declared exemption comes before the blanks and outlives them: it says
	// which pass did not run, so a section it would have grounded reads as a
	// pass that was never there rather than as work a human has been left.
	if ex := cov.PassBExemption; ex != nil {
		gap(b)
		fmt.Fprintf(b, "pass B (transcripts): declared exempt — %s\n", sanitize(ex.Reason))
	}
	if !cov.Present {
		return
	}
	if cov.Degraded {
		gap(b)
		fmt.Fprintf(b, "coverage: %s\n", sanitize(cov.Note))
		return
	}
	if len(cov.Blanks) == 0 {
		return
	}
	total := cov.Summary.Grounded + cov.Summary.Partial + cov.Summary.Blank
	gap(b)
	fmt.Fprintf(b, "coverage blanks — answer these first (%d of %d sections):\n", len(cov.Blanks), total)
	for _, bl := range cov.Blanks {
		line := "  ? " + sanitize(string(bl.Section))
		if bl.Kind == KindHumanOwned {
			line += "   (human-owned — yours to write)"
		}
		b.WriteString(line + "\n")
		if bl.Question != "" {
			fmt.Fprintf(b, "      %s\n", sanitize(bl.Question))
		}
	}
}

// renderPlanSummary writes the "would write N files" block: the create count, a
// per-family breakdown in the canonical family order, and the marker action.
func renderPlanSummary(b *strings.Builder, targetDir string, planned []PlannedEmbark, marker MarkerResult) {
	creates := 0
	for _, p := range planned {
		if p.Action == ActionCreate {
			creates++
		}
	}
	gap(b)
	fmt.Fprintf(b, "would write %d file(s) into %s:\n", creates, sanitize(targetDir))
	for _, fam := range embarkFamilies {
		c, u := 0, 0
		for _, p := range planned {
			if p.Family != fam.Name {
				continue
			}
			switch p.Action {
			case ActionCreate:
				c++
			case ActionUnchanged:
				u++
			}
		}
		if c > 0 {
			fmt.Fprintf(b, "    %-8s %d create\n", fam.Name, c)
		}
		if u > 0 {
			fmt.Fprintf(b, "    %-8s %d unchanged\n", fam.Name, u)
		}
	}
	if marker.Target == "" {
		fmt.Fprintf(b, "  marker: %s\n", marker.Action)
	} else {
		fmt.Fprintf(b, "  marker: %s → %s\n", sanitize(marker.Target), marker.Action)
	}
	if marker.Note != "" {
		fmt.Fprintf(b, "    note: %s\n", sanitize(marker.Note))
	}
}

// renderConflictList writes the single bulk conflict block (never a per-file
// barrage). note is the parenthetical/heading qualifier (differs between the
// probe prediction and the from refusal).
func renderConflictList(b *strings.Builder, conflicts []Conflict, note string) {
	if len(conflicts) == 0 {
		return
	}
	gap(b)
	fmt.Fprintf(b, "%d conflict(s) (%s):\n", len(conflicts), note)
	for _, c := range conflicts {
		fmt.Fprintf(b, "  • %s  (%s)\n", sanitize(c.Path), c.Kind)
	}
}

// renderIgnoredSummary writes the one-line tally of files not embarked.
func renderIgnoredSummary(b *strings.Builder, ignored []IgnoredFile) {
	var ro, um, uk int
	for _, ig := range ignored {
		switch ig.Reason {
		case IgnoredReportOnly:
			ro++
		case IgnoredUnmapped:
			um++
		case IgnoredUnknown:
			uk++
		}
	}
	gap(b)
	fmt.Fprintf(b, "ignored (not embarked): %d report-only, %d unmapped, %d unknown\n", ro, um, uk)
}

// familiesSuffix renders "  (adrs 1, issues 3, specs 1)" in canonical family
// order, or "" when nothing was written.
func familiesSuffix(families map[string]int) string {
	if len(families) == 0 {
		return ""
	}
	var parts []string
	for _, fam := range embarkFamilies {
		if n := families[fam.Name]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", fam.Name, n))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "  (" + strings.Join(parts, ", ") + ")"
}

// markerVerb renders the marker action as a past-tense outcome for the from
// summary.
func markerVerb(a MarkerAction) string {
	switch a {
	case MarkerActionInstall:
		return "installed"
	case MarkerActionRefresh:
		return "refreshed"
	case MarkerActionCurrent:
		return "already current"
	case MarkerActionSkip:
		return "skipped"
	}
	return string(a)
}
