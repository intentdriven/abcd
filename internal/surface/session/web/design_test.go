package web

import (
	"regexp"
	"strings"
	"testing"
)

// The page has two designs the team can try against each other: Guided, the
// default, and Shared. Ctrl+[ and Ctrl+] switch between them on every
// platform, the Mac included (never Cmd), and each viewer's choice is
// remembered in this browser alone. Both designs toggle their sidebars with
// Alt+Shift+D and Alt+Shift+S, and Escape closes the overlay on top.

// designHarness is the page's design and shortcut functions, with storage,
// the toast and the layout stubbed.
func designHarness(t *testing.T) string {
	js := pageScript(t)
	var b strings.Builder
	b.WriteString(`
const stored = new Map();
let storageThrows = false;
globalThis.localStorage = {
  getItem: (k) => { if (storageThrows) throw new Error("blocked"); return stored.has(k) ? stored.get(k) : null; },
  setItem: (k, v) => { if (storageThrows) throw new Error("blocked"); stored.set(k, String(v)); },
  removeItem: (k) => stored.delete(k),
};
const WORDS = new Proxy({}, { get: (_, k) => String(k) });
const toasts = [];
function toast(text) { toasts.push(text); }
let laidOut = 0;
function applyLayout() { laidOut++; }
let design = "";
`)
	for _, c := range []string{"DESIGNS", "KEY_DESIGN"} {
		b.WriteString(constLine(t, js, c) + "\n")
	}
	for _, fn := range []string{"recall", "remember", "storedDesign", "switchDesign", "shortcut"} {
		b.WriteString(functionBody(t, js, fn) + "\n")
	}
	return b.String()
}

func TestThePageRemembersEachViewersDesignAndDefaultsToGuided(t *testing.T) {
	script := designHarness(t) + `
const out = {};
design = storedDesign();
out.first = design;
switchDesign(1);
out.next = design;
out.stored = stored.get(KEY_DESIGN);
design = storedDesign();
out.reloaded = design;
switchDesign(-1);
out.back = design;
stored.set(KEY_DESIGN, "<img src=x>");
out.garbage = storedDesign();
storageThrows = true;
out.blocked = storedDesign();
switchDesign(1);
out.switchedBlocked = design;
out.toasts = toasts;
out.laidOut = laidOut;
console.log(JSON.stringify(out));
`
	var out struct {
		First, Next, Stored, Reloaded, Back, Garbage, Blocked, SwitchedBlocked string
		Toasts                                                                 []string
		LaidOut                                                                int
	}
	runPage(t, script, &out)
	if out.First != "guided" {
		t.Errorf("with nothing stored the page opens on %q, want guided", out.First)
	}
	if out.Next != "shared" || out.Stored != "shared" || out.Reloaded != "shared" {
		t.Errorf("switching: now %q, stored %q, after a reload %q; want shared each time", out.Next, out.Stored, out.Reloaded)
	}
	if out.Back != "guided" {
		t.Errorf("switching back gave %q, want guided", out.Back)
	}
	if out.Garbage != "guided" || out.Blocked != "guided" {
		t.Errorf("an unknown stored design gave %q and blocked storage %q; want guided for both", out.Garbage, out.Blocked)
	}
	if out.SwitchedBlocked != "shared" {
		t.Errorf("with storage blocked the switch gave %q, want shared for this page view", out.SwitchedBlocked)
	}
	if len(out.Toasts) != 3 || out.Toasts[0] != "designShared" || out.Toasts[1] != "designGuided" || out.Toasts[2] != "designShared" {
		t.Errorf("toasts %v, want each switch to name the design it switched to", out.Toasts)
	}
	if out.LaidOut != 3 {
		t.Errorf("the layout was applied %d times for 3 switches", out.LaidOut)
	}
	js := pageScript(t)
	for _, w := range []string{"designGuided", "designShared"} {
		if !regexp.MustCompile(`(?m)^  ` + w + `: "[^"]+",$`).MatchString(js) {
			t.Errorf("the page has no words for %s", w)
		}
	}
}

func TestTheShortcutsAreCtrlBracketsOnEveryPlatformAndAltShiftLetters(t *testing.T) {
	script := designHarness(t) + `
const k = (o) => shortcut(Object.assign({ ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, key: "", code: "" }, o));
console.log(JSON.stringify({
  ctrlLeft: k({ ctrlKey: true, code: "BracketLeft", key: "[" }),
  ctrlRight: k({ ctrlKey: true, code: "BracketRight", key: "]" }),
  ctrlKeyOnly: k({ ctrlKey: true, code: "Digit8", key: "[" }),
  cmdRight: k({ metaKey: true, code: "BracketRight", key: "]" }),
  ctrlCmd: k({ ctrlKey: true, metaKey: true, code: "BracketLeft", key: "[" }),
  ctrlAlt: k({ ctrlKey: true, altKey: true, code: "BracketLeft", key: "[" }),
  altLeft: k({ altKey: true, code: "BracketLeft", key: "“" }),
  docs: k({ altKey: true, shiftKey: true, code: "KeyD", key: "Î" }),
  side: k({ altKey: true, shiftKey: true, code: "KeyS", key: "Í" }),
  docsNoShift: k({ altKey: true, code: "KeyD", key: "∂" }),
  docsCtrl: k({ altKey: true, shiftKey: true, ctrlKey: true, code: "KeyD", key: "D" }),
  escape: k({ key: "Escape", code: "Escape" }),
  plain: k({ key: "d", code: "KeyD" }),
}));
`
	var out map[string]string
	runPage(t, script, &out)
	want := map[string]string{
		"ctrlLeft": "design-prev", "ctrlRight": "design-next", "ctrlKeyOnly": "design-prev",
		"cmdRight": "", "ctrlCmd": "", "ctrlAlt": "", "altLeft": "",
		"docs": "left", "side": "right", "docsNoShift": "", "docsCtrl": "",
		"escape": "escape", "plain": "",
	}
	for name, w := range want {
		if out[name] != w {
			t.Errorf("%s: shortcut gave %q, want %q", name, out[name], w)
		}
	}
	// The shortcuts are named the same on every platform, in the tooltips.
	js := pageScript(t)
	for _, w := range []string{`"Alt+Shift+D"`, `"Alt+Shift+S"`} {
		if !strings.Contains(js, w) {
			t.Errorf("session.js does not name the shortcut %s", w)
		}
	}
	for _, bad := range []string{"⌥", "⌘", "navigator.platform"} {
		if strings.Contains(js, bad) {
			t.Errorf("session.js carries %q: the shortcuts are the same on every platform", bad)
		}
	}
}

func TestEscapeClosesTheOverlayOnTop(t *testing.T) {
	js := pageScript(t)
	script := `
const calls = [];
const ui = { trust: { hidden: false }, send: { hidden: false }, suggestForm: { hidden: false } };
function trustAnswered(yes) { calls.push("trust:" + yes); ui.trust.hidden = true; }
function applyLayout() {}
let width = 400;
function viewWidth() { return width; }
function inWorkspace() { return true; }
let design = "shared";
const layout = { left: true, right: true, pleft: true, pright: false, sheet: "", invite: true };
` + constLine(t, js, "PHONE_MAX") + "\n" + constLine(t, js, "SLIDE_MAX") + "\n" + functionBody(t, js, "closeTop") + `
const steps = [];
const snap = () => JSON.stringify({ trust: ui.trust.hidden, send: ui.send.hidden, invite: layout.invite, form: ui.suggestForm.hidden, pleft: layout.pleft, right: layout.right });
for (let i = 0; i < 5; i++) { steps.push(closeTop() + " " + snap()); }
width = 1100;
steps.push(closeTop() + " " + snap());
steps.push(closeTop() + " " + snap());
width = 1440; layout.right = true;
steps.push(closeTop() + " " + snap());
design = "guided"; layout.sheet = "people"; width = 390;
steps.push(closeTop() + " sheet=" + layout.sheet);
console.log(JSON.stringify({ steps, calls }));
`
	var out struct {
		Steps []string
		Calls []string
	}
	runPage(t, script, &out)
	want := []string{
		`true {"trust":true,"send":false,"invite":true,"form":false,"pleft":true,"right":true}`,
		`true {"trust":true,"send":true,"invite":true,"form":false,"pleft":true,"right":true}`,
		`true {"trust":true,"send":true,"invite":false,"form":false,"pleft":true,"right":true}`,
		`true {"trust":true,"send":true,"invite":false,"form":true,"pleft":true,"right":true}`,
		`true {"trust":true,"send":true,"invite":false,"form":true,"pleft":false,"right":true}`,
		`true {"trust":true,"send":true,"invite":false,"form":true,"pleft":false,"right":false}`,
		`false {"trust":true,"send":true,"invite":false,"form":true,"pleft":false,"right":false}`,
		`false {"trust":true,"send":true,"invite":false,"form":true,"pleft":false,"right":true}`,
		`true sheet=`,
	}
	if len(out.Steps) != len(want) {
		t.Fatalf("steps %v", out.Steps)
	}
	for i := range want {
		if out.Steps[i] != want[i] {
			t.Errorf("Escape %d: %s, want %s", i+1, out.Steps[i], want[i])
		}
	}
	if len(out.Calls) != 1 || out.Calls[0] != "trust:false" {
		t.Errorf("the confirmation was answered %v, want one cancel", out.Calls)
	}
}

// Each design's colours are custom properties, and the cobalt accent is one
// token in each theme: no rule spells the accent's value out again.
func TestEachDesignTakesTheCobaltAccentAsOneToken(t *testing.T) {
	raw, err := assets.ReadFile("session.css")
	if err != nil {
		t.Fatal(err)
	}
	css := strings.ToLower(string(raw))
	for _, want := range []string{
		`system-ui, -apple-system, "segoe ui", roboto, "noto sans", ubuntu, cantarell, sans-serif`,
		`[data-design="guided"]`, `[data-design="shared"]`,
		`@media (prefers-color-scheme: dark)`, `@media (prefers-reduced-motion: reduce)`,
		`:focus-visible`,
	} {
		if !strings.Contains(css, want) {
			t.Errorf("session.css does not carry %s", want)
		}
	}
	for _, accent := range []string{"#2457d6", "#7aa2ff"} {
		decl := regexp.MustCompile(`--accent:\s*` + accent + `;`)
		if n := len(decl.FindAllString(css, -1)); n == 0 {
			t.Errorf("no --accent token is %s", accent)
		}
		if strings.Count(css, accent) != len(decl.FindAllString(css, -1)) {
			t.Errorf("%s is spelled out outside the --accent token", accent)
		}
	}
	for _, m := range regexp.MustCompile(`--accent:\s*([^;]+);`).FindAllStringSubmatch(css, -1) {
		if v := strings.TrimSpace(m[1]); v != "#2457d6" && v != "#7aa2ff" {
			t.Errorf("--accent is set to %s; the accent is cobalt", v)
		}
	}
	// Shared has no "What happens next": every element carrying those words
	// belongs to the guided design alone.
	js := pageScript(t)
	uses := 0
	for _, line := range strings.Split(js, "\n") {
		if !strings.Contains(line, "WORDS.whatNext") {
			continue
		}
		uses++
		if !strings.Contains(line, "g-only") {
			t.Errorf("What happens next is not the guided design's alone: %s", strings.TrimSpace(line))
		}
	}
	if uses == 0 {
		t.Error("the guided design has no What happens next")
	}
	if !strings.Contains(css, `[data-design="shared"] .g-only`) || !strings.Contains(css, `[data-design="guided"] .s-only`) {
		t.Error("session.css does not hide each design's own elements in the other")
	}
}
