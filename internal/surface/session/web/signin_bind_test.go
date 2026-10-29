package web

import (
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
function recheckRelayList() {}
function startDeviceSignIn() { toasts.push("device"); }
function takeRejoin() { return false; }
async function readRelayList() { return { state: anchor.state, relays: anchor.relays }; }
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
globalThis.fetch = async (url, init) => {
  requests.push({ url, body: init && init.body ? JSON.parse(init.body) : null });
  if (url.endsWith("/config")) return { ok: true, status: 200, json: async () => config };
  return { ok: true, status: 200, json: async () => ({ access_token: "ghu_x" }) };
};
`)
	// Each piece is taken when session.js has it, so the harness runs the
	// page as it stands; a test asserts on what the page does.
	for _, c := range []string{"GITHUB_AUTHORIZE", "KEY_SIGN_IN", "KEY_TRUSTED", "CONFIRM_LIFE", "CALLBACK_PATH", "CODE_ALPHABET"} {
		if regexp.MustCompile(`(?m)^const ` + c + ` = `).MatchString(js) {
			b.WriteString(constLine(t, js, c) + "\n")
		}
	}
	for _, fn := range []string{
		"recall", "remember", "stashSignIn", "takeSignIn", "trustedRelays", "keepTrusted", "liveConfirmations", "confirmedRelays", "confirmRelay", "forgetRelay",
		"pageURL", "relayOrigin", "normalizeCode", "b64url", "randomText", "isLoopbackOrigin", "relayConfig",
		"relayStanding", "relayCleared", "relayTokenCleared", "standingRefusal", "relayClearedFor",
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
