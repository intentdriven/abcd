package web

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// wordsBlock is the WORDS table's source.
func wordsBlock(t *testing.T, js string) string {
	t.Helper()
	i := strings.Index(js, "const WORDS = Object.freeze({")
	j := strings.Index(js[i:], "\n});\n")
	if i < 0 || j < 0 {
		t.Fatal("session.js has no WORDS table")
	}
	return js[i : i+j]
}

// TestThePageAsksForNoRelayAddressOrToken: the page has no field for a relay
// address or a token, never says either word to the person, and does not
// print the invite link, which may carry a relay (adr-2610161400542916,
// iss-2610161332024554).
func TestThePageAsksForNoRelayAddressOrToken(t *testing.T) {
	js := pageScript(t)
	for _, gone := range []string{`type: "password"`, `id: "relay"`, `id: "token"`, "function useToken", "pasteToken", "showTokenForm", "KEY_RELAY", "relayLabel"} {
		if strings.Contains(js, gone) {
			t.Errorf("session.js still carries %s", gone)
		}
	}
	values := regexp.MustCompile(`(?m)^  [A-Za-z0-9]+: "([^"]*)",$`).FindAllStringSubmatch(wordsBlock(t, js), -1)
	if len(values) < 40 {
		t.Fatalf("read %d words; the check is not looking at the table", len(values))
	}
	for _, v := range values {
		low := strings.ToLower(v[1])
		for _, word := range []string{"relay", "token", "personal access", "https://", "websocket"} {
			if strings.Contains(low, word) {
				t.Errorf("the page says %q to the person: %q", word, v[1])
			}
		}
	}
	if strings.Contains(functionBody(t, js, "renderSide"), `class: "link"`) {
		t.Error("the invite card prints the link")
	}
	if strings.Contains(functionBody(t, js, "standingRefusal"), "relay") {
		t.Error("a refusal names the relay")
	}
}

// TestAPageTheRelayServesIsItsOwnRelay: a local session's relay serves the
// page with a marker, and the page then takes its own origin as its relay,
// with no relay list to read; the static page never carries the marker.
func TestAPageTheRelayServesIsItsOwnRelay(t *testing.T) {
	served := RelayFiles()
	if bytes.Count(served["index.html"], []byte(RelayMarker)) != 1 {
		t.Fatal("the relay's page does not carry the marker once")
	}
	if _, ok := served["404.html"]; ok {
		t.Error("the relay serves a 404.html")
	}
	for _, f := range Files("/") {
		if f.Path != "session.js" && bytes.Contains(f.Data, []byte("abcd-session-relay")) {
			t.Errorf("the static page's %s carries the relay marker", f.Path)
		}
		if f.Path != "index.html" && f.Path != "404.html" && !bytes.Equal(served[f.Path], f.Data) {
			t.Errorf("the relay serves %s other than as written", f.Path)
		}
	}
	js := pageScript(t)
	if !strings.Contains(functionBody(t, js, "selfServed"), `document.querySelector('meta[name="abcd-session-relay"]')`) {
		t.Error("the page does not read the relay marker")
	}
	choose := functionBody(t, js, "chooseRelay")
	if !strings.Contains(choose, `if (anchor.state === "self") return location.origin;`) {
		t.Errorf("a page the relay serves does not take its own origin:\n%s", choose)
	}
	standing := functionBody(t, js, "relayStanding")
	if !strings.Contains(standing, `if (anchor.state === "self") return relay === location.origin ? "self" : "refused";`) {
		t.Errorf("a page the relay serves would talk to another relay:\n%s", standing)
	}
}

// TestThePageFindsItsRelay: with no relay in the link, the first relay the
// repository's list names; where the list cannot be read, one the person
// confirmed for this repository; and with none, plain words saying live
// sessions are not set up, and nothing to type.
func TestThePageFindsItsRelay(t *testing.T) {
	js := pageScript(t)
	choose := functionBody(t, js, "chooseRelay")
	for _, want := range []string{
		"if (fromLink) return fromLink;",
		`if (anchor.state === "listed") return anchor.relays[0] || "";`,
		`if (anchor.state === "unknown") return confirmedRelays()[0] || "";`,
	} {
		if !strings.Contains(choose, want) {
			t.Errorf("chooseRelay does not carry %s:\n%s", want, choose)
		}
	}
	start := functionBody(t, js, "renderStart")
	for _, want := range []string{"WORDS.notSetUp", "WORDS.privateNoLink", "WORDS.cannotCheck", "ui.startCard.hidden = !usable", "ui.joinCard.hidden = !usable"} {
		if !strings.Contains(start, want) {
			t.Errorf("renderStart does not carry %s", want)
		}
	}
	if !strings.Contains(js, `notSetUp: "Live sessions are not set up for this project yet. Ask your technical facilitator to set them up.",`) {
		t.Error("the page does not say plainly that live sessions are not set up")
	}
}

// TestTheInviteCarriesTheRelayOnlyWhenNeeded: the invite link names the
// relay only when a guest's page could not find the same one itself.
func TestTheInviteCarriesTheRelayOnlyWhenNeeded(t *testing.T) {
	js := pageScript(t)
	frag := functionBody(t, js, "inviteFragment")
	if !strings.Contains(frag, `const own = anchor.state === "self" || (anchor.state === "listed" && anchor.relays[0] === session.relay);`) ||
		!strings.Contains(frag, `(own ? "" : "relay=" + encodeURIComponent(session.relay) + "&")`) {
		t.Errorf("inviteFragment names the relay where the page finds it alone:\n%s", frag)
	}
	for _, fn := range []string{"inviteURL", "updateFragment"} {
		if !strings.Contains(functionBody(t, js, fn), "inviteFragment()") {
			t.Errorf("%s builds its fragment by hand", fn)
		}
	}
}

// TestTheSessionOpensOnThePressRelease: once joined or started, the page
// opens the brief's press release, never asking the person to pick a file.
func TestTheSessionOpensOnThePressRelease(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(functionBody(t, js, "firstFile"), "/press-release/i") {
		t.Error("firstFile does not look for the press release")
	}
	if strings.Count(js, "openFile(firstFile())") != 1 || !strings.Contains(js, "const first = firstFile();") {
		t.Error("the session does not open on firstFile")
	}
	if strings.Contains(js, "[...files.keys()].sort()[0]") {
		t.Error("a session still opens on the first file by name")
	}
}

// TestDeviceSignInGoesThroughTheRelay: the device flow's requests go to the
// relay, which keeps the device code; the page shows only a well-formed user
// code, polls no faster than five seconds, and opens GitHub's own device page
// in a new tab without an opener.
func TestDeviceSignInGoesThroughTheRelay(t *testing.T) {
	js := pageScript(t)
	start := functionBody(t, js, "startDeviceSignIn")
	for _, want := range []string{`relayPost("/auth/device", {})`, `/^[A-Z0-9]{4}-[A-Z0-9]{4}$/.test(out.user_code)`, "interval: Math.max(5, Number(out.interval) || 5)", "ui.deviceCode.textContent = out.user_code;"} {
		if !strings.Contains(start, want) {
			t.Errorf("startDeviceSignIn does not carry %s:\n%s", want, start)
		}
	}
	if strings.Contains(js, "device_code") {
		t.Error("the page handles a device code, which the relay keeps")
	}
	poll := functionBody(t, js, "pollDevice")
	if !strings.Contains(poll, `relayPost("/auth/device/token", { handle: d.handle })`) || !strings.Contains(poll, "out.interval >= 5") {
		t.Errorf("pollDevice does not poll the relay at GitHub's pace:\n%s", poll)
	}
	if !strings.Contains(js, `ui.deviceOpen = el("a", { class: "button primary", href: GITHUB_DEVICE, target: "_blank", rel: "noopener noreferrer"`) {
		t.Error("the page does not open GitHub's device page in a new tab without an opener")
	}
	if !strings.Contains(functionBody(t, js, "relayPost"), `credentials: "omit"`) {
		t.Error("relayPost sends credentials")
	}
}
