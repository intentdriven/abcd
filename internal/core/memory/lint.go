package memory

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// lint.go — the `abcd memory lint` verb (fn-39): a full-store curator
// health-check. Page-local checks (MR001/MS001/MS002/ML001/MQ001/MQ003) per
// typed page, a residue pass (MR001) over the sources registry and the text
// kept-originals, plus a corpus pass (MQ002 + per-source MQ003) that rebuilds
// the regenerable .coverage_index.json. Writes ONE run-log report; mutates no
// memory-store state. Exit contract: blocker -> 1; warn/info/clean -> 0.

// Finding is a single memory-lint finding.
type Finding struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// LintSummary tallies findings by severity.
type LintSummary struct {
	Blockers int `json:"blockers"`
	Warnings int `json:"warnings"`
	Infos    int `json:"infos"`
}

// LintRequest is the input to Lint.
type LintRequest struct {
	RepoRoot string
	Now      time.Time
}

// LintResult is the structured result of Lint.
type LintResult struct {
	Findings      []Finding      `json:"findings"`
	Summary       LintSummary    `json:"summary"`
	CoverageIndex map[string]any `json:"coverage_index"`
	ReportDir     string         `json:"report_dir"`
	GeneratedAt   string         `json:"generated_at"`
	StorePath     string         `json:"store_path"`
	ExitCode      int            `json:"exit_code"`
}

var defaultSeverities = map[string]string{
	"MQ001": "warn",
	"MQ002": "warn",
	"MQ003": "info",
	"MS001": "info",
	"MS002": "blocker",
	"ML001": "blocker",
	"MR001": "blocker",
}

func severityFor(code string) string {
	if s, ok := defaultSeverities[code]; ok {
		return s
	}
	return "warn"
}

// ---------------------------------------------------------------------------
// Typed-page gate
// ---------------------------------------------------------------------------

// The typed-page gate is isTypedMemoryPage (store.go), applied to a page the
// store handle has already read.

// ---------------------------------------------------------------------------
// Page-local linter
// ---------------------------------------------------------------------------

type memoryLinter struct {
	pagePath    string
	store       *storeHandle // the one way the checks read the store
	content     string
	frontmatter map[string]any
	sourceLine  int
	findings    []Finding
	// redactor is the store redactor's read side for MR001; nil when the
	// scanner is degraded, in which case Lint has already emitted the blocker
	// that says so and the page-local residue check stands down.
	redactor *storeRedactor
}

func newMemoryLinter(pagePath string, store *storeHandle, content string, redactor *storeRedactor) *memoryLinter {
	fm, err := parseFrontmatter(content)
	if err != nil {
		fm = map[string]any{}
	}
	return &memoryLinter{
		pagePath:    pagePath,
		store:       store,
		content:     content,
		frontmatter: fm,
		sourceLine:  frontmatterKeyLine(content, "source"),
		redactor:    redactor,
	}
}

func (l *memoryLinter) emit(code, message string, line int, suggestion string) {
	l.findings = append(l.findings, Finding{
		Code:       code,
		Severity:   severityFor(code),
		File:       l.pagePath,
		Line:       line,
		Message:    message,
		Suggestion: suggestion,
	})
}

func (l *memoryLinter) sourceBlock() map[string]any {
	if src, ok := l.frontmatter["source"].(map[string]any); ok {
		return src
	}
	return map[string]any{}
}

// derivedClasses returns every source class the page declares — the UNION of
// the scalar `source.class` and the plural `source.classes` list, scalar first,
// deduplicated. The two shapes are mutually exclusive only on the write path,
// where validateSourceBlock refuses a block that mixes them; lint reads a page's
// raw on-disk bytes by design and never runs that validator. Reading the plural
// list in an else-if therefore let it SHADOW the scalar: a page declaring
// `class: external_pdf` beside `classes: [session_memory]` presented only the
// harmless internal class to every check sharing this derivation, and slipped
// past the ML001 licence blocker. The union is the fail-closed reading, and the
// honest one — the page really does declare both.
func (l *memoryLinter) derivedClasses() []string {
	src := l.sourceBlock()
	var raw []string
	if single, ok := src["class"].(string); ok {
		raw = append(raw, single)
	}
	if classes, ok := src["classes"].([]any); ok {
		for _, c := range classes {
			if s, ok := c.(string); ok {
				raw = append(raw, s)
			}
		}
	}
	var out []string
	for _, c := range raw {
		if !contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}

func (l *memoryLinter) run() []Finding {
	l.checkResidue()
	l.checkSourceClasses()
	l.checkLicence()
	l.checkQuotation()
	return l.findings
}

// ---------------------------------------------------------------------------
// Stored-residue lint (MR001) — the read side of the store redactor
// ---------------------------------------------------------------------------

// The write side redacts every body and every leaf a write introduces
// (GHSA-j5f5, GHSA-x46m); it cannot reach text that was stored before it
// existed or planted by hand, and a store that carries a secret propagates it
// through every ask --file-back clone. MR001 is the same detector run over
// what is ALREADY on disk: it reports, and it never rewrites the store
// (adr-13 — lint judges, the operator repairs).

const residueSuggestion = "If the credential is real, rotate it. Then remove the value by hand or re-ingest the source (every leaf a write introduces is redacted on the way in); lint reports and never rewrites the store."

// residueFindings turns the store redactor's read-side verdict on one stored
// text into MR001 findings. The message carries the kind and the line, never
// the matched span: unlike scanner.Finding, whose MarshalJSON masks, a memory
// Finding is free text that lands verbatim in report.json and report.md.
func residueFindings(r *storeRedactor, text, file string) []Finding {
	var out []Finding
	for _, f := range r.residue(text, file) {
		out = append(out, Finding{
			Code: "MR001", Severity: severityFor("MR001"), File: file, Line: f.Line,
			Message:    fmt.Sprintf("stored text carries a %s span the store redactor would refuse or rewrite — a secret or identity leak committed with the memory store.", f.Kind),
			Suggestion: residueSuggestion,
		})
	}
	return out
}

func (l *memoryLinter) checkResidue() {
	if l.redactor == nil {
		return
	}
	l.findings = append(l.findings, pageNameResidue(l.redactor, filepath.Base(l.pagePath), l.pagePath, 0)...)
	l.findings = append(l.findings, residueFindings(l.redactor, l.content, l.pagePath)...)
}

const pageNameSuggestion = "If the credential is real, rotate it. Then rename the page by hand and repair every place its name is recorded — index.md, log.md and the sources registry back-link; lint reports and never rewrites the store."

// pageNameResidue is MR001 for a page NAME already in the store
// (iss-2609090642035097's read side of iss-2609020321100138). The free-text
// scan cannot see a token a name carries: '_' is a word character, so
// `topic_auth_ghp_…` has no boundary before the token and the anchored pattern
// never matches — in the page's own name, or in the registry back-link that
// repeats it. The name is therefore judged by the write side's own verdict,
// filenameHardFailKinds, which splits it into its components and its
// underscore suffixes and holds the hard_fail bar a prose-shaped name needs.
// file and line locate where the name was found; the message carries the kind,
// never the span. The finding names the file as the write-side refusal names
// the page: a report that withheld it would leave nothing to repair.
func pageNameResidue(r *storeRedactor, name, file string, line int) []Finding {
	var out []Finding
	for _, kind := range r.filenameHardFailKinds(name) {
		out = append(out, Finding{
			Code: "MR001", Severity: severityFor("MR001"), File: file, Line: line,
			Message:    fmt.Sprintf("page name carries a %s span the store redactor refuses at the write boundary — a secret committed as a file name and repeated in index.md, log.md and the sources registry back-link.", kind),
			Suggestion: pageNameSuggestion,
		})
	}
	return out
}

// storedBackLinks returns every page name the registry's back-link lists
// (`<content-hash>.consumers.<consumer>.pages`, the list registryBackLinkPath
// names on the write side) hold, deduplicated, each with the 1-based line of
// its first quoted occurrence in raw. A registry that does not parse yields
// none: its bytes are still scanned as text.
func storedBackLinks(raw []byte) []backLink {
	var reg map[string]any
	if json.Unmarshal(raw, &reg) != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []backLink
	for _, hash := range sortedKeys(reg) {
		entry, _ := reg[hash].(map[string]any)
		consumers, _ := entry["consumers"].(map[string]any)
		for _, c := range sortedKeys(consumers) {
			consumer, _ := consumers[c].(map[string]any)
			for _, name := range anyToStrings(consumer["pages"]) {
				if seen[name] {
					continue
				}
				seen[name] = true
				out = append(out, backLink{name: name, line: quotedLine(string(raw), name)})
			}
		}
	}
	return out
}

// maskBackLinks blanks every quoted back-link that is a well-formed page name
// out of the registry text before its free-text scan (iss-2609230851376499). A
// back-link is an identifier the store resolves, not acquired text: the write
// side excludes it from the leaf walk (registryBackLinkPath) and judges it as a
// filename, at the hard_fail bar, because a prose-shaped name such as
// `topic_home_migrating-off-the-nas.md` matches net_device_hostname at warn and
// the free-text bar promotes that to a blocker. The read side holds the same
// line: pageNameResidue judges each back-link, and the text scan does not see
// it. Only a name ParsePageFilename accepts is masked — its charset is bounded
// by pageNameRe, so the quoted form is its exact JSON encoding — and a
// hand-edited back-link that is not a page name stays in the text scan. The
// match is registry-wide: any quoted string byte-equal to a well-formed
// back-link is masked wherever it sits, not only inside consumers.*.pages. The
// only findings that can be hidden that way are warn-level identity and network
// kinds on bytes the write side accepts as a page name, since a hard-fail span
// in those bytes is still reported by pageNameResidue on the identical name; a
// free-text value that differs from every back-link is still scanned. The
// mask is spaces of the same length, so every other finding keeps its line.
func maskBackLinks(text string, links []backLink) string {
	for _, bl := range links {
		if _, _, _, ok := ParsePageFilename(bl.name); !ok {
			continue
		}
		text = strings.ReplaceAll(text, `"`+bl.name+`"`, `"`+strings.Repeat(" ", len(bl.name))+`"`)
	}
	return text
}

type backLink struct {
	name string
	line int
}

// quotedLine is the 1-based line of the first JSON-quoted occurrence of s in
// text, or 0 when there is none (a name the encoder escaped).
func quotedLine(text, s string) int {
	i := strings.Index(text, `"`+s+`"`)
	if i < 0 {
		return 0
	}
	return strings.Count(text[:i], "\n") + 1
}

// residueOfStoreFiles scans the store's untouched leaves — the sources
// registry and each text kept-original — which no page-local check reads.
// The derived siblings (index.md, contradictions.md, log.md) are regenerated
// from the pages by reconcile, so a page finding covers them. A binary
// kept-original (a PDF) cannot be scanned span-wise and is skipped, as the
// write side declines to rewrite it; its distilled pages are scanned instead.
func residueOfStoreFiles(r *storeRedactor, store *storeHandle) []Finding {
	var out []Finding
	const indexName = ".sources_index.json"
	index := store.path(indexName)
	if raw, err := store.read(indexName, maxRegistryBytes); err == nil {
		links := storedBackLinks(raw)
		out = append(out, residueFindings(r, maskBackLinks(string(raw), links), index)...)
		for _, bl := range links {
			out = append(out, pageNameResidue(r, bl.name, index, bl.line)...)
		}
	}
	if store.root == nil {
		return out
	}
	// ReadDir through the handle: a symlinked sources/ fails to open as a
	// directory inside the root rather than being listed through.
	if fi, err := store.root.Lstat("sources"); err != nil || fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return out
	}
	entries, err := fs.ReadDir(store.root.FS(), "sources")
	if err != nil {
		return out
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		rel := "sources/" + e.Name()
		raw, err := store.read(rel, maxFetchBytes)
		if err != nil || !isRedactableText(raw) {
			continue
		}
		out = append(out, residueFindings(r, string(raw), store.path(rel))...)
	}
	return out
}

func (l *memoryLinter) checkSourceClasses() {
	classes := l.derivedClasses()
	switch {
	case len(classes) == 1:
		l.emit("MS001",
			fmt.Sprintf("memory page synthesised from a single source class (%s) — low cross-validation (advisory).", classes[0]),
			l.sourceLine,
			"Cross-validate the page against a second source class when one becomes available; advisory only.")
	case len(classes) >= 2:
		note, ok := l.sourceBlock()["weighting_note"].(string)
		if !ok || strings.TrimSpace(note) == "" {
			l.emit("MS002",
				fmt.Sprintf("memory page mixes %d source classes (%s) without a `source.weighting_note` acknowledging the asymmetric trust gradient between class types.", len(classes), strings.Join(classes, ", ")),
				l.sourceLine,
				"Add `weighting_note: \"<text>\"` under `source:` explaining how the classes are weighted against each other.")
		}
	}
}

func hasLicence(block map[string]any) bool {
	lic, ok := block["licence"].(string)
	return ok && strings.TrimSpace(lic) != ""
}

func (l *memoryLinter) checkLicence() {
	src := l.sourceBlock()
	// Classes accounted for by a per-source entry: those carry their licence on
	// the entry, so the page-level pass below must not demand a second one.
	var covered []string
	if sources, ok := src["sources"].([]any); ok {
		covered = deriveClasses(sources)
		for idx, e := range sources {
			em, ok := e.(map[string]any)
			if !ok {
				continue
			}
			if isExternalClass(em["class"]) && !hasLicence(em) {
				l.emit("ML001",
					fmt.Sprintf("source.sources[%d] (class %v) has no per-source `licence` field; explicit `licence: unknown` is acceptable — missing is the violation.", idx, em["class"]),
					l.sourceLine,
					fmt.Sprintf("Add `licence: <spdx-id|declared-by-user|unknown>` to source.sources[%d].", idx))
			}
		}
	}
	// Derive the page's classes the way every sibling check does (MS001/MS002,
	// SourceClasses, the index rendering): the scalar `class` AND the plural
	// `classes` list. Reading only the scalar let an external_* page declared
	// `classes: [external_pdf]` with no licence pass the blocker unseen, while
	// MS001 in the same run asserted the page carried an external class.
	//
	// This pass runs even when a `sources` list is present. The two shapes are
	// mutually exclusive only on the write path (validateSourceBlock), which the
	// lint path never runs — so returning out of the branch above let a page
	// carrying a scalar `class: external_pdf` alongside an empty or junk
	// `sources: []` skip the page-level check entirely.
	for _, cls := range l.derivedClasses() {
		if contains(covered, cls) {
			continue
		}
		if isExternalClass(cls) && !hasLicence(src) {
			l.emit("ML001",
				fmt.Sprintf("memory page with source class `%s` has no `licence` field; explicit `licence: unknown` is acceptable — missing is the violation.", cls),
				l.sourceLine,
				"Add `licence: <spdx-id|declared-by-user|unknown>` under `source:`.")
		}
	}
}

func (l *memoryLinter) checkQuotation() {
	spans := extractQuotedSpans(l.content)
	if len(spans) == 0 {
		return
	}
	src := l.sourceBlock()
	externals := externalSourceHashes(src)
	if len(externals) == 0 {
		l.emit("MQ003",
			"[reason: no_external_source] page has quoted spans but no `external_*` source to budget — quotation budget skipped (coverage unavailable).",
			l.sourceLine, "")
		return
	}
	budget := loadQuotationBudget(l.store)
	for _, span := range spans {
		if span.tokenCount > budget.MaxContiguousQuoteWords {
			l.emit("MQ001",
				fmt.Sprintf("contiguous quoted span of %d words exceeds the %d-word ceiling.", span.tokenCount, budget.MaxContiguousQuoteWords),
				span.line,
				"Summarise the passage in your own words; quote only the load-bearing fragment.")
		}
	}
	pageTokens := pageQuotedTokenTotal(spans)
	registry, err := l.store.registry()
	if err != nil {
		registry = nil // corrupt index -> every lookup degrades to malformed
	}
	for _, sh := range externals {
		tokenCount, reason := lookupSourceTokenCount(registry, sh)
		if reason != "" {
			fix := "Re-ingest the source (or backfill `source_token_count` in `.sources_index.json`)."
			if reason == reasonMalformedEntry {
				fix = "Repair the corrupt registry entry in `.abcd/memory/.sources_index.json`."
			}
			l.emit("MQ003",
				fmt.Sprintf("[reason: %s] coverage unavailable for source %s — per-page quotation budget skipped for this source.", reason, short12(sh)),
				l.sourceLine, fix)
			continue
		}
		pct := float64(pageTokens) / float64(tokenCount)
		if pct > budget.PerPagePct {
			l.emit("MQ001",
				fmt.Sprintf("page quotes %s of source %s (%d/%d tokens) — over the %.0f%% per-page budget.", fmtPct(pct), short12(sh), pageTokens, tokenCount, budget.PerPagePct*100),
				l.sourceLine,
				"Summarise quoted passages in your own words until the page is back under the per-page budget.")
		}
	}
}

// ---------------------------------------------------------------------------
// Full-corpus coverage lint (MQ002 + per-source MQ003)
// ---------------------------------------------------------------------------

func runMemoryCoverageLint(repoRoot string, store *storeHandle) ([]Finding, map[string]any, error) {
	indexPath := CoverageIndexPath(repoRoot)
	report := map[string]any{
		"path":            indexPath,
		"stale":           false,
		"old_fingerprint": nil,
		"new_fingerprint": nil,
		"written":         false,
	}
	// The caller's store handle refused a symlinked store DIRECTORY when it was
	// opened (GHSA-72rp), and every read below and the index write resolve
	// inside the directory it vetted (iss-2608291814572914,
	// iss-2609252100150846), so a store swapped after the open can neither feed
	// the coverage nor receive the index. It is the handle the page lint read
	// through: one Lint opens the store once.
	if !store.present() {
		return nil, report, nil
	}
	pages := store.typedPages()

	budget := loadQuotationBudget(store)
	registry, regErr := store.registry()
	if regErr != nil {
		registry = nil
	}
	result := buildCoverage(pages, registry, budget)

	oldFP := readStoredFingerprint(store)
	if _, err := writeCoverageIndex(store, result, budget); err != nil {
		return nil, report, err
	}
	report["stale"] = oldFP != "" && oldFP != result.fingerprint
	if oldFP != "" {
		report["old_fingerprint"] = oldFP
	}
	report["new_fingerprint"] = result.fingerprint
	report["written"] = true

	var findings []Finding
	unavailKeys := make([]string, 0, len(result.unavailable))
	for sh := range result.unavailable {
		unavailKeys = append(unavailKeys, sh)
	}
	sort.Strings(unavailKeys)
	for _, sh := range unavailKeys {
		reason := result.unavailable[sh]
		fix := "Re-ingest the source (or backfill `source_token_count` in `.sources_index.json`)."
		if reason == reasonMalformedEntry {
			fix = "Repair the corrupt registry entry in `.abcd/memory/.sources_index.json`."
		}
		findings = append(findings, Finding{
			Code: "MQ003", Severity: "info", File: indexPath,
			Message:    fmt.Sprintf("[reason: %s] cumulative coverage unavailable for source %s — MQ002 skipped for this source.", reason, short12(sh)),
			Suggestion: fix,
		})
	}

	srcKeys := make([]string, 0, len(result.sources))
	for sh := range result.sources {
		srcKeys = append(srcKeys, sh)
	}
	sort.Strings(srcKeys)
	warnPct := budget.CumulativeWarnPct
	blockPct := budget.CumulativeBlockPct
	for _, sh := range srcKeys {
		cov := result.sources[sh]
		pagesStr := strings.Join(cov.Pages, ", ")
		switch {
		case cov.CoverageUnambiguous >= blockPct:
			findings = append(findings, Finding{
				Code: "MQ002", Severity: "blocker", File: indexPath,
				Message: fmt.Sprintf("cumulative quoted coverage of source %s is %s — unambiguous single-source attribution alone is %s, at or over the %.0f%% block threshold. Pages: %s.",
					short12(sh), fmtPct(cov.CoverageTotal), fmtPct(cov.CoverageUnambiguous), blockPct*100, pagesStr),
				Suggestion: "Rewrite quoted passages as summaries until coverage drops below the threshold.",
			})
		case cov.CoverageTotal >= blockPct:
			findings = append(findings, Finding{
				Code: "MQ002", Severity: "warn", File: indexPath,
				Message: fmt.Sprintf("cumulative quoted coverage of source %s is %s (>= the %.0f%% block threshold) but is driven by AMBIGUOUS multi-source attribution (unambiguous: %s) — capped at warn; only unambiguous single-source coverage can block. Pages: %s.",
					short12(sh), fmtPct(cov.CoverageTotal), blockPct*100, fmtPct(cov.CoverageUnambiguous), pagesStr),
				Suggestion: "Review the multi-source pages quoting this source; rewrite quotes as summaries to reduce coverage.",
			})
		case cov.CoverageTotal >= warnPct:
			findings = append(findings, Finding{
				Code: "MQ002", Severity: "warn", File: indexPath,
				Message: fmt.Sprintf("cumulative quoted coverage of source %s is %s — in the warn band [%.0f%%, %.0f%%). Pages: %s.",
					short12(sh), fmtPct(cov.CoverageTotal), warnPct*100, blockPct*100, pagesStr),
				Suggestion: "Rewrite quoted passages as summaries before coverage reaches the block threshold.",
			})
		}
	}
	return findings, report, nil
}

// ---------------------------------------------------------------------------
// Lint orchestration
// ---------------------------------------------------------------------------

// Lint runs the full-store curator health-check and writes one run-log report.
// Mutates no memory-store state (only the regenerable coverage index + report).
func Lint(req LintRequest) (LintResult, error) {
	root := req.RepoRoot
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	// The store handle refuses a symlinked store DIRECTORY before any crawl or
	// coverage write (GHSA-72rp), and every read below goes through it
	// (iss-2608291814572914).
	store, err := openStore(root)
	if err != nil {
		return LintResult{}, err
	}
	defer store.Close()
	mem := store.dir

	var findings []Finding
	if store.present() {
		// The store redactor's read side (GHSA-xj89-cc2c-wgwr). A degraded
		// scanner is a blocker finding against the store rather than an error:
		// lint's contract is to always crawl and write its report, and the exit
		// is nonzero either way.
		redactor, rerr := openStoreRedactor(root)
		if rerr != nil {
			findings = append(findings, Finding{
				Code: "MR001", Severity: severityFor("MR001"), File: mem,
				Message:    "secret scan unavailable (" + rerr.Error() + "): stored pages, the sources registry and kept originals were not checked for residue.",
				Suggestion: "Repair or remove the per-repo scanner override at .abcd/config/pii.json and re-run lint.",
			})
		}
		for _, p := range store.typedPages() {
			findings = append(findings, newMemoryLinter(store.path(p.rel), store, p.text, redactor).run()...)
		}
		if redactor != nil {
			findings = append(findings, residueOfStoreFiles(redactor, store)...)
		}
	}

	corpusFindings, coverageReport, err := runMemoryCoverageLint(root, store)
	if err != nil {
		return LintResult{}, err
	}
	findings = append(findings, corpusFindings...)

	summary := LintSummary{}
	for _, f := range findings {
		switch f.Severity {
		case "blocker":
			summary.Blockers++
		case "warn":
			summary.Warnings++
		case "info":
			summary.Infos++
		}
	}
	exitCode := 0
	if summary.Blockers > 0 {
		exitCode = 1
	}
	generatedAt := now.Format("2006-01-02T15:04:05Z")
	coverageIndex := map[string]any{}
	if coverageReport != nil {
		coverageIndex = coverageReport
	}

	reportDir, err := lintReportDir(root, now)
	if err != nil {
		return LintResult{}, err
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return LintResult{}, err
	}
	reportFields := map[string]any{
		"findings":       findingsToMaps(findings),
		"summary":        map[string]any{"blockers": summary.Blockers, "warnings": summary.Warnings, "infos": summary.Infos},
		"coverage_index": coverageIndex,
		"generated_at":   generatedAt,
		"store_path":     mem,
	}
	if err := writeStringAtomic(filepath.Join(reportDir, "report.json"), marshalIndentNoEscape(reportFields)); err != nil {
		return LintResult{}, err
	}
	if err := writeStringAtomic(filepath.Join(reportDir, "report.md"), renderLintReportMD(reportFields)); err != nil {
		return LintResult{}, err
	}

	return LintResult{
		Findings:      findings,
		Summary:       summary,
		CoverageIndex: coverageIndex,
		ReportDir:     reportDir,
		GeneratedAt:   generatedAt,
		StorePath:     mem,
		ExitCode:      exitCode,
	}, nil
}

func lintReportDir(repoRoot string, now time.Time) (string, error) {
	ts := now.Format("20060102T150405.000000Z")
	// Runtime artefacts live in the gitignored .abcd/.work.local/logs/ tier, not
	// the retired runtime location (iss-36/iss-56 adjudication, iss-73).
	logs := filepath.Join(repoRoot, ".abcd", ".work.local", "logs", "memory")
	base := filepath.Join(logs, "lint-"+ts)
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base, nil
	}
	for n := 1; n < 1000; n++ {
		candidate := filepath.Join(logs, fmt.Sprintf("lint-%s-%03d", ts, n))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not allocate a unique lint run-log dir for %s", ts)
}

func findingsToMaps(findings []Finding) []any {
	out := make([]any, len(findings))
	for i, f := range findings {
		out[i] = map[string]any{
			"code": f.Code, "severity": f.Severity, "file": f.File,
			"line": f.Line, "message": f.Message, "suggestion": f.Suggestion,
		}
	}
	return out
}

// LintReportHeading heads the run-log report, on the same rule as
// AskReportHeading: core names the binary invocation, never one front door.
const LintReportHeading = "abcd memory lint"

// renderLintReportMD renders report.md, the local-tier file an operator opens in
// a pager. Every free-text field — the store path, each finding's file, message
// and suggestion — goes through termsafe.Sanitize, the primitive the CLI render
// applies to the same findings: a degraded-scanner MR001 message carries a
// pattern name read from the per-repo pii.json, and a finding's file is a name
// the store holds, so either can carry a control sequence (iss-2609020239068243).
func renderLintReportMD(fields map[string]any) string {
	summary, _ := fields["summary"].(map[string]any)
	cov, _ := fields["coverage_index"].(map[string]any)
	lines := []string{
		"# " + LintReportHeading + " — curator health-check",
		"",
		fmt.Sprintf("Generated: %v", fields["generated_at"]),
		"Store: " + termsafe.Sanitize(fmt.Sprintf("%v", fields["store_path"])),
		fmt.Sprintf("Summary: %d blocker(s), %d warning(s), %d info(s)",
			toInt(summary["blockers"]), toInt(summary["warnings"]), toInt(summary["infos"])),
	}
	if written, _ := cov["written"].(bool); written {
		newFP, _ := cov["new_fingerprint"].(string)
		if stale, _ := cov["stale"].(bool); stale {
			oldFP, _ := cov["old_fingerprint"].(string)
			lines = append(lines, fmt.Sprintf("Coverage index: rebuilt (fingerprint %s); drift detected against the stored index (was %s) — the fresh crawl is authoritative.", short12(newFP), short12(oldFP)))
		} else {
			lines = append(lines, fmt.Sprintf("Coverage index: rebuilt (fingerprint %s); no drift against the stored index.", short12(newFP)))
		}
	} else {
		lines = append(lines, "Coverage index: not written (no memory store present).")
	}
	lines = append(lines, "Exit contract: blockers exit nonzero; warnings are curator-advisory (exit 0, non-blocking); infos never affect exit.", "")
	findings, _ := fields["findings"].([]any)
	if len(findings) == 0 {
		lines = append(lines, "No findings — store is clean.")
	} else {
		lines = append(lines, "## Findings", "")
		for _, sev := range []string{"blocker", "warn", "info"} {
			for _, fa := range findings {
				f, _ := fa.(map[string]any)
				if f["severity"] != sev {
					continue
				}
				loc := termsafe.Sanitize(fmt.Sprintf("%v", f["file"]))
				if line := toInt(f["line"]); line != 0 {
					loc += fmt.Sprintf(":%d", line)
				}
				lines = append(lines, fmt.Sprintf("- [%s] %v %s — %s", sev, f["code"], loc, termsafe.Sanitize(fmt.Sprintf("%v", f["message"]))))
				if sug, _ := f["suggestion"].(string); sug != "" {
					lines = append(lines, "  fix: "+termsafe.Sanitize(sug))
				}
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func short12(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
