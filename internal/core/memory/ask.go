package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// ask.go — deterministic native recall (fn-38 .6). Retrieval (QueryPages) is
// read-only token-overlap ranking; synthesis defaults to RenderCitedMatches (no
// LLM). The optional file-back (default OFF) routes through the SAME dedup +
// WritePages seams as ingest.

// AskTopN is the pinned default retrieval depth.
const AskTopN = 5

var askFilterRe = regexp.MustCompile(`(?i)\b(class|domain):([A-Za-z0-9_-]+)`)

const filedBy = "abcd memory ask --file-back"

// AskCitation carries the per-source provenance facts of one matched page.
// Fields are "" when absent (a legacy / backfilled page), never invented.
type AskCitation struct {
	SourceClass string         `json:"source_class"`
	Citation    map[string]any `json:"citation"`
	SourceHash  string         `json:"source_hash"`
	Licence     string         `json:"licence"`
	IngestedAt  string         `json:"ingested_at"`
}

// MatchedPage is one retrieval hit.
type MatchedPage struct {
	Filename  string        `json:"filename"`
	Score     int           `json:"score"`
	Classes   []string      `json:"classes"`
	Domain    string        `json:"domain"`
	Summary   string        `json:"summary"`
	Body      string        `json:"body"`
	Citations []AskCitation `json:"citations"`
}

// Synthesizer turns matches into answer prose; nil uses RenderCitedMatches.
type Synthesizer func(question string, matches []MatchedPage) string

// FileBackDecision is consulted after validation and before any write; false
// declines (no partial write).
type FileBackDecision func(page DistilledPage) bool

// FileBackResult records what the file-back branch did.
type FileBackResult struct {
	Status         string       `json:"status"`
	Pages          []string     `json:"pages"`
	Linked         [][2]string  `json:"linked"`
	Contradictions [][2]string  `json:"contradictions"`
	WriteReport    *WriteReport `json:"write_report"`
}

// AskRequest is the input to Ask.
type AskRequest struct {
	RepoRoot       string
	Question       string
	TopN           int
	Synthesizer    Synthesizer
	FileBackPage   map[string]any
	DecideFileBack FileBackDecision
	Now            time.Time
}

// AskResult is the structured result of one Ask call.
type AskResult struct {
	Question string          `json:"question"`
	Matches  []MatchedPage   `json:"matches"`
	Answer   string          `json:"answer"`
	FileBack *FileBackResult `json:"file_back"`
}

// Ask runs deterministic retrieval -> synthesis -> optional file-back.
func Ask(req AskRequest) (AskResult, error) {
	root := req.RepoRoot
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	topN := req.TopN
	if topN == 0 {
		topN = AskTopN
	}
	matches, err := QueryPages(root, req.Question, topN)
	if err != nil {
		return AskResult{}, err
	}
	// The question is argv — untrusted terminal input — and it is echoed by
	// every render and carried in AskResult.Question, the --json field. It is
	// sanitised ONCE here and that one value feeds both return paths, so the
	// empty-store branch cannot drift from the matched branch again: the
	// no-matches render used to take the raw question while RenderCitedMatches
	// sanitised its own copy, and a raw ESC/C1/bidi rune reached stdout from
	// exactly the branch a first-time user hits (GHSA-4fmm-95pf-32c6).
	// Retrieval above still tokenises the raw question — masking runes to '?'
	// before the overlap ranking would change what a question matches.
	question := termsafe.Sanitize(req.Question)
	if len(matches) == 0 {
		if req.FileBackPage != nil {
			return AskResult{}, newAskError("no matching memory pages — a file-back without cited matches would write an unattributable page; nothing was written")
		}
		return AskResult{Question: question, Matches: nil, Answer: RenderNoMatches(question)}, nil
	}
	synth := req.Synthesizer
	if synth == nil {
		synth = RenderCitedMatches
	}
	answer := synth(question, matches)
	if strings.TrimSpace(answer) == "" {
		return AskResult{}, newAskError("synthesizer returned no answer text")
	}
	var fb *FileBackResult
	if req.FileBackPage != nil {
		result, err := fileBack(root, matches, req.FileBackPage, req.DecideFileBack, now)
		if err != nil {
			return AskResult{}, err
		}
		fb = &result
	}
	return AskResult{Question: question, Matches: matches, Answer: answer, FileBack: fb}, nil
}

func parseQuestion(question string) ([]string, string, string) {
	var classFilter, domainFilter string
	remainder := askFilterRe.ReplaceAllStringFunc(question, func(m string) string {
		sub := askFilterRe.FindStringSubmatch(m)
		if strings.EqualFold(sub[1], "class") {
			classFilter = strings.ToLower(sub[2])
		} else {
			domainFilter = strings.ToLower(sub[2])
		}
		return " "
	})
	set := map[string]bool{}
	for _, t := range wordRe.FindAllString(remainder, -1) {
		set[strings.ToLower(t)] = true
	}
	tokens := make([]string, 0, len(set))
	for t := range set {
		tokens = append(tokens, t)
	}
	sort.Strings(tokens)
	return tokens, classFilter, domainFilter
}

func citationsFromSource(source map[string]any) []AskCitation {
	one := func(entry map[string]any) AskCitation {
		cite := AskCitation{}
		if s, ok := entry["class"].(string); ok {
			cite.SourceClass = s
		}
		if c, ok := entry["citation"].(map[string]any); ok {
			cite.Citation = deepCopyMap(c)
		} else {
			cite.Citation = map[string]any{}
		}
		if s, ok := entry["source_hash"].(string); ok {
			cite.SourceHash = s
		}
		if s, ok := entry["licence"].(string); ok {
			cite.Licence = s
		}
		if s, ok := entry["ingested_at"].(string); ok {
			cite.IngestedAt = s
		}
		return cite
	}
	if entries, ok := source["sources"].([]any); ok {
		var cites []AskCitation
		for _, e := range entries {
			if em, ok := e.(map[string]any); ok {
				cites = append(cites, one(em))
			}
		}
		if len(cites) > 0 {
			return cites
		}
	}
	return []AskCitation{one(source)}
}

// QueryPages is the read-only deterministic retrieval: tokenise the question,
// score by token overlap against each page's index-line facts, apply optional
// class:/domain: filters, rank by overlap (filename tie-break), take top-N.
func QueryPages(repoRoot, question string, topN int) ([]MatchedPage, error) {
	tokens, classFilter, domainFilter := parseQuestion(question)
	tokenSet := map[string]bool{}
	for _, t := range tokens {
		tokenSet[t] = true
	}
	// Refuse a symlinked store DIRECTORY up front (GHSA-72rp): the leaf-guarded
	// reads below only bind the leaf, so a committed `.abcd/memory` symlink would
	// otherwise be walked and its out-of-repo pages disclosed.
	mem, present, err := safeMemoryDir(repoRoot)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, nil
	}
	entries, err := os.ReadDir(mem)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var matches []MatchedPage
	for _, e := range entries {
		if !e.Type().IsRegular() || !IsMemoryPageName(e.Name()) {
			continue
		}
		// ReadGuarded re-checks regular-file on the open fd (closing the
		// ReadDir→open symlink-swap TOCTOU) and caps the size.
		raw, err := fsutil.ReadGuarded(filepath.Join(mem, e.Name()), maxMemoryPageBytes)
		if err != nil {
			continue
		}
		text := string(raw)
		page := parsePage(text)
		info := pageInfoOf(e.Name(), page)
		if classFilter != "" {
			ok := false
			for _, c := range info.Classes {
				if strings.ToLower(c) == classFilter {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if domainFilter != "" && strings.ToLower(info.Domain) != domainFilter {
			continue
		}
		haystack := strings.Join(append(append([]string{}, info.Classes...), info.Domain, info.Summary), " ")
		pageTokens := map[string]bool{}
		for _, t := range wordRe.FindAllString(haystack, -1) {
			pageTokens[strings.ToLower(t)] = true
		}
		score := 0
		for t := range tokenSet {
			if pageTokens[t] {
				score++
			}
		}
		filterOnly := len(tokens) == 0 && (classFilter != "" || domainFilter != "")
		if score == 0 && !filterOnly {
			continue
		}
		matches = append(matches, MatchedPage{
			Filename:  e.Name(),
			Score:     score,
			Classes:   info.Classes,
			Domain:    info.Domain,
			Summary:   info.Summary,
			Body:      page.body,
			Citations: citationsFromSource(page.source()),
		})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		return matches[i].Filename < matches[j].Filename
	})
	if topN < 0 {
		topN = 0
	}
	if topN < len(matches) {
		matches = matches[:topN]
	}
	return matches, nil
}

// AskReportHeading heads every ask render. It is the BINARY invocation, not the
// `/abcd:memory ask` slash command, because this is core: the same bytes are
// rendered whether the caller came through the plugin surface, the bare CLI, or
// a later transport, and a heading naming one front door is a falsehood in the
// other two (iss-44). A reader who ran `abcd memory ask` in a terminal must not
// be told they invoked a plugin command they may not even have installed.
const AskReportHeading = "abcd memory ask"

// RenderCitedMatches is the default deterministic synthesizer — a
// citation-renderer, not an LLM. Missing provenance renders as explicit (none).
func RenderCitedMatches(question string, matches []MatchedPage) string {
	lines := []string{
		"# " + AskReportHeading + " — " + termsafe.Sanitize(question),
		"",
		fmt.Sprintf("Matched pages (%d, overlap-ranked):", len(matches)),
		"",
	}
	for _, m := range matches {
		// Filename and Summary are page-derived (repo content); sanitise each field
		// before it joins the multi-line answer — masking the whole answer wholesale
		// would clobber its legitimate newlines.
		summary := termsafe.Sanitize(m.Summary)
		if summary == "" {
			summary = "(no summary)"
		}
		lines = append(lines, fmt.Sprintf("- `%s` (score %d) — %s", termsafe.Sanitize(m.Filename), m.Score, summary))
		for _, c := range m.Citations {
			// Every citation field is page-derived content from the same untrusted
			// ingest boundary as Summary/Filename above, so each is sanitised before
			// it joins the answer. encoding/json (compactJSONSorted) escapes only C0
			// and a couple of runes — C1 (U+009B), bidi overrides (U+202E) and
			// zero-width runes reach the terminal raw from the citation JSON unless
			// masked here (gh-250). class/source_hash are charset-constrained upstream,
			// but sanitising them too matches the sibling treatment and defends the
			// render even if that constraint ever weakens.
			cls := termsafe.Sanitize(c.SourceClass)
			if cls == "" {
				cls = "(none)"
			}
			sh := termsafe.Sanitize(c.SourceHash)
			if sh == "" {
				sh = "(none)"
			}
			cj := "(none)"
			if len(c.Citation) > 0 {
				cj = termsafe.Sanitize(compactJSONSorted(c.Citation))
			}
			lines = append(lines, fmt.Sprintf("  - cites: class=%s | source_hash=%s | citation=%s", cls, sh, cj))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// RenderNoMatches is the explicit empty-result render. It sanitises the
// question itself, as RenderCitedMatches does, so a direct caller is covered
// and the two renders cannot disagree on what reaches the terminal; Sanitize
// is idempotent, so the copy Ask already masked costs nothing here.
func RenderNoMatches(question string) string {
	return "# " + AskReportHeading + " — " + termsafe.Sanitize(question) + "\n\n" +
		"No matching memory pages (token overlap found nothing; an empty or absent store matches nothing).\n" +
		"Try different terms, an explicit class:<source-class> / domain:<domain> filter, or ingest a source first.\n"
}

// ---------------------------------------------------------------------------
// File-back (the one mutating branch of ask)
// ---------------------------------------------------------------------------

func fileBackSource(matches []MatchedPage, ingestedAt string) (map[string]any, error) {
	var entries []map[string]any
	seen := map[string]bool{}
	for _, m := range matches {
		for _, c := range m.Citations {
			if c.SourceClass == "" || len(c.Citation) == 0 || c.Licence == "" || c.SourceHash == "" || c.IngestedAt == "" {
				continue
			}
			if seen[c.SourceHash] {
				continue
			}
			seen[c.SourceHash] = true
			entries = append(entries, map[string]any{
				"class": c.SourceClass, "citation": deepCopyMap(c.Citation),
				"licence": c.Licence, "source_hash": c.SourceHash, "ingested_at": c.IngestedAt,
			})
		}
	}
	if len(entries) == 0 {
		return nil, newAskError("file-back needs provenance: none of the cited pages carries a complete source entry (class + citation + licence + source_hash + ingested_at) — supply an explicit source block on the file-back page")
	}
	if len(entries) == 1 {
		e := entries[0]
		return buildSingleSource(e["class"].(string), e["citation"].(map[string]any), e["licence"].(string), e["source_hash"].(string), e["ingested_at"].(string))
	}
	classes := deriveClasses(toAnyMaps(entries))
	note := ""
	if len(classes) >= 2 {
		note = fmt.Sprintf("filed back by %s on %s: the synthesis cites sources across classes %s; weighting follows the cited pages' own provenance", filedBy, ingestedAt, strings.Join(classes, " + "))
	}
	return buildMultiSource(entries, note)
}

func fileBack(root string, matches []MatchedPage, rawPage map[string]any, decide FileBackDecision, now time.Time) (FileBackResult, error) {
	ingestedAt := now.Format("2006-01-02")
	if _, ok := rawPage["source"]; !ok {
		src, err := fileBackSource(matches, ingestedAt)
		if err != nil {
			return FileBackResult{}, err
		}
		merged := map[string]any{}
		for k, v := range rawPage {
			merged[k] = v
		}
		merged["source"] = src
		rawPage = merged
	}
	page, err := ValidateDistilledPage(rawPage)
	if err != nil {
		return FileBackResult{}, err
	}
	if decide != nil && !decide(page) {
		return FileBackResult{Status: "declined"}, nil
	}

	mem := Dir(root)
	existing := existingPageFrontmatter(mem)
	plan, err := ResolveDistilledPages(existing, []DistilledPage{page})
	if err != nil {
		return FileBackResult{}, err
	}

	registry, err := LoadRegistry(SourcesIndexPath(root))
	if err != nil {
		return FileBackResult{}, err
	}
	sourceMeta := map[string]map[string]any{}
	var metaEntries []map[string]any
	if _, ok := page.Source["class"]; ok {
		metaEntries = []map[string]any{page.Source}
	} else if raw, ok := page.Source["sources"].([]any); ok {
		for _, e := range raw {
			if em, ok := e.(map[string]any); ok {
				metaEntries = append(metaEntries, em)
			}
		}
	}
	for _, e := range metaEntries {
		if h, ok := e["source_hash"].(string); ok {
			if _, exists := sourceMeta[h]; !exists {
				sourceMeta[h] = e
			}
		}
	}

	// Capture the ingest events for the cited hashes that already have a
	// registry entry. The actual merge is re-run under the store lock against
	// the freshly-read registry (lost-update fix) — never against this pre-lock
	// snapshot. origin/licence read here only seed MergeIngest, which preserves
	// any non-empty value already on the locked entry.
	var events []IngestEvent
	var planHashes []string
	for h := range plan.RegistryPages {
		planHashes = append(planHashes, h)
	}
	sort.Strings(planHashes)
	for _, contentHash := range planHashes {
		oldEntry, ok := registry[contentHash].(map[string]any)
		if !ok {
			continue
		}
		meta := sourceMeta[contentHash]
		class := "session_memory"
		citation := map[string]any{}
		if meta != nil {
			if c, ok := meta["class"].(string); ok && c != "" {
				class = c
			}
			if c, ok := meta["citation"].(map[string]any); ok {
				citation = c
			}
		}
		origin := ""
		if o, ok := oldEntry["origin"].(string); ok {
			origin = o
		}
		licence := "unknown"
		if l, ok := oldEntry["licence"].(string); ok && l != "" {
			licence = l
		}
		events = append(events, IngestEvent{
			ContentHash: contentHash, Consumer: "memory", SourceClass: class,
			Citation: citation, Origin: origin, Licence: licence, IngestedAt: ingestedAt,
			Pages: plan.RegistryPages[contentHash],
		})
	}

	status := "written"
	if len(plan.Linked) > 0 && len(plan.Writes) == 0 {
		status = "linked"
	}
	var report *WriteReport
	if len(plan.Writes) > 0 || len(events) > 0 {
		var pageWrites []PageWrite
		for _, w := range plan.Writes {
			pageWrites = append(pageWrites, PageWrite{Filename: w.Filename, Frontmatter: w.Frontmatter, Body: w.Body})
		}
		var merge RegistryMerge
		if len(events) > 0 {
			merge = func(current map[string]any) (map[string]any, error) {
				nr := current
				for _, ev := range events {
					merged, err := MergeIngest(nr, ev)
					if err != nil {
						return nil, err
					}
					nr = merged
				}
				return nr, nil
			}
		}
		r, err := WritePages(root, pageWrites, merge, now)
		if err != nil {
			return FileBackResult{}, err
		}
		report = &r
	}

	var pages []string
	if len(plan.Writes) > 0 {
		for _, w := range plan.Writes {
			pages = append(pages, w.Filename)
		}
	} else {
		seen := map[string]bool{}
		for _, l := range plan.Linked {
			if !seen[l[1]] {
				seen[l[1]] = true
				pages = append(pages, l[1])
			}
		}
	}
	return FileBackResult{
		Status: status, Pages: pages, Linked: plan.Linked,
		Contradictions: plan.Contradictions, WriteReport: report,
	}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func compactJSONSorted(m map[string]any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(m)
	return strings.TrimRight(buf.String(), "\n")
}

func toAnyMaps(entries []map[string]any) []any {
	out := make([]any, len(entries))
	for i, e := range entries {
		out[i] = e
	}
	return out
}
