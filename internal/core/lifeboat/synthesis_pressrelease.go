package lifeboat

// synthesis_pressrelease.go — the PRESS-RELEASE seam of M6 (itd-88). It composes,
// or ingests, press-release.json (+ press-release.md) for a packed lifeboat: the
// embark interview contract, a single document (not a list of entries).
//
// ComposePressRelease has two modes on one entrypoint:
//
//   - Deterministic (raw == nil): composed from the packed brief press-release
//     section (brief/01-product/01-press-release.md), falling back to rescue/spine.md,
//     falling back to a grounded-nothing placeholder. Always written, byte-identical
//     across re-runs.
//   - Delegated (raw != nil): a validated untrusted whole document. Its evidence
//     must carry ≥1 ref resolving to a packed path restricted to brief/**,
//     rescue/spine.md, or principles.json; a press release citing nothing resolvable
//     is a WHOLE-DOCUMENT refusal (ErrPressReleaseUncited, exit 2) that leaves the
//     previously-derived file untouched — mirroring memory ingest's "refusing to
//     write an unattributable page". Prose fields are sanitised and capped.
//
// Kept OUT of manifest_sha256 (see embark_types.go's exclusion sets).

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// briefPressReleasePath is the packed brief's press-release section, the primary
// deterministic composition source (mirrors the mapping.go leaf).
const briefPressReleasePath = "brief/01-product/01-press-release.md"

// spineRelPath is the rescue spine, the deterministic fallback source.
const spineRelPath = "rescue/spine.md"

// ErrPressReleaseUncited is the whole-document refusal a delegated press release
// earns when its evidence resolves to nothing in the restricted packed-path set.
// The surface maps it to exit 2; the previously-derived deterministic file is left
// untouched (the document is one unit — there is no per-entry granularity to drop).
var ErrPressReleaseUncited = errors.New("press release cites no resolvable evidence; refusing to replace the derived press release")

// ComposePressRelease builds or ingests press-release.json (+ press-release.md)
// for a packed lifeboat. raw == nil composes deterministically; raw != nil
// validates the untrusted delegated document. It is transport-agnostic. The
// deterministic file is always written; the delegated document is written whole or
// refused whole (ErrPressReleaseUncited), never partially.
func ComposePressRelease(lifeboatDir string, raw []byte) (PressReleaseResult, error) {
	abs, _, err := gateSynthLifeboat(lifeboatDir)
	if err != nil {
		return PressReleaseResult{}, err
	}

	root, err := os.OpenRoot(abs)
	if err != nil {
		return PressReleaseResult{}, err
	}
	defer root.Close()
	paths, err := buildLifeboatPathSet(root, pressReleaseOwnOutput)
	if err != nil {
		return PressReleaseResult{}, err
	}

	var file PressReleaseFile
	if raw == nil {
		file = deterministicPressRelease(abs, paths)
	} else {
		file, err = validateDelegatedPressRelease(abs, raw, paths)
		if err != nil {
			return PressReleaseResult{}, err
		}
	}

	if file.Evidence == nil {
		file.Evidence = []string{}
	}
	file.SchemaVersion = PressReleaseSchemaVersion
	data, err := marshalSynth(file)
	if err != nil {
		return PressReleaseResult{}, err
	}
	if err := writeIntoLifeboat(root, abs, "press-release.json", data); err != nil {
		return PressReleaseResult{}, err
	}
	if err := writeIntoLifeboat(root, abs, "press-release.md", []byte(renderPressReleaseMarkdown(file))); err != nil {
		return PressReleaseResult{}, err
	}

	return PressReleaseResult{
		LifeboatDir:      fsutil.RedactHome(abs),
		Mode:             file.Mode,
		EvidenceRefs:     len(file.Evidence),
		PressReleasePath: "press-release.json",
		RenderPath:       "press-release.md",
	}, nil
}

// deterministicPressRelease composes the document from the packed brief
// press-release section, else the spine, else a grounded-nothing placeholder.
// Evidence cites the packed path it derived from (or, when neither is present, the
// expected-but-absent spine as a diagnostic pointer — a deterministic entry is not
// cite-gated). If principles.json is packed, it is appended as evidence.
func deterministicPressRelease(abs string, paths map[string]bool) PressReleaseFile {
	f := PressReleaseFile{Mode: ModeDeterministic}

	var source string
	var data []byte
	if paths[briefPressReleasePath] {
		if d, ok := readSynthSource(abs, briefPressReleasePath); ok {
			source, data = briefPressReleasePath, d
		}
	}
	if source == "" && paths[spineRelPath] {
		if d, ok := readSynthSource(abs, spineRelPath); ok {
			source, data = spineRelPath, d
		}
	}

	if source == "" {
		f.Headline = "(no press release grounded)"
		f.Body = cleanSynthProseN(
			"Searched "+briefPressReleasePath+" and "+spineRelPath+"; neither was packed.",
			maxPressReleaseBodyBytes)
		f.Evidence = []string{spineRelPath}
		return f
	}

	headline, body := prHeadlineAndBody(data)
	f.Headline = cleanSynthProse(headline)
	if f.Headline == "" {
		f.Headline = "(no press release grounded)"
	}
	f.Body = cleanSynthProseN(body, maxPressReleaseBodyBytes)
	f.Evidence = []string{source}
	if paths["principles.json"] {
		f.Evidence = append(f.Evidence, "principles.json")
	}
	return f
}

// validateDelegatedPressRelease reads the untrusted whole document behind the
// synthesis guards, then gates it: structural faults (oversize, unknown field,
// schema/mode/prompt_version) are fatal; evidence resolving to nothing in the
// restricted packed-path set is the whole-document refusal (ErrPressReleaseUncited).
// Prose fields are sanitised and capped; the document is written whole on success.
func validateDelegatedPressRelease(abs string, raw []byte, paths map[string]bool) (PressReleaseFile, error) {
	if len(raw) > maxSynthesisBytes {
		return PressReleaseFile{}, fmt.Errorf("press-release payload exceeds the %d-byte cap", maxSynthesisBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var pf PressReleaseFile
	if err := dec.Decode(&pf); err != nil {
		// The decoder names an undeclared field by the payload's own key:
		// redacted, never raw (iss-2609290218032954).
		return PressReleaseFile{}, fmt.Errorf("malformed press-release JSON: %s", scanner.RedactRefusal(abs, err.Error()))
	}
	if err := synthSchemaGate("press-release", pf.SchemaVersion, PressReleaseSchemaVersion); err != nil {
		return PressReleaseFile{}, err
	}
	if err := synthModeGate(pf.Mode); err != nil {
		return PressReleaseFile{}, err
	}
	if !promptVersionRe.MatchString(pf.PromptVersion) {
		return PressReleaseFile{}, errors.New("delegated press-release payload is missing a semver prompt_version")
	}

	// The restricted P set: only brief/**, rescue/spine.md, and principles.json are
	// admissible press-release evidence (the plan's "brief, spine, principles").
	restricted := map[string]bool{}
	for p := range paths {
		if strings.HasPrefix(p, "brief/") || p == spineRelPath || p == "principles.json" {
			restricted[p] = true
		}
	}
	evidence := filterSynthEvidence(pf.Evidence, restricted)
	if len(evidence) == 0 {
		return PressReleaseFile{}, ErrPressReleaseUncited
	}

	out := PressReleaseFile{
		Mode:          ModeDelegated,
		PromptVersion: pf.PromptVersion,
		Headline:      cleanSynthProse(pf.Headline),
		Subhead:       cleanSynthProse(pf.Subhead),
		Body:          cleanSynthProseN(pf.Body, maxPressReleaseBodyBytes),
		Evidence:      evidence,
	}
	if out.Headline == "" {
		out.Headline = "(untitled)"
	}
	quotes := pf.Quotes
	if len(quotes) > maxPressReleaseQuotes {
		quotes = quotes[:maxPressReleaseQuotes]
	}
	for _, q := range quotes {
		text := cleanSynthProse(q.Text)
		if text == "" {
			continue
		}
		out.Quotes = append(out.Quotes, PressReleaseQuote{
			Attribution: cleanSynthProse(q.Attribution),
			Text:        text,
		})
	}
	return out, nil
}

// prHeadlineAndBody extracts the headline (first Markdown heading) and body (first
// block of consecutive non-heading, non-blank lines beneath it) from a source
// document. Both are returned raw; the caller sanitises and caps them.
func prHeadlineAndBody(data []byte) (headline, body string) {
	lines := strings.Split(string(data), "\n")
	i := 0
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "#") {
			headline = strings.TrimSpace(strings.TrimLeft(t, "#"))
			i++
			break
		}
		// No leading heading: the first non-blank line is the headline.
		headline = t
		i++
		break
	}
	var buf []string
	for ; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" {
			if len(buf) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(t, "#") {
			if len(buf) > 0 {
				break
			}
			continue
		}
		buf = append(buf, t)
	}
	return headline, strings.Join(buf, " ")
}

// readSynthSource reads one packed source file behind the contained,
// symlink-refusing, bounded SourceContext read surface.
func readSynthSource(abs, rel string) ([]byte, bool) {
	ctx, err := newSourceContext(abs)
	if err != nil {
		return nil, false
	}
	defer ctx.Close()
	return ctx.ReadFile(rel)
}

// Render is the deterministic, sanitised human summary of a press-release run.
func (r PressReleaseResult) Render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "press release for %s (%s)\n", sanitize(r.LifeboatDir), sanitize(string(r.Mode)))
	fmt.Fprintf(&b, "  evidence refs: %d  (press-release.json)\n", r.EvidenceRefs)
	return b.String()
}

// renderPressReleaseMarkdown is the sanitised, deterministic press-release.md
// written into the lifeboat.
func renderPressReleaseMarkdown(f PressReleaseFile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", mdInline(f.Headline))
	// The subhead is a paragraph of its own under the headline — never wrapped
	// in the render's own emphasis, which a value carrying the delimiter could
	// close early (iss-2609251355497247).
	if sub := mdBlock(f.Subhead); sub != "" {
		fmt.Fprintf(&b, "%s\n\n", sub)
	}
	b.WriteString(renderSynthModeLine(f.Mode, f.PromptVersion))
	if body := mdBlock(f.Body); body != "" {
		fmt.Fprintf(&b, "%s\n\n", body)
	}
	for _, q := range f.Quotes {
		fmt.Fprintf(&b, "> %s\n>\n> — %s\n\n", mdBlock(q.Text), mdInline(q.Attribution))
	}
	if refs := mdCodeList(f.Evidence); refs != "" {
		fmt.Fprintf(&b, "Evidence: %s\n", refs)
	}
	return b.String()
}
