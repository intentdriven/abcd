package statusline

// The user-level setting: ~/.abcd/statusline.json.
//
// It holds four things and no more (spc-70): the off switch, the per-element
// switches, the presence badge's foreground and background, and the status
// command that was configured before abcd took the row over. The defaults ship
// as DATA in this package — one embedded JSON asset, parsed and validated
// once at init, so a malformed default is a build-time panic rather than a
// runtime surprise. That is how internal/core/rules ships its bundled rule set
// and the reason is the same: the shipped configuration and the user's
// override are then the same shape, read by the same parser and merged by one
// per-field rule.
//
// It is USER-LEVEL rather than repo-level because the status line is a
// facilitator's own surface. A product thinker never sees it (adr-56), so the
// badge's colours belong where the facilitator's settings live, and a
// repository must not be able to dictate the colours of a row rendered on
// somebody else's machine.
//
// READING IT IS A TRUST-BOUNDARY READ, and there is ONE reader
// (ReadSettingsFile), guarded exactly the way the two other home-scoped
// declarations abcd reads are guarded (rules.trustedRootDeclared,
// history.localDeclared): lstat first and refuse anything that is not a
// regular file, refuse a file group- or other-writable, refuse a file this
// session's uid does not own, and then read it through fsutil.ReadGuarded
// under a byte cap — one open, O_NOFOLLOW, size-checked against both the
// fstat and the bytes actually read. Those three refusals are NOTES rather
// than errors, because a file that is not the caller's word declares nothing
// and the shipped defaults are the right answer; a file that IS the caller's
// word and is malformed is an error, because silently rendering defaults over
// a configuration somebody wrote would be a setting that appears to have been
// taken and was not. The install and uninstall steps in ahoy read the file
// through the same reader, so the previous command they record and restore is
// never taken from a file this guard would refuse.

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SettingsRelPath is the user-level setting, relative to the caller's home.
const SettingsRelPath = ".abcd/statusline.json"

// SettingsFileName is its leaf name.
const SettingsFileName = "statusline.json"

// SettingsDisplay names the file in a diagnostic: the tilde form, never the
// expanded path, so a message a user pastes into a shell works and no
// diagnostic carries the caller's home path (iss-81, fsutil.RedactHome).
const SettingsDisplay = "~/" + SettingsRelPath

// maxSettingsBytes caps the read. The file is four small fields; 64 KiB bounds
// a planted device or an endless file without ever refusing a real one — the
// same cap the two sibling home-scoped declarations use.
const maxSettingsBytes = 64 << 10

// ownerUID is the package's view of fsutil.OwnerUID, held as a var for the
// reason rules/root.go holds its own: the foreign-owner branch cannot be
// provoked on a host where the test process can create only its own files, so
// substituting the lookup is the only way a detector can prove the refusal.
var ownerUID = fsutil.OwnerUID

// Pair is a badge's foreground and background, as hex.
type Pair struct {
	Foreground string `json:"foreground"`
	Background string `json:"background"`
}

// Settings is the user-level configuration of the row.
type Settings struct {
	SchemaVersion int `json:"schema_version"`
	// Disabled is the off switch: abcd contributes nothing anywhere.
	Disabled bool `json:"disabled"`
	// Elements switches each element after the badge on or off. The badge is
	// element one and is not switchable, so a key for it is ignored.
	Elements map[ElementKey]bool `json:"elements"`
	// Presence is the presence badge's pair. Held to ContrastBar at read time.
	Presence Pair `json:"presence"`
	// PreviousCommand is the status command the harness was pointed at before
	// abcd took the row, recorded at install and restored at uninstall. It is
	// what makes "managed repositories only" true under a single harness-wide
	// setting: outside a managed repository the status verb runs it with the
	// same payload, so the user's own line is unchanged everywhere else.
	PreviousCommand string `json:"previous_command"`
	// Installed reports whether these settings were READ FROM A FILE the caller
	// wrote, as against merged over an absent one. It is never serialised: it
	// is a fact about the load, not a setting. The set form of the mode verb
	// needs it (ac-7): "the host has no status surface" is this being false, or
	// Disabled being true, and a load that merged defaults over nothing is
	// otherwise indistinguishable from one that read a file holding them.
	Installed bool `json:"-"`
}

// Enabled reports whether an element renders. The badge always does.
func (s Settings) Enabled(k ElementKey) bool {
	if k == KeyPresence {
		return true
	}
	on, named := s.Elements[k]
	if !named {
		return true
	}
	return on
}

//go:embed defaults/statusline.json
var defaultsJSON []byte

// defaultSettings is parsed once at init; a malformed embedded asset is a
// build error surfaced as a panic (it can never happen at runtime).
var defaultSettings = mustParseDefaults()

func mustParseDefaults() Settings {
	var s Settings
	if err := json.Unmarshal(defaultsJSON, &s); err != nil {
		panic("statusline: the bundled defaults are malformed: " + err.Error())
	}
	if err := Validate(s); err != nil {
		panic("statusline: the bundled defaults fail validation: " + err.Error())
	}
	return s
}

// Defaults returns a deep copy of the binary-bundled defaults, safe for the
// caller to mutate.
func Defaults() Settings { return clone(defaultSettings) }

func clone(s Settings) Settings {
	out := s
	out.Elements = make(map[ElementKey]bool, len(s.Elements))
	for k, v := range s.Elements {
		out.Elements[k] = v
	}
	return out
}

// Validate checks the structural invariants the render relies on:
// schema_version is 1, every switchable element is named, and the presence
// pair parses and clears the bar. It guards the BUNDLED defaults, where any of
// those is a build error; Load reaches it with a user's file already merged
// over them and its unusable parts already replaced.
func Validate(s Settings) error {
	if s.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1, got %d", s.SchemaVersion)
	}
	for _, k := range order {
		if k == KeyPresence {
			if _, named := s.Elements[k]; named {
				return fmt.Errorf("elements names %q, which is not switchable", k)
			}
			continue
		}
		if _, named := s.Elements[k]; !named {
			return fmt.Errorf("elements names no switch for %q", k)
		}
	}
	ratio, err := Contrast(s.Presence.Foreground, s.Presence.Background)
	if err != nil {
		return fmt.Errorf("presence: %w", err)
	}
	if ratio < ContrastBar {
		return fmt.Errorf("presence measures %s, below the %s bar", FormatRatio(ratio), FormatRatio(ContrastBar))
	}
	return nil
}

// Load reads the user-level setting from the caller's own home.
func Load() (Settings, []string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return Settings{}, nil, fmt.Errorf("statusline: cannot resolve the caller's home directory: %v", err)
	}
	return LoadFrom(home)
}

// LoadFrom reads the setting from an explicit home, merges it over the bundled
// defaults per field, and returns the result together with the notes the read
// had to make.
//
// The notes are the out-of-band diagnostic channel this codebase already uses
// for exactly this (rules.Resolve, history.Resolve): core never writes to a
// stream, so a declaration it declined to honour is RETURNED for the front
// door to print on stderr. A refusal that printed nothing would be
// indistinguishable from a setting that was taken.
func LoadFrom(home string) (Settings, []string, error) {
	out := Defaults()
	path := filepath.Join(home, filepath.FromSlash(SettingsRelPath))

	raw, why, err := ReadSettingsFile(path)
	if err != nil {
		return Settings{}, nil, err
	}
	if why != "" {
		return out, []string{ignoredSetting(why)}, nil
	}
	if raw == nil {
		return out, nil, nil // no settings file is the ordinary case.
	}

	var over Settings
	if err := json.Unmarshal(raw, &over); err != nil {
		return Settings{}, nil, fmt.Errorf("statusline: %s is not valid JSON: %w", SettingsDisplay, err)
	}
	if over.SchemaVersion != 1 {
		return Settings{}, nil, fmt.Errorf("statusline: %s: schema_version must be 1, got %d", SettingsDisplay, over.SchemaVersion)
	}

	var notes []string
	out.Installed = true
	out.Disabled = over.Disabled
	if over.PreviousCommand != "" {
		out.PreviousCommand = over.PreviousCommand
	}
	notes = append(notes, mergeElements(&out, over.Elements)...)
	notes = append(notes, mergePresence(&out, over.Presence)...)
	return out, notes, nil
}

// mergeElements overlays the switches the file names, per field: a key it sets
// wins, a key it omits inherits the default. A key naming the badge and a key
// naming nothing are both ignored, each with a note — a switch that appears to
// have been taken and was not is the failure this channel exists for.
func mergeElements(out *Settings, over map[ElementKey]bool) []string {
	known := make(map[ElementKey]bool, len(order))
	for _, k := range order {
		known[k] = true
	}
	var ignored []string
	for k, v := range over {
		switch {
		case k == KeyPresence:
			ignored = append(ignored, string(k)+" (the badge is element one and is not switchable)")
		case !known[k]:
			ignored = append(ignored, termsafe.Sanitize(string(k))+" (no such element)")
		default:
			out.Elements[k] = v
		}
	}
	if len(ignored) == 0 {
		return nil
	}
	sort.Strings(ignored)
	notes := make([]string, 0, len(ignored))
	for _, name := range ignored {
		notes = append(notes, "statusline: IGNORED the element switch "+name+" in "+SettingsDisplay)
	}
	return notes
}

// mergePresence takes the configured pair, or refuses it.
//
// This is ac-9, and the refusal happens AT READ TIME rather than at render
// time for two reasons. The render is pure and has no channel to report a
// measurement on; and a pair checked once per read is checked once, where a
// pair checked at render is re-checked on every status refresh of every
// session.
//
// The refusal NAMES THE MEASURED RATIO. "Below the bar" would tell a user
// their colours were rejected; "2.82:1, below the 4.50:1 bar" tells them how
// far they have to move, which is the difference between a refusal they can
// act on and one they can only obey. The default renders in its place, so the
// row is never left unbadged because a setting was wrong.
func mergePresence(out *Settings, over Pair) []string {
	if over == (Pair{}) {
		return nil // the file said nothing; the default stands.
	}
	ratio, err := Contrast(over.Foreground, over.Background)
	if err != nil {
		return []string{refusedPresence(termsafe.Sanitize(err.Error()), out.Presence)}
	}
	if ratio < ContrastBar {
		return []string{refusedPresence(
			"it measures "+FormatRatio(ratio)+", below the "+FormatRatio(ContrastBar)+" bar",
			out.Presence)}
	}
	out.Presence = over
	return nil
}

// refusedPresence renders the one-line refusal, naming the file in tilde form
// and the default that renders in its place.
func refusedPresence(why string, fallback Pair) string {
	return "statusline: REFUSED the presence colours in " + SettingsDisplay + " — " + why +
		"; the default " + fallback.Foreground + " on " + fallback.Background + " renders instead"
}

// ReadSettingsFile performs the trust-boundary read of the user-level setting
// at path. It is the ONE reader of that file: Load reads through it to render
// the row, and ahoy's install and uninstall steps read through it to record
// and restore the previous command. The second consumer is why it is exported.
// `previous_command` is a shell command the harness runs on every refresh,
// and a reader that took it from a file others can write, or that is not the
// caller's own, would hand the harness a command somebody else chose — the
// guard has to be the same guard wherever the file is read, or the weakest
// reader is the one that runs.
//
// Three outcomes, and the caller phrases the consequence of each:
//
//   - (nil, "", nil): there is no file — the ordinary case, not a diagnostic;
//   - (nil, why, nil): a file is present but is NOT the caller's word — not a
//     regular file, writable by group or others, or owned by another uid —
//     and why says which, as a clause after the file's name ("it is writable
//     by others, ..."). Nothing is read from such a file: a caller that
//     renders substitutes the defaults, a caller that would write refuses;
//   - (nil, "", err): a file that IS the caller's word cannot be read (over
//     the cap, an I/O error). The error names the file in tilde form.
//
// The guard is the one the two sibling home-scoped declarations use
// (rules.trustedRootDeclared, history.localDeclared): lstat first, the three
// refusals above, then fsutil.ReadGuarded under the byte cap — one open,
// O_NOFOLLOW, size-checked against both the fstat and the bytes read.
func ReadSettingsFile(path string) (raw []byte, why string, err error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, "", nil
	}
	switch {
	case !fi.Mode().IsRegular():
		return nil, "it is not a regular file", nil
	case fi.Mode().Perm()&0o022 != 0:
		return nil, "it is writable by others, so its contents are not necessarily yours", nil
	}
	if owner, err := ownerUID(path); err != nil || owner != uint32(os.Getuid()) {
		return nil, "it is not owned by this session's uid", nil
	}
	raw, err = fsutil.ReadGuarded(path, maxSettingsBytes)
	switch {
	case err == nil:
		return raw, "", nil
	case errors.Is(err, fsutil.ErrTooBig):
		return nil, "", fmt.Errorf("statusline: %s exceeds the %d-byte cap", SettingsDisplay, maxSettingsBytes)
	case errors.Is(err, fsutil.ErrNotRegular):
		return nil, "it is not a regular file", nil
	default:
		return nil, "", fmt.Errorf("statusline: reading %s: %s", SettingsDisplay, termsafe.Sanitize(err.Error()))
	}
}

// ignoredSetting renders the one-line reason a present setting was not
// honoured, naming the file in tilde form so no home path is carried.
func ignoredSetting(why string) string {
	return "statusline: IGNORED " + SettingsDisplay + " — " + why + "; the bundled defaults render"
}
