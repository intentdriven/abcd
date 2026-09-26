package site

// record.json — the whole development record as one machine-readable file.
//
// It is a pure BUILD ARTIFACT and is never committed (spc-38): production
// cannot drift from the tree because the tree is what renders it. Determinism is
// therefore the property that has to hold — sorted inputs everywhere, fixed
// seeds, and no clock reading beyond the build stamp the caller injects — and CI
// proves it by building twice and diffing.
//
// The graph itself comes from the record-lint engine's own scan
// (lint.LoadRecordGraph); nothing here re-parses the record's shape. What IS
// read here is record BODIES, once each, for the mentions pass: a body naming
// another record's id is weaker evidence than a frontmatter field, so it is
// carried as a separate, weaker relation rather than being promoted into one.

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxRecordBodyBytes bounds one record read for the mentions pass.
const maxRecordBodyBytes = 1 << 20

// bodyHandleRe is the record handle as it appears in prose.
var bodyHandleRe = regexp.MustCompile(`(?i)\b(adr|itd|iss|spc)-(\d+)\b`)

// RecordExport is record.json.
type RecordExport struct {
	SchemaVersion int `json:"schema_version"`
	// Build is the only non-derived content in the file: what produced it.
	Build BuildStamp `json:"build"`
	// Nodes are every record, ordered by store then id.
	Nodes []ExportNode `json:"nodes"`
	// Edges are the record's typed links, each distinct link appearing once.
	Edges []ExportEdge `json:"edges"`
	// Mentions are body references between records that no typed link already
	// records. They are undirected and deduplicated against Edges.
	Mentions []ExportMention `json:"mentions"`
	// Counts are the corpus by store and by lifecycle bucket.
	Counts Counts `json:"counts"`
	// Releases are the dated CHANGELOG headings, newest first. Absent where the
	// repository records none.
	Releases []changelog.DatedRelease `json:"releases"`
	// Authorship is who wrote the history and what assisted.
	Authorship Authorship `json:"authorship"`
	// Health is the record's own reference hygiene, measured.
	Health Health `json:"health"`
	// Layout carries the two precomputed arrangements.
	Layout Arrangements `json:"layout"`
	// History summarises the git walk the dates came from.
	History HistoryMeta `json:"history"`
}

// BuildStamp is what produced this file. Every field is injected by the caller
// so a test can pin the whole export byte for byte.
type BuildStamp struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	GeneratedAt string `json:"generated_at"`
	// Preview marks a build of an untagged tree — main, typically, rendered to a
	// preview deployment. Such a build has no version, and saying so is the point
	// of the field: with the version falling back to the newest CHANGELOG
	// heading, a preview would otherwise stamp itself with a release it is not,
	// and a reader could go and verify that release against a different tree.
	//
	// The version field stays PRESENT and empty on a preview rather than being
	// omitted. A reader of the export, and a test comparing two builds, both get
	// one shape to think about instead of two, and "version": "" beside
	// "preview": true says the absence is deliberate — where a missing key looks
	// like an older export or a dropped field.
	Preview bool `json:"preview,omitempty"`
}

// ExportNode is one record.
type ExportNode struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Lifecycle string `json:"lifecycle"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	Status    string `json:"status,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Severity  string `json:"severity,omitempty"`
	// Date is the record's effective date: its own frontmatter date where it
	// carries one, else the day its file first appeared in git.
	Date string `json:"date"`
	// Derived is true for a record that declares NO frontmatter — its id is its
	// file name, its title its first heading, its dates git's. A page must not
	// present those as fields the record stated, and a chart must not read a
	// lifecycle it never had.
	Derived bool `json:"derived,omitempty"`
	// Dates are the file's three git dates.
	Dates FileDates `json:"dates"`
	// Degree is the weighted connectedness the chart sizes bubbles by.
	Degree float64 `json:"degree"`
}

// ExportEdge is one distinct typed link.
type ExportEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Rel  string `json:"rel"`
}

// ExportMention is one undirected body reference.
type ExportMention struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Counts is the corpus by store, by lifecycle bucket, and by declared status.
//
// The last of those is not a duplicate of the second. A store that grades its
// records by MOVING them between directories (intents, specs, issues) is counted
// by lifecycle; a FLAT store that grades them with a frontmatter field
// (decisions) has one empty lifecycle bucket and its whole shape in the status
// field. A page that only read the lifecycle would show a flat store as a single
// undifferentiated block.
type Counts struct {
	ByType      map[string]int            `json:"by_type"`
	ByLifecycle map[string]map[string]int `json:"by_lifecycle"`
	ByStatus    map[string]map[string]int `json:"by_status"`
}

// Health is the record's reference hygiene as measured, not as claimed.
type Health struct {
	// Unresolved names every typed reference the record cannot account for.
	Unresolved []ExportEdge `json:"unresolved"`
	// Retired are ids a record declares it pruned; a reference naming one
	// resolves to that declaration rather than to a file, so it is not
	// unresolved.
	Retired []string `json:"retired"`
	// BaselineCount is the committed ratchet's size, where the repository keeps
	// one. The ratchet itself is a check, not an export.
	BaselineCount int `json:"baseline_count"`
}

// HistoryMeta summarises the git walk.
type HistoryMeta struct {
	FirstCommit string `json:"first_commit"`
	LastCommit  string `json:"last_commit"`
	Commits     int    `json:"commits"`
}

// relationOf maps a frontmatter field to the relation it declares, and to
// whether it declares it from the far end. The record spells one link from both
// sides — an intent's `spec_id` and its spec's `intent` are the same link — so
// normalising direction here is what lets the pair collapse into one edge.
var relationOf = map[string]struct {
	rel      string
	reversed bool
}{
	"related_adrs":    {"related", false},
	"related_intents": {"related", false},
	"related_rfcs":    {"related", false},
	"builds_on":       {"builds_on", false},
	"blocked_by":      {"blocked_by", false},
	"supersedes":      {"supersedes", false},
	"superseded_by":   {"supersedes", true},
	"spec_id":         {"implements", true},
	"intent":          {"implements", false},
}

// symmetricRel names the relations that have no direction: recording one from
// each end is one fact stated twice, not two facts.
var symmetricRel = map[string]bool{"related": true, "implements": true}

// BuildRecordExport derives record.json from the record graph, the git history
// and the changelog.
func BuildRecordExport(repoRoot, baselineRel string, graph lint.RecordGraph, extra []lint.RecordNode, hist History, stamp BuildStamp, opts RecordOpts) (RecordExport, error) {
	nodes := graph.Nodes
	derived := map[string]bool{}
	if len(extra) > 0 {
		// The file-keyed stores (principles) join the graph here under the
		// handle the site publishes them by: they carry no typed references, so
		// they add nodes and nothing else, and the scan stays the one parser of
		// the record's typed shape (the scan's own principle nodes are dropped
		// before this, by withoutPrincipleNodes). They are MARKED, so a page can tell a field a
		// record declared from one this build worked out from its file.
		existing := make(map[string]bool, len(nodes))
		for _, n := range nodes {
			existing[n.ID] = true
		}
		for _, n := range extra {
			// A frontmatter-free store id that collides with a typed record's id
			// would silently overwrite that record in the export (index/derived are
			// keyed by id, last write wins), stripping its frontmatter panel and
			// rerouting every cross-reference. Refuse loudly rather than corrupt
			// the graph — the fix is to rename the store file.
			if existing[n.ID] {
				return RecordExport{}, fmt.Errorf("site: record id %q is claimed by both a typed record and a frontmatter-free store file; rename the store file", n.ID)
			}
		}
		nodes = append(append(nodes[:0:0], nodes...), extra...)
		for _, n := range extra {
			derived[n.ID] = true
		}
	}
	if !opts.IssueLedger {
		// The issue ledger is working-tier data (adr-32); publishing it is an
		// explicit per-repo opt-in, so without one it is not in the export at
		// all rather than merely hidden by the pages.
		kept := nodes[:0:0]
		for _, n := range nodes {
			if n.Type != "issue" {
				kept = append(kept, n)
			}
		}
		nodes = kept
	}

	index := make(map[string]int, len(nodes))
	for i, n := range nodes {
		index[n.ID] = i
	}

	edges, typedPairs := collapseEdges(graph.Edges, index)
	mentions, mentionPairs, err := scanMentions(repoRoot, nodes, index, typedPairs)
	if err != nil {
		return RecordExport{}, err
	}

	lay := make([]LayoutNode, len(nodes))
	exp := RecordExport{
		SchemaVersion: 1,
		Build:         stamp,
		Nodes:         make([]ExportNode, len(nodes)),
		Edges:         edges,
		Mentions:      mentions,
		Counts: Counts{ByType: map[string]int{}, ByLifecycle: map[string]map[string]int{},
			ByStatus: map[string]map[string]int{}},
		History: HistoryMeta{FirstCommit: hist.First, LastCommit: hist.Last, Commits: hist.Commits},
	}
	for i, n := range nodes {
		date := hist.EffectiveDate(n.Path, n.Date)
		exp.Nodes[i] = ExportNode{
			ID: n.ID, Type: n.Type, Lifecycle: n.Lifecycle, Title: n.Title, Path: n.Path,
			Status: n.Status, Kind: n.Kind, Severity: n.Severity,
			Date: date, Dates: hist.Files[n.Path], Derived: derived[n.ID],
		}
		lay[i] = LayoutNode{Type: n.Type, Date: date, Num: handleNum(n.ID), Touched: hist.Files[n.Path].Touched}
		exp.Counts.ByType[n.Type]++
		if exp.Counts.ByLifecycle[n.Type] == nil {
			exp.Counts.ByLifecycle[n.Type] = map[string]int{}
		}
		exp.Counts.ByLifecycle[n.Type][n.Lifecycle]++
		if n.Status != "" {
			if exp.Counts.ByStatus[n.Type] == nil {
				exp.Counts.ByStatus[n.Type] = map[string]int{}
			}
			exp.Counts.ByStatus[n.Type][n.Status]++
		}
	}

	exp.Layout = ComputeArrangements(lay, typedPairs, mentionPairs)
	for i := range exp.Nodes {
		exp.Nodes[i].Degree = exp.Layout.Degree[i]
	}

	releases, _, err := changelog.DatedReleases(repoRoot)
	if err != nil {
		return RecordExport{}, err
	}
	exp.Releases = releases
	if exp.Releases == nil {
		exp.Releases = []changelog.DatedRelease{}
	}

	exp.Authorship, err = LoadAuthorship(repoRoot)
	if err != nil {
		return RecordExport{}, err
	}

	exp.Health, err = measureHealth(repoRoot, baselineRel, graph, index)
	if err != nil {
		return RecordExport{}, err
	}
	return exp, nil
}

// collapseEdges normalises each typed reference to one direction and drops the
// duplicate a mirrored declaration produces, exactly as the script does: the
// spec link is recorded from both ends, and some `related` pairs are listed in
// both files, so each distinct link must render once.
func collapseEdges(refs []lint.RecordEdge, index map[string]int) ([]ExportEdge, [][2]int) {
	seen := map[[3]string]bool{}
	var out []ExportEdge
	var pairs [][2]int
	for _, r := range refs {
		si, sok := index[r.From]
		ti, tok := index[r.To]
		if !sok || !tok {
			continue
		}
		m, ok := relationOf[r.Field]
		if !ok {
			continue
		}
		from, to := r.From, r.To
		fi, ti2 := si, ti
		if m.reversed {
			from, to = to, from
			fi, ti2 = ti, si
		}
		key := [3]string{m.rel, from, to}
		if symmetricRel[m.rel] {
			if from > to {
				key = [3]string{m.rel, to, from}
			}
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ExportEdge{From: from, To: to, Rel: m.rel})
		pairs = append(pairs, [2]int{fi, ti2})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return lint.HandleLess(out[i].From, out[j].From)
		}
		if out[i].Rel != out[j].Rel {
			return out[i].Rel < out[j].Rel
		}
		return lint.HandleLess(out[i].To, out[j].To)
	})
	// The pair list feeds the layout, so it is rebuilt in the sorted order
	// rather than in discovery order: the arrangement must not depend on which
	// file the walk happened to read first.
	pairs = pairs[:0]
	for _, e := range out {
		pairs = append(pairs, [2]int{index[e.From], index[e.To]})
	}
	if out == nil {
		out = []ExportEdge{}
	}
	return out, pairs
}

// scanMentions reads every record body once and records the ids it names, minus
// the ones a typed link already carries. It is undirected: a body naming another
// record says the two are related, not which way.
func scanMentions(repoRoot string, nodes []lint.RecordNode, index map[string]int, typed [][2]int) ([]ExportMention, [][2]int, error) {
	already := map[[2]int]bool{}
	for _, p := range typed {
		already[[2]int{min(p[0], p[1]), max(p[0], p[1])}] = true
	}
	var out []ExportMention
	var pairs [][2]int
	seen := map[[2]int]bool{}
	// One containment root for the whole body scan: each record path resolves
	// inside it, so a symlinked ancestor cannot pull a body from outside the
	// repository into the mentions pass (gh #487).
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, nil, err
	}
	defer root.Close()
	for _, n := range nodes {
		data, err := fsutil.ReadGuardedInRoot(root, n.Path, maxRecordBodyBytes)
		if err != nil {
			// A record the scan listed but cannot read is a fault worth naming:
			// the graph came from the same tree a moment ago.
			return nil, nil, err
		}
		body, _ := StripFrontmatter(string(data))
		si := index[n.ID]
		for _, m := range bodyHandleRe.FindAllStringSubmatch(body, -1) {
			id := strings.ToLower(m[1]) + "-" + strings.TrimLeft(m[2], "0")
			if strings.HasSuffix(id, "-") {
				id += "0"
			}
			ti, ok := index[id]
			if !ok || ti == si {
				continue
			}
			key := [2]int{min(si, ti), max(si, ti)}
			if already[key] || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, ExportMention{From: nodes[key[0]].ID, To: nodes[key[1]].ID})
			pairs = append(pairs, key)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return lint.HandleLess(out[i].From, out[j].From)
		}
		return lint.HandleLess(out[i].To, out[j].To)
	})
	pairs = pairs[:0]
	for _, m := range out {
		pairs = append(pairs, [2]int{index[m.From], index[m.To]})
	}
	if out == nil {
		out = []ExportMention{}
	}
	return out, pairs, nil
}

// measureHealth counts the references the record cannot account for.
//
// An absent target is accounted for when some record declares it PRUNED — the
// ADR lifecycle deletes a superseded file and keeps the trace in its successor
// — with one exception: the declaration itself. `supersedes: adr-14` names a
// file that is gone by design, and that is precisely the reference the site's
// committed ratchet tracks, so it counts.
func measureHealth(repoRoot, baselineRel string, graph lint.RecordGraph, index map[string]int) (Health, error) {
	retired := map[string]bool{}
	for _, id := range graph.Retired {
		retired[id] = true
	}
	h := Health{Unresolved: []ExportEdge{}, Retired: graph.Retired}
	if h.Retired == nil {
		h.Retired = []string{}
	}
	for _, e := range graph.Dangling {
		if _, ok := index[e.From]; !ok {
			continue
		}
		if e.Field != "supersedes" && retired[e.To] {
			continue
		}
		rel := e.Field
		if m, ok := relationOf[e.Field]; ok {
			rel = m.rel
		}
		h.Unresolved = append(h.Unresolved, ExportEdge{From: e.From, To: e.To, Rel: rel})
	}
	sort.SliceStable(h.Unresolved, func(i, j int) bool {
		if h.Unresolved[i].From != h.Unresolved[j].From {
			return lint.HandleLess(h.Unresolved[i].From, h.Unresolved[j].From)
		}
		return lint.HandleLess(h.Unresolved[i].To, h.Unresolved[j].To)
	})
	n, err := baselineCount(repoRoot, baselineRel)
	if err != nil {
		return Health{}, err
	}
	h.BaselineCount = n
	return h, nil
}
