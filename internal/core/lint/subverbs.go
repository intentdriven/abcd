package lint

// The sub-verb pass of surface_coverage (spc-27, adr-40 decision 6): each
// surface file under the registry's directory carries a `## Sub-verbs` table
// recording two facts per verb — which bucket it is (lint / review / audit /
// gate, or — for a non-assessment verb) and whether it exists (shipped /
// staged) — and every row is checked against the committed command-tree
// snapshot in both directions. Every surface file carries the table, the
// bare command's and the host-delegated surfaces' included: their exemption
// is from the cobra comparison only (itd-122). This is the detector that
// removes the surface-grain check's blindness inside a row (iss-246): a
// `staged` row cannot be cited as live elsewhere without failing the build,
// and a sub-verb cannot ship without a row.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// subVerbRow is one parsed row of a surface file's `## Sub-verbs` table.
type subVerbRow struct {
	verb   string // sub-command path below the top-level verb ("review ingest")
	bucket string // lint | review | audit | gate | — (non-assessment)
	status string // shipped | staged
	line   int
}

// subVerbBuckets is adr-40's closed bucket list plus the non-assessment dash.
// PR-to-extend: registered under Reserved vocabulary in 02-constraints/04-naming.md.
var subVerbBuckets = map[string]bool{"lint": true, "review": true, "audit": true, "gate": true, "—": true}

// surfaceFileRe maps a registry-sibling filename to its top-level verb:
// NN-<verb>.md.
var surfaceFileRe = regexp.MustCompile(`^[0-9]+-(.+)\.md$`)

// subVerbHeadingRe anchors the table: the exact `## Sub-verbs` heading.
var subVerbHeadingRe = regexp.MustCompile(`^##\s+Sub-verbs\s*$`)

// snapshotCommand is one command row of the release surface snapshot.
type snapshotCommand struct {
	Path   string `json:"path"`
	Hidden bool   `json:"hidden"`
}

// snapshotFile is the subset of surface.json this check reads.
type snapshotFile struct {
	Commands []snapshotCommand `json:"commands"`
}

// checkSubVerbCoverage runs the sub-verb pass. Unarmed (no Snapshot
// configured) it is inert; armed, a missing or unreadable snapshot is a loud
// finding, never a silent skip.
func checkSubVerbCoverage(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	if cfg.Snapshot == "" || cfg.Registry == "" {
		return nil, nil
	}
	regDir := filepath.ToSlash(filepath.Dir(cfg.Registry))

	subsByVerb, err := loadSnapshotSubVerbs(repoRoot, cfg.Snapshot)
	if err != nil {
		// Strip a *PathError to its bare cause: the finding's File already
		// names the repo-relative snapshot, and the raw error embeds the
		// absolute repo root — the iss-29/iss-76 leak — which the binary's
		// scrub boundaries never see on the findings path.
		detail := err.Error()
		var pe *os.PathError
		if errors.As(err, &pe) {
			detail = pe.Err.Error()
		}
		return []Finding{{
			File: cfg.Snapshot, Line: 1, RuleID: "surface_coverage", Severity: cfg.Severity,
			Message: "sub-verb pass is armed but the command-tree snapshot is unreadable (" + detail +
				") — regenerate with `go generate ./internal/surface/cli`",
		}}, nil
	}

	hostDelegated := stringSet(cfg.HostDelegated)
	operatorInternal := stringSet(cfg.OperatorInternal)

	var out []Finding
	seenVerbs := map[string]bool{}

	entries, err := os.ReadDir(filepath.Join(repoRoot, regDir))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		m := surfaceFileRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		verb := m[1]
		seenVerbs[verb] = true
		rel := regDir + "/" + name
		// The cobra comparison is skipped for the bare command's own file (its
		// "sub-verbs" are the top-level verbs the registry itself covers) and
		// for host-delegated surfaces (no Go verb by design). Their tables are
		// still format-checked.
		exemptFromCobra := verb == cfg.BareCommand || hostDelegated[verb]

		rows, tableFound, dupLine, shortRows, ferr := parseSubVerbTable(repoRoot, rel)
		if ferr != nil {
			return nil, ferr
		}
		if dupLine > 0 {
			out = append(out, Finding{
				File: rel, Line: dupLine, RuleID: "surface_coverage", Severity: cfg.Severity,
				Message: "duplicate '## Sub-verbs' heading — only the first table is checked, so a second one could carry an unchecked claim; merge them",
			})
		}
		for _, line := range shortRows {
			out = append(out, Finding{
				File: rel, Line: line, RuleID: "surface_coverage", Severity: cfg.Severity,
				Message: "sub-verb table row has fewer than three cells (want | Verb | Bucket | Status |) — a malformed row may not silently drop its fact",
			})
		}
		regs := subsByVerb[verb]
		if !tableFound {
			// Every surface file carries a table (itd-122 ac-1, ac-5): a verb
			// with no sub-command records that with a header-only table, and
			// the exemptions reach the cobra comparison only, never presence.
			msg := fmt.Sprintf("surface '%s' has no '## Sub-verbs' table — every surface file carries one, header-only where the verb has no sub-verb", verb)
			if len(regs) > 0 && !exemptFromCobra {
				msg = fmt.Sprintf("surface '%s' has %d registered sub-command(s) but no '## Sub-verbs' table — the sub-verb grain may not be waved through in prose", verb, len(regs))
			}
			out = append(out, Finding{
				File: rel, Line: 1, RuleID: "surface_coverage", Severity: cfg.Severity,
				Message: msg,
			})
			continue
		}

		regSet := stringSet(regs)
		rowSet := map[string]bool{}
		for _, r := range rows {
			rowSet[r.verb] = true
			if !subVerbBuckets[r.bucket] {
				out = append(out, Finding{
					File: rel, Line: r.line, RuleID: "surface_coverage", Severity: cfg.Severity,
					Message: "sub-verb row '" + r.verb + "' has unknown bucket '" + r.bucket + "' (want lint|review|audit|gate, or — for a non-assessment verb; the list is closed, PR-to-extend per adr-40)",
				})
			}
			switch r.status {
			case "shipped":
				if !exemptFromCobra && !regSet[r.verb] {
					out = append(out, Finding{
						File: rel, Line: r.line, RuleID: "surface_coverage", Severity: cfg.Severity,
						Message: "sub-verb row '" + r.verb + "' is marked shipped but the command tree registers no `abcd " + verb + " " + r.verb + "`",
					})
				}
			case "staged":
				if !exemptFromCobra && regSet[r.verb] {
					out = append(out, Finding{
						File: rel, Line: r.line, RuleID: "surface_coverage", Severity: cfg.Severity,
						Message: "sub-verb row '" + r.verb + "' is marked staged but `abcd " + verb + " " + r.verb + "` is registered; mark it shipped",
					})
				}
			default:
				out = append(out, Finding{
					File: rel, Line: r.line, RuleID: "surface_coverage", Severity: cfg.Severity,
					Message: "sub-verb row '" + r.verb + "' has unknown status '" + r.status + "' (want shipped|staged)",
				})
			}
		}
		if !exemptFromCobra {
			for _, reg := range regs {
				if !rowSet[reg] {
					out = append(out, Finding{
						File: rel, Line: 1, RuleID: "surface_coverage", Severity: cfg.Severity,
						Message: "registered sub-command `abcd " + verb + " " + reg + "` has no row in the '## Sub-verbs' table",
					})
				}
			}
		}
	}

	// Reverse sweep: a sub-command-bearing top-level verb cannot hide by
	// having no surface file at all. The bare command and operator-internal
	// verbs are the only exclusions, both explicit.
	verbs := make([]string, 0, len(subsByVerb))
	for v := range subsByVerb {
		verbs = append(verbs, v)
	}
	sort.Strings(verbs)
	for _, v := range verbs {
		if len(subsByVerb[v]) == 0 || seenVerbs[v] || v == cfg.BareCommand || operatorInternal[v] {
			continue
		}
		out = append(out, Finding{
			File: cfg.Registry, Line: 1, RuleID: "surface_coverage", Severity: cfg.Severity,
			Message: fmt.Sprintf("verb '%s' registers %d sub-command(s) but has no surface file under %s — add NN-%s.md with a '## Sub-verbs' table (or list it in operator_internal)", v, len(subsByVerb[v]), regDir, v),
		})
	}
	return out, nil
}

// loadSnapshotSubVerbs reads the committed command-tree snapshot into a
// top-level-verb → sub-command-path map. A hidden command excludes its whole
// subtree (operator plumbing is not product surface), and cobra's auto-added
// help/completion are excluded structurally.
func loadSnapshotSubVerbs(repoRoot, snapshot string) (map[string][]string, error) {
	data, err := readRepoFile(repoRoot, snapshot, maxRepoFileBytes)
	if err != nil {
		return nil, err
	}
	var snap snapshotFile
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parsing %s: %v", snapshot, err)
	}
	hidden := map[string]bool{}
	for _, c := range snap.Commands {
		if c.Hidden {
			hidden[c.Path] = true
		}
	}
	hiddenSubtree := func(path string) bool {
		parts := strings.Fields(path)
		for i := 1; i <= len(parts); i++ {
			if hidden[strings.Join(parts[:i], " ")] {
				return true
			}
		}
		return false
	}
	out := map[string][]string{}
	for _, c := range snap.Commands {
		parts := strings.Fields(c.Path)
		if len(parts) < 2 || hiddenSubtree(c.Path) {
			continue
		}
		top := parts[1]
		if top == "help" || top == "completion" {
			continue
		}
		if _, ok := out[top]; !ok {
			out[top] = []string{}
		}
		if len(parts) > 2 {
			if parts[2] == "help" || parts[2] == "completion" {
				continue
			}
			out[top] = append(out[top], strings.Join(parts[2:], " "))
		}
	}
	for top := range out {
		sort.Strings(out[top])
	}
	return out, nil
}

// parseSubVerbTable reads one surface file and returns the rows of its
// `## Sub-verbs` table (tableFound=false when the heading is absent) plus the
// line of any DUPLICATE unfenced heading — only the first is parsed, so a
// second one could carry an unchecked lying table and must be a finding.
// Fenced code blocks are masked so an example table is never read as the real
// one, mirroring parseSurfaceRegistry. shortRows collects the lines of rows
// with fewer than three cells: a silently dropped row would unrecord its fact.
func parseSubVerbTable(repoRoot, rel string) (rows []subVerbRow, tableFound bool, dupLine int, shortRows []int, err error) {
	content, err := readRepoFile(repoRoot, rel, maxRepoFileBytes)
	if err != nil {
		return nil, false, 0, nil, err
	}
	lines := strings.Split(string(content), "\n")
	fenced := fenceMask(lines)

	headingIdx := -1
	for i, ln := range lines {
		if fenced[i] || !subVerbHeadingRe.MatchString(ln) {
			continue
		}
		if headingIdx == -1 {
			headingIdx = i
			continue
		}
		dupLine = i + 1
		break
	}
	if headingIdx == -1 {
		return nil, false, 0, nil, nil
	}

	inTable := false
	for i := headingIdx + 1; i < len(lines); i++ {
		if fenced[i] {
			continue
		}
		ln := strings.TrimSpace(lines[i])
		if strings.HasPrefix(ln, "#") {
			break // next heading ends the section
		}
		if !strings.HasPrefix(ln, "|") {
			if inTable {
				break
			}
			continue
		}
		cells := splitTableRow(ln)
		if len(cells) > 0 {
			lower := strings.ToLower(cells[0])
			if lower == "verb" || strings.HasPrefix(cells[0], "---") || strings.HasPrefix(cells[0], ":-") {
				inTable = true
				continue
			}
		}
		// A data row with fewer than three cells cannot be silently dropped —
		// its fact would go unrecorded (spec: any malformed row is a finding).
		if len(cells) < 3 {
			shortRows = append(shortRows, i+1)
			continue
		}
		rows = append(rows, subVerbRow{
			verb:   strings.Trim(cells[0], "`"),
			bucket: cells[1],
			status: strings.ToLower(cells[2]),
			line:   i + 1,
		})
	}
	return rows, true, dupLine, shortRows, nil
}

// splitTableRow splits a markdown pipe-row into trimmed cells.
func splitTableRow(ln string) []string {
	trimmed := strings.Trim(ln, "|")
	parts := strings.Split(trimmed, "|")
	cells := make([]string, 0, len(parts))
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

// stringSet builds a membership set.
func stringSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, s := range items {
		set[s] = true
	}
	return set
}
