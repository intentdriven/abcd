package web

import (
	"regexp"
	"strings"
	"testing"
)

// The page's layout, run for real: build() makes the page's elements in a
// small fake DOM, and the tests drive the sidebars, the overlays, the keys
// and the focus the way a person would. The fake DOM knows the stylesheet's
// two rules that decide whether an element can be seen at all: the hidden
// attribute, and each design hiding the other's .g-only or .s-only parts.

// fakeDOM is a document just big enough for build() and the layout.
const fakeDOM = `
class Node0 {
  constructor() { this.parentNode = null; this.childNodes = []; }
  get firstChild() { return this.childNodes[0] || null; }
  contains(n) { for (let x = n; x; x = x.parentNode) if (x === this) return true; return false; }
}
class Text0 extends Node0 {
  constructor(t) { super(); this.nodeType = 3; this.data = String(t); }
  get textContent() { return this.data; }
}
function dataName(k) { return "data-" + String(k).replace(/[A-Z]/g, (c) => "-" + c.toLowerCase()); }
function blurIfInside(n) { if (n.contains(document.activeElement)) document.activeElement = document.body; }
function detach(n) {
  const p = n.parentNode;
  if (!p) return;
  blurIfInside(n);
  p.childNodes.splice(p.childNodes.indexOf(n), 1);
  n.parentNode = null;
}
class El extends Node0 {
  constructor(tag) {
    super();
    this.nodeType = 1;
    this.tagName = tag.toUpperCase();
    this.attrs = new Map();
    this.listeners = {};
    this.style = { setProperty() {}, removeProperty() {} };
    const self = this;
    this.classList = {
      contains: (c) => self.className.split(/\s+/).includes(c),
      add: (c) => { if (!self.classList.contains(c)) self.className = (self.className + " " + c).trim(); },
      remove: (c) => { self.className = self.className.split(/\s+/).filter((x) => x && x !== c).join(" "); },
      toggle: (c, on) => { if (on === undefined) on = !self.classList.contains(c); if (on) self.classList.add(c); else self.classList.remove(c); return on; },
    };
    this.dataset = new Proxy({}, {
      get: (_, k) => { const v = self.getAttribute(dataName(k)); return v === null ? undefined : v; },
      set: (_, k, v) => { self.setAttribute(dataName(k), String(v)); return true; },
    });
  }
  get className() { return this.getAttribute("class") || ""; }
  set className(v) { this.setAttribute("class", v); }
  get id() { return this.getAttribute("id") || ""; }
  get title() { return this.getAttribute("title") || ""; }
  set title(v) { this.setAttribute("title", v); }
  setAttribute(k, v) { this.attrs.set(k, String(v)); }
  getAttribute(k) { return this.attrs.has(k) ? this.attrs.get(k) : null; }
  removeAttribute(k) { this.attrs.delete(k); }
  hasAttribute(k) { return this.attrs.has(k); }
  get hidden() { return this.attrs.has("hidden"); }
  set hidden(v) { if (v) this.attrs.set("hidden", ""); else this.attrs.delete("hidden"); }
  append(...ns) {
    for (let n of ns) {
      if (typeof n === "string") n = new Text0(n);
      detach(n);
      n.parentNode = this;
      this.childNodes.push(n);
    }
  }
  removeChild(n) { detach(n); return n; }
  get children() { return this.childNodes.filter((c) => c.nodeType === 1); }
  get childElementCount() { return this.children.length; }
  get textContent() { return this.childNodes.map((c) => c.textContent).join(""); }
  set textContent(v) { while (this.childNodes.length) this.removeChild(this.childNodes[0]); if (v !== "" && v !== null && v !== undefined) this.append(String(v)); }
  get isConnected() { return document.documentElement.contains(this); }
  addEventListener(t, f) { (this.listeners[t] = this.listeners[t] || []).push(f); }
  click() { for (const f of this.listeners.click || []) f({ preventDefault() {}, target: this }); }
  focus() { if (shownFake(this)) document.activeElement = this; }
  blur() { if (document.activeElement === this) document.activeElement = document.body; }
  getBoundingClientRect() { return { left: 0, top: 0, width: 0, height: 0, right: 0, bottom: 0 }; }
  getClientRects() { return [{}]; }
  get scrollWidth() { return 0; }
  get clientWidth() { return 0; }
  all() { const out = []; const walk = (n) => { for (const c of n.children) { out.push(c); walk(c); } }; walk(this); return out; }
}
// shownFake says whether the stylesheet lets an element be seen: no hidden
// ancestor, and none of the other design's own parts.
function shownFake(node) {
  if (!node || !node.isConnected) return false;
  const other = document.documentElement.dataset.design === "shared" ? "g-only" : "s-only";
  for (let n = node; n && n.nodeType === 1; n = n.parentNode) {
    if (n.hidden || n.classList.contains(other)) return false;
  }
  return true;
}
let width = 1440;
const html = new El("html");
Object.defineProperty(html, "clientWidth", { get: () => width });
const body = new El("body");
const appRoot = new El("div");
appRoot.setAttribute("id", "app");
html.append(body);
body.append(appRoot);
globalThis.document = {
  documentElement: html, body, activeElement: body, title: "",
  createElement: (t) => new El(t),
  createElementNS: (_, t) => new El(t),
  createTextNode: (t) => new Text0(t),
  getElementById: (id) => (id === "app" ? appRoot : null),
};
const windowListeners = {};
globalThis.window = { innerWidth: 0, addEventListener: (t, f) => { windowListeners[t] = f; } };
// keydown sends a key press through the page's own window listener.
function keydown(o) {
  let prevented = false;
  const e = Object.assign({ key: "", code: "", ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, isComposing: false, keyCode: 0,
    target: document.activeElement, getModifierState: () => false,
    preventDefault() { prevented = true; }, stopPropagation() {} }, o);
  windowListeners.keydown(e);
  return prevented;
}
`

// pageStubs is the rest of the page that the layout touches, held still.
const pageStubs = `
const stored = new Map();
globalThis.localStorage = {
  getItem: (k) => (stored.has(k) ? stored.get(k) : null),
  setItem: (k, v) => stored.set(k, String(v)),
  removeItem: (k) => stored.delete(k),
};
const toasts = [];
function toast(t) { toasts.push(t); }
const ICONS = {};
const route = { owner: "acme", repo: "harbour", slug: "acme/harbour" };
const session = { relay: "http://127.0.0.1:8790", code: "", name: "", login: "", role: "", myId: "p1", roster: [],
  member: false, host: false, inRoom: false, synced: false, seeding: false, everJoined: false, over: false, pending: "" };
let anchor = { state: "self", relays: [] };
let signInOffered = true;
let linkCode = "";
let heldToken = "";
let trustPending = null;
let sending = false;
let design = "guided";
let placedFor = "";
let statusText = "";
let statusBad = false;
let view = null;
let currentPath = "";
const ui = {};
const notes = [];
const mine = [];
const files = new Map();
const suggestions = new Map();
const suggestionsDone = new Map();
const doc = { transact(f) { f(); } };
const meta = { set() {} };
function relayStanding() { return "self"; }
function standingRefusal(s) { return "refused:" + s; }
function pageURL() { return "https://abcdesign.app/acme/harbour/brief/"; }
function framed() { return false; }
function qrcode() { throw new Error("no QR code in the fake DOM"); }
function inviteURL() { return "https://abcdesign.app/acme/harbour/brief/#code=" + session.code; }
function effectiveMeta() { return {}; }
function validSuggestion() { return true; }
function sendJSON() {}
function renderSend() {}
function showSendResult() {}
function refreshSignIn() {}
function confirmRelay() {}
function startSignIn() {}
function openFile() {}
function firstFile() { return ""; }
// fakeView is an editor whose text area sits in the page's editor slot.
function fakeView() {
  const dom = new El("div");
  dom.className = "cm-editor";
  const contentDOM = new El("div");
  contentDOM.className = "cm-content";
  dom.append(contentDOM);
  ui.editor.append(dom);
  return { dom, contentDOM, focus() { contentDOM.focus(); }, requestMeasure() {}, destroy() {},
    state: { selection: { main: { empty: false, from: 0, to: 4 } }, sliceDoc: () => "text" } };
}
// live puts the page in a session: a member who hosts, or a guest.
function live(role) {
  session.role = role; session.member = role === "member"; session.host = role === "member";
  session.inRoom = true; session.synced = true; session.everJoined = true; session.code = "ABC234";
  session.name = "Carol"; session.roster = [{ id: "p1", name: "Carol", role, host: session.host }];
  view = fakeView();
  renderAll();
}
function setWidth(w) { width = w; applyLayout(); }
function state() { return ui.app.dataset.left + "/" + ui.app.dataset.right; }
function focused() {
  const a = document.activeElement;
  for (const [k, v] of Object.entries(ui)) if (v === a) return k;
  if (view && a === view.contentDOM) return "editor";
  return a === document.body ? "body" : a.tagName + "." + a.className;
}
`

// pageFunctions are the functions of session.js the harness runs for real.
var pageFunctions = []string{
	"el", "clear", "recall", "remember", "say", "icon", "avatar", "badge", "prettyName", "fileIcon",
	"closeButton", "areaHead", "nextSteps", "ownerNotice",
	"storedDesign", "switchDesign", "shortcut", "onKey", "viewWidth", "inWorkspace", "tier",
	"closeTop", "defaultSheet", "toggleSide", "openSheet", "sideShown", "applyLayout", "placeParts", "fitTiles",
	"hiddenNow", "focusFallback", "focusOn", "settleFocus", "noteOpener", "returnFocus",
	"toggleInvite", "closeInvite", "submitSession", "closeSend", "askTrust", "trustAnswered",
	"setStatus", "paintStatus", "build", "renderTop", "renderStart", "renderFiles", "renderGuestBar", "renderSide",
	"openSuggestForm", "renderAll",
}

// pageConsts are the constants of session.js the harness takes as they are.
var pageConsts = []string{
	"DESIGNS", "KEY_DESIGN", "PHONE_MAX", "SLIDE_MAX", "SVG_NS", "MAX_REPLACEMENT", "GITHUB_DEVICE",
	"COLOURS", "BRIEF_DIR", "BRIEF_PREFIX", "layout", "openers",
}

// livePage is the page built in the fake DOM. With realWords the page
// shows its own words; without, each word is its key, so a test can find
// an element by the word it shows.
func livePage(t *testing.T, realWords bool) string {
	t.Helper()
	js := pageScript(t)
	var b strings.Builder
	b.WriteString(fakeDOM)
	if realWords {
		b.WriteString(wordsBlock(t, js) + "\n});\n")
	} else {
		b.WriteString("const WORDS = new Proxy({}, { get: (_, k) => String(k) });\n")
	}
	b.WriteString(pageStubs)
	for _, c := range pageConsts {
		if m := regexp.MustCompile(`(?m)^const ` + c + ` = .*;$`).FindString(js); m != "" {
			b.WriteString(m + "\n")
		}
	}
	for _, fn := range pageFunctions {
		// A function the page does not have yet is left out, so the test
		// that needs it fails on what the page does, not on the harness.
		if regexp.MustCompile(`(?m)^(?:async )?function ` + fn + `\(`).MatchString(js) {
			b.WriteString(functionBody(t, js, fn) + "\n")
		}
	}
	b.WriteString("build();\n")
	return b.String()
}

// Each sidebar keeps its state through every change of screen and design.
// A wide window and the slide-over widths share the documents' state; the
// right panel at the slide-over widths, and both sidebars on a phone, keep
// their own, so an Escape at one width never hides a panel at another.
func TestSidebarStateIsKeptPerWidthAndAcrossDesignSwitches(t *testing.T) {
	script := livePage(t, false) + `
const out = {};
live("member");
const altS = () => keydown({ altKey: true, shiftKey: true, code: "KeyS", key: "S" });
const altD = () => keydown({ altKey: true, shiftKey: true, code: "KeyD", key: "D" });
const esc = () => keydown({ key: "Escape", code: "Escape", target: document.body });
setWidth(1440);
out.wide = state();
altS();
out.wideRightOff = state();
setWidth(1100);
out.slide = state();
out.slideEsc = esc();
out.slideAfterEsc = state();
setWidth(1440);
out.wideAgain = state();
altS();
out.wideRightOn = state();
setWidth(1100);
out.slideStillClosed = state();
setWidth(390);
out.phone = state();
altD();
out.phoneDocsOff = state();
setWidth(1440);
out.wideAfterPhone = state();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketRight", key: "}" });
out.sharedWide = state();
setWidth(1280);
out.sharedSlide = state();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketLeft", key: "{" });
out.guided1280 = state();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketRight", key: "}" });
setWidth(390);
out.sharedPhone = state();
altD();
out.sharedPhoneDocs = state();
altS();
out.sharedPhoneSide = state();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketLeft", key: "{" });
out.guidedPhone = state();
console.log(JSON.stringify(out));
`
	var out map[string]any
	runPage(t, script, &out)
	want := map[string]string{
		"wide": "shown/shown", "wideRightOff": "shown/hidden",
		"slide": "shown/shown", "slideAfterEsc": "shown/hidden",
		"wideAgain": "shown/hidden", "wideRightOn": "shown/shown", "slideStillClosed": "shown/hidden",
		"phone": "shown/hidden", "phoneDocsOff": "hidden/hidden", "wideAfterPhone": "shown/shown",
		"sharedWide": "shown/shown", "sharedSlide": "shown/hidden", "guided1280": "shown/shown",
		"sharedPhone": "hidden/hidden", "sharedPhoneDocs": "shown/hidden", "sharedPhoneSide": "hidden/shown",
		"guidedPhone": "hidden/hidden",
	}
	for k, w := range want {
		if out[k] != w {
			t.Errorf("%s: sidebars %v, want %s", k, out[k], w)
		}
	}
	if out["slideEsc"] != true {
		t.Errorf("Escape at a slide-over width did not close the open slide-over")
	}
}

// The keys: each shortcut the page takes is its own (preventDefault), AltGr
// typing a character is never one, a composition's Escape and the editor's
// own Escape are left alone, and Escape closes only what can be seen.
func TestTheKeysAreThePagesOnlyWhenTheyAskForIt(t *testing.T) {
	script := livePage(t, false) + `
const out = {};
setWidth(1440);
out.sideOutside = keydown({ altKey: true, shiftKey: true, code: "KeyD", key: "D" });
out.sideOutsideState = state();
out.design = keydown({ ctrlKey: true, shiftKey: true, code: "BracketRight", key: "}" });
out.designNow = design;
out.altGr = keydown({ ctrlKey: false, altKey: false, shiftKey: true, code: "BracketLeft", key: "{", getModifierState: (m) => m === "AltGraph" });
out.altGrBoth = keydown({ ctrlKey: true, altKey: true, shiftKey: true, code: "BracketLeft", key: "{" });
// Plain Ctrl+[ and Ctrl+] are the editor's indent and outdent.
out.plainLeft = keydown({ ctrlKey: true, code: "BracketLeft", key: "[" });
out.plainRight = keydown({ ctrlKey: true, code: "BracketRight", key: "]" });
out.altGrCtrl = keydown({ ctrlKey: true, shiftKey: true, code: "BracketLeft", key: "{", getModifierState: (m) => m === "AltGraph" });
out.altGrSide = keydown({ altKey: true, shiftKey: true, code: "KeyD", key: "D", getModifierState: (m) => m === "AltGraph" });
out.afterAltGr = design;
live("member");
out.side = keydown({ altKey: true, shiftKey: true, code: "KeyD", key: "D" });
out.sideState = state();
out.altDBack = keydown({ altKey: true, shiftKey: true, code: "KeyD", key: "D" });
// A slide-over is open: the editor's Escape and a composition's are theirs.
setWidth(1100);
view.contentDOM.focus();
out.editorEsc = keydown({ key: "Escape", code: "Escape", target: view.contentDOM });
out.composeEsc = keydown({ key: "Escape", code: "Escape", isComposing: true, target: document.body });
out.ime229 = keydown({ key: "Escape", code: "Escape", keyCode: 229, target: document.body });
out.stillOpen = state();
out.pageEsc = keydown({ key: "Escape", code: "Escape", target: document.body });
out.closed = state();
out.nothingLeft = keydown({ key: "Escape", code: "Escape", target: document.body });
// State that belongs to another design or width is not on top of anything.
setWidth(1440);
ui.inviteToggle.click();
out.inviteOpen = ui.app.dataset.overlay;
keydown({ ctrlKey: true, shiftKey: true, code: "BracketLeft", key: "{" });
out.guidedOverlay = ui.app.dataset.overlay;
out.staleInvite = keydown({ key: "Escape", code: "Escape", target: document.body });
setWidth(390);
openSheet("people");
keydown({ ctrlKey: true, shiftKey: true, code: "BracketRight", key: "}" });
out.staleSheet = keydown({ key: "Escape", code: "Escape", target: document.body });
keydown({ altKey: true, shiftKey: true, code: "KeyD", key: "D" });
out.sharedPhoneDocs = state();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketLeft", key: "{" });
out.guidedSheetAgain = keydown({ key: "Escape", code: "Escape", target: document.body });
out.stalePleft = keydown({ key: "Escape", code: "Escape", target: document.body });
console.log(JSON.stringify(out));
`
	var out map[string]any
	runPage(t, script, &out)
	want := map[string]any{
		"sideOutside": false, "sideOutsideState": "shown/shown",
		"design": true, "designNow": "shared",
		"altGr": false, "altGrBoth": false, "plainLeft": false, "plainRight": false, "altGrCtrl": false, "altGrSide": false, "afterAltGr": "shared",
		"side": true, "sideState": "hidden/shown",
		"editorEsc": false, "composeEsc": false, "ime229": false, "stillOpen": "shown/shown",
		"pageEsc": true, "closed": "shown/hidden", "nothingLeft": false,
		"inviteOpen": "invite", "guidedOverlay": "", "staleInvite": false,
		"staleSheet": false, "sharedPhoneDocs": "shown/hidden", "guidedSheetAgain": true, "stalePleft": false,
	}
	for k, w := range want {
		if out[k] != w {
			t.Errorf("%s: %v, want %v", k, out[k], w)
		}
	}
}

// Escape closes the overlay on top, in order: the relay confirmation (which
// is refused), the Send panel, the invite card, the guest's suggestion,
// then a phone's sheet or sidebar, then a slide-over.
func TestEscapeClosesTheOverlayOnTop(t *testing.T) {
	script := livePage(t, false) + `
const steps = [];
const esc = () => keydown({ key: "Escape", code: "Escape", target: document.body });
const snap = () => [ui.trust.hidden ? "" : "trust", ui.app.dataset.overlay, ui.suggestForm.hidden ? "" : "form", ui.app.dataset.sheet, state()].join(" ");
live("member");
keydown({ ctrlKey: true, shiftKey: true, code: "BracketRight", key: "}" });
setWidth(1100);
ui.inviteToggle.click();
submitSession();
askTrust("join");
for (let i = 0; i < 5; i++) steps.push(esc() + " " + snap());
setWidth(1440);
steps.push(esc() + " " + snap());
keydown({ ctrlKey: true, shiftKey: true, code: "BracketLeft", key: "{" });
setWidth(390);
openSheet("people");
steps.push(esc() + " " + snap());
steps.push(esc() + " " + snap());
console.log(JSON.stringify({ steps }));
`
	var out struct{ Steps []string }
	runPage(t, script, &out)
	want := []string{
		"true  send   shown/shown",
		"true  invite   shown/shown",
		"true     shown/shown",
		"true     shown/hidden",
		"false     shown/hidden",
		"false     shown/shown",
		"true     shown/hidden",
		"false     shown/hidden",
	}
	if len(out.Steps) != len(want) {
		t.Fatalf("steps %q", out.Steps)
	}
	for i := range want {
		if out.Steps[i] != want[i] {
			t.Errorf("Escape %d: %q, want %q", i+1, out.Steps[i], want[i])
		}
	}
}

// Focus: the editor keeps it across a design switch, each dialog takes it
// when it opens and gives it back to what opened it, and a focused control
// a layout change hides hands it to the control that does its job now.
func TestFocusStaysWherePeopleCanSeeIt(t *testing.T) {
	script := livePage(t, false) + `
const out = {};
const esc = () => keydown({ key: "Escape", code: "Escape", target: document.activeElement });
setWidth(1440);
renderAll();
ui.signInStart.focus();
askTrust("signin");
out.trustOpen = focused();
esc();
out.trustClosed = focused();
live("member");
view.focus();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketRight", key: "}", target: view.contentDOM });
out.editorShared = focused();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketLeft", key: "{", target: view.contentDOM });
out.editorGuided = focused();
ui.submitG.focus();
ui.submitG.click();
out.sendOpen = ui.send.contains(document.activeElement);
esc();
out.sendClosed = focused();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketRight", key: "}" });
ui.submit.focus();
ui.submit.click();
out.sendOpenShared = ui.send.contains(document.activeElement);
[...ui.send.all()].find((b) => b.className === "close").click();
out.sendClosedShared = focused();
ui.inviteToggle.focus();
ui.inviteToggle.click();
out.inviteOpen = ui.sShare.contains(document.activeElement) && document.activeElement !== ui.sShare;
esc();
out.inviteClosed = focused();
ui.inviteToggle.click();
[...ui.sShare.children].find((b) => b.className === "close").click();
out.inviteClosedByX = focused();
// A control the layout hides hands focus on.
ui.sLeft.focus();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketLeft", key: "{" });
out.toggleAcross = focused();
ui.gRight.focus();
keydown({ ctrlKey: true, shiftKey: true, code: "BracketRight", key: "}" });
out.rightAcross = focused();
const tab = ui.files.children[0] || (() => { const b = new El("button"); ui.files.append(b); return b; })();
tab.focus();
out.inSidebar = ui.sSidebar.contains(document.activeElement);
keydown({ altKey: true, shiftKey: true, code: "KeyD", key: "D" });
out.sidebarHidden = focused();
console.log(JSON.stringify(out));
`
	var out map[string]any
	runPage(t, script, &out)
	want := map[string]any{
		"trustOpen": "trustCancel", "trustClosed": "signInStart",
		"editorShared": "editor", "editorGuided": "editor",
		"sendOpen": true, "sendClosed": "submitG", "sendOpenShared": true, "sendClosedShared": "submit",
		"inviteOpen": true, "inviteClosed": "inviteToggle", "inviteClosedByX": "inviteToggle",
		"toggleAcross": "gLeft", "rightAcross": "sRight", "inSidebar": true, "sidebarHidden": "sLeft",
	}
	for k, w := range want {
		if out[k] != w {
			t.Errorf("%s: %v, want %v", k, out[k], w)
		}
	}
}

// The sidebar toggles say what they are and whether they are on: a stable
// name, and aria-pressed. The name never flips to Hide or Show.
func TestTheSidebarTogglesKeepTheirNames(t *testing.T) {
	script := livePage(t, false) + `
live("member");
setWidth(1440);
const names = () => [ui.gLeft, ui.gRight, ui.sLeft, ui.sRight].map((b) => b.getAttribute("aria-label") + "=" + b.getAttribute("aria-pressed") + "|" + (b.title || b.dataset.tip)).join(" ");
const out = { before: names() };
keydown({ altKey: true, shiftKey: true, code: "KeyD", key: "D" });
keydown({ altKey: true, shiftKey: true, code: "KeyS", key: "S" });
out.after = names();
console.log(JSON.stringify(out));
`
	var out struct{ Before, After string }
	runPage(t, script, &out)
	wantBefore := "documents=true|documents (Alt+Shift+D) reviewPeople=true|reviewPeople (Alt+Shift+S) documents=true|documents sugShort=true|sugShort"
	wantAfter := "documents=false|documents (Alt+Shift+D) reviewPeople=false|reviewPeople (Alt+Shift+S) documents=false|documents sugShort=false|sugShort"
	if out.Before != wantBefore {
		t.Errorf("before: %s\nwant    %s", out.Before, wantBefore)
	}
	if out.After != wantAfter {
		t.Errorf("after: %s\nwant   %s", out.After, wantAfter)
	}
}

// "What happens next" is the guided design's alone. It shows on the start
// screen whenever the page can go on (or sessions are not set up yet), on
// the device sign-in, and, live, only for a guest; never in Shared.
func TestWhatHappensNextIsTheGuidedDesignsAlone(t *testing.T) {
	script := livePage(t, false) + `
const out = {};
// count is how many "What happens next" headings show, in each design.
function count() {
  const res = {};
  for (const d of ["guided", "shared"]) {
    design = d; applyLayout();
    res[d] = ui.app.all().filter((n) => n.tagName === "H2" && n.textContent === "whatNext" && shownFake(n)
      && (shownFake(ui.device) ? ui.device.contains(n) : true)).length;
  }
  design = "guided"; applyLayout();
  return res.guided + "/" + res.shared;
}
// Every one of them sits in a part only the guided design shows.
out.unmarked = ui.app.all().filter((n) => n.textContent === "whatNext" && n.tagName === "H2").filter((n) => {
  for (let x = n; x && x.nodeType === 1; x = x.parentNode) if (x.classList.contains("g-only")) return false;
  return true;
}).length;
renderAll();
out.start = count();
linkCode = "ABC234"; renderAll();
out.invited = count();
linkCode = ""; session.relay = ""; anchor = { state: "none", relays: [] }; renderAll();
out.notSetUp = count();
anchor = { state: "checking", relays: [] }; renderAll();
out.checking = count();
anchor = { state: "unavailable", relays: [] }; renderAll();
out.unreachable = count();
session.relay = "http://127.0.0.1:8790"; anchor = { state: "self", relays: [] }; renderAll();
ui.device.hidden = false;
out.device = count();
ui.device.hidden = true;
live("member");
out.member = count();
live("guest");
out.guest = count();
console.log(JSON.stringify(out));
`
	var out map[string]any
	runPage(t, script, &out)
	want := map[string]any{
		"unmarked": float64(0),
		"start":    "1/0", "invited": "1/0", "notSetUp": "1/0", "checking": "0/0", "unreachable": "0/0",
		"device": "1/0", "member": "0/0", "guest": "1/0",
	}
	for k, w := range want {
		if out[k] != w {
			t.Errorf("%s: What happens next shows %v (guided/shared), want %v", k, out[k], w)
		}
	}
}

// Shared's tab order follows what it shows: the toolbar along the top, then
// the documents, the brief, and the suggestions.
func TestSharedReadsInTheOrderItShows(t *testing.T) {
	script := livePage(t, false) + `
const shared = ui.app.all().find((n) => n.classList.contains("s-live"));
console.log(JSON.stringify(shared.children.map((c) => c.className.split(" ")[0])));
`
	var out []string
	runPage(t, script, &out)
	order := strings.Join(out, " ")
	if !strings.HasPrefix(order, "toolbar sidebar main inspector") {
		t.Errorf("Shared's parts are in the order %s; want the toolbar, the sidebar, the brief, then the inspector", order)
	}
}

// A phone's tab bar names each tab whole, and the guided app bar keeps the
// project's name: the sign-in shrinks to its icon and the name to an ellipsis.
func TestThePhoneBarsCutNothing(t *testing.T) {
	script := livePage(t, true) + `
const out = { member: [], guest: [] };
live("member");
out.member = ui.tabs.filter((b) => !b.hidden).map((b) => b.children.find((c) => c.tagName === "SPAN").textContent);
live("guest");
out.guest = ui.tabs.filter((b) => !b.hidden).map((b) => b.children.find((c) => c.tagName === "SPAN").textContent);
out.signIn = ui.signInTop.all().map((n) => n.tagName + "." + n.className);
out.signInName = ui.signInTop.textContent;
console.log(JSON.stringify(out));
`
	var out struct {
		Member, Guest, SignIn []string
		SignInName            string
	}
	runPage(t, script, &out)
	// Four tabs share a 366 px bar on a 390 px phone; at the tab's 11.5 px
	// type, eleven characters is the most that shows whole.
	for _, labels := range [][]string{out.Member, out.Guest} {
		if len(labels) == 0 {
			t.Fatal("no tabs show")
		}
		for _, l := range labels {
			if n := len([]rune(l)); n > 11 {
				t.Errorf("the tab %q is %d characters; a phone cuts it", l, n)
			}
		}
	}
	if !strings.Contains(strings.Join(out.SignIn, " "), "SVG.i") || !strings.Contains(strings.Join(out.SignIn, " "), "SPAN.tb-label") {
		t.Errorf("the app bar's sign-in is %v; want an icon and a label a phone hides", out.SignIn)
	}
	if out.SignInName != "Sign in with GitHub" {
		t.Errorf("the sign-in's name is %q", out.SignInName)
	}
	raw, err := assets.ReadFile("session.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(raw)
	project := regexp.MustCompile(`\.top \.project \{[^}]*\}`).FindString(css)
	if !strings.Contains(project, "text-overflow: ellipsis") || !strings.Contains(project, "white-space: nowrap") {
		t.Errorf("the project's name in the app bar does not end in an ellipsis: %s", project)
	}
	names := regexp.MustCompile(`\.top \.names \{[^}]*\}`).FindString(css)
	if !regexp.MustCompile(`min-width: [1-9]`).MatchString(names) {
		t.Errorf("the project's name has no minimum width: %s", names)
	}
	if strings.Contains(regexp.MustCompile(`\.tab span \{[^}]*\}`).FindString(css), "ellipsis") {
		t.Error("the phone's tab labels are cut with an ellipsis")
	}
}

// Every word in the WORDS table is one the page shows.
func TestEveryWordIsUsed(t *testing.T) {
	js := pageScript(t)
	block := wordsBlock(t, js)
	rest := strings.Replace(js, block, "", 1)
	for _, m := range regexp.MustCompile(`(?m)^  ([A-Za-z0-9]+):`).FindAllStringSubmatch(block, -1) {
		if !regexp.MustCompile(`WORDS\.` + m[1] + `\b`).MatchString(rest) {
			t.Errorf("WORDS.%s is never shown", m[1])
		}
	}
}
