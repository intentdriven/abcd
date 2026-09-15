package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/statusline"
)

// harnessFixture stands up the host harness's user-level configuration
// directory under a temp dir and points CLAUDE_CONFIG_DIR at it. An empty
// settings body means NO harness: the directory exists but holds no
// settings.json, which is the state detection must read as "nothing to offer".
// It returns the settings.json path.
func harnessFixture(t *testing.T, settings string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	path := filepath.Join(dir, "settings.json")
	if settings != "" {
		if err := os.WriteFile(path, []byte(settings), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// harnessSettingsWith renders a realistic settings.json around one statusLine
// value: other top-level keys the wiring must preserve, including a hook
// command carrying `&&`, which a default JSON encoder would rewrite as a \u0026 escape.
// An empty statusLine omits the key.
func harnessSettingsWith(statusLine string) string {
	body := `{
  "permissions": {"allow": ["Bash(git log:*)", "Bash(cd foo && make)"]},
  "hooks": {"Stop": [{"hooks": [{"type": "command", "command": "echo a && echo b"}]}]},
  "padding": 1`
	if statusLine != "" {
		body += `,
  "statusLine": ` + statusLine
	}
	return body + "\n}\n"
}

const previousStatusCommand = "bash /tmp/prev-status.sh"

// scriptedPrompter answers confirms through a function of the question and
// prompts from a map (default otherwise), recording everything asked.
type scriptedPrompter struct {
	confirm func(q string) bool
	answers map[string]string
	asked   []string
	// onPrompt, when set, runs before each element prompt is answered — the
	// seam a test uses to act as the live harness writing its settings file
	// while the user is still answering.
	onPrompt func(key string)
}

func (p *scriptedPrompter) Confirm(q string) bool {
	p.asked = append(p.asked, q)
	if p.confirm == nil {
		return false
	}
	return p.confirm(q)
}

func (p *scriptedPrompter) Prompt(key string, _ []string, def string) string {
	p.asked = append(p.asked, key)
	if p.onPrompt != nil {
		p.onPrompt(key)
	}
	if v, ok := p.answers[key]; ok {
		return v
	}
	return def
}

// offerPrompter approves every category question and answers the status-line
// offer itself with consent.
func offerPrompter(consent bool, answers map[string]string) *scriptedPrompter {
	return &scriptedPrompter{
		confirm: func(q string) bool {
			if strings.HasPrefix(q, "Apply ") {
				return true
			}
			return consent
		},
		answers: answers,
	}
}

// installedRepo adopts a fresh repo under --yes so that the only work left for
// a later interactive install is the optional status-line offer.
func installedRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "clean" {
		t.Fatalf("baseline install status = %q (remaining=%v, notes=%v)", res.Status, res.Remaining, res.Notes)
	}
	return repo
}

func readJSONFile(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%s is not a JSON object: %v\n%s", path, err, raw)
	}
	return doc
}

func statusLineOf(t *testing.T, settingsPath string) (map[string]any, bool) {
	t.Helper()
	doc := readJSONFile(t, settingsPath)
	v, ok := doc["statusLine"]
	if !ok {
		return nil, false
	}
	obj, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("statusLine is not an object: %#v", v)
	}
	return obj, true
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func statusLineGaps(gaps []Gap) []string {
	var out []string
	for _, g := range gaps {
		if strings.HasPrefix(g.ID, "statusline.") {
			out = append(out, g.ID)
		}
	}
	return out
}

// TestDetectStatusLineNoHarnessOffersNothing pins the fail-closed default: no
// settings file means no harness was detected, so nothing is offered and an
// install writes nothing under ~/.abcd for the status line.
func TestDetectStatusLineNoHarnessOffersNothing(t *testing.T) {
	home, _ := setupHermetic(t)
	harnessFixture(t, "")
	repo := installedRepo(t)

	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got := det.Signals["statusline"]; got != "no-harness" {
		t.Errorf("signals[statusline] = %v, want no-harness", got)
	}
	if ids := statusLineGaps(det.Gaps); len(ids) != 0 {
		t.Errorf("gaps raised with no harness: %v", ids)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "statusline.json")); err == nil {
		t.Error("~/.abcd/statusline.json was written with no harness present")
	}
}

// TestDetectStatusLineStates pins the signal vocabulary and which state raises
// which gap.
func TestDetectStatusLineStates(t *testing.T) {
	cases := []struct {
		name     string
		settings func(entry string) string
		state    string
		gaps     []string
	}{
		{"absent", func(string) string { return harnessSettingsWith("") }, "absent", []string{"statusline.offered"}},
		{"foreign command", func(string) string {
			return harnessSettingsWith(`{"type": "command", "command": "` + previousStatusCommand + `"}`)
		}, "foreign", []string{"statusline.offered"}},
		{"foreign type", func(string) string {
			return harnessSettingsWith(`{"type": "static", "text": "hi"}`)
		}, "foreign", []string{"statusline.offered"}},
		{"installed", func(entry string) string {
			return harnessSettingsWith(`{"type": "command", "command": "'` + entry + `' statusline"}`)
		}, "installed", nil},
		{"installed unquoted", func(entry string) string {
			return harnessSettingsWith(`{"type": "command", "command": "` + entry + ` statusline"}`)
		}, "installed", nil},
		{"dangling", func(string) string {
			return harnessSettingsWith(`{"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}`)
		}, "dangling", []string{"statusline.dangling"}},
		{"unreadable", func(string) string { return "{not json" }, "unreadable", nil},
		{"not an object", func(string) string { return "[1, 2]" }, "unreadable", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupHermetic(t)
			entry := os.Getenv("ABCD_BIN_TARGET")
			// The fixture lands AFTER the baseline install: a dangling line is a
			// required repair that install would otherwise close on the way.
			repo := installedRepo(t)
			harnessFixture(t, tc.settings(entry))
			det, err := Detect(repo)
			if err != nil {
				t.Fatal(err)
			}
			if got := det.Signals["statusline"]; got != tc.state {
				t.Errorf("signals[statusline] = %v, want %s", got, tc.state)
			}
			if got := statusLineGaps(det.Gaps); strings.Join(got, ",") != strings.Join(tc.gaps, ",") {
				t.Errorf("status-line gaps = %v, want %v", got, tc.gaps)
			}
			for _, g := range det.Gaps {
				switch g.ID {
				case "statusline.offered":
					if g.Required || !g.Resolvable || g.Category != StatusLine || g.Scope != "machine" {
						t.Errorf("offer gap has the wrong contract: %+v", g)
					}
				case "statusline.dangling":
					if !g.Required || !g.Resolvable || g.Category != ConfigChange || g.Scope != "machine" {
						t.Errorf("dangling gap has the wrong contract: %+v", g)
					}
				}
				if strings.Contains(g.Detail, os.Getenv("HOME")) || strings.Contains(g.Title, os.Getenv("HOME")) {
					t.Errorf("gap %s carries the home path: %+v", g.ID, g)
				}
			}
		})
	}
}

// TestStatusLineOfferDeclinedWritesNothing is ac-8's decline half: the whole
// fixture home and the harness directory are byte-identical after a declined
// offer, and the decline is not persisted — the next install offers again.
func TestStatusLineOfferDeclinedWritesNothing(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(""))
	repo := installedRepo(t)
	harnessDir := filepath.Dir(settings)

	homeBefore := treeHash(t, home)
	harnessBefore := treeHash(t, harnessDir)
	repoBefore := treeHash(t, repo)

	p := offerPrompter(false, nil)
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	// The dependency step reports the scanners it would have the user install
	// under Writes too ("dependency: brew install …"), and on a machine without
	// them that entry is legitimate and unrelated to the offer; the tree hashes
	// below are what prove nothing was written. Here only the two files the
	// offer owns may not appear.
	for _, w := range res.Writes {
		if strings.Contains(w, statusline.SettingsFileName) || strings.Contains(w, harnessSettingsFile) {
			t.Errorf("a declined offer wrote: %v", res.Writes)
		}
	}
	if msg, ok := sameTree(homeBefore, treeHash(t, home)); !ok {
		t.Errorf("home changed after a declined offer: %s", msg)
	}
	if msg, ok := sameTree(harnessBefore, treeHash(t, harnessDir)); !ok {
		t.Errorf("harness directory changed after a declined offer: %s", msg)
	}
	if msg, ok := sameTree(repoBefore, treeHash(t, repo)); !ok {
		t.Errorf("repo changed after a declined offer: %s", msg)
	}
	// The reason was given before the answer was taken, in one question.
	var offered bool
	for _, q := range p.asked {
		if !strings.HasPrefix(q, "Apply ") && strings.Contains(q, "badge") {
			offered = true
			if strings.Contains(q, "\n\n") {
				t.Errorf("the reason spans more than one paragraph:\n%s", q)
			}
		}
	}
	if !offered {
		t.Errorf("the offer question never named the badge: %q", p.asked)
	}
	// Not persisted: it is offered again.
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, "statusline.offered") {
		t.Error("a declined offer was persisted as a decline; it must be re-offered")
	}
}

// TestStatusLineConsentWiresBothFiles is ac-8's consent half: the user-level
// setting is written 0600 with the element switches taken, the previous
// command is recorded, and the harness is pointed at the entry with every
// other key preserved verbatim.
func TestStatusLineConsentWiresBothFiles(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(
		`{"type": "command", "command": "`+previousStatusCommand+`", "padding": 0}`))
	repo := installedRepo(t)
	entry := os.Getenv("ABCD_BIN_TARGET")

	p := offerPrompter(true, map[string]string{"statusline.branch": "off"})
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range res.Notes {
		if strings.Contains(n, "status line") {
			t.Errorf("unexpected status-line note: %s", n)
		}
	}
	wrote := strings.Join(res.Writes, "\n")
	if !strings.Contains(wrote, "~/.abcd/statusline.json") {
		t.Errorf("writes do not name the user-level setting in tilde form: %v", res.Writes)
	}
	if !strings.Contains(wrote, "settings.json") || strings.Contains(wrote, home) {
		t.Errorf("writes do not name the harness settings safely: %v", res.Writes)
	}

	// Every element after the badge was asked, in order, on/off, default on.
	var keys []string
	for _, q := range p.asked {
		if strings.HasPrefix(q, "statusline.") {
			keys = append(keys, strings.TrimPrefix(q, "statusline."))
		}
	}
	var want []string
	for _, k := range statusline.Order() {
		if k != statusline.KeyPresence {
			want = append(want, string(k))
		}
	}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Errorf("element prompts = %v, want %v", keys, want)
	}

	// The user-level setting.
	settingPath := filepath.Join(home, ".abcd", "statusline.json")
	fi, err := os.Stat(settingPath)
	if err != nil {
		t.Fatalf("user-level setting not written: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("setting mode = %o, want 0600", fi.Mode().Perm())
	}
	set, notes, err := statusline.LoadFrom(home)
	if err != nil || len(notes) != 0 {
		t.Fatalf("the written setting does not load cleanly: err=%v notes=%v", err, notes)
	}
	if set.PreviousCommand != previousStatusCommand {
		t.Errorf("previous_command = %q, want %q", set.PreviousCommand, previousStatusCommand)
	}
	if set.Enabled(statusline.KeyBranch) {
		t.Error("branch was switched off at the prompt but is enabled")
	}
	for _, k := range statusline.Order() {
		if k != statusline.KeyBranch && !set.Enabled(k) {
			t.Errorf("%s is off; only branch was switched off", k)
		}
	}
	if set.Disabled {
		t.Error("the fresh setting is disabled")
	}

	// The harness wiring.
	line, ok := statusLineOf(t, settings)
	if !ok {
		t.Fatal("statusLine key missing after consent")
	}
	if line["type"] != "command" {
		t.Errorf("statusLine.type = %v", line["type"])
	}
	if got, want := line["command"], shSingleQuote(entry)+" statusline"; got != want {
		t.Errorf("statusLine.command = %v, want %v", got, want)
	}
	if line["padding"] != float64(0) {
		t.Errorf("statusLine.padding not preserved: %v", line["padding"])
	}
	doc := readJSONFile(t, settings)
	if doc["padding"] != float64(1) {
		t.Errorf("top-level padding not preserved: %v", doc["padding"])
	}
	if _, ok := doc["permissions"]; !ok {
		t.Error("permissions key dropped")
	}
	raw, _ := os.ReadFile(settings)
	if !strings.Contains(string(raw), "echo a && echo b") || strings.Contains(string(raw), "\\u0026") {
		t.Errorf("the rewrite escaped a hook command:\n%s", raw)
	}
	if fi, err := os.Stat(settings); err != nil || fi.Mode().Perm() != 0o644 {
		t.Errorf("harness settings mode not preserved: %v", fi.Mode())
	}

	// Detection now reads installed, offers nothing, and a re-run is a no-op.
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if det.Signals["statusline"] != "installed" || len(statusLineGaps(det.Gaps)) != 0 {
		t.Errorf("after consent: signal=%v gaps=%v", det.Signals["statusline"], statusLineGaps(det.Gaps))
	}
	again, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != "already_up_to_date" || strings.Contains(strings.Join(again.OptionalSkipped, ","), StatusLineOfferGapID) {
		t.Errorf("second install = %q optional_skipped=%v, want already_up_to_date with the offer no longer pending", again.Status, again.OptionalSkipped)
	}
}

// TestStatusLineYesSkipsTheOfferAndSaysSo: --yes approves every category but
// never the offer, which rewrites a harness-wide setting; the skip is reported.
func TestStatusLineYesSkipsTheOfferAndSaysSo(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(""))
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(settings)
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(res.OptionalSkipped, StatusLineOfferGapID) {
		t.Errorf("optional_skipped = %v, want it to name %s", res.OptionalSkipped, StatusLineOfferGapID)
	}
	after, _ := os.ReadFile(settings)
	if string(before) != string(after) {
		t.Error("--yes rewrote the harness settings")
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "statusline.json")); err == nil {
		t.Error("--yes wrote the user-level setting")
	}
	again, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != "already_up_to_date" {
		t.Errorf("re-install under --yes = %q, want already_up_to_date", again.Status)
	}
	if !containsString(again.OptionalSkipped, StatusLineOfferGapID) {
		t.Errorf("re-install optional_skipped = %v, want it to name %s", again.OptionalSkipped, StatusLineOfferGapID)
	}
}

// TestStatusLineForeignCommandIsOfferedNeverReplaced: a status line abcd does
// not own is never overwritten without the offer's consent, and on consent the
// foreign command is what is recorded as the previous one.
func TestStatusLineForeignCommandIsOfferedNeverReplaced(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(
		`{"type": "command", "command": "`+previousStatusCommand+`"}`))
	repo := installedRepo(t)
	line, _ := statusLineOf(t, settings)
	if line["command"] != previousStatusCommand {
		t.Fatalf("--yes replaced a foreign status line: %v", line)
	}
	if _, err := Install(repo, InstallOptions{}, offerPrompter(true, nil)); err != nil {
		t.Fatal(err)
	}
	set, _, err := statusline.LoadFrom(home)
	if err != nil {
		t.Fatal(err)
	}
	if set.PreviousCommand != previousStatusCommand {
		t.Errorf("previous_command = %q, want the foreign command", set.PreviousCommand)
	}
}

// TestStatusLineUnknownTypeIsRefused: a statusLine of a type abcd does not
// understand is refused with a note and nothing is written on either side.
func TestStatusLineUnknownTypeIsRefused(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(`{"type": "static", "text": "hi"}`))
	repo := installedRepo(t)
	before, _ := os.ReadFile(settings)
	res, err := Install(repo, InstallOptions{}, offerPrompter(true, nil))
	if err != nil {
		t.Fatal(err)
	}
	refusals := 0
	for _, n := range res.Notes {
		if strings.Contains(n, "status line") {
			refusals++
			if !strings.Contains(n, `"static"`) || strings.Contains(n, home) {
				t.Errorf("refusal does not name the type, or carries the home path: %s", n)
			}
		}
	}
	if refusals != 1 {
		t.Errorf("notes = %v, want exactly one status-line refusal", res.Notes)
	}
	after, _ := os.ReadFile(settings)
	if string(before) != string(after) {
		t.Error("the harness settings were rewritten")
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "statusline.json")); err == nil {
		t.Error("the user-level setting was written despite the refusal")
	}
}

// TestStatusLineMalformedSettingsIsRefused: a settings.json that does not parse
// raises no offer and is never rewritten.
func TestStatusLineMalformedSettingsIsRefused(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, "{not json")
	repo := installedRepo(t)
	det0, _ := Detect(repo)
	if det0.Signals["statusline"] != "unreadable" || len(statusLineGaps(det0.Gaps)) != 0 {
		t.Errorf("signal=%v gaps=%v, want unreadable and nothing offered", det0.Signals["statusline"], statusLineGaps(det0.Gaps))
	}
	if _, err := Install(repo, InstallOptions{}, offerPrompter(true, nil)); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(settings)
	if string(after) != "{not json" {
		t.Error("the malformed settings were rewritten")
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "statusline.json")); err == nil {
		t.Error("the user-level setting was written")
	}
	// The apply-time re-read refuses too: corrupt the file between detection
	// and the step by driving the step directly against a stale gap set.
	if err := os.WriteFile(settings, []byte(harnessSettingsWith("")), 0o644); err != nil {
		t.Fatal(err)
	}
	det, _ := Detect(repo)
	if !hasGap(det.Gaps, "statusline.offered") {
		t.Fatal("precondition: offer expected once the file parses")
	}
	if err := os.WriteFile(settings, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{StatusLine: true},
		gapPresent: gapIDSet(det.Gaps), prompter: offerPrompter(true, nil), binTarget: os.Getenv("ABCD_BIN_TARGET")}
	a.stepStatusLine()
	if len(a.notes) != 1 || strings.Contains(a.notes[0], home) {
		t.Errorf("notes = %v, want one refusal without the home path", a.notes)
	}
	if len(a.writes) != 0 {
		t.Errorf("wrote %v after a refusal", a.writes)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "statusline.json")); err == nil {
		t.Error("the user-level setting was written although the harness file was refused")
	}
}

// TestStatusLineExistingSettingIsLeftAlone: a user-level setting that already
// exists is the user's configuration; the wiring only fills an empty
// previous_command and rewrites nothing else.
func TestStatusLineExistingSettingIsLeftAlone(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(
		`{"type": "command", "command": "`+previousStatusCommand+`"}`))
	repo := installedRepo(t)
	settingPath := filepath.Join(home, ".abcd", "statusline.json")
	body := "{\n  \"schema_version\": 1,\n  \"disabled\": true,\n  \"elements\": {\"model\": false},\n  \"previous_command\": \"\"\n}\n"
	if err := os.MkdirAll(filepath.Dir(settingPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	det, _ := Detect(repo)
	// The offer is not raised while the setting exists; drive the step against
	// the gap it would carry.
	a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{StatusLine: true},
		gapPresent: map[string]bool{StatusLineOfferGapID: true},
		prompter:   offerPrompter(true, map[string]string{"statusline.branch": "off"}),
		binTarget:  os.Getenv("ABCD_BIN_TARGET")}
	a.stepStatusLine()
	if len(a.notes) != 0 {
		t.Fatalf("notes = %v", a.notes)
	}
	doc := readJSONFile(t, settingPath)
	if doc["disabled"] != true {
		t.Error("the user's off switch was reset")
	}
	el, _ := doc["elements"].(map[string]any)
	if el["model"] != false || len(el) != 1 {
		t.Errorf("the user's element switches were rewritten: %v", el)
	}
	if doc["previous_command"] != previousStatusCommand {
		t.Errorf("previous_command = %v, want it filled from the harness", doc["previous_command"])
	}
	if line, _ := statusLineOf(t, settings); line["command"] != shSingleQuote(a.binTarget)+" statusline" {
		t.Errorf("harness not pointed at abcd: %v", line)
	}
}

// TestStatusLineDanglingIsRepaired: a status command naming an abcd that is
// gone blanks the line in every repository, so it is a required repair —
// repointed at the current entry, or restored to the previous command when
// abcd has no entry to offer.
func TestStatusLineDanglingIsRepaired(t *testing.T) {
	t.Run("repointed at the current entry", func(t *testing.T) {
		setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'/nowhere/at/all/abcd' statusline", "padding": 2}`))
		repo := t.TempDir()
		if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		res, err := Install(repo, installOpts(), RefusingPrompter{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Status != "clean" {
			t.Fatalf("status = %q remaining=%v notes=%v", res.Status, res.Remaining, res.Notes)
		}
		line, _ := statusLineOf(t, settings)
		if want := shSingleQuote(os.Getenv("ABCD_BIN_TARGET")) + " statusline"; line["command"] != want {
			t.Errorf("command = %v, want %v", line["command"], want)
		}
		if line["padding"] != float64(2) {
			t.Errorf("padding not preserved: %v", line["padding"])
		}
		det, _ := Detect(repo)
		if det.Signals["statusline"] != "installed" {
			t.Errorf("after repair: %v", det.Signals["statusline"])
		}
	})
	t.Run("restored when abcd has no entry", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}`))
		if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".abcd", "statusline.json"),
			[]byte(`{"schema_version": 1, "previous_command": "`+previousStatusCommand+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		repo := t.TempDir()
		det, _ := Detect(repo)
		a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{ConfigChange: true},
			gapPresent: map[string]bool{"statusline.dangling": true}, prompter: RefusingPrompter{}}
		a.det.pluginRoot = ""
		a.stepStatusLine()
		line, ok := statusLineOf(t, settings)
		if !ok || line["command"] != previousStatusCommand {
			t.Errorf("statusLine = %v, want the previous command restored", line)
		}
	})
	t.Run("removed when nothing was recorded and abcd has no entry", func(t *testing.T) {
		setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}`))
		repo := t.TempDir()
		det, _ := Detect(repo)
		a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{ConfigChange: true},
			gapPresent: map[string]bool{"statusline.dangling": true}, prompter: RefusingPrompter{}}
		a.det.pluginRoot = ""
		a.stepStatusLine()
		if _, ok := statusLineOf(t, settings); ok {
			t.Error("a dangling line with no previous command was left in place")
		}
		if doc := readJSONFile(t, settings); doc["padding"] != float64(1) {
			t.Error("the rest of the file was not preserved")
		}
	})
}

// TestUninstallRestoresTheStatusLine covers the three uninstall outcomes.
func TestUninstallRestoresTheStatusLine(t *testing.T) {
	t.Run("previous command restored", func(t *testing.T) {
		setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "`+previousStatusCommand+`", "padding": 0}`))
		repo := installedRepo(t)
		if _, err := Install(repo, InstallOptions{}, offerPrompter(true, nil)); err != nil {
			t.Fatal(err)
		}
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if !receipt.StatusLine.Restored {
			t.Errorf("receipt = %+v, want restored", receipt.StatusLine)
		}
		line, ok := statusLineOf(t, settings)
		if !ok || line["command"] != previousStatusCommand || line["type"] != "command" || line["padding"] != float64(0) {
			t.Errorf("statusLine = %v, want the previous command with its keys", line)
		}
		if doc := readJSONFile(t, settings); doc["padding"] != float64(1) {
			t.Error("top-level keys not preserved")
		}
		// The user's configuration stays.
		if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".abcd", "statusline.json")); err != nil {
			t.Error("uninstall removed ~/.abcd/statusline.json")
		}
	})
	t.Run("key removed when there was none before", func(t *testing.T) {
		setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(""))
		repo := installedRepo(t)
		if _, err := Install(repo, InstallOptions{}, offerPrompter(true, nil)); err != nil {
			t.Fatal(err)
		}
		if _, ok := statusLineOf(t, settings); !ok {
			t.Fatal("precondition: consent should have wired the line")
		}
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if !receipt.StatusLine.Restored {
			t.Errorf("receipt = %+v", receipt.StatusLine)
		}
		if _, ok := statusLineOf(t, settings); ok {
			t.Error("statusLine key left after uninstall although none existed before")
		}
	})
	t.Run("foreign command left alone", func(t *testing.T) {
		setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "`+previousStatusCommand+`"}`))
		repo := installedRepo(t)
		before, _ := os.ReadFile(settings)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.StatusLine.Restored || receipt.StatusLine.Note == "" {
			t.Errorf("receipt = %+v, want untouched with a note", receipt.StatusLine)
		}
		after, _ := os.ReadFile(settings)
		if string(before) != string(after) {
			t.Error("a foreign status line was rewritten")
		}
	})
	t.Run("no harness", func(t *testing.T) {
		setupHermetic(t)
		harnessFixture(t, "")
		repo := installedRepo(t)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.StatusLine.Restored || receipt.StatusLine.Note == "" || strings.Contains(receipt.StatusLine.Note, os.Getenv("HOME")) {
			t.Errorf("receipt = %+v", receipt.StatusLine)
		}
	})
}

// TestStatusLineCategoryIsOptionalForYes pins the category's place in the
// approval protocol: present in the canonical order, and never written by --yes
// even though --yes approves the category.
func TestStatusLineCategoryIsOptionalForYes(t *testing.T) {
	found := false
	for i, c := range categoryPromptOrder {
		if c == StatusLine {
			found = true
			if i == 0 || categoryPromptOrder[i-1] != ConfigChange {
				t.Errorf("status-line is asked after %q, want after config-change", categoryPromptOrder[i-1])
			}
		}
	}
	if !found {
		t.Fatal("StatusLine is not in categoryPromptOrder")
	}
	gaps := []Gap{{ID: StatusLineOfferGapID, Category: StatusLine, Resolvable: true}}
	if got := optionalSkipped(InstallOptions{Yes: true}, gaps); strings.Join(got, ",") != StatusLineOfferGapID {
		t.Errorf("optionalSkipped under --yes = %v", got)
	}
	if got := optionalSkipped(InstallOptions{}, gaps); len(got) != 0 {
		t.Errorf("optionalSkipped without --yes = %v, want none (it is offered)", got)
	}
}

// statusLineNotes picks the status-line refusals out of an install's notes.
func statusLineNotes(notes []string) []string {
	var out []string
	for _, n := range notes {
		if strings.Contains(n, "status line") {
			out = append(out, n)
		}
	}
	return out
}

// TestStatusLineRefusesToRecordItselfAsPrevious is the record-time half of
// the recursion guard. A hand-wired `abcd statusline`, or one reached through
// a path detection does not recognise as abcd's, is FOREIGN to detection and
// so is offered — and on consent it would be recorded as the previous command,
// which the status verb runs outside a managed checkout, which runs the
// previous command, which is abcd's status verb. Recording it forks the
// machine, so the wiring refuses in any spelling: nothing written on either
// side, and a note naming the command and the fix.
func TestStatusLineRefusesToRecordItselfAsPrevious(t *testing.T) {
	for _, cmd := range []string{
		"abcd statusline",
		"~/.local/bin/abcd statusline",
		`\"abcd\" statusline | head -c 200`,
		"'~/.local/bin/abcd' statusline",
	} {
		t.Run(cmd, func(t *testing.T) {
			home, _ := setupHermetic(t)
			settings := harnessFixture(t, harnessSettingsWith(`{"type": "command", "command": "`+cmd+`"}`))
			repo := installedRepo(t)
			det, _ := Detect(repo)
			if !hasGap(det.Gaps, StatusLineOfferGapID) {
				t.Fatalf("precondition: a hand-wired spelling is foreign to detection and is offered; gaps=%v", statusLineGaps(det.Gaps))
			}
			before, _ := os.ReadFile(settings)
			res, err := Install(repo, InstallOptions{}, offerPrompter(true, nil))
			if err != nil {
				t.Fatal(err)
			}
			refusals := statusLineNotes(res.Notes)
			if len(refusals) != 1 {
				t.Fatalf("notes = %v, want exactly one status-line refusal", res.Notes)
			}
			n := refusals[0]
			if !strings.Contains(n, "statusline") || !strings.Contains(n, "remove") || strings.Contains(n, home) {
				t.Errorf("the refusal does not name the command and the fix, or carries the home path: %s", n)
			}
			if after, _ := os.ReadFile(settings); string(after) != string(before) {
				t.Error("the harness settings were rewritten")
			}
			if _, err := os.Stat(filepath.Join(home, ".abcd", "statusline.json")); err == nil {
				t.Error("the user-level setting was written with abcd's own verb as the previous command")
			}
		})
	}
}

// TestStatusLineRestoreRefusesARecordedCommandThatIsAbcd: the two paths that
// hand a RECORDED previous command back to the harness — the dangling repair
// with no entry to offer, and uninstall — refuse one that reaches abcd's own
// status verb, since writing it into the harness is the same fork from the
// other end. The harness is left as it is and the note says why.
func TestStatusLineRestoreRefusesARecordedCommandThatIsAbcd(t *testing.T) {
	writeRecorded := func(t *testing.T, home string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".abcd", "statusline.json"),
			[]byte(`{"schema_version": 1, "previous_command": "abcd statusline"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("dangling repair", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'/nowhere/at/all/abcd' statusline"}`))
		writeRecorded(t, home)
		before, _ := os.ReadFile(settings)
		repo := t.TempDir()
		det, _ := Detect(repo)
		a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{ConfigChange: true},
			gapPresent: map[string]bool{"statusline.dangling": true}, prompter: RefusingPrompter{}}
		a.det.pluginRoot = ""
		a.stepStatusLine()
		if len(a.notes) != 1 || !strings.Contains(a.notes[0], "statusline") || strings.Contains(a.notes[0], home) {
			t.Errorf("notes = %v, want one refusal naming the recorded command without the home path", a.notes)
		}
		if after, _ := os.ReadFile(settings); string(after) != string(before) {
			t.Error("the dangling line was rewritten with abcd's own verb")
		}
	})
	t.Run("uninstall", func(t *testing.T) {
		home, _ := setupHermetic(t)
		repo := installedRepo(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'`+os.Getenv("ABCD_BIN_TARGET")+`' statusline"}`))
		writeRecorded(t, home)
		before, _ := os.ReadFile(settings)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.StatusLine.Restored || !strings.Contains(receipt.StatusLine.Note, "statusline") || strings.Contains(receipt.StatusLine.Note, home) {
			t.Errorf("receipt = %+v, want untouched with a note naming the recorded command", receipt.StatusLine)
		}
		if after, _ := os.ReadFile(settings); string(after) != string(before) {
			t.Error("uninstall wrote abcd's own verb back into the harness")
		}
	})
}

// TestStatusLineSettingNotTheCallersWordIsRefused: the user-level setting is
// a trust-boundary read everywhere it is read. A file others can write is not
// the caller's word, and its previous_command is a shell command the harness
// would run on every refresh — so uninstall leaves the harness untouched and
// says why, and install refuses rather than filling or trusting it.
func TestStatusLineSettingNotTheCallersWordIsRefused(t *testing.T) {
	writeWorldWritable := func(t *testing.T, home, body string) string {
		t.Helper()
		p := filepath.Join(home, ".abcd", "statusline.json")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o666); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, 0o666); err != nil {
			t.Fatal(err)
		}
		return p
	}
	t.Run("uninstall leaves the harness untouched", func(t *testing.T) {
		home, _ := setupHermetic(t)
		repo := installedRepo(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "'`+os.Getenv("ABCD_BIN_TARGET")+`' statusline"}`))
		writeWorldWritable(t, home, `{"schema_version": 1, "previous_command": "`+previousStatusCommand+`"}`)
		before, _ := os.ReadFile(settings)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.StatusLine.Restored {
			t.Errorf("receipt = %+v, want the restore refused", receipt.StatusLine)
		}
		if !strings.Contains(receipt.StatusLine.Note, "writable by others") || strings.Contains(receipt.StatusLine.Note, home) {
			t.Errorf("note = %q, want the reason without the home path", receipt.StatusLine.Note)
		}
		if after, _ := os.ReadFile(settings); string(after) != string(before) {
			t.Error("uninstall rewrote the harness from a setting that is not the caller's word")
		}
	})
	t.Run("install refuses", func(t *testing.T) {
		home, _ := setupHermetic(t)
		settings := harnessFixture(t, harnessSettingsWith(
			`{"type": "command", "command": "`+previousStatusCommand+`"}`))
		repo := installedRepo(t)
		body := "{\n  \"schema_version\": 1,\n  \"previous_command\": \"\"\n}\n"
		settingPath := writeWorldWritable(t, home, body)
		det, _ := Detect(repo)
		before, _ := os.ReadFile(settings)
		a := &applyCtx{cwd: repo, det: det, approved: map[GapCategory]bool{StatusLine: true},
			gapPresent: map[string]bool{StatusLineOfferGapID: true},
			prompter:   offerPrompter(true, nil), binTarget: os.Getenv("ABCD_BIN_TARGET")}
		a.stepStatusLine()
		if len(a.notes) != 1 || !strings.Contains(a.notes[0], "writable by others") || strings.Contains(a.notes[0], home) {
			t.Errorf("notes = %v, want one refusal giving the reason without the home path", a.notes)
		}
		if len(a.writes) != 0 {
			t.Errorf("wrote %v after a refusal", a.writes)
		}
		if after, _ := os.ReadFile(settings); string(after) != string(before) {
			t.Error("the harness was wired although the setting was refused")
		}
		if raw, _ := os.ReadFile(settingPath); string(raw) != body {
			t.Error("the refused setting was rewritten")
		}
	})
}

// TestStatusLineWiringMergesIntoTheLiveHarnessFile: the harness settings are
// read before the consent prompt and the element prompts, and the live harness
// may write the file while the user is answering. The wiring re-reads the file
// immediately before it writes, and when only something OTHER than the status
// line moved, merges abcd's statusLine into the fresh document — so a
// permissions rule the harness wrote during the prompts survives.
func TestStatusLineWiringMergesIntoTheLiveHarnessFile(t *testing.T) {
	setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(""))
	repo := installedRepo(t)
	p := offerPrompter(true, nil)
	p.onPrompt = func(key string) {
		if key != elementPromptPrefix+string(statusline.KeyModel) {
			return
		}
		doc := readJSONFile(t, settings)
		doc["permissions"] = map[string]any{"deny": []any{"Bash(rm -rf:*)"}}
		raw, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(settings, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	if n := statusLineNotes(res.Notes); len(n) != 0 {
		t.Errorf("unexpected status-line notes: %v", n)
	}
	doc := readJSONFile(t, settings)
	perms, _ := doc["permissions"].(map[string]any)
	if _, ok := perms["deny"]; !ok {
		t.Errorf("the permissions rule the harness wrote during the prompts was reverted: %v", doc["permissions"])
	}
	line, ok := statusLineOf(t, settings)
	if !ok || line["command"] != shSingleQuote(os.Getenv("ABCD_BIN_TARGET"))+" statusline" {
		t.Errorf("statusLine = %v, want abcd's wiring", line)
	}
}

// TestStatusLineWiringRefusesAStatusLineChangedDuringThePrompts: when the
// STATUS LINE itself moved between the read and the write, the consent was
// given over a state that no longer holds, so nothing is written and the note
// names the file and asks for a re-run.
func TestStatusLineWiringRefusesAStatusLineChangedDuringThePrompts(t *testing.T) {
	home, _ := setupHermetic(t)
	settings := harnessFixture(t, harnessSettingsWith(""))
	repo := installedRepo(t)
	concurrent := harnessSettingsWith(`{"type": "command", "command": "bash /tmp/other.sh"}`)
	p := offerPrompter(true, nil)
	p.onPrompt = func(key string) {
		if key == elementPromptPrefix+string(statusline.KeyModel) {
			if err := os.WriteFile(settings, []byte(concurrent), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	refusals := statusLineNotes(res.Notes)
	if len(refusals) != 1 || !strings.Contains(refusals[0], "settings.json") || !strings.Contains(refusals[0], "re-run") || strings.Contains(refusals[0], home) {
		t.Errorf("notes = %v, want one refusal naming the file and asking for a re-run", res.Notes)
	}
	if after, _ := os.ReadFile(settings); string(after) != concurrent {
		t.Errorf("the harness file is not what the concurrent writer left:\n%s", after)
	}
	if _, err := os.Stat(filepath.Join(home, ".abcd", "statusline.json")); err == nil {
		t.Error("the user-level setting was written although the wiring was refused")
	}
}
