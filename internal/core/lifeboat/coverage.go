package lifeboat

import (
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// Coverage is one repository's probe result: which brief sections a lifeboat
// could ground from it, at what confidence, citing what. It is the M2
// experiment's per-repo readout, and it aggregates across repositories (see
// Aggregate) — that cross-repo table is what answers "is the brief structure
// sound."
type Coverage struct {
	SchemaVersion int               `json:"schema_version"`
	Repo          RepoInfo          `json:"repo"`
	TiersPresent  []Tier            `json:"tiers_present"`
	Sections      []SectionCoverage `json:"sections"`
	Summary       Summary           `json:"summary"`

	// IncludedIgnored records that this probe read files git ignores. Only the
	// WIDE scan is stated: the narrow one is the default and every existing
	// report already means it, so omitempty keeps a default report byte-identical
	// and no schema bump is owed.
	//
	// It is stated because scope is not visible in the result. A section that
	// came back blank looks the same either way, and a reader of a packed
	// lifeboat cannot otherwise tell "nothing there" from "not looked at".
	IncludedIgnored bool `json:"included_ignored,omitempty"`
}

// RepoInfo identifies the probed repository.
type RepoInfo struct {
	Name    string `json:"name"`
	RootSHA string `json:"root_sha,omitempty"`
	Commits int    `json:"commits"`
}

// SectionCoverage is the probe result for one brief section, and — from schema
// v2 (adr-36) — a fillable object across the disembark→embark round-trip. A
// grounded or partial row cites evidence; a blank row carries what was searched,
// the question a human must answer, its Kind (extractable coverage debt vs a
// human-owned prompt), a Resolution that tracks whether it has been answered or
// deferred, and, once answered, an authored Answer whose provenance is a person
// and a date — never a file it did not come from.
type SectionCoverage struct {
	Name       Section    `json:"name"`
	Kind       Kind       `json:"kind"`
	Status     Status     `json:"status"`
	Confidence Confidence `json:"confidence,omitempty"`
	Tier       Tier       `json:"tier,omitempty"`
	Evidence   []string   `json:"evidence,omitempty"`
	Searched   []string   `json:"searched,omitempty"`
	Question   string     `json:"question,omitempty"`
	// Reason says why a partial row is partial: what the probe found and what
	// would ground the section. It is set on every partial and on nothing else;
	// omitempty keeps a grounded or blank row byte-identical, so no schema bump
	// is owed.
	Reason string `json:"reason,omitempty"`
	// Resolution tracks a blank's fill state across the round-trip. It is
	// meaningful only for a blank; a grounded/partial section leaves it empty.
	Resolution Resolution `json:"resolution,omitempty"`
	// Answer is present only once a human has answered the blank. Its provenance
	// is authored-by, structurally distinct from an extracted citation.
	Answer *Answer `json:"answer,omitempty"`
}

// Resolution is the fill state of a blank as it travels with the lifeboat.
type Resolution string

const (
	// ResolutionOpen is a blank that has not been answered yet — the default the
	// probe stamps on every blank.
	ResolutionOpen Resolution = "open"
	// ResolutionAnswered is a blank a human has answered; Answer is set.
	ResolutionAnswered Resolution = "answered"
	// ResolutionDeferred is a blank a human chose not to answer now; it
	// re-surfaces at embark rather than being dropped.
	ResolutionDeferred Resolution = "deferred"
)

// Answer is a human's authored answer to a blank. Its provenance is a person and
// a date — an honest non-file source, per adr-36 — never a file the answer did
// not come from.
type Answer struct {
	Text          string `json:"text"`
	AuthoredBy    string `json:"authored_by"`
	AuthoredAt    string `json:"authored_at"`
	AgentAssisted bool   `json:"agent_assisted,omitempty"`
}

// Summary counts sections by status. Blank is counted, not hidden — a blank is
// a result.
type Summary struct {
	Grounded int `json:"grounded"`
	Partial  int `json:"partial"`
	Blank    int `json:"blank"`
}

// Render returns the human-readable per-repo coverage report.
func (c Coverage) Render() string {
	var b strings.Builder
	tiers := make([]string, len(c.TiersPresent))
	for i, t := range c.TiersPresent {
		tiers[i] = sanitize(string(t))
	}
	fmt.Fprintf(&b, "coverage for %s", sanitize(c.Repo.Name))
	if c.Repo.Commits > 0 {
		fmt.Fprintf(&b, " (%d commits)", c.Repo.Commits)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "tiers present: %s\n", tiersOrNone(tiers))
	if c.IncludedIgnored {
		b.WriteString("scope: WIDE — files git ignores were read\n")
	}
	fmt.Fprintf(&b, "grounded %d · partial %d · blank %d  (of %d sections)\n\n",
		c.Summary.Grounded, c.Summary.Partial, c.Summary.Blank, len(c.Sections))

	for _, s := range c.Sections {
		mark := statusGlyph(s.Status)
		fmt.Fprintf(&b, "%s %-32s %s", mark, sanitize(string(s.Name)), sanitize(string(s.Status)))
		if s.Confidence != "" {
			fmt.Fprintf(&b, " (%s, %s)", sanitize(string(s.Tier)), sanitize(string(s.Confidence)))
		}
		b.WriteString("\n")
		if len(s.Evidence) > 0 {
			fmt.Fprintf(&b, "    evidence: %s\n", strings.Join(sanitizeAll(s.Evidence), ", "))
		}
		if s.Status == StatusPartial && s.Reason != "" {
			fmt.Fprintf(&b, "    why partial: %s\n", sanitize(s.Reason))
		}
		if s.Status == StatusBlank {
			if s.Kind == KindHumanOwned {
				fmt.Fprintf(&b, "    (human-owned — yours to write, not an extraction)\n")
			}
			if len(s.Searched) > 0 {
				fmt.Fprintf(&b, "    searched: %s\n", strings.Join(sanitizeAll(s.Searched), ", "))
			}
			if s.Question != "" {
				fmt.Fprintf(&b, "    ? %s\n", sanitize(s.Question))
			}
		}
	}
	return b.String()
}

func statusGlyph(s Status) string {
	switch s {
	case StatusGrounded:
		return "+"
	case StatusPartial:
		return "~"
	default:
		return "-"
	}
}

func tiersOrNone(t []string) string {
	if len(t) == 0 {
		return "(none)"
	}
	return strings.Join(t, ", ")
}

// sanitize strips terminal-display attack characters from repository-derived text
// (evidence, searched entries, questions, the repo name — all commit-subject/path/
// ref content a hostile or archived repo controls) before it reaches the human
// report. It is the canonical termsafe primitive. The JSON output is NOT
// covered by encoding/json's own escaping (only C0 and U+2028/9 are escaped;
// DEL, C1, bidi and zero-width runes pass raw — see termsafe.Sanitize's note).
func sanitize(s string) string { return termsafe.Sanitize(s) }

// sanitizeAll sanitizes every member of a slice.
func sanitizeAll(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = sanitize(s)
	}
	return out
}

// AggregateReport is the cross-repo readout: one row per brief section, one
// column per probed repository, each cell the section's status in that repo.
// This is the artefact the M2 gate reads to decide which brief sections survive.
type AggregateReport struct {
	SchemaVersion int              `json:"schema_version"`
	Repos         []AggregateRepo  `json:"repos"`
	Sections      []AggregateRow   `json:"sections"`
	Verdict       []SectionVerdict `json:"verdict"`
}

// AggregateRepo is a probed repository's identity in the aggregate header.
type AggregateRepo struct {
	Name         string `json:"name"`
	Commits      int    `json:"commits"`
	TiersPresent []Tier `json:"tiers_present"`
}

// AggregateRow is one brief section across every repository.
type AggregateRow struct {
	Section Section           `json:"section"`
	Cells   map[string]Status `json:"cells"` // repo name -> status
}

// SectionVerdict summarises a section across the corpus: how the record-rich
// repos fared versus the record-poor ones. It is the quantified answer to "what
// is the record worth" for that section.
type SectionVerdict struct {
	Section       Section `json:"section"`
	GroundedCount int     `json:"grounded_count"`
	PartialCount  int     `json:"partial_count"`
	BlankCount    int     `json:"blank_count"`
	// AlwaysBlank is true when no probed repo grounded or partially grounded the
	// section — evidence the section may not be derivable from a repository at
	// all, and belongs to a human rather than an extraction.
	AlwaysBlank bool `json:"always_blank"`
}

// Aggregate reduces per-repo coverage reports to the cross-repo table. Repos
// keep the order given; sections keep the mapping's canonical order.
func Aggregate(covs []Coverage) AggregateReport {
	repos := make([]AggregateRepo, 0, len(covs))
	names := make([]string, 0, len(covs))
	seen := map[string]bool{}
	for _, c := range covs {
		// Disambiguate a duplicate repo name so its column is not overwritten,
		// probing past a suffix that itself collides with another repo's real
		// name (e.g. "foo", "foo#2", "foo") — the cells map is keyed by name, so
		// any collision would silently drop a column.
		name := sanitize(c.Repo.Name)
		if seen[name] {
			for n := 2; ; n++ {
				cand := fmt.Sprintf("%s#%d", name, n)
				if !seen[cand] {
					name = cand
					break
				}
			}
		}
		seen[name] = true
		repos = append(repos, AggregateRepo{
			Name: name, Commits: c.Repo.Commits, TiersPresent: c.TiersPresent,
		})
		names = append(names, name)
	}

	rows := make([]AggregateRow, 0, len(Table))
	verdicts := make([]SectionVerdict, 0, len(Table))
	for _, m := range Table {
		cells := map[string]Status{}
		v := SectionVerdict{Section: m.Section}
		for i, c := range covs {
			st := statusOf(c, m.Section)
			cells[names[i]] = st
			switch st {
			case StatusGrounded:
				v.GroundedCount++
			case StatusPartial:
				v.PartialCount++
			default:
				v.BlankCount++
			}
		}
		v.AlwaysBlank = len(covs) > 0 && v.GroundedCount == 0 && v.PartialCount == 0
		rows = append(rows, AggregateRow{Section: m.Section, Cells: cells})
		verdicts = append(verdicts, v)
	}
	return AggregateReport{
		SchemaVersion: SchemaVersion,
		Repos:         repos,
		Sections:      rows,
		Verdict:       verdicts,
	}
}

func statusOf(c Coverage, section Section) Status {
	for _, s := range c.Sections {
		if s.Name == section {
			return s.Status
		}
	}
	return StatusBlank
}

// Render returns the human-readable cross-repo aggregate table.
func (a AggregateReport) Render() string {
	var b strings.Builder
	b.WriteString("cross-repo brief coverage\n\n")
	// A coverage report is decoded from an untrusted cross-repo file, so every
	// repo-derived string (name, tier, section label, cell status) is sanitised
	// before it reaches the terminal; the raw r.Name is still the cell-map key.
	repoNames := make([]string, len(a.Repos))
	for i, r := range a.Repos {
		repoNames[i] = sanitize(r.Name)
	}
	for i, r := range a.Repos {
		tiers := make([]string, len(r.TiersPresent))
		for j, t := range r.TiersPresent {
			tiers[j] = sanitize(string(t))
		}
		fmt.Fprintf(&b, "  %s — %d commits, tiers: %s\n", repoNames[i], r.Commits, tiersOrNone(tiers))
	}
	b.WriteString("\n")

	sectionLabels := make([]string, len(a.Sections))
	for i, row := range a.Sections {
		sectionLabels[i] = sanitize(string(row.Section))
	}

	// Column widths, computed from the sanitised strings so alignment holds.
	nameW := len("brief section")
	for _, label := range sectionLabels {
		if len(label) > nameW {
			nameW = len(label)
		}
	}
	colW := make([]int, len(a.Repos))
	for i := range a.Repos {
		colW[i] = len(repoNames[i])
		if colW[i] < 8 {
			colW[i] = 8
		}
	}

	fmt.Fprintf(&b, "%-*s", nameW, "brief section")
	for i := range a.Repos {
		fmt.Fprintf(&b, "  %-*s", colW[i], repoNames[i])
	}
	b.WriteString("  verdict\n")

	for ri, row := range a.Sections {
		fmt.Fprintf(&b, "%-*s", nameW, sectionLabels[ri])
		for i, r := range a.Repos {
			fmt.Fprintf(&b, "  %-*s", colW[i], sanitize(string(row.Cells[r.Name])))
		}
		v := a.Verdict[ri]
		if v.AlwaysBlank {
			b.WriteString("  always-blank")
		}
		b.WriteString("\n")
	}

	alwaysBlank := 0
	for _, v := range a.Verdict {
		if v.AlwaysBlank {
			alwaysBlank++
		}
	}
	fmt.Fprintf(&b, "\n%d of %d sections are blank in every probed repo.\n", alwaysBlank, len(a.Sections))
	return b.String()
}

// AlwaysBlankSections returns the sections no probed repo could ground, in the
// mapping's canonical order.
func (a AggregateReport) AlwaysBlankSections() []Section {
	var out []Section
	for _, v := range a.Verdict {
		if v.AlwaysBlank {
			out = append(out, v.Section)
		}
	}
	return out
}
