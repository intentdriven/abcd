package scanner

import (
	"strings"
	"testing"
)

// TestCallerHomeTrimsTheTrailingSlash pins the one normalisation every store
// relies on when it sweeps the literal home: a HOME exported with a trailing
// slash resolves to the same string as one without, so "$HOME/x" matches
// either way.
func TestCallerHomeTrimsTheTrailingSlash(t *testing.T) {
	t.Setenv("HOME", "/base/zzhomeuser42/")
	if got := CallerHome(); got != "/base/zzhomeuser42" {
		t.Errorf("CallerHome() = %q, want the trailing slash trimmed", got)
	}
}

// TestBlockingResidualGatesOnKindNotSeverityAlone pins the stage-two rule the
// stores share: a warn-severity identity or network span refuses the write, a
// warn-severity span of any other kind does not, and a hard_fail span always
// does.
func TestBlockingResidualGatesOnKindNotSeverityAlone(t *testing.T) {
	findings := []Finding{
		{Kind: kindNetLANHost, Severity: SeverityWarn},
		{Kind: "generic:warn_only", Severity: SeverityWarn},
		{Kind: "generic:token", Severity: SeverityHardFail},
	}
	got := BlockingResidual(findings)
	if len(got) != 2 {
		t.Fatalf("BlockingResidual kept %d findings, want 2 (the identity span and the hard_fail one): %+v", len(got), got)
	}
	if got[0].Kind != kindNetLANHost || got[1].Kind != "generic:token" {
		t.Errorf("BlockingResidual kept the wrong spans: %+v", got)
	}
	if len(BlockingResidual(nil)) != 0 {
		t.Error("a clean rescan must not block")
	}
}

// TestSweepCallerHomeIsAnchoredOnAPathBoundary pins the literal $HOME sweep:
// the home is replaced where it stands as a path — at the start of a token and
// followed by a separator or the end — and left alone where it is merely a
// prefix of something else. An unanchored replace turned "/rootfs/etc/hosts"
// into "~fs/etc/hosts" and "the /root-cause" into "the ~-cause" under
// HOME=/root, and "/home/abc/x" into "~bc/x" under HOME=/home/a, silently
// corrupting the committed page.
func TestSweepCallerHomeIsAnchoredOnAPathBoundary(t *testing.T) {
	cases := []struct{ home, in, want string }{
		{"/root", "/root/x", "~/x"},
		{"/root", "HOME=/root", "HOME=~"},
		{"/root", "cd /root", "cd ~"},
		{"/root", "(/root)", "(~)"},
		{"/root", "/rootfs/etc/hosts", "/rootfs/etc/hosts"},
		// A '-' or '.' suffix is the home with a suffix, not another name: over-
		// sweeping "/root-cause" is the safe side, and the unanchored sweep did it.
		{"/root", "the /root-cause analysis", "the ~-cause analysis"},
		{"/root", "/var/root/x", "/var/root/x"},
		{"/root", "~/root/x", "~/root/x"},
		{"/home/a", "/home/abc/x", "/home/abc/x"},
		// A multi-segment home under a longer root is still the caller's home.
		{"/home/a", "/Volumes/T7/home/a/x", "/Volumes/T7~/x"},
		{"/home/a", "/Volumes/T7/home/a_snapshot/x", "/Volumes/T7~_snapshot/x"},
		{"/home/a", "HOME=/home/a /home/a/x", "HOME=~ ~/x"},
		{"", "/root/x", "/root/x"},
		// Punctuation after the home is not a longer name — only a letter or
		// digit is.
		{"/root", "saved under /root.", "saved under ~."},
		{"/root", "saved under /root. Then", "saved under ~. Then"},
		{"/root", "was /root, then", "was ~, then"},
		{"/root", "/root.old", "~.old"},
		// '_' is a word rune to the local_username rule, so a home followed by
		// '_' is one nothing else would catch: the sweep takes it, over-redacting
		// "/root_2" rather than committing the home in "/root_backup/x".
		{"/root", "/root_2", "~_2"},
		{"/root", "/root_backup/x", "~_backup/x"},
		// An empty segment before the home (file:///, a doubled slash) is not a parent.
		{"/root", "open file:///root/x", "open file://~/x"},
		{"/root", "//root/x", "/~/x"},
	}
	for _, c := range cases {
		if got := SweepCallerHome(c.in, c.home); got != c.want {
			t.Errorf("SweepCallerHome(%q, home=%q) = %q, want %q", c.in, c.home, got, c.want)
		}
	}
}

// TestSurvivingCallerHomeIgnoresAPrefixMatch pins the survivor check to the
// same boundary as the sweep: "/rootfs" is not the home "/root" and must not
// refuse a write the sweep correctly left alone, while a home that stands as a
// path is still reported.
func TestSurvivingCallerHomeIgnoresAPrefixMatch(t *testing.T) {
	if _, got := SurvivingCallerHome("/rootfs/etc/hosts", "/root"); len(got) != 0 {
		t.Errorf("a prefix match was reported as the surviving home: %+v", got)
	}
	if _, got := SurvivingCallerHome("cd /root/x", "/root"); len(got) == 0 {
		t.Error("the literal home standing as a path was not reported")
	}
	if out, _ := SurvivingCallerHome("saved under /Users/zzhomeuser42.", "/base/zzhomeuser42"); strings.Contains(out, "zzhomeuser42") { // abcd-audit:allow
		t.Errorf("the caller's username segment followed by sentence punctuation was not rewritten: %q", out)
	}
}

// TestLocalUsernameIsNotSuppressedUnderADroppedHomeSpan pins the suppression
// to the same anchor as the detection: a home_path_self span the detector
// drops (the home is a prefix of a longer name) must not go on suppressing the
// local_username finding underneath it, or neither fires and the username is
// committed verbatim.
func TestLocalUsernameIsNotSuppressedUnderADroppedHomeSpan(t *testing.T) {
	t.Setenv("HOME", "/Users/zzhomeuser42") // abcd-audit:allow
	sc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"config lives at /Users/zzhomeuser42-backup/notes.md\n", // abcd-audit:allow
		"config lives at /Users/zzhomeuser42_backup/notes.md\n", // abcd-audit:allow
	} {
		redacted, _ := Redact(text, sc.ScanText(text, "t"))
		if strings.Contains(redacted, "zzhomeuser42") {
			t.Errorf("the caller's username survived under a dropped home span:\n%s", redacted)
		}
	}
}

// TestHomePathSelfDetectionIsAnchored pins the stage-one detector to the same
// anchor as the sweep: under HOME=/root the caller's home is "/root/x", not
// "/rootfs/etc/hosts", so the first is a home_path_self finding and the second
// is left alone rather than redacted to "~fs/etc/hosts". (A bare "root" word
// elsewhere on the line is the local_username rule's business, not this one's,
// so the fixture carries none.)
func TestHomePathSelfDetectionIsAnchored(t *testing.T) {
	t.Setenv("HOME", "/root")
	sc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	text := "copied /rootfs/etc/hosts to /root/x\n"
	findings := sc.ScanText(text, "t")
	var self int
	for _, f := range findings {
		if f.Kind == kindHomeSelf {
			self++
		}
	}
	if self != 1 {
		t.Fatalf("want exactly one home_path_self finding (the /root/x path), got %d: %+v", self, findings)
	}
	redacted, _ := Redact(text, findings)
	if !strings.Contains(redacted, "/rootfs/etc/hosts") {
		t.Errorf("a path that merely starts with the home was redacted:\n%s", redacted)
	}
	if strings.Contains(redacted, "/root/x") {
		t.Errorf("the caller's home survived:\n%s", redacted)
	}
}

// TestALongerNameThanTheHomeIsStillAnotherUsersPath pins the third consumer of
// the home regex: the home_path_other skip. A /Users/<name> path whose name
// merely starts with the caller's home basename is not the caller's home (the
// anchored detector declines it), so it must fall through to home_path_other
// rather than be skipped as "the caller's own" — or nothing reports it at all
// and it is committed verbatim.
func TestALongerNameThanTheHomeIsStillAnotherUsersPath(t *testing.T) {
	t.Setenv("HOME", "/Users/zzhomeuser42") // abcd-audit:allow
	sc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"see /Users/zzhomeuser42andra/reports/q3.md\n", // abcd-audit:allow
		"see /Users/zzhomeuser42_old/notes\n",          // abcd-audit:allow
	} {
		findings := sc.ScanText(text, "t")
		if len(findings) == 0 {
			t.Errorf("no detector reported %q", text)
			continue
		}
		redacted, _ := Redact(text, findings)
		if strings.Contains(redacted, "zzhomeuser42") {
			t.Errorf("the longer name survived redaction:\n%s", redacted)
		}
	}
}

// TestHomeBehindAURLHostIsSweptNotRefused is the differential case both
// reviews found from opposite sides. Behind a URL host the byte before the
// caller's home is the host's last letter — a path-segment byte — so the
// anchored detector declined it, home_path_other declined it at its leading
// boundary, and local_username is URL-suppressed: nothing reported it, where
// the unanchored detector had. On the store side the backstop then either
// refused the whole write (a /Users or /home home) or knew nothing (any other
// root). A home inside a URL must be swept like any other, and the backstop
// must rewrite a surviving user segment rather than refuse the write.
func TestHomeBehindAURLHostIsSweptNotRefused(t *testing.T) {
	for _, home := range []string{"/Users/zzhomeuser42", "/home/zzhomeuser42", "/workspaces/zzhomeuser42", "/var/root"} { // abcd-audit:allow
		t.Run(home, func(t *testing.T) {
			t.Setenv("HOME", home)
			sc, err := New(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			text := "artifact at https://ci.example.com" + home + "/build.log and it failed\n"
			findings := sc.ScanText(text, "t")
			redacted, _ := Redact(text, findings)
			redacted = SweepCallerHome(redacted, home)
			redacted, resid := SurvivingCallerHome(redacted, home)
			if len(resid) > 0 {
				t.Errorf("the write was refused instead of rewritten: %+v", resid)
			}
			if strings.Contains(redacted, home) {
				t.Errorf("the literal home behind the URL host reached the output:\n%s", redacted)
			}
			if !strings.Contains(redacted, "and it failed") {
				t.Errorf("the rest of the line was lost:\n%s", redacted)
			}
		})
	}
	// The anchor still holds outside a URL: a longer name is not the home.
	if got := SweepCallerHome("/rootfs/etc/hosts", "/root"); got != "/rootfs/etc/hosts" {
		t.Errorf("a longer name outside a URL was swept: %q", got)
	}
}
