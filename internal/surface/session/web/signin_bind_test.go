package web

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// The web sign-in's code goes only to the relay GitHub returned the person
// to: the team's app lists the relay's own callback, the relay sends the
// person on to the page naming itself, and the page posts the code to that
// relay alone, never to one the invite link or the tab's storage names. A
// relay the person confirmed, where the repository's list cannot be read,
// may run the sign-in: GitHub sends the code only to a callback the team's
// app lists. The token then goes for verification only to the relay GitHub
// returned the person to.

// pageHarness is the page's sign-in, storage and trust functions, with the
// browser around them stubbed: storage, location, history and fetch, which
// records every request.
func pageHarness(t *testing.T) string {
	js := pageScript(t)
	var b strings.Builder
	b.WriteString(`
const store = () => { const m = new Map(); return { getItem: (k) => m.has(k) ? m.get(k) : null, setItem: (k, v) => m.set(k, String(v)), removeItem: (k) => m.delete(k), m }; };
globalThis.sessionStorage = store();
globalThis.localStorage = store();
const WORDS = new Proxy({}, { get: (_, k) => String(k) });
const toasts = [];
function toast(text) { toasts.push(text); }
function renderAll() {}
function setStatus() {}
function updateFragment() {}
function framed() { return false; }
function readStartForm() {}
function askTrust(action) { toasts.push("askTrust:" + action); }
let linkRelay = "";
let rechecking = false;
async function refreshSignIn() {}
function startDeviceSignIn() { toasts.push("device"); }
function takeRejoin() { return false; }
// listSeen, when set, is what the repository's list shows a token; else the
// anchor as it stands.
let listSeen = null;
async function readRelayList() { return listSeen || { state: anchor.state, relays: anchor.relays, app: anchor.app }; }
let relayApp = { relay: "", clientID: "", state: "" };
let appCheck = null;
const sent = [];
let socket = false;
function socketOpen() { return socket; }
function sendJSON(msg) { sent.push({ relay: session.relay, msg }); }
function openSocket() { sent.push({ relay: session.relay, msg: { type: "open" } }); }
let heldToken = "";
let githubToken = "";
let tokenRelay = "";
let signInOffered = null;
const requests = [];
let assigned = "";
const location = { origin: "https://abcdesign.app", pathname: "/acme/brief/brief/", search: "", hash: "", assign: (u) => { assigned = u; } };
globalThis.history = { replaceState: () => {} };
const route = { slug: "acme/brief" };
const session = { relay: "", code: "", name: "", everJoined: false, inRoom: false, verifying: false, rejoining: false };
const rejoinToken = "";
function proceed() {}
let anchor = { state: "listed", relays: [] };
let config = { client_id: "Iv1.abc", sign_in: true, app_type: "github-app", authorize_url: "https://github.com/login/oauth/authorize", flow: "web" };
// configAnswer, when set, is how the relay's /config fails instead: an HTTP
// status, or "throw" for a request that never reaches it.
let configAnswer = null;
globalThis.fetch = async (url, init) => {
  requests.push({ url, body: init && init.body ? JSON.parse(init.body) : null });
  if (url.endsWith("/config") && configAnswer === "throw") throw new Error("unreachable");
  if (url.endsWith("/config") && typeof configAnswer === "number") return { ok: false, status: configAnswer, json: async () => ({}) };
  if (url.endsWith("/config")) return { ok: true, status: 200, json: async () => config };
  return { ok: true, status: 200, json: async () => ({ access_token: "ghu_x" }) };
};
`)
	// Each piece is taken when session.js has it, so the harness runs the
	// page as it stands; a test asserts on what the page does.
	for _, c := range []string{"GITHUB_AUTHORIZE", "KEY_SIGN_IN", "KEY_TRUSTED", "CONFIRM_LIFE", "CALLBACK_PATH", "CODE_ALPHABET", "RELAY_WAKE"} {
		if regexp.MustCompile(`(?m)^const ` + c + ` = `).MatchString(js) {
			b.WriteString(constLine(t, js, c) + "\n")
		}
	}
	for _, fn := range []string{
		"recall", "remember", "stashSignIn", "takeSignIn", "trustedRelays", "keepTrusted", "liveConfirmations", "confirmedRelays", "confirmRelay", "forgetRelay",
		"pageURL", "relayOrigin", "normalizeCode", "b64url", "randomText", "isLoopbackOrigin", "relayConfig",
		"appRequired", "appStanding", "relayAnswersAs", "checkRelayApp",
		"relayStanding", "relayCleared", "relayTokenCleared", "standingRefusal", "relayClearedFor", "chooseRelay", "recheckRelayList",
		"startSignIn", "finishSignIn", "presentHeldToken", "connectSignedIn", "afterOpen",
	} {
		if regexp.MustCompile(`(?m)^(?:async )?function ` + fn + `\(`).MatchString(js) {
			b.WriteString(functionBody(t, js, fn) + "\n")
		}
	}
	return b.String()
}

func TestASignInCodeGoesOnlyToTheRelayGitHubReturnedTo(t *testing.T) {
	const relay, other = "https://brief-relay.onrender.com", "https://other-relay.example.org"
	script := pageHarness(t) + `
const pending = (r) => ({ state: "s1.cGFnZQ", verifier: "v".repeat(43), relay: r, code: "", name: "" });
async function attempt(name, { stored, returned, listed, confirmed, anchorState, linked }) {
  sessionStorage.m.clear();
  requests.length = 0; toasts.length = 0; sent.length = 0; heldToken = ""; githubToken = ""; tokenRelay = "";
  anchor = { state: anchorState || "listed", relays: listed || [] };
  // The relay the invite link named, which main chose before the return.
  session.relay = linked || "";
  if (confirmed) confirmRelay(confirmed);
  sessionStorage.setItem(KEY_SIGN_IN, JSON.stringify(pending(stored)));
  const q = new URLSearchParams({ code: "good-code", state: "s1.cGFnZQ" });
  if (returned !== undefined) q.set("relay", returned);
  location.search = "?" + q.toString();
  await finishSignIn();
  const posts = requests.filter((r) => r.url.endsWith("/auth/token"));
  out[name] = {
    posts: posts.map((p) => ({ url: p.url, redirect: p.body.redirect_uri })),
    verified: sent.filter((x) => x.msg.type === "open" || x.msg.type === "verify").map((x) => x.relay),
  };
}
const out = {};
(async () => {
  await attempt("returned", { stored: "` + relay + `", returned: "` + relay + `", listed: ["` + relay + `"] });
  await attempt("no-relay", { stored: "` + relay + `", listed: ["` + relay + `"] });
  await attempt("another-relay", { stored: "` + relay + `", returned: "` + other + `", listed: ["` + relay + `", "` + other + `"] });
  await attempt("stored-differs", { stored: "` + other + `", returned: "` + relay + `", listed: ["` + relay + `", "` + other + `"] });
  await attempt("unlisted", { stored: "` + other + `", returned: "` + other + `", listed: ["` + relay + `"] });
  await attempt("confirmed", { stored: "` + other + `", returned: "` + other + `", anchorState: "unknown", confirmed: "` + other + `" });
  // The invite link named another relay than the one GitHub returned to.
  await attempt("link-differs", { stored: "` + relay + `", returned: "` + relay + `", anchorState: "unknown", confirmed: "` + relay + `", linked: "` + other + `" });
  console.log(JSON.stringify(out));
})();
`
	type result struct {
		Posts    []struct{ URL, Redirect string }
		Verified []string
	}
	var out map[string]result
	runPage(t, script, &out)
	// The code, and then the token, go to the relay GitHub returned to: a
	// listed one, a confirmed one, and one other than the invite link named.
	for name, to := range map[string]string{"returned": relay, "confirmed": other, "link-differs": relay} {
		got := out[name]
		if len(got.Posts) != 1 || got.Posts[0].URL != to+"/auth/token" || got.Posts[0].Redirect != to+"/auth/callback" {
			t.Errorf("%s: posts %+v, want one post to %s/auth/token naming its callback", name, got.Posts, to)
		}
		if len(got.Verified) != 1 || got.Verified[0] != to {
			t.Errorf("%s: the token went to %v, want %s alone", name, got.Verified, to)
		}
	}
	for name, why := range map[string]string{
		"no-relay":       "a code arriving with no relay named",
		"another-relay":  "a code returned through a relay other than the one the sign-in started with",
		"stored-differs": "a code whose stored relay differs from the one GitHub returned to",
		"unlisted":       "a relay the repository's list does not name",
	} {
		if c := out[name]; len(c.Posts) != 0 || len(c.Verified) != 0 {
			t.Errorf("%s: %+v, want no code and no token sent anywhere", why, c)
		}
	}
}

// The token is presented for verification only to the relay it was signed
// in through: never to a relay the invite link named that GitHub did not
// return the person to, whichever way the page would present it.
func TestTheTokenGoesOnlyToTheRelayItWasSignedInThrough(t *testing.T) {
	const relay, other = "https://brief-relay.onrender.com", "https://other-relay.example.org"
	script := pageHarness(t) + `
const out = {};
(async () => {
  anchor = { state: "unknown", relays: [] };
  confirmRelay("` + relay + `");
  confirmRelay("` + other + `");
  tokenRelay = "` + relay + `";
  // A held token, with the page pointed at the link's relay.
  session.relay = "` + other + `";
  heldToken = "ghu_x";
  await presentHeldToken(false);
  out.present = { sent: sent.slice(), token: githubToken, held: heldToken };
  // A token already in hand, on an open connection and on a new one.
  githubToken = "ghu_x"; sent.length = 0;
  socket = true;
  connectSignedIn();
  out.connect = sent.length;
  githubToken = "ghu_x"; sent.length = 0;
  afterOpen();
  out.open = sent.filter((x) => x.msg.type === "verify").length;
  // Pointed back at the relay it was signed in through, it goes there.
  session.relay = "` + relay + `"; githubToken = "ghu_x"; sent.length = 0;
  connectSignedIn();
  out.own = sent.map((x) => x.relay + ":" + x.msg.type);
  console.log(JSON.stringify(out));
})();
`
	var out struct {
		Present struct {
			Sent        []any
			Token, Held string
		}
		Connect, Open int
		Own           []string
	}
	runPage(t, script, &out)
	if len(out.Present.Sent) != 0 || out.Present.Token != "" || out.Present.Held != "" {
		t.Errorf("a held token with the page on the link's relay: %+v, want it dropped and nothing sent", out.Present)
	}
	if out.Connect != 0 || out.Open != 0 {
		t.Errorf("a token in hand reached the link's relay: connect %d, open %d", out.Connect, out.Open)
	}
	if len(out.Own) != 1 || out.Own[0] != relay+":verify" {
		t.Errorf("the relay it was signed in through: %v, want one verify", out.Own)
	}
	for _, fn := range []string{"afterOpen", "connectSignedIn", "onJoined"} {
		if !strings.Contains(functionBody(t, pageScript(t), fn), "relayTokenCleared()") {
			t.Errorf("%s presents the token without checking the relay it was signed in through", fn)
		}
	}
}

func TestASignInStartsThroughAClearedRelayAndNamesItsCallback(t *testing.T) {
	const relay = "https://brief-relay.onrender.com"
	script := pageHarness(t) + `
const out = {};
(async () => {
  anchor = { state: "listed", relays: ["` + relay + `"] };
  session.relay = "` + relay + `";
  await startSignIn();
  const u = new URL(assigned);
  const state = u.searchParams.get("state");
  const page = Buffer.from(state.split(".")[1], "base64url").toString();
  const kept = JSON.parse(sessionStorage.getItem(KEY_SIGN_IN));
  out.listed = { redirect: u.searchParams.get("redirect_uri"), page, kept: kept.relay };

  // A relay the person confirmed, where the repository's list cannot be
  // read, runs the sign-in the same way: GitHub sends the code only to the
  // callback the team's app lists.
  assigned = ""; requests.length = 0; toasts.length = 0; sessionStorage.m.clear();
  anchor = { state: "unknown", relays: [] };
  session.relay = "https://other-relay.example.org";
  confirmRelay(session.relay);
  await startSignIn();
  out.confirmed = { redirect: assigned ? new URL(assigned).searchParams.get("redirect_uri") : "", toasts: toasts.slice() };
  // Unconfirmed, the page asks first and starts nothing.
  assigned = ""; sessionStorage.m.clear(); toasts.length = 0;
  await startSignIn();
  out.unconfirmed = { redirect: assigned, toasts: toasts.slice() };
  console.log(JSON.stringify(out));
})();
`
	var out struct {
		Listed struct {
			Redirect, Page, Kept string
		}
		Confirmed, Unconfirmed struct {
			Redirect string
			Toasts   []string
		}
	}
	runPage(t, script, &out)
	if out.Listed.Redirect != relay+"/auth/callback" || out.Listed.Page != "https://abcdesign.app/acme/brief/brief/" || out.Listed.Kept != relay {
		t.Errorf("a listed relay: %+v, want redirect_uri %s/auth/callback and the page in the state", out.Listed, relay)
	}
	if out.Confirmed.Redirect != "https://other-relay.example.org/auth/callback" {
		t.Errorf("a confirmed relay: %+v, want the sign-in to its own callback", out.Confirmed)
	}
	if out.Unconfirmed.Redirect != "" || len(out.Unconfirmed.Toasts) != 1 || out.Unconfirmed.Toasts[0] != "askTrust:signin" {
		t.Errorf("an unconfirmed relay: %+v, want the confirmation asked and no sign-in", out.Unconfirmed)
	}
}

// A confirmation lasts this tab, and lapses after a bounded time: it is kept
// in session storage, never local storage.
func TestAConfirmationLastsTheTabAndLapses(t *testing.T) {
	script := pageHarness(t) + `
const out = {};
confirmRelay("https://other-relay.example.org");
out.now = confirmedRelays();
out.local = localStorage.m.size;
const all = JSON.parse(sessionStorage.getItem(KEY_TRUSTED));
all["acme/brief"][0].at -= CONFIRM_LIFE + 1;
sessionStorage.setItem(KEY_TRUSTED, JSON.stringify(all));
out.lapsed = confirmedRelays();
console.log(JSON.stringify(out));
`
	var out struct {
		Now, Lapsed []string
		Local       int
	}
	runPage(t, script, &out)
	if len(out.Now) != 1 || out.Local != 0 {
		t.Errorf("a fresh confirmation: %+v, want it held in session storage alone", out)
	}
	if len(out.Lapsed) != 0 {
		t.Errorf("a confirmation older than CONFIRM_LIFE still counts: %v", out.Lapsed)
	}
	js := pageScript(t)
	if !strings.Contains(functionBody(t, js, "main"), "localStorage.removeItem(LEGACY_TRUSTED)") {
		t.Error("main does not drop a confirmation an earlier page kept in local storage")
	}
}

// A relay the repository's relays.json names must answer as the
// repository's own: its /config names the client id .abcd/relay/github-app.json
// records beside the list, and offers GitHub's web flow, or the page signs
// nobody in through it, sends it no token and opens no session on it
// (spc-2610230609005313 step 4). The app file holds a listed relay to the
// team's own app and to GitHub's web flow.

// appRelay is the relay the app-check tests hold to the app file.
const appRelay = "https://brief-relay.onrender.com"

// appProbe is a script that drives the page through one relay as main
// would, with the anchor and the relay's /config each case names: the app
// check, a sign-in started, a guest's join, a return from GitHub and the
// token it brings. It prints what reached the relay for each case.
func appProbe(t *testing.T, cases string) string {
	return pageHarness(t) + `
const RELAY = "` + appRelay + `";
async function probe({ anchorIs, cfg, confirmed }) {
  sessionStorage.m.clear(); requests.length = 0; toasts.length = 0; sent.length = 0;
  heldToken = ""; githubToken = ""; tokenRelay = ""; assigned = ""; listSeen = null;
  relayApp = { relay: "", clientID: "", state: "" }; appCheck = null;
  anchor = anchorIs; config = cfg;
  session.relay = RELAY;
  if (confirmed) confirmRelay(RELAY);
  // main checks the relay it chose before anything reaches it.
  await checkRelayApp();
  const r = { standing: relayStanding(RELAY) };
  await startSignIn();
  r.signIn = assigned !== "";
  r.join = relayClearedFor("join");
  requests.length = 0; sent.length = 0;
  sessionStorage.setItem(KEY_SIGN_IN, JSON.stringify({ state: "s1.cGFnZQ", verifier: "v".repeat(43), relay: RELAY, code: "", name: "" }));
  location.search = "?" + new URLSearchParams({ code: "good-code", state: "s1.cGFnZQ", relay: RELAY }).toString();
  await finishSignIn();
  r.code = requests.filter((q) => q.url.endsWith("/auth/token")).length;
  r.token = sent.filter((x) => x.msg.type === "open" || x.msg.type === "verify").length;
  r.toasts = toasts.slice();
  return r;
}
const ours = { client_id: "Iv1.abc", sign_in: true, app_type: "github-app", authorize_url: "https://github.com/login/oauth/authorize", flow: "web" };
const listedWith = (app) => ({ state: "listed", relays: [RELAY], app });
const theApp = { required: true, clientID: "Iv1.abc" };
const out = {};
(async () => {
` + cases + `
  console.log(JSON.stringify(out));
})();
`
}

// appProbeResult is what appProbe prints for one case.
type appProbeResult struct {
	Standing    string
	SignIn      bool
	Join        bool
	Code, Token int
	Toasts      []string
}

// refusedAsNotOurs reports how r falls short of a relay held off entirely,
// with the page saying it is not answering as the project's own.
func refusedAsNotOurs(r appProbeResult) string {
	if r.Standing != "foreign" || r.SignIn || r.Join || r.Code != 0 || r.Token != 0 {
		return "it was used"
	}
	for _, s := range r.Toasts {
		if s == "notOurs" {
			return ""
		}
	}
	return "the page did not say the session is not answering as the project's own"
}

// usedAsToday reports how r falls short of a relay the page uses fully.
func usedAsToday(r appProbeResult, standing string) string {
	if r.Standing != standing || !r.SignIn || !r.Join || r.Code != 1 || r.Token != 1 {
		return "it was not used as today"
	}
	return ""
}

func TestAListedRelayMustNameTheRepositorysApp(t *testing.T) {
	var out map[string]appProbeResult
	runPage(t, appProbe(t, `
  out.foreign = await probe({ anchorIs: listedWith(theApp), cfg: Object.assign({}, ours, { client_id: "Iv1.someoneelse" }) });
  out.noID = await probe({ anchorIs: listedWith(theApp), cfg: Object.assign({}, ours, { client_id: undefined }) });
  out.ours = await probe({ anchorIs: listedWith(theApp), cfg: ours });
`), &out)
	for _, name := range []string{"foreign", "noID"} {
		if why := refusedAsNotOurs(out[name]); why != "" {
			t.Errorf("%s: %s: %+v", name, why, out[name])
		}
	}
	if why := usedAsToday(out["ours"], "listed"); why != "" {
		t.Errorf("a relay naming the repository's app: %s: %+v", why, out["ours"])
	}
}

func TestAListedRelayMustOfferTheWebFlow(t *testing.T) {
	var out map[string]appProbeResult
	runPage(t, appProbe(t, `
  out.device = await probe({ anchorIs: listedWith(theApp), cfg: Object.assign({}, ours, { flow: "device" }) });
  out.none = await probe({ anchorIs: listedWith(theApp), cfg: Object.assign({}, ours, { flow: undefined, sign_in: false }) });
  out.unnamed = await probe({ anchorIs: listedWith(theApp), cfg: Object.assign({}, ours, { flow: undefined }) });
  out.oauth = await probe({ anchorIs: listedWith(theApp), cfg: Object.assign({}, ours, { app_type: "oauth-app" }) });
`), &out)
	for name, r := range out {
		if why := refusedAsNotOurs(r); why != "" {
			t.Errorf("%s: %s: %+v", name, why, r)
		}
	}
	if len(out) != 4 {
		t.Fatalf("got %d cases, want 4", len(out))
	}
}

func TestARepositoryWithoutAnAppFileBehavesAsBefore(t *testing.T) {
	var out map[string]appProbeResult
	runPage(t, appProbe(t, `
  const other = Object.assign({}, ours, { client_id: "Iv1.someoneelse" });
  out.noFile = await probe({ anchorIs: listedWith({ required: false, clientID: "" }), cfg: other });
  out.noAnswer = await probe({ anchorIs: { state: "listed", relays: [RELAY] }, cfg: other });
`), &out)
	for name, r := range out {
		if why := usedAsToday(r, "listed"); why != "" {
			t.Errorf("%s: %s: %+v", name, why, r)
		}
	}
	// readRelayList reads the app file only beside a list it could read, and
	// GitHub's 404 for it is a repository that records no app.
	got := readListWithApp(t, map[string]string{"absent": "404"})
	if c := got["absent"]; c.State != "listed" || c.App.Required || c.Standing[0] != "listed" {
		t.Errorf("no app file: %+v, want the list as today", c)
	}
}

func TestAConfirmedRelayIsNotHeldToTheAppFile(t *testing.T) {
	var out map[string]appProbeResult
	runPage(t, appProbe(t, `
  out.confirmed = await probe({ anchorIs: { state: "unknown", relays: [] }, confirmed: true, cfg: Object.assign({}, ours, { client_id: "Iv1.someoneelse" }) });
`), &out)
	if why := usedAsToday(out["confirmed"], "confirmed"); why != "" {
		t.Errorf("a relay the person confirmed: %s: %+v", why, out["confirmed"])
	}
}

func TestAnAppFileWithAClientSecretRefusesEveryListedRelay(t *testing.T) {
	got := readListWithApp(t, map[string]string{
		"secret":    `{"client_id": "Iv1.abc", "client_secret": "shh"}`,
		"nullsec":   `{"client_id": "Iv1.abc", "client_secret": null}`,
		"malformed": `{"client_id": "Iv1.abc"`,
		"badID":     `{"client_id": "Iv1 abc"}`,
		"array":     `[{"client_id": "Iv1.abc"}]`,
		"good":      `{"client_id": "Iv1.abc"}`,
		"forbidden": "403",
		"offline":   "throw",
	})
	for _, name := range []string{"secret", "nullsec", "malformed", "badID", "array"} {
		c := got[name]
		if c.State != "listed" || !c.App.Required || c.App.ClientID != "" {
			t.Errorf("%s: %+v, want an app file that names no usable client id", name, c)
		}
		for i, s := range c.Standing {
			if s != "foreign" {
				t.Errorf("%s: listed relay %d stands %q, want every listed relay refused", name, i, s)
			}
		}
	}
	if c := got["good"]; !c.App.Required || c.App.ClientID != "Iv1.abc" || c.Standing[0] != "listed" || c.Standing[1] != "listed" {
		t.Errorf("a well-formed app file: %+v, want its client id and both relays listed once each answers as it", c)
	}
	// An app file GitHub would not show is a list that could not be checked:
	// nothing reaches any relay until it can be.
	for _, name := range []string{"forbidden", "offline"} {
		if c := got[name]; c.State != "unavailable" {
			t.Errorf("%s: %+v, want the list unavailable", name, c)
		}
	}
	for name, c := range got {
		if !c.AppRead {
			t.Errorf("%s: the app file was not read from the repository's contents on api.github.com", name)
		}
	}
}

// listWithApp is what readListWithApp reports for one app-file answer.
type listWithApp struct {
	State string
	App   struct {
		Required bool
		ClientID string
	}
	// Standing is each listed relay's standing, once it answers with the
	// client id Iv1.abc and the web flow.
	Standing []string
	AppRead  bool
}

// readListWithApp runs readRelayList against a list naming two relays and,
// per case, an app file answered as given: an HTTP status, "throw" for a
// network error, or the file's contents.
func readListWithApp(t *testing.T, cases map[string]string) map[string]listWithApp {
	t.Helper()
	js := pageScript(t)
	enc := func(s string) string {
		b, _ := json.Marshal(map[string]string{"type": "file", "encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(s))})
		return string(b)
	}
	answers, _ := json.Marshal(cases)
	var b strings.Builder
	for _, c := range []string{"GITHUB_API", "RELAYS_FILE", "APP_FILE"} {
		b.WriteString(constLine(t, js, c) + "\n")
	}
	for _, fn := range []string{"readRelayList", "readAppFile", "appIn", "relaysIn", "decodeBlob", "relayOrigin", "repoPath", "appRequired", "appStanding", "relayStanding"} {
		b.WriteString(functionBody(t, js, fn) + "\n")
	}
	b.WriteString(`
const route = { owner: "acme", repo: "brief", slug: "acme/brief" };
const location = { origin: "https://abcdesign.app" };
function confirmedRelays() { return []; }
const RELAYS = ["https://brief-relay.onrender.com", "https://second-relay.onrender.com"];
const listBody = ` + enc(`{"relays": ["https://brief-relay.onrender.com", "https://second-relay.onrender.com"]}`) + `;
const answers = ` + string(answers) + `;
const files = {};
for (const [k, v] of Object.entries(answers)) files[k] = /^[0-9]+$/.test(v) || v === "throw" ? v : { type: "file", encoding: "base64", content: Buffer.from(v).toString("base64") };
let answer = null;
let appRead = false;
globalThis.fetch = async (url) => {
  if (url === "https://api.github.com/repos/acme/brief/contents/.abcd/relay/relays.json") return { status: 200, json: async () => listBody };
  if (url === "https://api.github.com/repos/acme/brief/contents/.abcd/relay/github-app.json") {
    appRead = true;
    if (answer === "throw") throw new Error("offline");
    if (typeof answer === "string") return { status: Number(answer), json: async () => ({}) };
    return { status: 200, json: async () => answer };
  }
  return { status: 500, json: async () => ({}) };
};
let anchor = null;
let relayApp = null;
const out = {};
(async () => {
  for (const [name, file] of Object.entries(files)) {
    answer = file; appRead = false;
    anchor = await readRelayList("");
    const standing = [];
    for (const relay of RELAYS) {
      relayApp = { relay, clientID: "Iv1.abc", state: "ours" };
      standing.push(relayStanding(relay));
    }
    out[name] = { state: anchor.state, app: anchor.app || { required: false, clientID: "" }, standing, appRead };
  }
  console.log(JSON.stringify(out));
})();
`)
	var out map[string]listWithApp
	runPage(t, b.String(), &out)
	return out
}

// TestAHeldTokenWaitsForTheAppCheck: a token GitHub returned through a
// relay the person confirmed, in a private repository whose list the token
// then shows, becomes the token the page presents only once that relay
// answers as the repository's app: the check runs before the token is given.
func TestAHeldTokenWaitsForTheAppCheck(t *testing.T) {
	script := pageHarness(t) + `
const RELAY = "` + appRelay + `";
const ours = { client_id: "Iv1.abc", sign_in: true, app_type: "github-app", authorize_url: "https://github.com/login/oauth/authorize", flow: "web" };
async function held(cfg) {
  sessionStorage.m.clear(); requests.length = 0; toasts.length = 0; sent.length = 0;
  relayApp = { relay: "", clientID: "", state: "" }; appCheck = null; githubToken = "";
  anchor = { state: "unknown", relays: [] };
  confirmRelay(RELAY);
  listSeen = { state: "listed", relays: [RELAY], app: { required: true, clientID: "Iv1.abc" } };
  config = cfg;
  session.relay = RELAY; tokenRelay = RELAY; heldToken = "ghu_x";
  await presentHeldToken(false);
  return { asked: requests.filter((q) => q.url === RELAY + "/config").length, token: githubToken, held: heldToken, sent: sent.length, toasts: toasts.slice() };
}
const out = {};
(async () => {
  out.foreign = await held(Object.assign({}, ours, { client_id: "Iv1.someoneelse" }));
  out.ours = await held(ours);
  console.log(JSON.stringify(out));
})();
`
	var out map[string]struct {
		Asked, Sent int
		Token, Held string
		Toasts      []string
	}
	runPage(t, script, &out)
	if f := out["foreign"]; f.Asked != 1 || f.Token != "" || f.Held != "" || f.Sent != 0 || len(f.Toasts) != 1 || f.Toasts[0] != "notOurs" {
		t.Errorf("a held token for a relay naming another app: %+v, want it dropped, nothing sent, and the page saying so", f)
	}
	if o := out["ours"]; o.Asked != 1 || o.Sent != 1 {
		t.Errorf("a held token for the repository's own relay: %+v, want it presented once the relay answered", o)
	}
	present := functionBody(t, pageScript(t), "presentHeldToken")
	check, give := strings.Index(present, "await checkRelayApp();"), strings.Index(present, "githubToken = heldToken;")
	if check < 0 || give < 0 || check > give {
		t.Errorf("presentHeldToken gives the token before the app check:\n%s", present)
	}
}

// A listed relay whose /config does not answer, by a status other than 200
// or a request that never reaches it, gets nothing, and the page says it
// could not reach the session and offers to ask again: the start screen's
// Try again asks the relay again, and once it answers as the repository's
// app the page uses it.
func TestAListedRelayThatDoesNotAnswerCanBeAskedAgain(t *testing.T) {
	script := appProbe(t, `
  for (const [name, failing] of [["status", 503], ["thrown", "throw"]]) {
    sessionStorage.m.clear(); requests.length = 0; toasts.length = 0; sent.length = 0;
    heldToken = ""; githubToken = ""; tokenRelay = ""; assigned = ""; listSeen = null;
    relayApp = { relay: "", clientID: "", state: "" }; appCheck = null;
    anchor = listedWith(theApp); config = ours; configAnswer = failing;
    session.relay = RELAY;
    await checkRelayApp();
    const r = { standing: relayStanding(RELAY) };
    await startSignIn();
    r.signIn = assigned !== "";
    r.join = relayClearedFor("join");
    // Let a re-ask the refusal started settle while the relay still fails.
    await new Promise((done) => setTimeout(done, 20));
    r.sent = sent.length + requests.filter((q) => !q.url.endsWith("/config")).length;
    r.toasts = toasts.slice();
    // The relay answers now; Try again asks it again.
    configAnswer = null;
    await recheckRelayList();
    r.after = relayStanding(RELAY);
    out[name] = r;
  }
`)
	var out map[string]struct {
		Standing, After string
		SignIn, Join    bool
		Sent            int
		Toasts          []string
	}
	runPage(t, script, &out)
	for _, name := range []string{"status", "thrown"} {
		r := out[name]
		if r.Standing != "unanswered" || r.SignIn || r.Join || r.Sent != 0 {
			t.Errorf("%s: %+v, want the relay unanswered and nothing sent to it", name, r)
		}
		said := false
		for _, s := range r.Toasts {
			said = said || s == "sessionUnreachable"
		}
		if !said {
			t.Errorf("%s: the page said %v, want that the session could not be reached", name, r.Toasts)
		}
		if r.After != "listed" {
			t.Errorf("%s: after the relay answered, Try again left it %q, want listed", name, r.After)
		}
	}
}

// startSignIn signs in with the repository's own app alone: a relay that
// answered as that app at the check and names another client id by the
// time the person signs in gets no sign-in.
func TestASignInUsesOnlyTheRepositorysClientID(t *testing.T) {
	var out map[string]appProbeResult
	runPage(t, appProbe(t, `
  sessionStorage.m.clear(); requests.length = 0; toasts.length = 0; sent.length = 0; assigned = "";
  relayApp = { relay: "", clientID: "", state: "" }; appCheck = null;
  anchor = listedWith(theApp); config = ours; session.relay = RELAY;
  await checkRelayApp();
  config = Object.assign({}, ours, { client_id: "Iv1.someoneelse" });
  await startSignIn();
  out.changed = { standing: relayStanding(RELAY), signIn: assigned !== "", toasts: toasts.slice() };
  config = ours; toasts.length = 0; assigned = "";
  await startSignIn();
  out.same = { standing: relayStanding(RELAY), signIn: assigned !== "", toasts: toasts.slice() };
`), &out)
	if c := out["changed"]; c.SignIn || len(c.Toasts) != 1 || c.Toasts[0] != "notOurs" {
		t.Errorf("a relay naming another client id at sign-in: %+v, want no sign-in and the page saying so", c)
	}
	if c := out["same"]; !c.SignIn {
		t.Errorf("the repository's own client id: %+v, want the sign-in started", c)
	}
}
