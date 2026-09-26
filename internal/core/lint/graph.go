package lint

// The exported record graph: the corpus as nodes and typed references, read out
// of the record_schema rule's own scan.
//
// The rule already walks all four stores, parses every cross-reference field in
// both YAML sequence spellings, and knows which handles resolve — that is the
// whole graph, and it was locked inside a lint rule. A consumer that wanted it
// (the site build's record export) had exactly two options: re-implement the
// walk, or use this. There is one parser of the record's shape in this binary
// and this is the door onto it; a second one would drift the moment a store
// gained a bucket.
//
// This is a READ of the same scan, never a judgement: it returns what the record
// says, including the references that do not resolve, and reports no findings.
// Whether an unresolved reference is a fault is the rule's question, not this
// function's — the rule excuses a pruned ADR id, and a consumer measuring
// unresolved references against a committed baseline needs to see it anyway.

import (
	"sort"
	"strconv"
	"strings"
)

// RecordNode is one record in the corpus.
type RecordNode struct {
	// ID is the record's prose handle (adr-47, itd-135, spc-38, iss-100).
	ID string `json:"id"`
	// Type is the store's name: adr, intent, spec, issue.
	Type string `json:"type"`
	// Lifecycle is the bucket directory the record sits in (drafts, planned,
	// shipped, open, resolved …). It is empty for a flat store, whose records
	// carry their state in frontmatter rather than in a directory.
	Lifecycle string `json:"lifecycle"`
	// Title is the record's H1, or its first body line where the store's records
	// carry no heading, or the handle where the file is empty.
	Title string `json:"title"`
	// Path is the record file, repo-relative and slash-separated.
	Path string `json:"path"`
	// Date is the frontmatter `date` where the record carries one, else empty.
	// A record without one is dated from git by the consumer, not here.
	Date string `json:"date,omitempty"`
	// Status, Kind and Severity are the frontmatter fields the stores use to
	// grade a record, carried where present and empty where not.
	Status   string `json:"status,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// RecordEdge is one typed reference: the record that declares it, the handle it
// names, and the frontmatter field it was read out of. The field is kept rather
// than collapsed into a relation name because the same relation is spelled from
// both ends (an intent's `spec_id` and a spec's `intent` are one link) and only
// the consumer knows whether it wants them folded.
type RecordEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Field string `json:"field"`
}

// RecordGraph is the whole corpus: every record, every typed reference that
// resolves to one, and every reference that names a record no file carries.
type RecordGraph struct {
	Nodes    []RecordNode `json:"nodes"`
	Edges    []RecordEdge `json:"edges"`
	Dangling []RecordEdge `json:"dangling"`
	// Retired are the ids some record declares it superseded, bounded by each
	// store's allocation high-water mark exactly as the rule bounds them. The
	// ADR lifecycle prunes a superseded file and keeps the trace in its
	// successor's `supersedes`, so a handle naming a retired id is accounted for
	// even though no file answers to it — and a consumer counting unresolved
	// references needs that distinction to avoid reporting the record's own
	// bookkeeping as breakage.
	Retired []string `json:"retired"`
}

// LoadRecordGraph reads every configured record store once and returns the
// corpus as a graph. It is the record_schema rule's scan, exported: cfg is the
// same lint configuration the rule runs from, and a store the configuration does
// not name (or whose directory is absent) contributes nothing, exactly as it
// contributes no findings.
//
// The result is fully ordered — nodes by store then id number, edges by their
// two endpoints and field — so a caller that serialises it gets the same bytes
// from the same tree.
func LoadRecordGraph(cfg Config, repoRoot string) (RecordGraph, error) {
	records, _, err := scanRecordStores(repoRoot, cfg.Rules[ruleRecordSchema])
	if err != nil {
		return RecordGraph{}, err
	}

	// present keys on the rendered handle, so a slug-keyed record (a principle,
	// prn-<stem>) is present under the handle it is cited by rather than under a
	// (prefix, 0) pair every principle would share.
	present := make(map[string]bool, len(records))
	for _, r := range records {
		present[r.handle()] = true
	}
	retired := retiredHandles(records)

	g := RecordGraph{
		Nodes:    make([]RecordNode, 0, len(records)),
		Edges:    []RecordEdge{},
		Dangling: []RecordEdge{},
		Retired:  retired,
	}
	seen := map[RecordEdge]bool{}
	for _, r := range records {
		handle := r.handle()
		title := r.title
		if title == "" {
			title = handle
		}
		g.Nodes = append(g.Nodes, RecordNode{
			ID:        handle,
			Type:      r.store.nodeType,
			Lifecycle: r.bucket,
			Title:     title,
			Path:      filepathToSlash(r.rel),
			Date:      fieldValue(r.fields, "date"),
			Status:    fieldValue(r.fields, "status"),
			Kind:      fieldValue(r.fields, "kind"),
			Severity:  fieldValue(r.fields, "severity"),
		})
		for _, field := range recordParsedFields {
			for _, h := range r.refs[field] {
				e := RecordEdge{From: handle, To: h.String(), Field: field}
				if e.From == e.To || seen[e] {
					continue
				}
				seen[e] = true
				if present[h.String()] {
					g.Edges = append(g.Edges, e)
				} else {
					g.Dangling = append(g.Dangling, e)
				}
			}
		}
	}

	sort.Slice(g.Nodes, func(i, j int) bool {
		return HandleLess(g.Nodes[i].ID, g.Nodes[j].ID)
	})
	sortEdges(g.Edges)
	sortEdges(g.Dangling)
	return g, nil
}

// sortEdges orders edges by source, then field, then target, so the export is a
// function of the tree and not of the walk.
func sortEdges(edges []RecordEdge) {
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.From != b.From {
			return HandleLess(a.From, b.From)
		}
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		return HandleLess(a.To, b.To)
	})
}

// HandleLess orders two record handles by store prefix, then NUMERICALLY — so
// adr-9 precedes adr-10, which a plain string sort gets backwards and which a
// reader of any rendered list notices immediately.
//
// It is exported alongside the graph because every consumer that renders a
// record list needs exactly this order, and the obvious second implementation
// is a plain string sort that looks right until the store passes nine records.
func HandleLess(a, b string) bool {
	pa, na := splitHandle(a)
	pb, nb := splitHandle(b)
	if pa != pb {
		return pa < pb
	}
	if na != nb {
		return na < nb
	}
	return a < b
}

// splitHandle divides a handle into its prefix and number. A value that is not a
// handle sorts by its whole text under an empty prefix.
func splitHandle(h string) (string, int) {
	i := strings.LastIndex(h, "-")
	if i < 0 {
		return h, 0
	}
	n, err := strconv.Atoi(h[i+1:])
	if err != nil {
		return h, 0
	}
	return h[:i], n
}

// fieldValue reads one frontmatter field as a plain scalar: quotes stripped,
// an explicit null read as absent. The record spells the same value three ways
// (`minor`, `"minor"`, `'minor'`) and a consumer should see one.
func fieldValue(fields map[string]fmField, name string) string {
	f, ok := fields[name]
	if !ok {
		return ""
	}
	v := strings.TrimSpace(f.value)
	if isNull(v) {
		return ""
	}
	return strings.Trim(v, `"'`)
}

// filepathToSlash renders a scanned relative path with forward slashes, so the
// export reads the same on every platform.
func filepathToSlash(rel string) string {
	return strings.ReplaceAll(rel, "\\", "/")
}
