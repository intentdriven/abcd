package launch

// The citation-baseline release gate (spc-17's surfacing clause).
//
// The refresh verb is manual. spc-17's answer to "then who remembers to run it"
// is nagging at the two places a maintainer already looks: `ahoy` for the daily
// glance, and the release preflight for the moment it actually matters. This is
// the second.
//
// The measurement is NOT taken here. Grading the baseline needs the docs-lint
// engine, and internal/core/lint imports this package for its semver — so a call
// the other way would be an import cycle. The front door that already holds both
// takes the summary and hands it in as data, the same shape surface/cli uses to
// feed the cobra command tree into core/surface. The dependency never points
// back.

import "strconv"

// CitationPreflight is the citation baseline's state, measured by the caller.
// A nil pointer means the repo has not armed the citation gate, and the gate
// reports itself as not run rather than silently passing.
type CitationPreflight struct {
	// Unreadable, when non-empty, means the measurement could not be taken at
	// all — a docs-lint config that arms the rule but does not parse, or a
	// baseline the loader refuses. It is a separate state from "not armed"
	// because reporting a broken gate as an absent one is a false statement
	// about a real requirement, and it refuses.
	Unreadable string
	// Present reports whether a baseline exists at all.
	Present bool
	// Cited, Recorded and Missing describe coverage.
	Cited    int
	Recorded int
	Missing  int
	// Broken counts citations recorded as dead.
	Broken int
	// Approaching counts entries inside the nag window before the blocking
	// threshold; Overdue counts those past it.
	Approaching int
	Overdue     int
	// ApproachingURLs and OverdueURLs name them, so a nag and a refusal tell an
	// operator WHICH links to deal with rather than only how many.
	ApproachingURLs []string
	OverdueURLs     []string
}

// citationGate renders the gate's dry-run disposition.
func citationGate(pre *CitationPreflight) GateSummary {
	if pre == nil {
		return GateSummary{
			Name:   "citation-baseline",
			Status: "not_implemented",
			Detail: "citation_baseline is not armed in this repo's docs-lint config",
		}
	}
	if pre.Unreadable != "" {
		return GateSummary{Name: "citation-baseline", Status: "ran",
			Detail: "the citation baseline could not be measured: " + pre.Unreadable}
	}
	if !pre.Present && pre.Cited > 0 {
		return GateSummary{Name: "citation-baseline", Status: "ran",
			Detail: strconv.Itoa(pre.Cited) + " cited, no baseline — run `abcd docs cite refresh`"}
	}

	detail := strconv.Itoa(pre.Cited) + " cited, " + strconv.Itoa(pre.Recorded) + " with receipts"
	if pre.Missing > 0 {
		detail += ", " + strconv.Itoa(pre.Missing) + " unreceipted"
	}
	if pre.Broken > 0 {
		detail += ", " + strconv.Itoa(pre.Broken) + " broken"
	}
	if pre.Overdue > 0 {
		detail += ", " + strconv.Itoa(pre.Overdue) + " overdue"
	}
	// The nag: named here, in the informational half, precisely because it does
	// NOT refuse. A maintainer gets a month's notice while releases still cut.
	if pre.Approaching > 0 {
		detail += ", " + strconv.Itoa(pre.Approaching) + " approaching the staleness blocker (" +
			joinURLs(pre.ApproachingURLs) + ")"
	}
	return GateSummary{Name: "citation-baseline", Status: "ran", Detail: detail}
}

// citationRefusals lists what the citation baseline would stop a release on.
// Approaching entries are deliberately absent: they nag, they do not block.
func citationRefusals(pre *CitationPreflight) []string {
	if pre == nil {
		return nil
	}
	if pre.Unreadable != "" {
		return []string{"citation baseline: could not be measured (" + pre.Unreadable +
			"); a release must not proceed on a gate that did not run"}
	}
	var out []string
	if !pre.Present && pre.Cited > 0 {
		return []string{"citation baseline: absent, but " + strconv.Itoa(pre.Cited) +
			" citation(s) need one — run `abcd docs cite refresh`"}
	}
	if pre.Missing > 0 {
		out = append(out, "citation baseline: "+strconv.Itoa(pre.Missing)+
			" cited URL(s) have no receipt — run `abcd docs cite refresh`")
	}
	if pre.Broken > 0 {
		out = append(out, "citation baseline: "+strconv.Itoa(pre.Broken)+
			" citation(s) recorded broken — fix or replace the source")
	}
	if pre.Overdue > 0 {
		out = append(out, "citation baseline: "+strconv.Itoa(pre.Overdue)+
			" citation(s) past the staleness threshold ("+joinURLs(pre.OverdueURLs)+
			") — run `abcd docs cite refresh`")
	}
	return out
}

// joinURLs renders a bounded list. A release blocked by forty stale citations
// must not print forty lines into a gate summary; the count above it is the
// complete number, and `abcd lint docs` names every one.
func joinURLs(urls []string) string {
	const max = 3
	if len(urls) == 0 {
		return "none"
	}
	shown := urls
	suffix := ""
	if len(urls) > max {
		shown = urls[:max]
		suffix = ", and " + strconv.Itoa(len(urls)-max) + " more"
	}
	out := ""
	for i, u := range shown {
		if i > 0 {
			out += ", "
		}
		out += u
	}
	return out + suffix
}
