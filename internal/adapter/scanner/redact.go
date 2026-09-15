package scanner

import "strings"

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
// place (it reuses the same maskSecret fingerprint and the same per-line
// strings.ReplaceAll approach as Finding.MarshalJSON).
//
// Secret tokens are masked to a non-reversible fingerprint by AUTHORITATIVE BYTE
// SPAN (reusing sealLine), and identity kinds to a neutral placeholder (a self
// home path collapses to "~"). Byte-span masking is what makes two PARTIALLY
// overlapping secret spans safe: substring replacement, longest-first, used to
// let the wider match consume the narrower one's leading bytes so the narrower
// ReplaceAll found nothing and its raw tail survived — sealLine instead forces
// every overlap byte to '*'. Identity placeholders keep the substring rewrite
// (they are length-changing) and run AFTER the secrets are sealed, when no raw
// secret bytes remain to be shifted. Redact is only stage one; the caller MUST
// re-scan the result and fail closed if any hard_fail span survived.
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
// byte position (sealLine), so overlapping matches cannot leak a raw tail;
// identity kinds get their readable placeholders by substring replacement,
// longest-first, applied after the secret bytes are already masked.
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
	if len(secretIdx) > 0 {
		if sealed := sealLine(line, fs, secretIdx); sealed != line {
			changed += len(secretIdx)
			line = sealed
		}
	}
	sortByMatchedLenDesc(identity)
	for _, f := range identity {
		var next string
		if f.Kind == kindHomeSelf {
			// The caller's home is a path, and a longer path that merely
			// starts with it ("/rootfs" under HOME=/root) is not the home: the
			// substring rewrite collapsed both to "~", so the home is swept
			// where it stands as a path, by the anchor the detector applies
			// (the sweep's placeholder is the "~" redactionReplacement gives
			// this kind).
			next = SweepCallerHome(line, f.Matched)
		} else {
			next = strings.ReplaceAll(line, f.Matched, redactionReplacement(f))
		}
		if next != line {
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

// sortByMatchedLenDesc orders findings by descending Matched byte length
// (stable, deterministic on ties via the existing sortFindings key would be
// overkill here — insertion is small and ties are handled by stability).
func sortByMatchedLenDesc(fs []Finding) {
	for i := 1; i < len(fs); i++ {
		for j := i; j > 0 && len(fs[j].Matched) > len(fs[j-1].Matched); j-- {
			fs[j], fs[j-1] = fs[j-1], fs[j]
		}
	}
}
