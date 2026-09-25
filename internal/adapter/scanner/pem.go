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
// opened: an armour header, or a base64 run that is key material rather than
// a word, within a line or two of the header (pemBodyEvidence). A header that is only
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
// a frame this list does not know — ends the block early and survives, and
// the stage-two rescan cannot see a headerless base64 line any more than
// stage one could (iss-96 tracks the entropy residue). The rule is wide
// enough for a pasted, indented, quoted, diffed or line-numbered block, and
// for the renderings iss-2609020127210042 named: log-prefixed lines, CSV
// cells, one XML element per line, a trailing comment, a source-string
// concatenation, and (on the header's own line, in patterns.go) a JSON array
// and doubly escaped newlines. Its width is the price: inside a block that
// has opened, a line such as "[main] done" or "7,ok" is body-shaped too.
//
// The same-line pattern's `open` alternative (patterns.go) judges a short
// final chunk on its own alphabet, so a one-line key with no END marker and
// prose after it no longer keeps a padded or digit-bearing tail; a
// letters-only tail followed by prose is the part left to an entropy rule.
//
// The over-claim on the other side of the line — sub-residual (b) of that
// record — IS fixed here, and the shape of the fix is the point. The opener
// used to accept ANY run of 16+ base64-alphabet characters on a line in the
// window, which is a length threshold an ordinary identifier clears: a
// mentioned header followed by "CertificateRotationPolicy" alone in a fenced
// snippet opened a block, and the consumer spliced the fence, the identifier
// and the closing fence out of a committed record, silently, leaving an
// orphaned fence that broke the site render.
//
// The first answer to that was a WIDTH — a run of 40+, under RFC 7468's 64 and
// OpenSSH's 70 but over anything a person types — and width was the wrong
// axis. It is true of a canonically rendered block and false of the shape a
// transcript redactor actually faces: a key truncated by a log rotation or
// re-wrapped by a mail client can show ONE visible body line of 16 to 39
// characters, and that line stopped being redacted while the header pattern
// went on reporting a handled secret (iss-2609090932441377). No bar can
// separate the two, because the identifier is 25 characters and the dropped
// body is 32.
//
// So the opener asks for one of four things, and only one of them is a length
// (pemBodyOpened, pemBodyRunPair). An armour header. A run at a key body's
// CHARACTERISTIC WIDTH. Two CONSECUTIVE body-shaped lines that each carry a
// key-material run — a wrapped body shows them back to back where prose shows
// at most one. Or, on a single line, a narrower run that is not a WORD, under
// a header that was PASTED rather than named: the run must carry a base64 byte
// outside [A-Za-z], and the header's marker must be the whole content of its
// line. Neither of those last two is width, and each on its own refuses the
// case the width bar was raised for.
//
// What survives is narrower and named. A single identifier of 40+ characters
// alone on a line within two lines of a MENTIONED header still opens a block,
// and so do two such lines of 16+ back to back — the width rule and the pair
// rule do not ask whether the header was pasted, because a run at full width,
// or one that shows itself on two lines running, is evidence enough alone. In
// the other direction a real body of 16 to 39 characters is missed where its
// header carries prose after the marker, or where the run happens to be
// letters only — about three 16-byte runs in a hundred, one 32-byte run in a
// thousand. Only the entropy rule iss-96 tracks can tell any of these from key
// material by looking at the bytes, and where that bar sits is a decision for
// the maintainer rather than one to settle inside a block consumer.

// maxPEMBodyLines bounds the lines one block consumer may take after the
// header that carry no key material. A PGP private-key block with several
// subkeys runs to a few hundred lines; the bound clears that by an order of
// magnitude and still caps a pathological run of blank lines or single short
// tokens at a fraction of any transcript. Lines that do carry key material
// continue a block past it (pemBlockEnd), so an oversized key has no tail.
const maxPEMBodyLines = 4096

// pemGutter, pemPrefix, pemTrailer and pemComment are the optional frame a
// pasted line carries: the indentation, diff or quote marker, line number or
// opening quote in front of its content; the fields a log line or a CSV row
// puts before it; the quoting, escape and concatenation a serialiser or a
// source file leaves after it; and a trailing comment. One definition, shared
// by every rule below that reads a whole line — the body-shape rule, the
// armour rule and the pasted-header rule — so a frame one of them learns
// cannot be a frame another has never heard of.
//
// The prefix fields are named shapes, not "anything before the run": an ISO
// date and time, a clock time, a bracketed field ("[main]", "[WARN]"), a log
// level, a process tag ("sshd[42]:"), and a CSV cell ending in its separator.
// A word followed by a colon is deliberately NOT one — "Owner: platform." is a
// sentence, and an open block must not take it for a body line
// (iss-2609020127210042).
const (
	pemGutter  = `[\s\d+\-|>:"'` + "`" + `│(]*`
	pemPrefix  = `(?:(?:\d{4}-\d{2}-\d{2}(?:[T ]\d{2}:\d{2}(?::\d{2})?(?:[.,]\d+)?(?:Z|[+-]\d{2}:?\d{2})?)?|\d{2}:\d{2}:\d{2}(?:[.,]\d+)?|\[[^\]]{0,64}\]|(?:TRACE|DEBUG|INFO|NOTICE|WARN|WARNING|ERROR|FATAL|CRITICAL)|[A-Za-z][\w.-]{0,31}\[\d+\]:)\s+|(?:"[^"]{0,64}"|'[^']{0,64}'|[A-Za-z0-9_.-]{0,64})[,;]\s*)*`
	pemOpenTag = `(?:<[A-Za-z_][\w:.-]*(?:\s[^<>]*)?>)?`
	pemEndTag  = `(?:</[A-Za-z_][\w:.-]*>)?`
	pemTrailer = `(?:\\[nr]|[\s"',;\\+&.|)\]])*`
	pemComment = `(?:\s+(?:#|//).*)?`
	// pemFrameHead and pemFrameTail wrap a line's content in the whole frame.
	pemFrameHead = `^` + pemGutter + pemPrefix + `[\s"'` + "`" + `(]*` + pemOpenTag
	pemFrameTail = pemEndTag + pemTrailer + pemComment + `$`
	// pemArmourTags are the header tags a legacy encrypted PEM, a PGP block
	// and an RFC 4716 block carry ("x-" is RFC 4716's private-use prefix).
	pemArmourTags = `(?:Proc-Type|DEK-Info|Version|Comment|Charset|Hash|MessageID|Subject|[xX]-[A-Za-z0-9-]+):`
	// pemBegin and pemEnd are the two armour markers: the five-dash form of
	// RFC 7468, OpenSSH and PGP, and the four-dash form of RFC 4716, which an
	// SSH2 private key carries — the same pair the bundled pattern opens on.
	pemBegin = `(?:-----BEGIN (?:[A-Z0-9]+ )*PRIVATE KEY(?: BLOCK)?-----|---- BEGIN (?:[A-Z0-9]+ )*PRIVATE KEY ----)`
	pemEnd   = `(?:-----END (?:[A-Z0-9]+ )*PRIVATE KEY(?: BLOCK)?-----|---- END (?:[A-Z0-9]+ )*PRIVATE KEY ----)`
)

var (
	// pemEndRe is the END line of any PEM/PGP/RFC 4716 private-key block; a
	// line that carries it closes the block and is consumed with it.
	pemEndRe = regexp.MustCompile(pemEnd)
	// pemBodyLineRe is one body-shaped line: the frame, then a base64 run or
	// an armour header or nothing at all.
	pemBodyLineRe = regexp.MustCompile(pemFrameHead + `(?:[A-Za-z0-9+/=]+|` + pemArmourTags + `.*)?` + pemFrameTail)
	// pemPastedHeaderRe is a BEGIN marker that is the WHOLE content of its
	// line, behind the same frame every other rule here allows: the header of
	// a block someone PASTED, as against one NAMED inside a sentence. What
	// separates them is not the marker but what follows it, since a sentence
	// goes on and a pasted line does not — so a marker with prose after it is
	// a mention, and a marker that ends its line is a block opening. Reading
	// the line rather than the marker also keeps the rule blind to where the
	// paste came from: it survives the indentation, quoting, diff markers and
	// speaker punctuation a block picks up on its way into a transcript.
	pemPastedHeaderRe = regexp.MustCompile(pemFrameHead + pemBegin + pemFrameTail)
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
	// pasted block on its way into a transcript. A body wrapped or truncated
	// BELOW the bar is not this rule's to catch: pemBodyRunPair takes it where
	// two lines survive, pemNarrowKeyRun where only one does.
	pemOpenerRunRe = regexp.MustCompile(`[A-Za-z0-9+/=]{40,}`)
	// pemArmourRe is an armour header at the head of a line, behind the same
	// frame pemBodyLineRe allows.
	pemArmourRe = regexp.MustCompile(pemFrameHead + pemArmourTags)
)

// pemEvidenceWindow is how far past the header the consumer looks for evidence
// that a body opened. A block's first body line follows the header directly, or
// after one blank line; nothing further out is a block this consumer opened.
const pemEvidenceWindow = 2

// pemBodyOpened reports whether a line ALONE carries positive evidence that a
// key body opened on it. One line is a small amount of evidence, so what it
// takes to be conclusive on one line is a lot, and there are three ways to be:
// a PEM/PGP armour header; a base64 run at a key body's characteristic width
// (pemOpenerRunRe); or, where the header was PASTED rather than named, a
// narrower run that is not a word (pemNarrowKeyRun). A run of pure padding
// ("====…", a setext underline) is evidence of nothing on any of the three —
// base64 padding is at most two bytes and never stands alone — so every run
// must carry at least one alphanumeric byte.
//
// The third way is what keeps the second from costing coverage on a body the
// pair rule cannot see. A block truncated by a log rotation or re-wrapped by a
// narrow terminal can leave ONE visible body line under the characteristic
// width, and there is no second line to pair it with, so before this the
// canonical opening of a DER-encoded key survived a redaction that reported
// itself done (iss-2609090932441377). Width cannot be what admits it: the
// identifier that opened the block this bar was raised to close is 25
// characters and the body it dropped is 32, so any bar clearing one clears the
// other. Two things that are not width admit it instead, and both must hold.
// The header must be pasted — its marker the whole content of its line — since
// a mention is a sentence and a sentence goes on past the marker. And the run
// must carry a byte of the base64 alphabet that is NOT a letter: a digit, '+',
// '/' or the '=' pad. Key material is drawn from all 64 symbols, so a 16-byte
// run of it is letters-only about three times in a hundred and a 32-byte run
// about one time in a thousand; an identifier a person types is letters and
// word structure, and the ones that do carry a digit — SHA256WithRSAEncryption
// — are refused on the first half instead, since a document naming an
// algorithm names the header too.
func pemBodyOpened(line string, pasted bool) bool {
	if pemArmourRe.MatchString(line) || pemKeyRun(line, pemOpenerRunRe) {
		return true
	}
	return pasted && pemNarrowKeyRun(line)
}

// pemNarrowKeyRun reports whether line carries a run long enough to be key
// material that is not a word: it holds at least one alphanumeric byte, and at
// least one byte of the base64 alphabet outside [A-Za-z]. Both halves are
// load-bearing — the first refuses a setext underline, the second an
// identifier — and neither reads the run's ENTROPY. That is deliberate.
// Entropy would refuse this rule's own padding-tail case, which repeats one
// byte, and it is the open decision iss-96 holds for the maintainer rather
// than a discriminator to settle here.
func pemNarrowKeyRun(line string) bool {
	for _, run := range pemBase64RunRe.FindAllString(line, -1) {
		if strings.ContainsFunc(run, isBase64Alnum) && strings.ContainsFunc(run, isBase64NonAlpha) {
			return true
		}
	}
	return false
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

// isBase64NonAlpha reports whether r is a base64-alphabet byte that a word
// cannot contain: a digit or one of the three symbols. '-' and '_' of the
// URL-safe alphabet are deliberately absent — they are how identifiers are
// spelled outside CamelCase, and pemBase64RunRe does not read them anyway.
func isBase64NonAlpha(r rune) bool {
	return (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '='
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
// how much of it must be visible from the header. Whether the header was
// pasted or merely named is a property of the header's OWN line, so it is
// read once here and handed to every line the window reaches.
func pemBodyEvidence(lines []string, h int) bool {
	pasted := pemPastedHeaderRe.MatchString(lines[h])
	last := h + pemEvidenceWindow
	if last >= len(lines) {
		last = len(lines) - 1
	}
	for j := h + 1; j <= last; j++ {
		if pemBodyOpened(lines[j], pasted) || pemBodyRunPair(lines, j) {
			return true
		}
		if !pemBodyLineRe.MatchString(lines[j]) {
			return false
		}
	}
	return false
}

// pemContinues reports whether an RFC 4716 header line continues onto the
// next: its last non-blank byte is a backslash.
func pemContinues(line string) bool {
	return strings.HasSuffix(strings.TrimRight(line, " \t\r"), `\`)
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
//
// Two rules extend the walk. An armour header whose value ends in a backslash
// continues onto the next line (RFC 4716 §3.3), and that line is part of the
// header whatever its shape. And the bound governs lines that carry no key
// material: past maxPEMBodyLines the block goes on over lines that do, so the
// tail of an oversized key is consumed with it rather than written out
// verbatim, while a pathological run of blank lines or single short tokens
// still stops at the bound (iss-2609020127210042).
func pemBlockEnd(lines []string, h int) int {
	limit := h + 1 + maxPEMBodyLines
	if limit > len(lines) {
		limit = len(lines)
	}
	j := h + 1
	continued := false
	for ; j < limit; j++ {
		if pemEndRe.MatchString(lines[j]) {
			return j + 1 // closed: the END marker is the evidence
		}
		if continued {
			continued = pemContinues(lines[j])
			continue
		}
		if !pemBodyLineRe.MatchString(lines[j]) {
			break
		}
		continued = pemArmourRe.MatchString(lines[j]) && pemContinues(lines[j])
	}
	if j == limit {
		for ; j < len(lines); j++ {
			if pemEndRe.MatchString(lines[j]) {
				return j + 1
			}
			if !pemBodyLineRe.MatchString(lines[j]) || !pemKeyRun(lines[j], pemBase64RunRe) {
				break
			}
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
