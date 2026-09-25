package scanner

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// identity_span_test.go — iss-2609120446083912. Identity masking rewrites the
// BYTE SPANS the detector flagged, not every occurrence of the matched string.
// A lookalike the detector deliberately cleared — a reverse-DNS namespace
// component, a mid-word collision, an occurrence inside a URL — must survive
// byte-for-byte on a line that also carries a genuine mention.
//
// The fixtures below are invented: "com" and "me" are logins chosen because
// they collide with a namespace label, and "Wren" is a persona.

// TestIdentityMaskingIsSpanBased is the record's own measured case and its
// siblings: one line, one genuine mention, one lookalike the detector refused.
func TestIdentityMaskingIsSpanBased(t *testing.T) {
	cases := []struct {
		name string
		id   Identity
		line string
		want string
	}{
		{
			// The measured case in iss-2609120446083912: the bare mention is
			// masked, the bundle identifier the detector cleared
			// (isDottedNamespaceComponent, iss-2609100505142469) survives whole.
			name: "reverse_dns_bundle_component_survives",
			id:   Identity{HomeUser: "com", HomePath: "/Users/com"},
			line: "com filed it: the crash is in the bundle com.acme.app",
			want: "[redacted-user] filed it: the crash is in the bundle com.acme.app",
		},
		{
			// Detection already refuses "com" inside "commit" (wordBounded);
			// the whole-string rewrite overrode it whenever a real mention
			// shared the line.
			name: "midword_collision_survives",
			id:   Identity{HomeUser: "com", HomePath: "/Users/com"},
			line: "the commit was authored by com",
			want: "the commit was authored by [redacted-user]",
		},
		{
			// TWO genuine mentions on one line are BOTH masked — the one thing
			// the whole-string rewrite caught by accident, now caught on
			// purpose, with a cleared lookalike standing between them.
			name: "both_genuine_mentions_masked_lookalike_between",
			id:   Identity{HomeUser: "com", HomePath: "/Users/com"},
			line: "com filed it: com.acme.app crashed, and com confirmed",
			want: "[redacted-user] filed it: com.acme.app crashed, and [redacted-user] confirmed",
		},
		{
			// real_name: the bare mention is masked; the occurrence inside a URL
			// span (suppressed by the detector) and the longer word that merely
			// starts with the name (wordBounded) both survive.
			name: "real_name_masks_only_the_flagged_span",
			id:   Identity{GitUserName: "Wren"},
			line: "Wren wrote it; see https://example.com/Wren/notes and the Wrenches",
			want: "[redacted-name] wrote it; see https://example.com/Wren/notes and the Wrenches",
		},
		{
			// github_username shares the mechanism and must behave identically.
			name: "github_username_masks_only_the_flagged_span",
			id:   Identity{GitRemoteUsername: "Wren"},
			line: "Wren wrote it; see https://example.com/Wren/notes and the Wrenches",
			want: "[redacted-user] wrote it; see https://example.com/Wren/notes and the Wrenches",
		},
		{
			// Mixed kinds with placeholders of different lengths on one line:
			// the offsets of the later spans must not drift as earlier ones are
			// replaced by longer text.
			name: "mixed_kinds_length_changing_placeholders",
			id:   Identity{HomeUser: "com", HomePath: "/Users/com", GitUserEmail: "alice@example.com"},
			line: "com pinged alice@example.com then com replied",
			want: "[redacted-user] pinged [redacted-email] then [redacted-user] replied",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := tc.line + "\n"
			findings := ScanText(text, tc.id, DefaultPatterns(), DefaultIdentitySeverities(), "t")
			got, n := Redact(text, findings)
			if n == 0 {
				t.Fatalf("Redact rewrote nothing for %q (findings %+v)", tc.line, findings)
			}
			if got != tc.want+"\n" {
				t.Errorf("span-based identity masking:\n got  %q\n want %q\n findings %+v", got, tc.want+"\n", findings)
			}
		})
	}
}

// TestDetectorFlagsEveryGenuineMentionOnALine is the detection half of the
// two-mention case. Span-based masking can only mask both mentions if the
// detector emits a finding for each, at its own column — and none for the
// lookalike between them. The whole-string rewrite masked the second mention
// as a side effect of replacing the first; nothing rested on the detector
// seeing it. Now everything does.
func TestDetectorFlagsEveryGenuineMentionOnALine(t *testing.T) {
	line := "com filed it: com.acme.app crashed, and com confirmed\n"
	id := Identity{HomeUser: "com", HomePath: "/Users/com"}
	var cols []int
	for _, f := range ScanText(line, id, DefaultPatterns(), DefaultIdentitySeverities(), "t") {
		if f.Kind == kindLocalUser {
			cols = append(cols, f.Column)
		}
	}
	if len(cols) != 2 {
		t.Fatalf("want a local_username finding for each genuine mention, got columns %v", cols)
	}
	// 1-based columns of the two bare mentions; the bundle component at column
	// 15 must NOT appear.
	if cols[0] != 1 || cols[1] != 41 {
		t.Errorf("flagged columns %v; want [1 41] (the two bare mentions, not the bundle at 15)", cols)
	}
}

// TestIdentitySpansKeepOffsetsAcrossManyMasks pins the offset discipline
// requirement 4 names: four genuine mentions of a three-byte login, each
// replaced by a fifteen-byte placeholder, with three cleared lookalikes
// interleaved. Every survivor must come out byte-for-byte, which can only hold
// if no span is applied against shifted offsets.
func TestIdentitySpansKeepOffsetsAcrossManyMasks(t *testing.T) {
	id := Identity{HomeUser: "com", HomePath: "/Users/com"}
	line := "com then com.acme.app then com then commit then com then io.com.x then com"
	want := "[redacted-user] then com.acme.app then [redacted-user] then commit then " +
		"[redacted-user] then io.com.x then [redacted-user]"
	got, _ := Redact(line+"\n", ScanText(line+"\n", id, DefaultPatterns(), DefaultIdentitySeverities(), "t"))
	if got != want+"\n" {
		t.Errorf("offsets drifted across length-changing masks:\n got  %q\n want %q", got, want+"\n")
	}
}

// TestSealLineIsByteLengthPreserving verifies — rather than assumes — the
// property the span rewrite rests on: identity spans are applied at the byte
// offsets the detector recorded on the ORIGINAL line, after the secret seal has
// already run over that line. That is only sound while the seal changes no byte
// COUNT. sealLine only assigns into a copy of the source bytes, and the seal
// must also leave valid UTF-8 behind.
func TestSealLineIsByteLengthPreserving(t *testing.T) {
	lines := []string{
		"token ghp_abcdefghijklmnopqrstuvwxyz0123456789 used by com",
		"key=AKIAIOSFODNN7EXAMPLE and key2=ghp_abcdefghijklmnopqrstuvwxyz0123456789",
		"naïve ghp_abcdefghijklmnopqrstuvwxyz0123456789 üñïçødé tail",
	}
	for _, line := range lines {
		findings := ScanText(line+"\n", Identity{}, DefaultPatterns(), DefaultIdentitySeverities(), "t")
		var idxs []int
		for i := range findings {
			if !IsIdentityKind(findings[i].Kind) && findings[i].Matched != "" {
				idxs = append(idxs, i)
			}
		}
		if len(idxs) == 0 {
			t.Fatalf("no secret finding on %q; the fixture no longer exercises the seal", line)
		}
		sealed := sealLine(line, findings, idxs)
		if len(sealed) != len(line) {
			t.Errorf("sealLine changed the byte length of %q: %d -> %d", line, len(line), len(sealed))
		}
		if !utf8.ValidString(sealed) {
			t.Errorf("sealLine produced invalid UTF-8 for %q: %q", line, sealed)
		}
	}
}

// TestIdentitySpanOffsetsSurviveTheSecretSeal is the end-to-end of the property
// above: a secret and an identity mention on one line, with a cleared
// namespace lookalike after both. The identity span is applied at an offset
// recorded before the seal ran, so a seal that shifted bytes would mask the
// wrong span — and the lookalike must survive regardless.
func TestIdentitySpanOffsetsSurviveTheSecretSeal(t *testing.T) {
	id := Identity{HomeUser: "com", HomePath: "/Users/com"}
	line := "token ghp_abcdefghijklmnopqrstuvwxyz0123456789 used by com on com.acme.app"
	got, _ := Redact(line+"\n", ScanText(line+"\n", id, DefaultPatterns(), DefaultIdentitySeverities(), "t"))
	if strings.Contains(got, "ghp_abcdefghijklmnopqrstuvwxyz0123456789") {
		t.Fatalf("the secret survived the seal: %q", got)
	}
	if !strings.Contains(got, "used by [redacted-user] on com.acme.app\n") {
		t.Errorf("identity span masked at the wrong offset after the seal, or the bundle was mangled:\n%q", got)
	}
}

// TestIdentitySpanColumnsAreByteOffsets pins the property the whole rewrite
// stands on and that the whole-string rewrite never needed: Finding.Column is a
// BYTE offset into the line, not a rune count. Every fixture here puts
// multi-byte runes BEFORE the match, so a rune-counted column would land the
// placeholder short of the span and mask the wrong bytes. Nothing else in the
// package asserts it, because until now nothing read the column to rewrite by.
func TestIdentitySpanColumnsAreByteOffsets(t *testing.T) {
	cases := []struct {
		name string
		id   Identity
		line string
		want string
	}{
		{
			name: "local_username_after_multibyte_runes",
			id:   Identity{HomeUser: "com", HomePath: "/Users/com"},
			line: "naïve — üñï com said com.acme.app",
			want: "naïve — üñï [redacted-user] said com.acme.app",
		},
		{
			name: "real_name_after_multibyte_runes",
			id:   Identity{GitUserName: "Wren"},
			line: "naïve — üñï Wren and the Wrenches",
			want: "naïve — üñï [redacted-name] and the Wrenches",
		},
		{
			name: "home_path_self_after_multibyte_runes",
			id:   Identity{HomeUser: "com", HomePath: "/Users/com"},
			line: "naïve — üñï /Users/com/x",
			want: "naïve — üñï ~/x",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := tc.line + "\n"
			findings := ScanText(text, tc.id, DefaultPatterns(), DefaultIdentitySeverities(), "t")
			got, n := Redact(text, findings)
			if n == 0 {
				t.Fatalf("Redact rewrote nothing for %q (findings %+v)", tc.line, findings)
			}
			if got != tc.want+"\n" {
				t.Errorf("byte-offset span masking:\n got  %q\n want %q\n findings %+v", got, tc.want+"\n", findings)
			}
		})
	}
}

// TestSpanMaskingFailsOpenOnClearedLookalikes pins the cost the decision
// ACCEPTED, so it is asserted rather than discovered. Each line below carries a
// genuine mention (masked) and an occurrence the detector deliberately clears.
// Under the whole-string rewrite the cleared occurrence was masked ANYWAY, as a
// side effect; it no longer is, and the stage-two re-scan does not flag it
// either, because it is the same detector. If a case here ever starts coming
// out masked, the detector's suppression changed and this file is the record of
// what that means.
//
// Not residue: home_path_self and home_path_other (the detector flags every
// path-standing occurrence, so the span rewrite covers everything the literal
// SweepCallerHome backstop would), real_email (no suppression at all), and the
// network kinds (matched per occurrence, unsuppressed).
func TestSpanMaskingFailsOpenOnClearedLookalikes(t *testing.T) {
	cases := []struct {
		name     string
		id       Identity
		line     string
		survives string
		want     string
	}{
		{
			// local_username inside a URL span: the one residue with real
			// exposure, since a forge URL carries the login in the clear.
			name:     "local_username_inside_a_url",
			id:       Identity{HomeUser: "com", HomePath: "/Users/com"},
			line:     "com filed it, see https://example.com/com/repo",
			survives: "https://example.com/com/repo",
			want:     "[redacted-user] filed it, see https://example.com/com/repo",
		},
		{
			// github_username inside a URL span — the same suppression.
			name:     "github_username_inside_a_url",
			id:       Identity{GitRemoteUsername: "wren"},
			line:     "wren filed it, see https://example.com/wren/repo",
			survives: "https://example.com/wren/repo",
			want:     "[redacted-user] filed it, see https://example.com/wren/repo",
		},
		{
			// The reverse-DNS namespace component (isDottedNamespaceComponent):
			// the case the fix exists for, and a survival that is CORRECT.
			name:     "reverse_dns_component",
			id:       Identity{HomeUser: "com", HomePath: "/Users/com"},
			line:     "com filed it about com.acme.app",
			survives: "com.acme.app",
			want:     "[redacted-user] filed it about com.acme.app",
		},
		{
			// A mid-word collision (wordBounded) — also a correct survival.
			name:     "midword_collision",
			id:       Identity{HomeUser: "com", HomePath: "/Users/com"},
			line:     "com says the commit landed",
			survives: "commit",
			want:     "[redacted-user] says the commit landed",
		},
		{
			// A username that is also a system directory, as the top segment of
			// an absolute path (isSystemPathSegment).
			name: "system_path_segment",
			// HomePath is deliberately left empty: the fixture needs only the
			// bare-login matcher, and spelling a real home root beside this
			// login would put a live home path in a committed file.
			// "opt" rather than "dev": a generic account name is not reported
			// as a bare word at all (iss-236), so the genuine mention this
			// case needs would not be one.
			id:       Identity{HomeUser: "opt"},
			line:     "opt wrote to /opt/tools",
			survives: "/opt/tools",
			want:     "[redacted-user] wrote to /opt/tools",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := tc.line + "\n"
			got, _ := Redact(text, ScanText(text, tc.id, DefaultPatterns(), DefaultIdentitySeverities(), "t"))
			if got != tc.want+"\n" {
				t.Fatalf("accepted-residue case moved:\n got  %q\n want %q", got, tc.want+"\n")
			}
			if !strings.Contains(got, tc.survives) {
				t.Errorf("the cleared occurrence %q did not survive in %q", tc.survives, got)
			}
			// The stage-two re-scan is the same detector, so it does not flag
			// the survivor either. That is what makes this FAIL-OPEN rather
			// than merely deferred.
			for _, f := range ScanText(got, tc.id, DefaultPatterns(), DefaultIdentitySeverities(), "t") {
				if IsIdentityKind(f.Kind) {
					t.Errorf("the re-scan flagged %+v; the residue is not fail-open after all", f)
				}
			}
		})
	}
}

// TestIdentitySpanOverlappingASealedSecretIsMaskedInPlace pins the one path
// that could reach the whole-string fallback from a real scan: an identity
// mention whose bytes a secret span also covers (a repo-configured pattern
// enclosing a login, say). The seal stars those bytes first, so a span
// validated against the SEALED line no longer holds its Matched text, goes
// loose, and the fallback's ReplaceAll — unable to hit the starred span —
// rewrites every cleared lookalike on the line instead. Validating against the
// pre-seal line keeps the span, masks it in place, and leaves the lookalike.
func TestIdentitySpanOverlappingASealedSecretIsMaskedInPlace(t *testing.T) {
	line := "password=alice42 leaked, see alice.acme.app"
	findings := []Finding{
		{Line: 1, Column: 1, Kind: "custom_secret", Matched: "password=alice42", line: line},
		{Line: 1, Column: 10, Kind: kindLocalUser, Matched: "alice", line: line},
	}
	got, n := Redact(line+"\n", findings)
	if strings.Contains(got, "alice42") {
		t.Fatalf("the secret survived the seal: %q", got)
	}
	if !strings.HasSuffix(got, "leaked, see alice.acme.app\n") {
		t.Errorf("a cleared lookalike was rewritten by the fallback:\n%q", got)
	}
	if !strings.Contains(got, "[redacted-user]") {
		t.Errorf("the identity span overlapping the sealed secret was not masked in place:\n%q", got)
	}
	if n != 2 {
		t.Errorf("changed = %d, want 2 (one seal, one identity span)", n)
	}
}
