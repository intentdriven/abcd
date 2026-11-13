package lint

// The citations lint family: spc-17's zero-network commit gate.
//
// Every rule here reads only the committed markdown, the committed config, and
// the committed baseline. Nothing dials out. That is the whole design bet — a
// gate that touches the network is a gate that flakes, and a flaky gate gets
// disabled, so the live checking is banished to `abcd docs cite refresh` and
// what it learned arrives here as a reviewable committed record.
//
// Scoping note that governs three of the five rules: the CITATION CORPUS of a
// page is its footnote definitions, not its prose. A URL in a body paragraph is
// a link (links_resolve's business); a URL on a `[^id]: ...` line is a citation.
// Judging prose as citation is how a source-policy rule acquires a false-positive
// rate nobody will tolerate, so the parse draws that line once, here, and every
// syntax/policy/baseline rule inherits it.

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Citation rule ids. They are separate rules rather than one family switch so a
// repo can adopt the structural checks (which need no baseline) long before it
// has a baseline to enforce.
const (
	ruleCitationFootnotes      = "citation_footnotes"
	ruleCitationCrosswalkRows  = "citation_crosswalk_rows"
	ruleCitationURLSyntax      = "citation_url_syntax"
	ruleCitationSourcePolicy   = "citation_source_policy"
	ruleCitationBaseline       = "citation_baseline"
	ruleCitationBaselineOverdu = "citation_baseline_overdue"
)

// The staleness policy from spc-17's grill settlements. Both are defaults a
// repo may tighten in config; neither is invented here.
const (
	defaultCitationWarnDays  = 180
	defaultCitationBlockDays = 365
)

// CitationPolicy resolves the citation-baseline settings a repo's config
// declares, filling in spc-17's defaults for anything it leaves unset.
//
// It exists so the refresh verb and the gate read the SAME policy: the verb
// decides which entries are stale enough to re-check from the very number the
// lint will later warn on, rather than from a second copy of 180 that drifts.
//
// It is also the containment choke point for the one config value that steers a
// WRITE. `.abcd/docs-lint.json` is a committed file, so a contributor controls
// it; `abcd docs cite refresh` then hands `baseline` to SaveBaseline, which
// MkdirAll's the parent. Without the check below, a `baseline` of
// "../../../etc/x.json" turns a routine refresh into an arbitrary-file-write
// primitive that escapes the repository. Every reader and both writers resolve
// the path through here, so one check covers all four.
func CitationPolicy(cfg Config, repoRoot string) (CitationPolicySet, error) {
	rc := cfg.Rules[ruleCitationBaseline]
	rel := strings.TrimSpace(rc.Baseline)
	if rel == "" {
		rel = DefaultBaselinePath
	}
	// Two checks, because either alone is bypassable. The lexical one refuses a
	// path that is absolute or spells its way out with ".."; the resolved one
	// refuses a path that stays lexically inside and then leaves through a
	// committed SYMLINK, which is the same write primitive by a quieter route.
	if cerr := containedRepoPath(rel); cerr != nil {
		return CitationPolicySet{}, &configError{ruleCitationBaseline + ": baseline " + quote(rel) + " " + cerr.Error()}
	}
	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if cerr := resolvedInsideRoot(repoRoot, abs); cerr != nil {
		return CitationPolicySet{}, &configError{ruleCitationBaseline + ": baseline " + quote(rel) + " " + cerr.Error()}
	}

	set := CitationPolicySet{
		BaselineRel:  rel,
		BaselinePath: abs,
		WarnDays:     rc.WarnAfterDays,
		BlockDays:    rc.BlockAfterDays,
	}
	if set.WarnDays <= 0 {
		set.WarnDays = defaultCitationWarnDays
	}
	if set.BlockDays <= 0 {
		set.BlockDays = defaultCitationBlockDays
	}
	return set, nil
}

// CitationPolicySet is the resolved citation-baseline policy: where the record
// lives, in both the form messages use and the form I/O uses, and the two
// staleness thresholds.
type CitationPolicySet struct {
	// BaselineRel is repo-relative — the only form that may reach a message or
	// machine output, so no developer-identity path is ever rendered.
	BaselineRel string
	// BaselinePath is absolute, for reading and writing.
	BaselinePath string
	WarnDays     int
	BlockDays    int
}

// containedRepoPath refuses a config-supplied path that is not a plain,
// repo-relative location. Absolute paths, Windows volume and UNC forms, and any
// path whose cleaned form climbs out with ".." are all refused rather than
// normalised — a config that asks to write outside the repo is a mistake or an
// attack, and silently rewriting it to something safe would hide both.
func containedRepoPath(rel string) error {
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return errors.New("is absolute; it must be a repo-relative path")
	}
	if vol := filepath.VolumeName(filepath.FromSlash(rel)); vol != "" {
		return errors.New("names a volume; it must be a repo-relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("escapes the repository root")
	}
	return nil
}

// resolvedInsideRoot refuses a target that leaves the repository once symlinks
// are followed — the escape a purely lexical check cannot see, where a committed
// `.abcd/evil -> /somewhere` makes ".abcd/evil/x.json" a write outside the tree.
//
// The target usually does not exist yet (a first refresh writes it), so the walk
// resolves the deepest ancestor that DOES exist and re-attaches the remainder.
// It is the same shape internal/core/launch/bundle.go uses for symlinked bundle
// entries: EvalSymlinks both sides, then filepath.Rel and refuse on "..".
func resolvedInsideRoot(root, target string) error {
	_, err := containedRealPath(root, target)
	return err
}

// containedRealPath is resolvedInsideRoot's other half: it returns the RESOLVED
// path as well as the verdict, so a caller that is about to read the file can
// read the thing containment actually judged rather than re-resolving it.
//
// That distinction matters for the read guard. O_NOFOLLOW refuses every symlink
// leaf, which would break the legitimate in-repo bridge (CLAUDE.md -> AGENTS.md);
// opening the RESOLVED path instead keeps the no-follow protection meaningful
// while still allowing a link that lands inside the repository.
func containedRealPath(root, target string) (string, error) {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		realRoot = filepath.Clean(root)
	}

	probe := filepath.Clean(target)
	var tail []string
	for {
		if _, serr := os.Lstat(probe); serr == nil {
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break // reached the filesystem root without finding anything
		}
		tail = append([]string{filepath.Base(probe)}, tail...)
		probe = parent
	}
	realProbe, err := filepath.EvalSymlinks(probe)
	if err != nil {
		return "", errors.New("cannot be resolved: " + filepath.Base(probe) + ": " + bareCause(err))
	}
	real := filepath.Join(append([]string{realProbe}, tail...)...)

	relToRoot, err := filepath.Rel(realRoot, real)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", errors.New("resolves outside the repository root through a symlink")
	}
	return real, nil
}

// bareCause strips a *PathError's absolute path, leaving the reason. A message
// about a rejected path must not itself embed one (iss-81).
func bareCause(err error) string {
	var pe *os.PathError
	if errors.As(err, &pe) && pe.Err != nil {
		return pe.Err.Error()
	}
	return err.Error()
}

// defaultCrosswalkHeading is the narrowest identification heuristic that matches
// the real crosswalk: a table counts only when its nearest preceding heading
// names itself a crosswalk. Anchoring on the heading rather than on "any table
// in docs/" is what keeps every ordinary directory-map and comparison table in
// the corpus out of the rule's reach.
const defaultCrosswalkHeading = `(?i)crosswalk`

var (
	// A footnote definition opens a line: [^id]: ...
	footnoteDefRe = regexp.MustCompile(`^\s{0,3}\[\^([^\]\s]+)\]:`)
	// A footnote marker anywhere in prose: [^id]. The caret is what separates
	// this from a reference-style link definition ([label]: url), which is a
	// different syntax the rule must not claim.
	footnoteMarkerRe = regexp.MustCompile(`\[\^([^\]\s]+)\]`)
	// An ATX heading, used to find the section a table sits in.
	atxHeadingRe = regexp.MustCompile(`^\s{0,3}#{1,6}\s+(.*\S)\s*$`)
	// A bare URL. Citations in this corpus are written bare, never as markdown
	// links, so the scan is for the scheme rather than for link syntax.
	citationURLRe = regexp.MustCompile("https?://[^\\s<>\"'`]+")
	// A DOI mention: the word, a delimiter, and a token that at least STARTS
	// like a DOI. Anchoring on the "10" prefix is what keeps ordinary prose out
	// of a blocker-severity rule — "a paper with a DOI and a stable URL" and "no
	// DOI is assigned" both put an English word after the delimiter, and the
	// resolvable https://doi.org/10.x form is a URL the syntax check already
	// judges. A token that starts 10 but breaks the grammar is still caught.
	doiMentionRe = regexp.MustCompile(`(?i)\bdoi[:\s]\s*(10\S*)`)
	// The DOI grammar: a registrant prefix and a non-empty suffix.
	doiShapeRe = regexp.MustCompile(`^10\.\d{4,9}/\S+$`)
)

// footnoteRef is one marker or definition occurrence.
type footnoteRef struct {
	id   string
	line int // 1-based
}

// citedRef is one URL cited on a footnote-definition line, with where it was
// cited so a finding can point at it.
type citedRef struct {
	url  string
	file string
	line int // 1-based
}

// pageCitations is one page's citation parse, produced once and shared by every
// citation rule so no rule scans a page again for the same thing.
//
// markers, defs, urls and dois are what a use that can report a finding reads:
// all the page read line by line alone yields, then what the whole reading
// below a backtick left open yields beyond it (readProse), a second scan made
// only where the two readings differ on some line. alone is the page
// read line by line alone, on every line, for a use that can make a finding go
// away: a marker counting a definition as referenced, a definition counting a
// marker as defined, a cited URL admitting a hand receipt.
type pageCitations struct {
	markers []footnoteRef
	defs    []footnoteRef
	urls    []citedRef
	dois    []citedRef
	alone   citationReading
}

// citationReading is one reading's footnote structure and citation corpus.
type citationReading struct {
	markers []footnoteRef
	defs    []footnoteRef
	urls    []citedRef
	dois    []citedRef
}

// parseCitations reads a page's footnote structure and citation corpus.
//
// mask excludes fenced code; readProse (shared with the forbidden-synonym and
// link checks) excludes backticked spans, and below a backtick left open adds
// the whole reading beside the line read alone. Between them, a regexp
// character class like [^A-Za-z] in documentation is not mistaken for a
// citation — which matters because this repo's own records contain exactly
// that decoy.
func parseCitations(rel string, lines []string, mask []bool) pageCitations {
	pr := readProse(lines, mask)
	alone := readCitations(rel, lines, mask, pr.alone)
	p := pageCitations{markers: alone.markers, defs: alone.defs, urls: alone.urls, dois: alone.dois, alone: alone}
	if pr.differs {
		whole := readCitations(rel, lines, mask, pr.whole)
		p.markers = beyond(alone.markers, whole.markers)
		p.defs = beyond(alone.defs, whole.defs)
		p.urls = beyond(alone.urls, whole.urls)
		p.dois = beyond(alone.dois, whole.dois)
	}
	return p
}

// readCitations is parseCitations over one reading of each line, prose.
func readCitations(rel string, lines []string, mask []bool, prose []string) citationReading {
	var p citationReading
	inDef := false
	for i, raw := range lines {
		if mask[i] {
			inDef = false
			continue
		}
		line := prose[i]
		lineNo := i + 1

		// A definition line contributes its id, and its TAIL is citation corpus.
		// The head is excluded from marker scanning so a definition never counts
		// as a reference to itself.
		rest := line
		switch m := footnoteDefRe.FindStringSubmatchIndex(line); {
		case m != nil:
			p.defs = append(p.defs, footnoteRef{id: line[m[2]:m[3]], line: lineNo})
			rest = line[m[1]:]
			inDef = true
		case inDef && isDefContinuation(raw):
			// A wrapped definition is still the same citation, so its later
			// lines stay corpus. Dropping them would fail OPEN — a malformed or
			// baseline-broken address below the fold would never be examined.
		default:
			inDef = false
		}
		if inDef {
			p.urls = append(p.urls, scanCitedURLs(rel, rest, lineNo)...)
			p.dois = append(p.dois, scanCitedDOIs(rel, rest, lineNo)...)
		}
		for _, m := range footnoteMarkerRe.FindAllStringSubmatch(rest, -1) {
			p.markers = append(p.markers, footnoteRef{id: m[1], line: lineNo})
		}
	}
	return p
}

// isDefContinuation reports whether a line continues the footnote definition
// above it. Indented and non-blank is the convention: a blank line or a line
// starting at column zero ends the definition and returns the page to prose.
func isDefContinuation(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	return raw[0] == ' ' || raw[0] == '\t'
}

// scanCitedURLs extracts the bare URLs from a citation line, trimmed of the
// sentence punctuation that abuts them in real prose.
func scanCitedURLs(rel, s string, line int) []citedRef {
	var out []citedRef
	for _, raw := range citationURLRe.FindAllString(s, -1) {
		if u := trimCitedURL(raw); u != "" {
			out = append(out, citedRef{url: u, file: rel, line: line})
		}
	}
	return out
}

// scanCitedDOIs extracts DOI mentions from a citation line.
func scanCitedDOIs(rel, s string, line int) []citedRef {
	var out []citedRef
	for _, m := range doiMentionRe.FindAllStringSubmatch(s, -1) {
		out = append(out, citedRef{url: strings.Trim(m[1], ".,;:"), file: rel, line: line})
	}
	return out
}

// trimCitedURL strips trailing sentence punctuation from a bare URL.
//
// Citations end sentences, sit inside parentheses, and are separated by
// semicolons, so the raw scheme match over-reaches by a character or two almost
// every time. A closing bracket is stripped only when it is unbalanced — URLs
// legitimately contain balanced parentheses, and eating one would invent a
// broken address the baseline could never match.
func trimCitedURL(s string) string {
	for len(s) > 0 {
		switch s[len(s)-1] {
		case '.', ',', ';', ':', '!', '?', '"', '\'':
			s = s[:len(s)-1]
			continue
		case ')':
			if strings.Count(s, ")") > strings.Count(s, "(") {
				s = s[:len(s)-1]
				continue
			}
		case ']':
			if strings.Count(s, "]") > strings.Count(s, "[") {
				s = s[:len(s)-1]
				continue
			}
		}
		return s
	}
	return s
}

// checkCitationFootnotes enforces the marker/definition bijection.
//
// Both directions are findings. An unresolved marker is an obviously broken
// citation; an unreferenced definition is the subtler failure — a source the
// page still lists but no longer claims anything from, which is how a reference
// page quietly stops meaning what it says.
func checkCitationFootnotes(rel string, p pageCitations, cfg RuleConfig) []Finding {
	var out []Finding

	// What makes a finding go away reads each line alone (pageCitations):
	// a definition the page may show as code defines no marker, and a marker
	// it may show as code references no definition.
	defined := map[string]bool{}
	for _, d := range p.alone.defs {
		defined[d.id] = true
	}
	referenced := map[string]bool{}
	for _, m := range p.alone.markers {
		referenced[m.id] = true
	}

	declared := map[string]int{}
	for _, d := range p.defs {
		if n := declared[d.id]; n > 0 {
			out = append(out, Finding{
				File: rel, Line: d.line, RuleID: ruleCitationFootnotes, Severity: cfg.Severity,
				Message: "footnote definition [^" + d.id + "] is declared more than once; one definition per citation",
			})
		}
		declared[d.id]++
	}

	seen := map[string]bool{}
	for _, m := range p.markers {
		if !defined[m.id] && !seen[m.id] {
			seen[m.id] = true
			out = append(out, Finding{
				File: rel, Line: m.line, RuleID: ruleCitationFootnotes, Severity: cfg.Severity,
				Message: "footnote marker [^" + m.id + "] has no definition on this page",
			})
		}
	}
	for _, d := range p.defs {
		if !referenced[d.id] {
			out = append(out, Finding{
				File: rel, Line: d.line, RuleID: ruleCitationFootnotes, Severity: cfg.Severity,
				Message: "footnote definition [^" + d.id + "] is never referenced; a citation nothing claims from is dead weight",
			})
		}
	}
	return out
}

// checkCitationCrosswalkRows implements the rule deferred from spc-15: every
// body row of a crosswalk table carries at least one footnote.
//
// A crosswalk maps an outside term to this project's position on it. The whole
// artefact's authority rests on the outside half being sourced, so an uncited
// row is not a formatting slip — it is an unsupported claim about someone else's
// vocabulary. Identification is by the table's nearest preceding heading, the
// narrowest heuristic that selects the real crosswalk and no ordinary table.
func checkCitationCrosswalkRows(rel string, lines []string, mask []bool, cfg RuleConfig) ([]Finding, error) {
	pattern := cfg.CrosswalkHeading
	if strings.TrimSpace(pattern) == "" {
		pattern = defaultCrosswalkHeading
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, &configError{ruleCitationCrosswalkRows + ": crosswalk_heading is not a valid regexp: " + err.Error()}
	}

	var out []Finding
	inCrosswalk := false
	for i := 0; i < len(lines); i++ {
		if mask[i] {
			continue
		}
		if m := atxHeadingRe.FindStringSubmatch(lines[i]); m != nil {
			// A heading closes the previous section and opens a new one, so a
			// crosswalk's scope ends where the next heading begins.
			inCrosswalk = re.MatchString(m[1])
			continue
		}
		if !inCrosswalk || !isTableHeader(lines, mask, i) {
			continue
		}
		// i is the header row; i+1 is the separator; body rows follow.
		for r := i + 2; r < len(lines) && !mask[r] && strings.Contains(lines[r], "|"); r++ {
			if len(tableCells(lines[r])) == 0 {
				continue
			}
			// A row is read alone, as every use that can make a finding go
			// away is (readProse): a marker read inside a span the page may
			// show as code would let an uncited row through, so no line above
			// a row changes the row's spans.
			if footnoteMarkerRe.MatchString(stripInlineCodeLine(lines[r])) {
				continue
			}
			out = append(out, Finding{
				File: rel, Line: r + 1, RuleID: ruleCitationCrosswalkRows, Severity: cfg.Severity,
				Message: "crosswalk row carries no footnote; every row's established meaning needs a cited primary source",
			})
		}
		// Skip past the table just consumed.
		for i+1 < len(lines) && !mask[i+1] && strings.Contains(lines[i+1], "|") {
			i++
		}
	}
	return out, nil
}

// isTableHeader reports whether line i opens a markdown table — a pipe row
// followed by a separator row. Both conditions are required: a lone pipe in
// prose is not a table, and isTableSeparator alone accepts a blank line.
func isTableHeader(lines []string, mask []bool, i int) bool {
	if !strings.Contains(lines[i], "|") || i+1 >= len(lines) || mask[i+1] {
		return false
	}
	sep := lines[i+1]
	return strings.Contains(sep, "|") && strings.Contains(sep, "-") && isTableSeparator(sep)
}

// checkCitationURLSyntax checks that cited addresses are well-formed. It reads
// the citation corpus only — prose URLs belong to links_resolve.
func checkCitationURLSyntax(p pageCitations, cfg RuleConfig) []Finding {
	var out []Finding
	for _, ref := range p.urls {
		if why := malformedURL(ref.url); why != "" {
			out = append(out, Finding{
				File: ref.file, Line: ref.line, RuleID: ruleCitationURLSyntax, Severity: cfg.Severity,
				Message: "cited URL " + ref.url + " is not well-formed: " + why,
			})
		}
	}
	for _, ref := range p.dois {
		if !doiShapeRe.MatchString(ref.url) {
			out = append(out, Finding{
				File: ref.file, Line: ref.line, RuleID: ruleCitationURLSyntax, Severity: cfg.Severity,
				Message: "cited DOI " + ref.url + " is not well-formed: expected 10.<registrant>/<suffix>",
			})
		}
	}
	return out
}

// malformedURL returns why a cited URL is unusable, or "" when it is fine.
func malformedURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return err.Error()
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "scheme is not http(s)"
	}
	host := u.Hostname()
	if host == "" {
		return "no host"
	}
	if strings.ContainsAny(host, " \t") {
		return "host contains whitespace"
	}
	// A citation must name a resolvable public host. A dotless host is a
	// hostname on someone's private network, never a source anyone else can read.
	if !strings.Contains(host, ".") || strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return "host " + host + " is not a fully-qualified domain"
	}
	return ""
}

// checkCitationSourcePolicy refuses citations to configured aggregator domains.
//
// The list is committed config and ships EMPTY: this repo documents its
// admission rule in prose ("no single-author coinages, no aggregators") but
// nowhere names a domain, and a gate must not invent a blacklist its project
// never agreed to. The rule is the mechanism; the policy is the maintainer's.
func checkCitationSourcePolicy(p pageCitations, cfg RuleConfig) []Finding {
	if len(cfg.RefusedDomains) == 0 {
		return nil
	}
	refused := make([]string, 0, len(cfg.RefusedDomains))
	for _, d := range cfg.RefusedDomains {
		if d = normaliseHost(d); d != "" {
			refused = append(refused, d)
		}
	}

	var out []Finding
	for _, ref := range p.urls {
		u, err := url.Parse(ref.url)
		if err != nil {
			continue // malformed addresses are citation_url_syntax's finding, not a second one here
		}
		host := normaliseHost(u.Hostname())
		for _, d := range refused {
			// Subdomains count: refusing an aggregator must not be defeated by
			// citing its blog. subdomain host.
			if host == d || strings.HasSuffix(host, "."+d) {
				out = append(out, Finding{
					File: ref.file, Line: ref.line, RuleID: ruleCitationSourcePolicy, Severity: cfg.Severity,
					Message: "cited source " + ref.url + " is on the refused domain " + d + "; cite the primary source instead",
				})
				break
			}
		}
	}
	return out
}

// normaliseHost lower-cases a host and drops a leading www., so one config entry
// covers the ways the same publisher is written.
func normaliseHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.TrimSuffix(h, ".")
	return strings.TrimPrefix(h, "www.")
}

// checkCitationBaseline enforces the committed baseline offline.
//
// now is a parameter rather than a wall-clock read so the staleness arithmetic
// is testable at its exact boundaries; the caller (LintAt) owns the clock.
func checkCitationBaseline(repoRoot string, refs []citedRef, cfg RuleConfig, now time.Time) ([]Finding, error) {
	if len(refs) == 0 {
		// Nothing cites, so there is nothing to have a receipt for. A repo that
		// enables the rule before it has any citations is not in violation.
		return nil, nil
	}
	// Resolve through the shared policy so the containment check binds here too:
	// a config that points the baseline outside the repo is refused by the gate,
	// not only by the verb that writes it.
	policy, err := CitationPolicy(Config{Rules: map[string]RuleConfig{ruleCitationBaseline: cfg}}, repoRoot)
	if err != nil {
		return nil, err
	}
	rel := policy.BaselineRel

	base, err := LoadBaseline(policy.BaselinePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Fail closed, but as ONE finding: a per-URL storm here would bury
			// the single action that fixes all of them.
			return []Finding{{
				File: rel, Line: 0, RuleID: ruleCitationBaseline, Severity: cfg.Severity,
				Message: "citation baseline not found, but " + strconv.Itoa(len(refs)) +
					" citation(s) need one; run `abcd docs cite refresh` to create it",
			}}, nil
		}
		return nil, err
	}

	warnDays, blockDays := policy.WarnDays, policy.BlockDays
	overdueSeverity := cfg.OverdueSeverity
	if strings.TrimSpace(overdueSeverity) == "" {
		// The commit gate never calendar-blocks (spc-17). The release gate is
		// what promotes this, by setting the field.
		overdueSeverity = severityWarn
	}

	var out []Finding
	// One citation site is one problem. A URL repeated on the same line (a
	// source and its mirror) would otherwise emit byte-identical findings.
	reported := map[citedRef]bool{}
	for _, ref := range refs {
		if reported[ref] {
			continue
		}
		reported[ref] = true
		e, ok := base.Entries[ref.url]
		if !ok {
			out = append(out, Finding{
				File: ref.file, Line: ref.line, RuleID: ruleCitationBaseline, Severity: cfg.Severity,
				// Both verbs, deliberately. The ONE state that produces a missing
				// entry is a source that refuses automated fetchers: the refresh
				// queued it and wrote nothing by design, so naming refresh alone
				// sends the maintainer round a loop that cannot terminate.
				Message: "cited URL " + ref.url + " is not in the citation baseline; run " +
					"`abcd docs cite refresh` — or, if the source refuses automated fetchers, " +
					"open it and record it with `abcd docs cite confirm " + ref.url + "`",
			})
			continue
		}
		switch {
		case e.Outcome == OutcomeBroken:
			out = append(out, Finding{
				File: ref.file, Line: ref.line, RuleID: ruleCitationBaseline, Severity: cfg.Severity,
				Message: "cited URL " + ref.url + " is recorded broken in the citation baseline (checked " + e.LastChecked + ")",
			})
		case e.FinalURL != ref.url:
			// Offline drift: the baseline knows where this address ends up, and
			// the page still cites the old one. Citing the pre-redirect address
			// is how a citation survives its own source moving, right up until
			// the redirect is retired.
			out = append(out, Finding{
				File: ref.file, Line: ref.line, RuleID: ruleCitationBaseline, Severity: cfg.Severity,
				Message: "cited URL " + ref.url + " resolves to " + e.FinalURL +
					"; cite the final address the baseline recorded",
			})
		}

		checked, perr := e.Checked()
		if perr != nil {
			continue // unreachable: LoadBaseline validated the date
		}
		age := DaysBetween(checked, now)
		switch {
		case age >= blockDays:
			out = append(out, Finding{
				File: ref.file, Line: ref.line, RuleID: ruleCitationBaselineOverdu, Severity: overdueSeverity,
				Message: "citation " + ref.url + " was last checked " + e.LastChecked + " (" +
					strconv.Itoa(age) + " days ago, past the " + strconv.Itoa(blockDays) +
					"-day release threshold); run `abcd docs cite refresh`",
			})
		case age >= warnDays:
			out = append(out, Finding{
				File: ref.file, Line: ref.line, RuleID: ruleCitationBaseline, Severity: severityWarn,
				Message: "citation " + ref.url + " was last checked " + e.LastChecked + " (" +
					strconv.Itoa(age) + " days ago, past the " + strconv.Itoa(warnDays) +
					"-day staleness threshold); run `abcd docs cite refresh`",
			})
		}
	}
	return out, nil
}

// DaysBetween counts whole calendar days, both ends normalised to UTC midnight.
// Comparing instants would make the 180-day boundary depend on the hour the gate
// happened to run.
//
// It is exported because the refresh verb decides which entries are stale enough
// to re-check and MUST get the same answer as the gate that will later warn on
// them. Two copies of this arithmetic is two chances for a boundary to drift,
// which would surface as a URL the verb thinks current and the gate thinks
// overdue, with nothing to say which is right.
func DaysBetween(from, to time.Time) int {
	// Convert to UTC before taking the calendar date, don't merely restamp the
	// local Y/M/D at UTC: the gate reads time.Now() (local) while the baseline
	// dates are stamped time.Now().UTC(), so a bare Year()/Month()/Day() made
	// the boundary — and, under the release gate, the citation-overdue blocker —
	// depend on the maintainer's timezone, the very drift this shared function
	// exists to prevent.
	fu, tu := from.UTC(), to.UTC()
	f := time.Date(fu.Year(), fu.Month(), fu.Day(), 0, 0, 0, 0, time.UTC)
	t := time.Date(tu.Year(), tu.Month(), tu.Day(), 0, 0, 0, 0, time.UTC)
	return int(t.Sub(f).Hours() / 24)
}
