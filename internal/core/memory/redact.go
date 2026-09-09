package memory

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// redact.go — the write-time secret/PII sanitiser for the committed
// .abcd/memory/ store (GHSA-j5f5-phgm-9m73).
//
// capture, history and launch all route free text through the shared
// internal/adapter/scanner before it lands in a committed artefact; memory
// ingest did not. Ingest acquires a local file or URL, distils it, and writes
// page bodies (and, under --keep-original, the raw source bytes) under
// .abcd/memory/ — a tree .gitignore does NOT exclude, so an operator commits
// whatever it holds. A PAT or an absolute home path in the source therefore
// reached the repository with every secret-pattern lint gate green, because
// those gates are not this package.
//
// The fix reuses the ONE canonical detector (no second scanner) and mirrors
// history.Capture's fail-closed, two-stage discipline: refuse on a degraded
// scanner, redact-and-report through scanner.Redact, apply a literal $HOME
// backstop, then re-scan and refuse if a blocking span survived. Every
// acquired-text write passes through here before it is written: page BODIES
// inside WritePages, the one primitive every verb (ingest, ask --file-back)
// writes through, so no body lands unscanned whichever verb built it; the
// --keep-original copy in Ingest; and, transitively, since they are derived
// from the redacted bodies, index.md and log.md. Page FRONTMATTER and the
// registry leaves a write introduces go through the same detector by way of
// redactLeaves / redactRegistryLeaves (GHSA-x46m-mw9h-5jwj,
// iss-2608291941064448): a host-supplied
// citation title or recall entry, the licence the core lifts from an SPDX line
// or a License: header, and a redirect-controlled origin are acquired text as
// much as a body is, and the one place every verb writes through is where they
// are judged. contradictions.md is derived from the redacted frontmatter.

// storeRedactor holds a per-repo scanner plus the caller's resolved $HOME for
// the deterministic literal backstop. Construct it with newStoreRedactor, which
// fails closed on a degraded scanner exactly as history.Capture does.
type storeRedactor struct {
	sc   *scanner.Scanner
	home string
}

// openStoreRedactor builds the shared scanner for repoRoot and says, as a plain
// error, why it cannot be trusted: init failed, or the per-repo pii.json left
// it degraded. ScanText/Redact cannot signal the unavailable state in-band
// (only ScanBundle does), so a caller that skipped this check would sanitise
// with a silently weakened pattern set — the exact fail-open this closes. The
// write side (newStoreRedactor) turns that error into a refusal; the lint
// (GHSA-xj89-cc2c-wgwr) turns it into a blocker finding, because its contract
// is to always crawl and write its report.
func openStoreRedactor(repoRoot string) (*storeRedactor, error) {
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("scanner init failed: %v", err)
	}
	if unavail, reason := sc.Unavailable(); unavail {
		return nil, fmt.Errorf("degraded scanner: %s", reason)
	}
	return &storeRedactor{sc: sc, home: scanner.CallerHome()}, nil
}

// newStoreRedactor is the write side of openStoreRedactor: it refuses the whole
// ingest on a broken per-repo pii.json, matching history's posture — a
// committed store must never be written with a detector known to be weakened.
func newStoreRedactor(repoRoot string) (*storeRedactor, error) {
	r, err := openStoreRedactor(repoRoot)
	if err != nil {
		return nil, newIngestError("refusing to ingest: %v", err)
	}
	return r, nil
}

// residueHomeKind labels the literal-home backstop's finding. It mirrors the
// scanner's own kind for the caller's home, so a report reads the same
// whichever detector saw the path.
const residueHomeKind = "home_path_self"

// residue is the read side of redactText, for text ALREADY in the store: the
// spans the write side would refuse or rewrite — every blocking finding of the
// canonical scanner (a hard_fail secret, any identity or network span whatever
// its severity) plus the deterministic literal-home backstop, reported per
// line and deduplicated against a scanner finding of the same kind on that
// line. It rewrites nothing: the lint reports and the operator repairs
// (GHSA-xj89-cc2c-wgwr). A finding carries the kind and the line; the caller
// must never put its Matched span into free text.
func (r *storeRedactor) residue(text, label string) []scanner.Finding {
	out := scanner.BlockingResidual(r.sc.ScanText(text, label))
	if r.home == "" {
		return out
	}
	seen := map[int]bool{}
	for _, f := range out {
		if f.Kind == residueHomeKind {
			seen[f.Line] = true
		}
	}
	for i, line := range strings.Split(text, "\n") {
		if seen[i+1] || scanner.SweepCallerHome(line, r.home) == line {
			continue
		}
		out = append(out, scanner.Finding{File: label, Line: i + 1, Kind: residueHomeKind, Matched: "~"})
	}
	return out
}

// redactText sanitises one free-text blob bound for the store. label is the
// logical name stamped into each finding. It returns the redacted text and the
// number of spans rewritten, or an IngestError when a blocking span (any
// identity/network span whatever its severity, plus every hard_fail one)
// survives redaction — the same fail-closed gate history.Capture applies.
func (r *storeRedactor) redactText(text, label string) (string, int, error) {
	findings := r.sc.ScanText(text, label)
	redacted, _ := scanner.Redact(text, findings)

	// Deterministic literal $HOME backstop, independent of the scanner
	// heuristic (defence in depth on this trust boundary): collapse every
	// remaining occurrence of the resolved home that stands as a path to "~",
	// rewrite any /Users/<user> or /home/<user> segment for the caller's own
	// username that the literal sweep cannot see, then fail closed only if the
	// literal home still shows. The same gate history.Capture holds.
	if r.home != "" {
		redacted = scanner.SweepCallerHome(redacted, r.home)
		var resid []scanner.Finding
		redacted, resid = scanner.SurvivingCallerHome(redacted, r.home)
		if len(resid) > 0 {
			return "", 0, newIngestError("refusing to write: the caller's home path survived redaction")
		}
	}
	if resid := scanner.BlockingResidual(r.sc.ScanText(redacted, label)); len(resid) > 0 {
		kinds := make([]string, 0, len(resid))
		for _, f := range resid {
			kinds = append(kinds, f.Kind)
		}
		return "", 0, newIngestError("refusing to write: redaction left %d blocking span(s) unresolved [%s]", len(resid), strings.Join(kinds, ", "))
	}
	return redacted, len(findings), nil
}

// redactLeaves is the PAGE-FRONTMATTER entry to the leaf walk: nothing in a
// page's frontmatter is an identifier the store resolves, so every string leaf
// it introduces is judged. `contradicts:` is deliberately included even though
// it lists page ids — the entries are host-supplied and validated only for
// non-emptiness (parseDistilledPage strips a trailing ".md" and checks nothing
// else), they are copied verbatim into contradictions.md, and NOTHING resolves
// them against the store, so a rewritten target is a dangling cross-reference
// rather than a deleted page. The registry's back-links are the opposite case
// on both counts — see redactRegistryLeaves.
func (r *storeRedactor) redactLeaves(current, target any, label string) error {
	return r.walkLeaves(nil, current, target, label, nil)
}

// redactRegistryLeaves is the REGISTRY entry to the same walk, with the store's
// page back-links excluded from it. `<content-hash>.consumers.<consumer>.pages`
// holds filenames the store RESOLVES: pruneOrphans deletes any page on disk
// that no back-link names, so rewriting one is not a redaction — it unlinks a
// file that keeps its real name and the next write silently deletes it. An
// ordinary slug is enough to trigger it, because a page name is prose-shaped:
// `topic_home_migrating-off-the-nas.md` carries `off-the-nas`, which the
// canonical scanner matches as a device hostname on the hyphen boundary.
// Excluding the leaf costs nothing: a back-link's charset is bounded by
// pageNameRe through validatePageFilename, so it is an identifier, not the
// acquired text this walk exists to judge.
func (r *storeRedactor) redactRegistryLeaves(current, target any, label string) error {
	return r.walkLeaves(nil, current, target, label, registryBackLinkPath)
}

// registryBackLinkPath reports whether path names the registry's page back-link
// list, `<content-hash>.consumers.<consumer>.pages`. The consumer is matched by
// position rather than by the literal "memory" so a later consumer's back-links
// carry the same protection by construction.
func registryBackLinkPath(path []string) bool {
	return len(path) == 4 && path[1] == "consumers" && path[3] == "pages"
}

// walkLeaves walks target — the nested map[string]any / []any / string shape
// the frontmatter dumper and the JSON registry share — IN PLACE, and sanitises
// through redactText every string leaf the write is INTRODUCING. A leaf is
// introduced when current holds no equal string, or none at all; a leaf present
// and equal in current is the store's already, not this write's to judge, and
// stays byte-identical — so a legacy registry carrying a dirty cached citation
// is neither refused on re-ingest (the one verb that repairs it) nor rewritten
// behind the operator's back; reporting it is the lint's job. A nil current
// introduces every leaf. An introduced KEY is judged too, by judgeKey — keys
// are NOT schema-fixed, and a key carrying a secret is refused rather than
// rewritten; non-string scalars pass through untouched. A subtree structural
// reports at its map path is skipped whole: it holds identifiers the store
// resolves, and rewriting one breaks the thing it names.
//
// Mutating in place is the contract, not a shortcut: the map a RegistryMerge
// returned IS what the store then writes, so a caller that kept hold of it —
// the registry-only fast path reads its result licence and citation off the
// merged map — sees the written bytes rather than a pre-redaction copy.
func (r *storeRedactor) walkLeaves(path []string, current, target any, label string, structural func([]string) bool) error {
	switch v := target.(type) {
	case map[string]any:
		cm, _ := current.(map[string]any)
		for k, item := range v {
			var cv any
			known := false
			if cm != nil {
				cv, known = cm[k]
			}
			// A key the store does not already hold is this write's, exactly as
			// a leaf is, and is judged before its value: the key is the one part
			// of the shape a host controls that no value-side pass can see.
			if !known {
				if err := r.judgeKey(k, label); err != nil {
					return err
				}
			}
			at := append(append([]string(nil), path...), k)
			if structural != nil && structural(at) {
				continue
			}
			if s, ok := item.(string); ok {
				red, err := r.judgeLeaf(cv, s, label)
				if err != nil {
					return err
				}
				v[k] = red
				continue
			}
			if err := r.walkLeaves(at, cv, item, label, structural); err != nil {
				return err
			}
		}
	case []any:
		cl, _ := current.([]any)
		for i, item := range v {
			cv := storedListElement(cl, item, i)
			if s, ok := item.(string); ok {
				red, err := r.judgeLeaf(cv, s, label)
				if err != nil {
					return err
				}
				v[i] = red
				continue
			}
			if err := r.walkLeaves(path, cv, item, label, structural); err != nil {
				return err
			}
		}
	}
	return nil
}

// storedListElement pairs one element of a list the write is about to store
// with its counterpart in the baseline list BY VALUE, falling back to the same
// index when the baseline holds nothing equal. Index pairing alone contradicts
// the walk's own contract: a list that gains a front element shifts every later
// element, so a leaf the store already held is re-judged — and a legacy store's
// dirty recall entry is then rewritten, or the whole write refused, on a
// re-ingest that introduced nothing. A list is an unordered set of leaves as
// far as "did this write introduce it" is concerned; the index says nothing.
func storedListElement(baseline []any, item any, i int) any {
	for _, c := range baseline {
		if reflect.DeepEqual(c, item) {
			return c
		}
	}
	if i < len(baseline) {
		return baseline[i]
	}
	return nil
}

// judgeKey is redactLeaves' one KEY rule. Keys are not schema-fixed:
// validateSourceBlock rejects no unknown key in a page's source: block and the
// frontmatter dumper admits any identifier-shaped key, a `ghp_`-prefixed token
// among them, so a host distiller's page JSON can carry a credential as a YAML
// key. It is refused, never rewritten — renaming a key renames the field the
// reader looks up, and dropping it discards the value it names, so neither is a
// redaction. The refusal names the label and the kinds and NEVER the key
// itself: here the key IS the secret, and an error is the one artefact that
// reaches the terminal and the run log unredacted.
func (r *storeRedactor) judgeKey(key, label string) error {
	resid := r.residue(key, label)
	if len(resid) == 0 {
		return nil
	}
	kinds := make([]string, 0, len(resid))
	for _, f := range resid {
		kinds = append(kinds, f.Kind)
	}
	return newIngestError("refusing to write: a map key in %s carries %d blocking span(s) [%s]; a key cannot be redacted without renaming the field it names, so repair the source", label, len(resid), strings.Join(kinds, ", "))
}

// judgeFilename is the PAGE FILENAME rule (iss-2609020321100138). A page's name
// is host-supplied — a distiller returns `slug`, slugRe admits
// [A-Za-z0-9_-] and pageNameRe admits <type>_<domain>_<slug>.md — and no
// value-side or key-side pass can see it: the filename is not a leaf of the
// frontmatter, and the registry back-link that carries it is deliberately
// excluded from the leaf walk (redactRegistryLeaves, the pruneOrphans
// data-loss fix). So a slug of `ghp_<40 chars>` reached the committed tree
// four times over — as the file's own name, in index.md, in log.md, and as the
// back-link — with every other write-side detector green.
//
// Refused, never rewritten, for judgeKey's reason: a page name is the identity
// the store resolves, so renaming it is not a redaction — it makes a different
// page, and the back-link that names the old one then points at nothing.
//
// The BAR is narrower than every other write-side rule's, and that is the
// point. redactText and judgeKey refuse on scanner.BlockingResidual, which
// promotes any identity-or-network span to blocking whatever its severity so a
// warn-level hostname heuristic cannot slip through stage two. A filename is
// not free text: it is prose-shaped by construction, and
// `topic_home_migrating-off-the-nas.md` matches net_device_hostname at warn on
// the hyphen boundary — at BlockingResidual's bar every such ordinary page
// would be refused. This rule therefore selects on the scanner's own severity
// vocabulary alone, scanner.SeverityHardFail, which within a filename's
// charset is exactly the credential class: no '/', '@', ':' or '.' can appear
// inside a page name, so the address kinds and the home-path kinds are
// unreachable there and what remains at hard_fail is a secret pattern, the
// caller's own local username, or a banned real name.
//
// The components are judged as well as the joined name because '_' is a word
// character: `\bghp_...` has no word boundary after `topic_auth_`, so the
// joined form hides in the scanner exactly the token the slug carries plainly.
// The underscore SUFFIXES are judged for the same reason carried one step
// further — see filenameJudgeTexts.
func (r *storeRedactor) judgeFilename(filename string) error {
	seen := map[string]bool{}
	var kinds []string
	for _, text := range filenameJudgeTexts(filename) {
		for _, f := range r.hardFailResidue(text, filename) {
			if seen[f.Kind] {
				continue
			}
			seen[f.Kind] = true
			kinds = append(kinds, f.Kind)
		}
	}
	if len(kinds) == 0 {
		return nil
	}
	return newIngestError("refusing to write %s: the page filename carries %d hard-fail span(s) [%s]; a page name cannot be redacted without renaming the page the store resolves, so repair the slug at the source", filename, len(kinds), strings.Join(kinds, ", "))
}

// filenameJudgeTexts is the set of strings judgeFilename scans for one page
// name: the joined name, the three components pageNameRe parses out of it, and
// every SUFFIX that begins immediately after an '_'.
//
// The suffixes are what close the separator-straddling spelling. Judging the
// parsed components was a fix for the missing word boundary, but the component
// split is ITSELF on underscore and a credential prefix ends in one, so a
// credential whose own prefix ends one component and whose body begins the next
// hides from both earlier passes at once: `topic_ghp_<36>.md` parses as type
// `topic`, domain `ghp`, slug `<36>`, and the joined form has no boundary
// before `ghp`, the domain alone is three letters, and the slug alone carries
// no prefix. `sk_live_` splits the other way, into domain `sk` and a slug
// opening with the rest of the prefix.
//
// Suffixes rather than re-joined adjacent components, because slugRe admits
// '_': a token can begin at an underscore INSIDE the slug
// (`topic_auth_x_ghp_<36>.md`), a position no pair of parsed components starts
// at. And suffixes rather than normalising the separators away, because the
// prefixes this is hunting — `ghp_`, `sk_live_`, `github_pat_` — contain the
// very character such a normalisation would remove or replace, so it would
// destroy the tokens it was meant to expose.
//
// The set is COMPLETE for the class, not a longer list of guesses. Within a
// page name's charset the word characters are [A-Za-z0-9_], so a `\b`-anchored
// pattern can match at the string start, after a '-', or after a '.' — all
// three of which are real boundaries the joined pass already sees — or at a
// position the joined pass cannot see, which is exactly a position preceded by
// '_'. One suffix per underscore therefore covers every position where an
// anchored pattern could match if the underscore were a boundary.
//
// The components are kept alongside the suffixes rather than replaced by them.
// A suffix carries the rest of the name after its component, so a pattern with
// a TRAILING anchor that matches a component standing alone need not match it
// inside a suffix; dropping the component pass could therefore narrow the bar,
// and the bar is not this function's business. Widening it is not either: this
// changes only WHERE the patterns are matched, never which severities count —
// hardFailResidue still selects on scanner.SeverityHardFail alone.
//
// Offsets do not survive a suffix, and nothing downstream needs them to. Each
// scan is labelled with the whole `filename`, and judgeFilename reports the
// page by name and the findings by kind, never by position, so a suffix's
// shifted offsets cannot corrupt the refusal's ability to name the page.
// Indexing by byte is safe for the same reason it is exact: '_' is ASCII, so a
// split after one never lands inside a multi-byte rune.
func filenameJudgeTexts(filename string) []string {
	texts := []string{filename}
	if typ, domain, slug, ok := ParsePageFilename(filename); ok {
		texts = append(texts, typ, domain, slug)
	}
	for i := 0; i < len(filename); i++ {
		if filename[i] == '_' {
			texts = append(texts, filename[i+1:])
		}
	}
	return texts
}

// hardFailResidue is judgeFilename's narrow bar: the scanner's own hard_fail
// severity and nothing else. It is deliberately NOT scanner.BlockingResidual
// (see judgeFilename) and deliberately NOT a second severity notion — the
// selection is on scanner.SeverityHardFail, the level the scanner already
// defines. The literal-home backstop residue applies is skipped too: a home
// path cannot appear in a page name, which holds no '/'.
func (r *storeRedactor) hardFailResidue(text, label string) []scanner.Finding {
	var out []scanner.Finding
	for _, f := range r.sc.ScanText(text, label) {
		if f.Severity == scanner.SeverityHardFail {
			out = append(out, f)
		}
	}
	return out
}

// judgeLeaf is redactLeaves' one leaf rule: unchanged from current, keep it;
// otherwise it is this write's and goes through redactText.
func (r *storeRedactor) judgeLeaf(current any, leaf, label string) (string, error) {
	if c, ok := current.(string); ok && c == leaf {
		return leaf, nil
	}
	red, _, err := r.redactText(leaf, label)
	return red, err
}

// redactOriginalBytes sanitises the --keep-original source copy. A text source's
// raw bytes ARE its text, so they are redacted in place. A binary source (a PDF)
// cannot be redacted byte-wise without corrupting it, so its distilled text is
// scanned instead and the keep-original copy is REFUSED when that text carries a
// blocking span — a refused best-effort copy is recorded, never fatal, and never
// leaks. A binary source with no blocking span is stored verbatim.
func (r *storeRedactor) redactOriginalBytes(material sourceMaterial) ([]byte, error) {
	if isRedactableText(material.rawBytes) {
		red, _, err := r.redactText(string(material.rawBytes), "source")
		if err != nil {
			return nil, err
		}
		return []byte(red), nil
	}
	if _, _, err := r.redactText(material.text, "source"); err != nil {
		return nil, newIngestError("refusing to keep original: source carries a secret/PII span that cannot be redacted in a binary file")
	}
	return material.rawBytes, nil
}

// isRedactableText reports whether data is UTF-8 text with no NUL byte — the
// same shape decodeText accepts — so it can be safely redacted as a string. A
// binary blob (a PDF) is not, and must never be rewritten span-wise.
func isRedactableText(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return utf8.Valid(data)
}
