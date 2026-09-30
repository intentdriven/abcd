package lint

// The dependency-edge rules: stale_edge and edge_cycle read the intents'
// `builds_on` and `blocked_by` edges out of the record_schema scan (the one
// canonical walk of the stores, the same one LoadRecordGraph exports), so a
// block-spelled list is read by the same parser every other rule uses.
//
// record_schema already refuses an edge naming a record the corpus does not
// hold. It says nothing about two shapes an edge can take while every handle
// resolves: an edge naming a record that was superseded, and edges that close
// a cycle. Both leave the pick order and the build's blocked check reasoning
// over a dependency that no longer holds.
//
// Both rules land at warn. The tree they arrive in carries five findings that
// each wait on a decision, not on a repair: the drafts itd-22 (`blocked_by`)
// and itd-33 (`builds_on`) name itd-2, whose successor does not carry the
// in-session dispatch contract they depended on; itd-2609201916056194 and
// itd-2609201916151817 each build on the other (iss-2609300848016421); and so
// do itd-14 and itd-15, and itd-2609081951381895 and itd-2609170822093401
// (iss-2609300903551032). The rules are promoted to blocker once those are
// decided.

import (
	"slices"
	"sort"
	"strings"
)

const (
	// ruleStaleEdge reports a planned or draft intent whose builds_on or
	// blocked_by names a superseded intent.
	ruleStaleEdge = "stale_edge"
	// ruleEdgeCycle reports a cycle through builds_on and blocked_by together.
	ruleEdgeCycle = "edge_cycle"
)

// The intent buckets the edge rules distinguish.
const (
	bucketDrafts     = "drafts"
	bucketPlanned    = "planned"
	bucketSuperseded = "superseded"
)

// intentEdgeFields are the dependency edges both rules read. They are one
// relation for the cycle rule: an intent waiting on another cannot also be
// waited on by it, whichever of the two fields spells each half.
var intentEdgeFields = []string{"builds_on", "blocked_by"}

// SupersessionLink is one intent as a supersession chain reads it: its id, the
// bucket it sits in, and the successor its `superseded_by` names. It aliases the
// same unnamed struct intent.ChainLink does, so intent.SupersessionChainOf
// registers here as it is.
type SupersessionLink = struct{ ID, Bucket, SupersededBy string }

// supersessionChain is the intent package's chain reader,
// intent.SupersessionChainOf, registered by the front doors that run this gate
// (cmd/record-lint, and the CLI for `abcd lint`), because this package cannot
// import core/intent: intent's own tests import this package, and Go refuses the
// cycle. It is the one walk of a supersession chain, the one the build's blocked
// check follows; stale_edge never walks a chain itself. Unregistered, stale_edge
// says so in one finding instead of reporting nothing.
var supersessionChain func(links []SupersessionLink, id string) (chain []string, endID, endBucket, problem string)

// SetSupersessionChain registers the supersession chain reader stale_edge
// follows. Pass intent.SupersessionChainOf.
func SetSupersessionChain(fn func(links []SupersessionLink, id string) (chain []string, endID, endBucket, problem string)) {
	supersessionChain = fn
}

// scanIntentRecords reads the intent store through the record_schema scan,
// keyed on the stores rc names or, where it names none, record_schema's. The
// scan's own findings belong to record_schema and are dropped here.
func scanIntentRecords(repoRoot string, cfg Config, rc RuleConfig) ([]schemaRecord, string, error) {
	stores := rc.RecordStores
	if len(stores) == 0 {
		stores = cfg.Rules[ruleRecordSchema].RecordStores
	}
	if stores["itd"] == "" {
		return nil, "", nil
	}
	scanCfg := cfg.Rules[ruleRecordSchema]
	scanCfg.RecordStores = map[string]string{"itd": stores["itd"]}
	records, _, err := scanRecordStores(repoRoot, scanCfg)
	if err != nil {
		return nil, "", err
	}
	var out []schemaRecord
	for _, r := range records {
		if r.store.prefix == "itd" {
			out = append(out, r)
		}
	}
	return out, stores["itd"], nil
}

// checkStaleEdges implements stale_edge: a planned/ or drafts/ intent whose
// builds_on or blocked_by names an intent in superseded/. One finding per edge,
// on the field's line, naming the record the target's supersession chain ends
// at (its live successor), or why the chain cannot be finished.
//
// WHY planned/ and drafts/ alone: those are the records whose edges still steer
// work (the pick order, the build's blocked check). A shipped record's edges
// are history, and a superseded record's edges bind nothing.
func checkStaleEdges(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {
	records, store, err := scanIntentRecords(repoRoot, cfg, rc)
	if err != nil || len(records) == 0 {
		return nil, err
	}
	if supersessionChain == nil {
		return []Finding{{
			File: store, Line: 0, RuleID: ruleStaleEdge, Severity: rc.Severity,
			Message: "stale_edge follows supersession chains through intent.SupersessionChainOf, and the front door " +
				"running it registered none (lint.SetSupersessionChain), so no edge was checked; " +
				"register it where the gate is wired, as cmd/record-lint and the CLI do",
		}}, nil
	}
	links := make([]SupersessionLink, 0, len(records))
	bucketOf := map[string]string{}
	for _, r := range records {
		l := SupersessionLink{ID: r.handle(), Bucket: r.bucket}
		if hs := r.refs["superseded_by"]; len(hs) > 0 {
			l.SupersededBy = hs[0].String()
		}
		links = append(links, l)
		bucketOf[l.ID] = r.bucket
	}
	var out []Finding
	for _, r := range records {
		if r.bucket != bucketPlanned && r.bucket != bucketDrafts {
			continue
		}
		for _, field := range intentEdgeFields {
			for _, h := range r.refs[field] {
				target := h.String()
				if h.prefix != "itd" || bucketOf[target] != bucketSuperseded {
					continue
				}
				chain, endID, endBucket, problem := supersessionChain(links, target)
				path := strings.Join(chain, " → ")
				msg := field + " names " + target + ", which is superseded; "
				if problem == "" {
					msg += "its supersession chain " + path + " ends at " + endID + " (" + endBucket + "): " +
						"repoint the edge there if that record carries what this one depends on, or drop the edge"
				} else {
					msg += "its supersession chain " + path + " cannot be followed to a live intent (" + problem + "): " +
						"drop the edge, or repoint it at the record that carries what this one depends on"
				}
				line := r.fields[field].line
				if line == 0 {
					line = 1
				}
				out = append(out, Finding{File: r.rel, Line: line, RuleID: ruleStaleEdge, Severity: rc.Severity, Message: msg})
			}
		}
	}
	return out, nil
}

// checkEdgeCycles implements edge_cycle: a strongly connected set of intents
// under builds_on ∪ blocked_by (two or more records, or one naming itself) is
// one finding, on the first record of the set in handle order, naming every
// record in it.
//
// WHY every bucket but superseded/: a superseded record's edges bind nothing,
// and an edge into one is stale_edge's finding, not a cycle.
func checkEdgeCycles(repoRoot string, cfg Config, rc RuleConfig) ([]Finding, error) {
	records, _, err := scanIntentRecords(repoRoot, cfg, rc)
	if err != nil {
		return nil, err
	}
	byID := map[string]schemaRecord{}
	for _, r := range records {
		if r.bucket == bucketSuperseded {
			continue
		}
		byID[r.handle()] = r
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return HandleLess(ids[i], ids[j]) })

	succ := map[string][]string{}
	for _, id := range ids {
		seen := map[string]bool{}
		for _, field := range intentEdgeFields {
			for _, h := range byID[id].refs[field] {
				to := h.String()
				if _, ok := byID[to]; ok && h.prefix == "itd" && !seen[to] {
					seen[to] = true
					succ[id] = append(succ[id], to)
				}
			}
		}
		sort.Slice(succ[id], func(i, j int) bool { return HandleLess(succ[id][i], succ[id][j]) })
	}

	var out []Finding
	for _, comp := range stronglyConnected(ids, succ) {
		if len(comp) == 1 && !slices.Contains(succ[comp[0]], comp[0]) {
			continue
		}
		sort.Slice(comp, func(i, j int) bool { return HandleLess(comp[i], comp[j]) })
		first := byID[comp[0]]
		line := 0
		for _, field := range intentEdgeFields {
			if l := first.fields[field].line; l > 0 && (line == 0 || l < line) {
				line = l
			}
		}
		if line == 0 {
			line = 1
		}
		out = append(out, Finding{
			File: first.rel, Line: line, RuleID: ruleEdgeCycle, Severity: rc.Severity,
			Message: "builds_on/blocked_by cycle through " + strings.Join(comp, ", ") +
				": they depend on one another, so no order satisfies every edge; drop the edge that does not hold",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out, nil
}

// stronglyConnected returns the strongly connected components of the graph
// (Tarjan's algorithm), visiting nodes and successors in the order given so
// the result is a function of the graph.
func stronglyConnected(nodes []string, succ map[string][]string) [][]string {
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	var comps [][]string
	next := 0
	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range succ[v] {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var comp []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			comps = append(comps, comp)
		}
	}
	for _, v := range nodes {
		if _, seen := index[v]; !seen {
			visit(v)
		}
	}
	return comps
}
