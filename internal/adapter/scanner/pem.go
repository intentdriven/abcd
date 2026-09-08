package scanner

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// pem.go — the block consumer for a PEM private key (GHSA-gmp7-9rvm-qcr3,
// GHSA-5qr6-f78x-g2cx, GHSA-29jw-3jg9-qmhx).
//
// The scanner is line-oriented and the bundled pem_private_key pattern keys on
// the BEGIN line, which is what makes a key block DETECTABLE: the header is
// single-line and self-identifying, while the base64 body after it has no
// prefix, no fixed width and — with no entropy rule in the bundled set —
// matches nothing. Detection is not redaction, though. Redact masks matched
// spans, so masking the header alone wrote every body line and the END line
// verbatim into every store, and the stage-two rescan, running the same
// header rule over a fingerprinted header, was clean by construction.
//
// So the pattern reaches over whatever body shares the header's OWN line (a
// resolve note, a JSON or K8s secret dump with literal \n escapes — see
// patterns.go), and Redact consumes the block's FOLLOWING lines through the
// END line here. The consumer is bounded three ways. An OPEN block — one with
// no END marker within the bound — consumes NOTHING until a body demonstrably
// opened: a base64 run long enough to be key material, or an armour header,
// within a line or two of the header (pemBodyEvidence). A header that is only
// NAMED in a rotation note or a runbook opens no block, and "body-shaped" on
// its own accepts a blank line, a code fence, a setext underline, a bare
// number and a single-token list item. A CLOSED block needs no such evidence:
// an END marker reached over an unbroken run of body-shaped lines is the proof
// that the run was a body, and a real block can hold its key material deeper
// than the window reaches (a leading blank line, a short prefix chunk). Either
// route opens the block; neither means nothing is taken. What is taken is
// body-shaped lines only — base64 runs, the armour headers
// a legacy encrypted PEM or a PGP block carries, blank lines, each behind an
// optional gutter (indentation, a diff or quote marker, a line number, a
// quote) — so a truncated block with no END line never swallows the prose
// after it. And it stops at maxPEMBodyLines, so no block consumes a record
// without limit. The consumed lines collapse
// into one placeholder that says how many lines went; the header span itself
// is masked whole (maskedWhole), since a head/tail fingerprint of a span that
// ends in body bytes would keep two bytes of key.
//
// Residuals, stated rather than hidden. A body line the shape rule declines —
// a gutter this list does not know — ends the block early and survives, and
// the stage-two rescan cannot see a headerless base64 line any more than
// stage one could (iss-96 tracks the entropy residue). The rule is wide
// enough for a pasted, indented, quoted, diffed or line-numbered block.
//
// One more. The same-line pattern's `open` alternative (patterns.go) takes a
// short final padding chunk only where it ENDS the line, so a one-line key
// rendering with no END marker that is followed by prose keeps that chunk —
// "… QQQ= and then prose" stores "QQQ=". That is residual (a) of
// iss-2609020127210042 and is not fixed here.
//
// The over-claim on the other side of the line — sub-residual (b) of that
// record — IS fixed here, and the shape of the fix is the point. The opener
// used to accept ANY run of 16+ base64-alphabet characters on a line in the
// window, which is a length threshold an ordinary identifier clears: a
// mentioned header followed by "CertificateRotationPolicy" alone in a fenced
// snippet opened a block, and the consumer spliced the fence, the identifier
// and the closing fence out of a committed record, silently, leaving an
// orphaned fence that broke the site render. A mention is not a block, so the
// opener now asks for one of three things instead of a bare length
// (pemBodyOpened, pemBodyRunPair): an armour header, a run at a key body's
// CHARACTERISTIC WIDTH, or two CONSECUTIVE body-shaped lines that each carry a
// key-material run. Raising a bar on a redactor loosens a security control, so
// the second and third exist to keep the loosening from costing coverage — see
// pemOpenerRunRe and pemBodyRunPair for what each is holding. What survives is
// narrower and named: a single identifier of 40+ characters, alone on a line
// within two lines of a mentioned header, still opens a block, and so do two
// such lines of 16+ back to back. Only the entropy rule iss-96 tracks can tell
// those from key material by looking at the bytes.

// maxPEMBodyLines bounds the lines one block consumer may take after the
// header. A PGP private-key block with several subkeys runs to a few hundred
// lines; the bound clears that by an order of magnitude and still caps a
// pathological block at a fraction of any transcript.
const maxPEMBodyLines = 4096

var (
	// pemEndRe is the END line of any PEM/PGP private-key block; a line that
	// carries it closes the block and is consumed with it.
	pemEndRe = regexp.MustCompile(`-----END (?:[A-Z0-9]+ )*PRIVATE KEY(?: BLOCK)?-----`)
	// pemBodyLineRe is one body-shaped line: an optional gutter, then a base64
	// run or an armour header or nothing at all, then optional quoting.
	pemBodyLineRe = regexp.MustCompile(`^[\s\d+\-|>:"'` + "`" + `│]*(?:[A-Za-z0-9+/=]+|(?:Proc-Type|DEK-Info|Version|Comment|Charset|Hash|MessageID):.*)?[\s"',;\\]*$`)
	// pemBase64RunRe is a run from the base64 alphabet long enough to be key
	// material — the SAME rule the same-line pattern applies before it reaches
	// past a header (patterns.go).
	pemBase64RunRe = regexp.MustCompile(`[A-Za-z0-9+/=]{16,}`)
	// pemOpenerRunRe is a run at a key body's characteristic width: what one
	// line of a rendered PEM body looks like, as against what a long word looks
	// like. Every generator that writes these blocks wraps the body wider than
	// this — RFC 7468 mandates 64, OpenSSH writes 70, MIME-wrapped armour 76 —
	// so no canonically rendered body line falls under the bar, while an
	// identifier a person types into a sentence or a list almost never reaches
	// it. 40 characters is 30 bytes of material, and the margin below 64 is
	// there for the re-wrapping a mail client or a narrow terminal does to a
	// pasted block on its way into a transcript.
	pemOpenerRunRe = regexp.MustCompile(`[A-Za-z0-9+/=]{40,}`)
	// pemArmourRe is an armour header at the head of a line, behind the same
	// optional gutter pemBodyLineRe allows.
	pemArmourRe = regexp.MustCompile(`^[\s\d+\-|>:"'` + "`" + `│]*(?:Proc-Type|DEK-Info|Version|Comment|Charset|Hash|MessageID):`)
)

// pemEvidenceWindow is how far past the header the consumer looks for evidence
// that a body opened. A block's first body line follows the header directly, or
// after one blank line; nothing further out is a block this consumer opened.
const pemEvidenceWindow = 2

// pemBodyOpened reports whether a line ALONE carries positive evidence that a
// key body opened on it: a PEM/PGP armour header, or a base64 run at a key
// body's characteristic width (pemOpenerRunRe). One line is a small amount of
// evidence, so what it takes to be conclusive on one line is a lot: the old
// bar of 16 characters is a length an ordinary word clears, and reading a
// mentioned header plus one long identifier as a key deleted the identifier
// and the fence around it from a record nobody was warned about. A run of pure
// padding ("====…", a setext underline) is not evidence either — base64
// padding is at most two bytes and never stands alone — so the run must carry
// at least one alphanumeric byte.
func pemBodyOpened(line string) bool {
	return pemArmourRe.MatchString(line) || pemKeyRun(line, pemOpenerRunRe)
}

// pemBodyRunPair reports whether lines[j] and the line after it are BOTH
// body-shaped and both carry a run long enough to be key material. This is the
// other way to be conclusive, and it is why raising the single-line bar to a
// characteristic width costs no coverage: a block re-wrapped narrower than
// that width — hard-wrapped in a mail quote, folded by a viewer — shows two
// such lines back to back where prose, which is what the bar is meant to
// spare, shows at most one. Both members must be body-shaped, since a line
// pemBlockEnd would refuse to consume is no evidence about a block it would
// not have taken.
func pemBodyRunPair(lines []string, j int) bool {
	if j+1 >= len(lines) {
		return false
	}
	for _, k := range [2]int{j, j + 1} {
		if !pemBodyLineRe.MatchString(lines[k]) || !pemKeyRun(lines[k], pemBase64RunRe) {
			return false
		}
	}
	return true
}

// pemKeyRun reports whether line carries a run matched by re that holds at
// least one alphanumeric byte.
func pemKeyRun(line string, re *regexp.Regexp) bool {
	for _, run := range re.FindAllString(line, -1) {
		if strings.ContainsFunc(run, isBase64Alnum) {
			return true
		}
	}
	return false
}

func isBase64Alnum(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

// pemBodyEvidence reports whether a body opened after the header at lines[h].
// The shape rule alone cannot answer this: "body-shaped" accepts a blank line,
// a code fence, a setext underline, a bare number and a single-token list item,
// so a sentence that merely NAMES a BEGIN marker — a rotation note, a runbook,
// an issue record — used to consume the lines after it and could leave an
// unbalanced fence where a fenced block had been. Evidence is required before
// anything is taken; a line that is not even body-shaped ends the search.
//
// Either form of evidence counts, on any line the window reaches. The pair
// form is read from a line in the window, so its second member may sit one
// line past the window's edge: the window bounds where a body may START, not
// how much of it must be visible from the header.
func pemBodyEvidence(lines []string, h int) bool {
	last := h + pemEvidenceWindow
	if last >= len(lines) {
		last = len(lines) - 1
	}
	for j := h + 1; j <= last; j++ {
		if pemBodyOpened(lines[j]) || pemBodyRunPair(lines, j) {
			return true
		}
		if !pemBodyLineRe.MatchString(lines[j]) {
			return false
		}
	}
	return false
}

// pemBodyPlaceholder is the one line a consumed block collapses to.
func pemBodyPlaceholder(n int) string {
	return fmt.Sprintf("[redacted-pem-body: %d lines]", n)
}

// consumePEMBodies collapses, for every pem_private_key finding whose span did
// not reach an END marker on the header's own line, the body-shaped lines that
// follow the header through the END line into one placeholder line. It returns
// the rewritten lines and the number of blocks collapsed. Headers are handled
// from the last line upward so an earlier header's indices stay valid.
//
// original is the text as it was BEFORE the per-line redaction; lines is the
// sealed slice the placeholder is written into. Where a block ends is a
// question about the source text — a seal writes '*' bytes, which are not
// body-shaped, so a second finding on a body line would otherwise close the
// block at that line and leave the rest of the key behind. The two slices are
// the same length by construction (per-line redaction never adds or removes a
// line); if they ever diverge, the sealed slice is the safe fallback.
func consumePEMBodies(original, lines []string, findings []Finding) ([]string, int) {
	if len(original) != len(lines) {
		original = lines
	}
	var headers []int
	seen := map[int]bool{}
	for _, f := range findings {
		if f.Kind != kindPEMPrivateKey || pemEndRe.MatchString(f.Matched) {
			continue
		}
		idx := f.Line - 1
		if idx < 0 || idx >= len(lines) || seen[idx] {
			continue
		}
		seen[idx] = true
		headers = append(headers, idx)
	}
	if len(headers) == 0 {
		return lines, 0
	}
	sort.Sort(sort.Reverse(sort.IntSlice(headers)))
	blocks := 0
	for _, h := range headers {
		end := pemBlockEnd(original, h)
		if end <= h+1 {
			continue
		}
		rest := append([]string{pemBodyPlaceholder(end - (h + 1))}, lines[end:]...)
		lines = append(lines[:h+1:h+1], rest...)
		blocks++
	}
	return lines, blocks
}

// pemBlockEnd returns the exclusive index of the last line the block whose
// header is lines[h] consumes: through the END line when one is reached within
// the bound, else through the last body-shaped line — with trailing blank
// lines given back, since they belong to the prose after a truncated block.
//
// Two independent routes open a block, and either alone is enough. A CLOSED
// block needs no evidence near its header: an END marker reached within the
// bound over an unbroken run of body-shaped lines is itself the proof that the
// run was a body, wherever the key material sits inside it — a real block
// whose first lines are blank or carry a short prefix chunk keeps its evidence
// deeper than pemEvidenceWindow can see, and demanding evidence there wrote
// the whole body and the END line out verbatim. An OPEN block — no END marker
// within the bound — has no such proof, so it opens only on evidence in the
// window (pemBodyEvidence), which is what keeps a header that is merely NAMED
// in a rotation note from swallowing the prose after it. Only when there is
// neither is nothing consumed.
func pemBlockEnd(lines []string, h int) int {
	limit := h + 1 + maxPEMBodyLines
	if limit > len(lines) {
		limit = len(lines)
	}
	j := h + 1
	for ; j < limit; j++ {
		if pemEndRe.MatchString(lines[j]) {
			return j + 1 // closed: the END marker is the evidence
		}
		if !pemBodyLineRe.MatchString(lines[j]) {
			break
		}
	}
	if !pemBodyEvidence(lines, h) {
		return h + 1 // the header was named, not opened: take nothing
	}
	for j > h+1 && strings.TrimSpace(lines[j-1]) == "" {
		j--
	}
	return j
}
