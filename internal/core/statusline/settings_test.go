package statusline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeSettings lays a settings file at <home>/.abcd/statusline.json.
func writeSettings(t *testing.T, home, body string) string {
	t.Helper()
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, SettingsFileName)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestDefaultsAreWellFormed holds the bundled data asset to the contract the
// render relies on: schema 1, every switchable element named and on, a
// presence pair that parses, and the pair iss-168 settled on 2026-08-29.
func TestDefaultsAreWellFormed(t *testing.T) {
	d := Defaults()
	if d.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", d.SchemaVersion)
	}
	if d.Disabled {
		t.Fatal("the shipped defaults are disabled")
	}
	if d.PreviousCommand != "" {
		t.Fatalf("previous_command = %q, want empty until install records one", d.PreviousCommand)
	}
	for _, k := range Order() {
		if k == KeyPresence {
			if _, named := d.Elements[k]; named {
				t.Fatal("the defaults carry a switch for the badge, which is not switchable")
			}
			continue
		}
		on, named := d.Elements[k]
		if !named {
			t.Fatalf("the defaults name no switch for %q", k)
		}
		if !on {
			t.Fatalf("%q ships switched off; every element after the badge is on by default", k)
		}
	}
	if d.Presence.Foreground != "#f0c052" || d.Presence.Background != "#444444" {
		t.Fatalf("presence pair = %+v, want the house yellow on dark grey", d.Presence)
	}
	if err := Validate(d); err != nil {
		t.Fatalf("the shipped defaults do not validate: %v", err)
	}
}

// TestDefaultsAreACopy: a caller that switches an element off must not change
// what the next caller gets.
func TestDefaultsAreACopy(t *testing.T) {
	a := Defaults()
	a.Elements[KeyBranch] = false
	a.Presence.Foreground = "#000000"
	b := Defaults()
	if !b.Elements[KeyBranch] {
		t.Fatal("mutating one Defaults() changed the next")
	}
	if b.Presence.Foreground != "#f0c052" {
		t.Fatal("mutating one Defaults().Presence changed the next")
	}
}

// TestLoadWithNoFileReturnsTheDefaults is the ordinary case: no settings file
// is not a diagnostic, it is the shipped configuration.
func TestLoadWithNoFileReturnsTheDefaults(t *testing.T) {
	home := t.TempDir()
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("an absent settings file produced notes: %v", notes)
	}
	if got.Presence != Defaults().Presence || len(got.Elements) != len(Defaults().Elements) {
		t.Fatalf("LoadFrom = %+v, want the defaults", got)
	}
}

// TestLoadMergesPerField is the rules.json merge contract applied here: a
// field the file sets wins, a field it omits inherits the default.
func TestLoadMergesPerField(t *testing.T) {
	home := t.TempDir()
	writeSettings(t, home, `{"schema_version":1,"elements":{"branch":false},"previous_command":"/bin/echo hi"}`)
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("unexpected notes: %v", notes)
	}
	if got.Enabled(KeyBranch) {
		t.Fatal("branch is on, but the file switched it off")
	}
	if !got.Enabled(KeyModel) {
		t.Fatal("model is off, but the file said nothing about it")
	}
	if got.Presence != Defaults().Presence {
		t.Fatalf("presence = %+v, want the default the file did not override", got.Presence)
	}
	if got.PreviousCommand != "/bin/echo hi" {
		t.Fatalf("previous_command = %q", got.PreviousCommand)
	}
}

// TestLoadOffSwitch is ac-6 at the setting: the kill switch reads back.
func TestLoadOffSwitch(t *testing.T) {
	home := t.TempDir()
	writeSettings(t, home, `{"schema_version":1,"disabled":true}`)
	got, _, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if !got.Disabled {
		t.Fatal("the off switch did not survive the load")
	}
	if !Render(Input{State: StateManaged}, got).Empty() {
		t.Fatal("a disabled setting still rendered a row")
	}
}

// TestLoadRefusesAPairBelowTheBar is ac-9: a configured pair below the
// contrast bar is refused AT READ TIME, the refusal names the measured ratio,
// and the default renders in its place.
func TestLoadRefusesAPairBelowTheBar(t *testing.T) {
	cases := []struct {
		name   string
		fg, bg string
	}{
		// The worst case: a colour on itself, 1.00:1.
		{name: "identical", fg: "#444444", bg: "#444444"},
		// Plausible, legible-looking, and still under the bar.
		{name: "plausible but low", fg: "#8a8a8a", bg: "#444444"},
		// Just under: the boundary must be refused from below.
		{name: "just under the bar", fg: "#ffffff", bg: "#797979"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			measured, err := Contrast(tc.fg, tc.bg)
			if err != nil {
				t.Fatal(err)
			}
			if measured >= ContrastBar {
				t.Fatalf("the fixture pair measures %.2f, which clears the bar — pick a different one", measured)
			}
			home := t.TempDir()
			writeSettings(t, home, `{"schema_version":1,"presence":{"foreground":"`+tc.fg+`","background":"`+tc.bg+`"}}`)
			got, notes, err := LoadFrom(home)
			if err != nil {
				t.Fatalf("LoadFrom: %v", err)
			}
			if got.Presence != Defaults().Presence {
				t.Fatalf("presence = %+v, want the default to have been substituted", got.Presence)
			}
			if len(notes) != 1 {
				t.Fatalf("notes = %v, want exactly one refusal", notes)
			}
			note := notes[0]
			if !strings.Contains(note, FormatRatio(measured)) {
				t.Fatalf("the refusal does not report the measured ratio %s: %q", FormatRatio(measured), note)
			}
			if !strings.Contains(note, FormatRatio(ContrastBar)) {
				t.Fatalf("the refusal does not name the bar: %q", note)
			}
			if !strings.Contains(note, SettingsDisplay) {
				t.Fatalf("the refusal does not name the file in tilde form: %q", note)
			}
		})
	}
}

// TestLoadAdmitsAPairAtOrAboveTheBar: a legal pair reaches the render
// untouched, and the boundary is inclusive.
func TestLoadAdmitsAPairAtOrAboveTheBar(t *testing.T) {
	home := t.TempDir()
	writeSettings(t, home, `{"schema_version":1,"presence":{"foreground":"#ffffff","background":"#767676"}}`)
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	ratio, err := Contrast("#ffffff", "#767676")
	if err != nil {
		t.Fatal(err)
	}
	if ratio < ContrastBar {
		t.Fatalf("the fixture pair measures %.2f, which is below the bar — pick a different one", ratio)
	}
	if len(notes) != 0 {
		t.Fatalf("a legal pair produced notes: %v", notes)
	}
	if got.Presence.Foreground != "#ffffff" || got.Presence.Background != "#767676" {
		t.Fatalf("presence = %+v, want the configured pair", got.Presence)
	}
}

// TestLoadRefusesAMalformedPair: an unparseable colour is refused the same way
// a low-contrast one is — with a note, and the default in its place.
func TestLoadRefusesAMalformedPair(t *testing.T) {
	home := t.TempDir()
	writeSettings(t, home, `{"schema_version":1,"presence":{"foreground":"gold","background":"#444444"}}`)
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got.Presence != Defaults().Presence {
		t.Fatalf("presence = %+v, want the default", got.Presence)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "gold") {
		t.Fatalf("notes = %v, want one refusal naming the value", notes)
	}
}

// TestLoadIgnoresABadgeSwitch: the badge is not switchable, so a file that
// names it is ignored — and says so, because a silently ignored setting is
// indistinguishable from one that was honoured.
func TestLoadIgnoresABadgeSwitch(t *testing.T) {
	home := t.TempDir()
	writeSettings(t, home, `{"schema_version":1,"elements":{"presence":false}}`)
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if !got.Enabled(KeyPresence) {
		t.Fatal("the badge was switched off by the settings file")
	}
	if len(notes) != 1 || !strings.Contains(notes[0], string(KeyPresence)) {
		t.Fatalf("notes = %v, want one note naming the badge", notes)
	}
}

// TestLoadNotesAnUnknownElementKey: a typo in a switch name would otherwise be
// a setting that appears to have been taken and was not.
func TestLoadNotesAnUnknownElementKey(t *testing.T) {
	home := t.TempDir()
	writeSettings(t, home, `{"schema_version":1,"elements":{"brnach":false}}`)
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if !got.Enabled(KeyBranch) {
		t.Fatal("a typo switched a real element off")
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "brnach") {
		t.Fatalf("notes = %v, want one note naming the unknown key", notes)
	}
}

// TestLoadRefusesAMalformedFile: the file is a trust-boundary read, so a
// broken one is an error the front door reports rather than a silent fallback
// that looks like a working configuration.
func TestLoadRefusesAMalformedFile(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "not json", body: `{`, want: "not valid JSON"},
		{name: "wrong schema", body: `{"schema_version":2}`, want: "schema_version"},
		{name: "missing schema", body: `{"disabled":true}`, want: "schema_version"},
		{name: "an array", body: `[]`, want: "not valid JSON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			writeSettings(t, home, tc.body)
			_, _, err := LoadFrom(home)
			if err == nil {
				t.Fatal("LoadFrom returned no error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to mention %q", err, tc.want)
			}
			if !strings.Contains(err.Error(), SettingsDisplay) {
				t.Fatalf("error = %v, want it to name the file in tilde form", err)
			}
		})
	}
}

// TestLoadRefusesANonRegularFile is the guarded-read contract shared with
// rules.trustedRootDeclared and history.localDeclared: a setting that is not a
// regular file is not the caller's word.
func TestLoadRefusesANonRegularFile(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(filepath.Join(dir, SettingsFileName), 0o700); err != nil {
		t.Fatal(err)
	}
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "regular file") {
		t.Fatalf("notes = %v, want one note about a non-regular file", notes)
	}
	if got.Presence != Defaults().Presence {
		t.Fatal("the defaults were not substituted")
	}
}

// TestLoadRefusesASymlink: O_NOFOLLOW at the leaf, so a planted link cannot
// redirect the read to a file the caller did not write.
func TestLoadRefusesASymlink(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".abcd")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, "elsewhere.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":1,"disabled":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, SettingsFileName)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got.Disabled {
		t.Fatal("the symlinked file was read")
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "regular file") {
		t.Fatalf("notes = %v, want one note about a non-regular file", notes)
	}
}

// TestLoadRefusesAWorldWritableFile: a file anyone can write is not
// necessarily the caller's, the same judgement rules/root.go makes.
func TestLoadRefusesAWorldWritableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits")
	}
	home := t.TempDir()
	path := writeSettings(t, home, `{"schema_version":1,"disabled":true}`)
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatal(err)
	}
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got.Disabled {
		t.Fatal("a world-writable settings file was honoured")
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "writable by others") {
		t.Fatalf("notes = %v, want one note about the permissions", notes)
	}
}

// TestLoadRefusesAForeignOwner: the owner-uid check, with the lookup
// substituted, because a test process can only create files it owns.
func TestLoadRefusesAForeignOwner(t *testing.T) {
	home := t.TempDir()
	writeSettings(t, home, `{"schema_version":1,"disabled":true}`)
	orig := ownerUID
	t.Cleanup(func() { ownerUID = orig })
	ownerUID = func(string) (uint32, error) { return 4242, nil }

	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got.Disabled {
		t.Fatal("a settings file owned by another uid was honoured")
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "uid") {
		t.Fatalf("notes = %v, want one note about the owner", notes)
	}
}

// TestLoadRefusesAnOversizeFile bounds the read the way the two sibling
// home-scoped declarations bound theirs.
func TestLoadRefusesAnOversizeFile(t *testing.T) {
	home := t.TempDir()
	pad := strings.Repeat("x", maxSettingsBytes+1)
	writeSettings(t, home, `{"schema_version":1,"previous_command":"`+pad+`"}`)
	_, _, err := LoadFrom(home)
	if err == nil {
		t.Fatal("LoadFrom returned no error for an oversize file")
	}
	if !strings.Contains(err.Error(), "cap") {
		t.Fatalf("error = %v, want it to name the size cap", err)
	}
}

// TestSettingsRoundTrip: what Load reads, install writes back, so the shape
// must survive a marshal/unmarshal cycle unchanged.
func TestSettingsRoundTrip(t *testing.T) {
	want := Defaults()
	want.PreviousCommand = "/usr/local/bin/mystatus"
	want.Elements[KeyModel] = false
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	writeSettings(t, home, string(data))
	got, notes, err := LoadFrom(home)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("a round trip of the defaults produced notes: %v", notes)
	}
	if got.PreviousCommand != want.PreviousCommand || got.Enabled(KeyModel) || got.Presence != want.Presence {
		t.Fatalf("round trip lost a field: %+v", got)
	}
}

// TestLoadReportsWhetherTheSettingWasInstalled: the set form of the mode verb
// prints its one-line notice only where the host has NO status surface
// (ac-7), and "no surface" is a fact about the setting file — absent, or
// present with the off switch thrown. A load that merged defaults over an
// absent file is indistinguishable from one that read a file that happens to
// hold the defaults, so the reader says which it was.
func TestLoadReportsWhetherTheSettingWasInstalled(t *testing.T) {
	home := t.TempDir()
	got, _, err := LoadFrom(home)
	if err != nil {
		t.Fatal(err)
	}
	if got.Installed {
		t.Fatal("no file on disk, yet the setting reads as installed")
	}
	if Defaults().Installed {
		t.Fatal("the bundled defaults must not read as installed: nothing wrote them")
	}
	writeSettings(t, home, `{"schema_version":1}`)
	got, _, err = LoadFrom(home)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Installed {
		t.Fatal("a present, honoured file must read as installed")
	}
	if b, _ := json.Marshal(got); strings.Contains(string(b), "nstalled") {
		t.Fatalf("Installed leaked into the serialised setting: %s", b)
	}
}
