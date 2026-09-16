package scanner

import (
	"sort"
	"strings"
)

// ScanText scans an in-memory string with THIS scanner's merged patterns,
// probed identity, and per-repo severity floors — the same detection
// ScanBundle runs on a file's content, but with no filesystem read. It is the
// entry point a write-time redactor (the history store) uses to find every
// secret/PII span in a transcript before it lands on disk. logicalName is the
// label stamped into each finding's File field.
//
// It intentionally exposes the merged config that the package-level ScanText
// cannot: a caller using the package-level function would bypass the
// .abcd/config/pii.json override that New folded in.
func (s *Scanner) ScanText(text, logicalName string) []Finding {
	return ScanText(text, s.identity, s.patterns, s.identSev, logicalName)
}

// Redact rewrites every finding's matched span out of text, returning the
// sanitised text and the count of spans actually changed. It is the shared
// write-time sanitiser: both the history transcript store and any other
// pre-disk redactor route through it so the masking discipline lives in ONE
// place (it reuses the same maskSecret fingerprint Finding.MarshalJSON applies
// to the serialized surface).
//
// EVERY kind is masked by AUTHORITATIVE BYTE SPAN — secret tokens to a
// non-reversible fingerprint (sealLine), identity kinds to a neutral
// placeholder (a self home path collapses to "~"). Byte-span masking is what
// makes two PARTIALLY overlapping secret spans safe: substring replacement,
// longest-first, used to let the wider match consume the narrower one's leading
// bytes so the narrower ReplaceAll found nothing and its raw tail survived —
// sealLine instead forces every overlap byte to '*'. Identity spans run AFTER
// the secrets are sealed, when no raw secret bytes remain and the seal has
// changed no byte COUNT, so the detector's offsets still describe the same
// bytes. Redact is only stage one; the caller MUST re-scan the result and fail
// closed if any hard_fail span survived.
//
// Identity masking WAS a per-line strings.ReplaceAll, recorded as deliberate
// because an identity placeholder is length-changing and a whole-string rewrite
// needs no offsets. iss-2609120446083912 retired that choice: a whole-string
// replace cannot say "this occurrence masked, that one left", so it overrode
// every refusal the detector had learned to make — a reverse-DNS bundle
// component, a mid-word collision, an occurrence inside a URL — whenever a
// genuine mention shared the line, corrupting the technical content the record
// exists to hold. maskIdentitySpans below states the mechanism and names the
// fail-open cost the reversal accepts.
func Redact(text string, findings []Finding) (string, int) {
	if len(findings) == 0 {
		return text, 0
	}
	lines := strings.Split(text, "\n")
	// The block consumer decides where a key block ENDS by the shape of its
	// lines, and a seal writes '*' bytes, which are not body-shaped. Judged on
	// the rewritten slice, a second finding on a body line — an AWS-key-shaped
	// run inside the base64 is enough — closed the block at that line and left
	// the rest of the key and its END marker in the record. So the boundaries
	// are computed on the ORIGINAL lines and applied to the rewritten ones.
	original := append([]string(nil), lines...)

	byLine := map[int][]Finding{}
	for _, f := range findings {
		byLine[f.Line] = append(byLine[f.Line], f)
	}
	rewritten := 0
	for lineno, fs := range byLine {
		idx := lineno - 1
		if idx < 0 || idx >= len(lines) {
			continue
		}
		line, n := redactLine(lines[idx], fs)
		lines[idx] = line
		rewritten += n
	}
	// A PEM private key is the one finding whose leak is not on the line the
	// pattern matched: the header names the block, and the body it announces
	// follows on the lines after it. Those are consumed here, through the END
	// line, bounded (pem.go) — the one place every store's redaction shares.
	lines, blocks := consumePEMBodies(original, lines, findings)
	rewritten += blocks
	return strings.Join(lines, "\n"), rewritten
}

// maskedWhole reports whether a finding's span is masked with no head/tail
// fingerprint: every identity kind, where the head and tail are what
// re-identify the machine, and a PEM private-key span, whose tail is body
// bytes once the pattern reaches past the header — two bytes of key are two
// bytes too many, and a header needs no fingerprint to be recognised for
// what it was.
func maskedWhole(kind string) bool {
	return IsIdentityKind(kind) || kind == kindPEMPrivateKey
}

// redactLine masks every finding on one source line. Secret spans are sealed by
// byte position (sealLine); identity kinds get their readable placeholders by
// byte position too, at the offsets the detector recorded on this line, applied
// after the secret bytes are already masked.
func redactLine(line string, fs []Finding) (string, int) {
	var secretIdx []int
	var identity []Finding
	for i, f := range fs {
		if f.Matched == "" {
			continue
		}
		if IsIdentityKind(f.Kind) {
			identity = append(identity, f)
		} else {
			secretIdx = append(secretIdx, i)
		}
	}
	changed := 0
	sealed := line
	if len(secretIdx) > 0 {
		if next := sealLine(line, fs, secretIdx); next != line {
			changed += len(secretIdx)
			sealed = next
		}
	}
	out, n := maskIdentitySpans(line, sealed, identity)
	return out, changed + n
}

// identitySpan is one identity finding resolved to the byte interval it occupies
// on the line, with the placeholder that replaces it.
type identitySpan struct {
	start, end int
	kind       string
	repl       string
}

// maskIdentitySpans replaces exactly the byte spans the identity detector
// flagged, and nothing else.
//
// WHY BY SPAN (iss-2609120446083912). The detector clears lookalikes on
// purpose: a whole component of a reverse-DNS identifier
// (isDottedNamespaceComponent), a collision in the middle of a longer word
// (wordBounded), an occurrence inside a URL span, a system path segment. The
// whole-string rewrite this replaces overrode every one of those refusals the
// moment a genuine mention shared the line. Masking the flagged spans is what
// makes the refusals real.
//
// THE COST, accepted with the decision and FAIL-OPEN. An occurrence the
// detector cleared now survives even on a line where the old rewrite masked it
// by accident — most visibly a login inside a URL ("github.com/<login>/repo"),
// which the URL suppression clears for every bare-token identity kind. Nothing
// masks it here any more; the stage-two re-scan does not flag it either, because
// it is the same detector. The caller's own home path is not part of that
// residue: the detector flags every occurrence by the same anchor the literal
// SweepCallerHome sweep uses, and history, memory and ideate additionally run
// that sweep after Redact (capture, decide, intent and reading do not).
//
// OFFSETS. No span is applied against a shifted offset: overlapping spans are
// merged into disjoint clusters, sorted ascending, and the line is REBUILT from
// the sealed bytes between them. Every offset used is therefore an offset on
// the untouched line, so a placeholder of any length is free to change it.
// (Right-to-left in-place application would serve equally; rebuilding makes the
// invariant structural rather than dependent on the order of application.)
// Spans are validated against `orig`, the line BEFORE the secret seal, and
// applied to `sealed`, the line after it: the seal is byte-length-preserving,
// so the two share every offset, and an identity span that overlaps a sealed
// secret (a repo-configured pattern enclosing a login, say) still holds its
// bytes on the original and is masked in place rather than falling to the
// whole-string rewrite below, which could not reach the starred bytes and
// would instead rewrite every cleared lookalike on the line.
//
// OVERLAP. Two identity findings can cover the same bytes — a local_username
// inside the home_path_other or the longer real_name that contains it. The
// cluster is masked once, with the placeholder of its WIDEST member, which is
// what the old longest-first ReplaceAll produced; and it counts as one rewrite,
// as it did then.
//
// FALLBACK. A finding whose recorded span does not hold the bytes it claims on
// the original line was not produced by this scanner over this text (every
// producer slices Matched out of the line at Column), so its offsets say
// nothing. It keeps the whole-string rewrite rather than being dropped:
// dropping it would fail open on a span that is genuinely present.
func maskIdentitySpans(orig, sealed string, fs []Finding) (string, int) {
	line := sealed
	if len(fs) == 0 {
		return line, 0
	}
	if len(orig) != len(sealed) {
		// The seal is length-preserving by construction; if that ever stops
		// being true the offsets below are meaningless, so validate against
		// the line the spans will be applied to and let the fallback carry
		// what no longer matches.
		orig = sealed
	}
	spans := make([]identitySpan, 0, len(fs))
	var loose []Finding
	for _, f := range fs {
		start := f.Column - 1
		end := start + len(f.Matched)
		if start < 0 || end > len(orig) || orig[start:end] != f.Matched {
			loose = append(loose, f)
			continue
		}
		spans = append(spans, identitySpan{start, end, f.Kind, redactionReplacement(f)})
	}
	// Ascending by start, widest first on a tie, then by kind so a cluster's
	// placeholder is deterministic when two kinds cover identical bytes.
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		if spans[i].end != spans[j].end {
			return spans[i].end > spans[j].end
		}
		return spans[i].kind < spans[j].kind
	})
	changed := 0
	var b strings.Builder
	from := 0
	for i := 0; i < len(spans); {
		start, end, repl := spans[i].start, spans[i].end, spans[i].repl
		widest := end - start
		j := i + 1
		for j < len(spans) && spans[j].start < end {
			if w := spans[j].end - spans[j].start; w > widest {
				widest, repl = w, spans[j].repl
			}
			if spans[j].end > end {
				end = spans[j].end
			}
			j++
		}
		if line[start:end] != repl {
			changed++
		}
		b.WriteString(line[from:start])
		b.WriteString(repl)
		from = end
		i = j
	}
	if len(spans) > 0 {
		b.WriteString(line[from:])
		line = b.String()
	}
	for _, f := range loose {
		if next := strings.ReplaceAll(line, f.Matched, redactionReplacement(f)); next != line {
			changed++
			line = next
		}
	}
	return line, changed
}

// IsIdentityKind reports whether a finding kind is a PII identity span (masked to
// a readable placeholder) rather than a secret token (masked to a fingerprint).
// The network kinds belong here too: an address, a MAC and a hostname are
// IDENTIFIERS, not high-entropy secrets. The secret fingerprint deliberately
// keeps the first three and last two runes so a leaked credential can be
// triaged, and on an identifier that is precisely the wrong trade — it preserves
// a MAC's vendor bytes and final octet, and a hostname's head and suffix, which
// is enough to re-identify the machine the redaction was meant to hide.
func IsIdentityKind(kind string) bool {
	for _, k := range identityKinds() {
		if k == kind {
			return true
		}
	}
	return false
}

// identityKinds enumerates every identity kind IsIdentityKind accepts. It is a
// function rather than a var so the byte-scan policy table can be checked
// against it in full (TestEveryIdentityKindIsClassifiedForBytes).
func identityKinds() []string {
	return []string{
		kindHomeSelf, kindHomeOther, kindRealEmail, kindRealName, kindGithubUser, kindLocalUser,
		kindNetIPv4, kindNetIPv6, kindNetMAC, kindNetLANHost, kindNetDeviceHost,
	}
}

// redactionReplacement maps a finding to the text that replaces its raw span.
// Identity kinds get readable placeholders (never the original value, so a
// re-scan cannot re-match); secret tokens get the maskSecret fingerprint.
func redactionReplacement(f Finding) string {
	switch f.Kind {
	case kindHomeSelf:
		return "~"
	case kindHomeOther:
		return "[redacted-path]"
	case kindRealEmail:
		return "[redacted-email]"
	case kindRealName:
		return "[redacted-name]"
	case kindGithubUser, kindLocalUser:
		return "[redacted-user]"
	case kindNetIPv4, kindNetIPv6:
		return "[redacted-address]"
	case kindNetMAC:
		return "[redacted-mac]"
	case kindNetLANHost, kindNetDeviceHost:
		return "[redacted-hostname]"
	default:
		return maskSecret(f.Matched)
	}
}
