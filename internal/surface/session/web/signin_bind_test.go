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
// relay the person confirmed, which the repository's list does not name,
// serves guest joins and never receives a code.

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
async function presentHeldToken() { toasts.push("presented"); }
function takeRejoin() { return false; }
let heldToken = "";
let signInOffered = null;
const requests = [];
let assigned = "";
const location = { origin: "https://abcdesign.app", pathname: "/acme/brief/brief/", search: "", hash: "", assign: (u) => { assigned = u; } };
globalThis.history = { replaceState: () => {} };
const route = { slug: "acme/brief" };
const session = { relay: "", code: "", name: "", everJoined: false };
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
		"relayStanding", "relayCleared", "relaySignInCleared", "standingRefusal", "relayClearedFor",
		"startSignIn", "finishSignIn",
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
async function attempt(name, { stored, returned, listed, confirmed, anchorState }) {
  sessionStorage.m.clear();
  requests.length = 0; toasts.length = 0; heldToken = "";
  anchor = { state: anchorState || "listed", relays: listed || [] };
  session.relay = "";
  if (confirmed) confirmRelay(confirmed);
  sessionStorage.setItem(KEY_SIGN_IN, JSON.stringify(pending(stored)));
  const q = new URLSearchParams({ code: "good-code", state: "s1.cGFnZQ" });
  if (returned !== undefined) q.set("relay", returned);
  location.search = "?" + q.toString();
  await finishSignIn();
  const posts = requests.filter((r) => r.url.endsWith("/auth/token"));
  out[name] = { posts: posts.map((p) => ({ url: p.url, redirect: p.body.redirect_uri })), held: heldToken !== "" };
}
const out = {};
(async () => {
  await attempt("returned", { stored: "` + relay + `", returned: "` + relay + `", listed: ["` + relay + `"] });
  await attempt("no-relay", { stored: "` + relay + `", listed: ["` + relay + `"] });
  await attempt("another-relay", { stored: "` + relay + `", returned: "` + other + `", listed: ["` + relay + `", "` + other + `"] });
  await attempt("stored-differs", { stored: "` + other + `", returned: "` + relay + `", listed: ["` + relay + `", "` + other + `"] });
  await attempt("unlisted", { stored: "` + other + `", returned: "` + other + `", listed: ["` + relay + `"] });
  await attempt("confirmed", { stored: "` + other + `", returned: "` + other + `", anchorState: "unknown", confirmed: "` + other + `" });
  console.log(JSON.stringify(out));
})();
`
	type result struct {
		Posts []struct{ URL, Redirect string }
		Held  bool
	}
	var out map[string]result
	runPage(t, script, &out)
	got := out["returned"]
	if len(got.Posts) != 1 || got.Posts[0].URL != relay+"/auth/token" || got.Posts[0].Redirect != relay+"/auth/callback" || !got.Held {
		t.Errorf("the relay GitHub returned to: %+v, want one post to %s/auth/token naming its callback", got, relay)
	}
	for name, why := range map[string]string{
		"no-relay":       "a code arriving with no relay named",
		"another-relay":  "a code returned through a relay other than the one the sign-in started with",
		"stored-differs": "a code whose stored relay differs from the one GitHub returned to",
		"unlisted":       "a relay the repository's list does not name",
		"confirmed":      "a relay the person confirmed but the repository does not list",
	} {
		if c := out[name]; len(c.Posts) != 0 || c.Held {
			t.Errorf("%s: %+v, want no code sent anywhere", why, c)
		}
	}
}

func TestASignInStartsOnlyThroughAListedRelayAndNamesItsCallback(t *testing.T) {
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

  // A relay the person confirmed, which the repository does not list: no
  // sign-in starts, nothing is asked of it, and the page says why.
  assigned = ""; requests.length = 0; toasts.length = 0; sessionStorage.m.clear();
  anchor = { state: "unknown", relays: [] };
  session.relay = "https://other-relay.example.org";
  confirmRelay(session.relay);
  await startSignIn();
  out.confirmed = { assigned, asked: requests.length, toasts: toasts.slice() };
  // Unconfirmed, the same: no confirmation is offered for a sign-in.
  sessionStorage.m.clear(); toasts.length = 0;
  await startSignIn();
  out.unconfirmed = { assigned, asked: requests.length, toasts: toasts.slice() };
  console.log(JSON.stringify(out));
})();
`
	var out struct {
		Listed struct {
			Redirect, Page, Kept string
		}
		Confirmed, Unconfirmed struct {
			Assigned string
			Asked    int
			Toasts   []string
		}
	}
	runPage(t, script, &out)
	if out.Listed.Redirect != relay+"/auth/callback" || out.Listed.Page != "https://abcdesign.app/acme/brief/brief/" || out.Listed.Kept != relay {
		t.Errorf("a listed relay: %+v, want redirect_uri %s/auth/callback and the page in the state", out.Listed, relay)
	}
	for name, c := range map[string]struct {
		Assigned string
		Asked    int
		Toasts   []string
	}{"confirmed": out.Confirmed, "unconfirmed": out.Unconfirmed} {
		if c.Assigned != "" || c.Asked != 0 || len(c.Toasts) != 1 || c.Toasts[0] != "signInUnlisted" {
			t.Errorf("%s relay: %+v, want no sign-in, nothing asked of it, and the plain reason", name, c)
		}
	}
	js := pageScript(t)
	if !strings.Contains(js, `signInUnlisted: "`) {
		t.Error("the page has no words for a sign-in through a relay its project does not list")
	}
}

// A confirmation covers guest joins, for this tab only, and lapses after a
// bounded time: it is kept in session storage, never local storage.
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
