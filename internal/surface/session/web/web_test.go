package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// The files the page is made of, and nothing else.
var wantFiles = []string{"404.html", "index.html", "session.css", "session.js", "vendor/LICENSES.txt", "vendor/collab.js"}

// listTree lists every entry under dir, slash-separated, directories marked
// with a trailing slash.
func listTree(t *testing.T, dir string) []string {
	t.Helper()
	var got []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			rel += "/"
		}
		got = append(got, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	return got
}

func TestPageWritesExactlyItsFilesAsEmbedded(t *testing.T) {
	out := filepath.Join(t.TempDir(), "page")
	res, err := Write(out, "")
	if err != nil {
		t.Fatal(err)
	}
	want := append([]string{"vendor/"}, wantFiles...)
	sort.Strings(want)
	if got := listTree(t, out); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("wrote %v, want %v", got, want)
	}
	if res.Replaced || len(res.Files) != len(wantFiles) {
		t.Fatalf("result = %+v", res)
	}
	var total int64
	for _, f := range Files("/") {
		got, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(f.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, f.Data) {
			t.Errorf("%s differs from the embedded asset", f.Path)
		}
		// Every embedded asset is the committed file beside this test.
		src := f.Path
		if src == "404.html" {
			src = "index.html"
		}
		disk, err := os.ReadFile(filepath.FromSlash(src))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, disk) {
			t.Errorf("%s differs from the committed %s", f.Path, src)
		}
		total += int64(len(got))
	}
	if res.Bytes != total {
		t.Errorf("reported %d bytes, wrote %d", res.Bytes, total)
	}

	// A second run replaces its own copy, and still writes nothing else.
	res, err = Write(out, "")
	if err != nil || !res.Replaced {
		t.Fatalf("rewrite: %+v, %v", res, err)
	}
	if got := listTree(t, out); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("rewrite left %v", got)
	}
}

func Test404IsTheIndex(t *testing.T) {
	byPath := map[string][]byte{}
	for _, f := range Files("/") {
		byPath[f.Path] = f.Data
	}
	if !bytes.Equal(byPath["404.html"], byPath["index.html"]) || len(byPath["index.html"]) == 0 {
		t.Fatal("404.html is not index.html byte for byte")
	}
}

// refused asserts that Write refused out and left it exactly as it was.
func refused(t *testing.T, out string) {
	t.Helper()
	before := listTree(t, out)
	_, err := Write(out, "")
	var ref *Refusal
	if !errors.As(err, &ref) {
		t.Fatalf("Write(%s) = %v, want a refusal", out, err)
	}
	if after := listTree(t, out); strings.Join(after, ",") != strings.Join(before, ",") {
		t.Fatalf("a refusal changed the directory: %v -> %v", before, after)
	}
}

func TestPageRefusesADirectoryItDidNotWrite(t *testing.T) {
	t.Run("a foreign file", func(t *testing.T) {
		out := t.TempDir()
		if err := os.WriteFile(filepath.Join(out, "notes.txt"), []byte("mine\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		refused(t, out)
	})
	t.Run("a foreign file beside an earlier copy", func(t *testing.T) {
		out := t.TempDir()
		if _, err := Write(out, ""); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(out, "vendor", "extra.js"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		refused(t, out)
	})
	t.Run("the page's names without its index", func(t *testing.T) {
		out := t.TempDir()
		if err := os.WriteFile(filepath.Join(out, "session.js"), []byte("someone else's\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		refused(t, out)
	})
	t.Run("an index.html the page did not write", func(t *testing.T) {
		out := t.TempDir()
		if err := os.WriteFile(filepath.Join(out, "index.html"), []byte("<!doctype html><title>mine</title>\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		refused(t, out)
	})
	t.Run("a symlink inside", func(t *testing.T) {
		out := t.TempDir()
		if _, err := Write(out, ""); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(filepath.Join(out, "vendor")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(t.TempDir(), filepath.Join(out, "vendor")); err != nil {
			t.Fatal(err)
		}
		refused(t, out)
	})
	t.Run("a symlink or a file as --out", func(t *testing.T) {
		dir := t.TempDir()
		link := filepath.Join(dir, "link")
		if err := os.Symlink(t.TempDir(), link); err != nil {
			t.Fatal(err)
		}
		if _, err := Write(link, ""); !errors.As(err, new(*Refusal)) {
			t.Fatalf("symlink --out: %v", err)
		}
		file := filepath.Join(dir, "file")
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Write(file, ""); !errors.As(err, new(*Refusal)) {
			t.Fatalf("file --out: %v", err)
		}
	})
	t.Run("no --out", func(t *testing.T) {
		if _, err := Write("", ""); !errors.As(err, new(*Refusal)) {
			t.Fatalf("empty --out: %v", err)
		}
	})
}

// provenance is the part of PROVENANCE.json the hash check reads.
type provenance struct {
	Bundle      string `json:"bundle"`
	SHA256      string `json:"sha256"`
	Entry       string `json:"entry"`
	EntrySHA256 string `json:"entry_sha256"`
	Packages    []struct {
		Name, Version, License, Integrity, Resolved string
	} `json:"packages"`
}

// checkProvenance proves the bundle and its entry module in dir are the bytes
// PROVENANCE.json records (adr-2610141949110653).
func checkProvenance(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "PROVENANCE.json"))
	if err != nil {
		return err
	}
	var p provenance
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if p.Bundle != "collab.js" || p.Entry != "collab.entry.js" || len(p.Packages) == 0 {
		return fmt.Errorf("PROVENANCE.json names %q and %q with %d packages", p.Bundle, p.Entry, len(p.Packages))
	}
	for _, c := range []struct{ file, want string }{{p.Bundle, p.SHA256}, {p.Entry, p.EntrySHA256}} {
		data, err := os.ReadFile(filepath.Join(dir, c.file))
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != c.want {
			return fmt.Errorf("%s has sha256 %s, PROVENANCE.json records %s", c.file, got, c.want)
		}
	}
	for _, pkg := range p.Packages {
		if pkg.License != "MIT" || !strings.HasPrefix(pkg.Integrity, "sha512-") || !strings.HasPrefix(pkg.Resolved, "https://registry.npmjs.org/") {
			return fmt.Errorf("package %s@%s: licence %q, integrity %q, resolved %q", pkg.Name, pkg.Version, pkg.License, pkg.Integrity, pkg.Resolved)
		}
	}
	return nil
}

func TestVendoredBundleMatchesItsProvenance(t *testing.T) {
	if err := checkProvenance("vendor"); err != nil {
		t.Fatal(err)
	}
	embedded, err := assets.ReadFile("vendor/collab.js")
	if err != nil {
		t.Fatal(err)
	}
	disk, err := os.ReadFile(filepath.Join("vendor", "collab.js"))
	if err != nil || !bytes.Equal(embedded, disk) {
		t.Fatalf("the embedded collab.js is not the committed one: %v", err)
	}
}

func TestProvenanceCheckCatchesATamperedBundle(t *testing.T) {
	for _, name := range []string{"collab.js", "collab.entry.js"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range []string{"PROVENANCE.json", "collab.js", "collab.entry.js"} {
				data, err := os.ReadFile(filepath.Join("vendor", f))
				if err != nil {
					t.Fatal(err)
				}
				if f == name {
					data = append(data, []byte("\n// tampered\n")...)
				}
				if err := os.WriteFile(filepath.Join(dir, f), data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			err := checkProvenance(dir)
			if err == nil || !strings.Contains(err.Error(), name+" has sha256") {
				t.Fatalf("a tampered %s passed the provenance check: %v", name, err)
			}
		})
	}
}

// cspWant is the page's content-security policy, in full.
const cspWant = `default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src https: wss: http://localhost:* ws://localhost:* http://127.0.0.1:* ws://127.0.0.1:*; base-uri 'none'; form-action 'none'`

func TestPageHTMLLoadsOnlyItsOwnFiles(t *testing.T) {
	html, err := assets.ReadFile("index.html")
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	for _, want := range []string{
		`<meta http-equiv="Content-Security-Policy" content="` + cspWant + `">`,
		`<meta name="referrer" content="no-referrer">`,
		Generator,
		`<script type="module" src="/session.js"></script>`,
		`<link rel="stylesheet" href="/session.css">`,
	} {
		if strings.Count(s, want) != 1 {
			t.Errorf("index.html does not carry %s exactly once", want)
		}
	}
	// Every src and href: the page's own files, named from the site's root
	// so /<owner>/<repo>/brief loads them with or without its trailing
	// slash, or an empty icon.
	allowed := map[string]bool{
		"/session.js": true, "/session.css": true, "/vendor/collab.js": true, "data:,": true,
	}
	refs := regexp.MustCompile(`(?i)\b(?:src|href)\s*=\s*"([^"]*)"`).FindAllStringSubmatch(s, -1)
	if len(refs) != 4 {
		t.Errorf("index.html has %d src/href attributes, want 4: %v", len(refs), refs)
	}
	for _, m := range refs {
		if !allowed[m[1]] {
			t.Errorf("index.html loads %q", m[1])
		}
	}
	if n := strings.Count(strings.ToLower(s), "<script"); n != 1 {
		t.Errorf("index.html has %d script elements, want the one module", n)
	}
	// Outside the policy's own connect-src list, the HTML names no address.
	rest := strings.Replace(s, cspWant, "", 1)
	for _, bad := range []string{"http://", "https://", "ws://", "wss://", "//cdn", "<iframe", "<base", "onload=", "onerror=", "javascript:"} {
		if strings.Contains(strings.ToLower(rest), bad) {
			t.Errorf("index.html contains %q", bad)
		}
	}
	css, err := assets.ReadFile("session.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"url(", "@import", "http://", "https://"} {
		if strings.Contains(string(css), bad) {
			t.Errorf("session.css contains %q: the stylesheet loads nothing", bad)
		}
	}
}

func TestPageScriptKeepsTheTokenAndTheDOMSafe(t *testing.T) {
	raw, err := assets.ReadFile("session.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(raw)
	// Text reaches the page through textContent, and nothing is evaluated.
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function", "console.", "srcdoc", "javascript:"} {
		if strings.Contains(js, bad) {
			t.Errorf("session.js contains %q", bad)
		}
	}
	// One import, the vendored bundle beside it.
	imports := regexp.MustCompile(`(?m)\bfrom\s+"([^"]+)"|\bimport\s*\(`).FindAllStringSubmatch(js, -1)
	if len(imports) != 1 || imports[0][1] != "./vendor/collab.js" {
		t.Errorf("session.js imports %v, want ./vendor/collab.js alone", imports)
	}
	// The only addresses it names are GitHub's API, its sign-in page and its
	// site, where a pre-filled issue opens.
	urls := regexp.MustCompile(`(?i)\b(?:https?|wss?)://[^\s"'`+"`"+`)]*`).FindAllString(js, -1)
	for _, u := range urls {
		if u != "https://api.github.com" && u != "https://github.com/login/oauth/authorize" && u != "https://github.com" && u != "https://" {
			t.Errorf("session.js names %q", u)
		}
	}
	// localStorage is written in one place, which remembers the relay, the
	// display name and the relays the person confirmed, and nothing else.
	if n := strings.Count(js, "localStorage.setItem("); n != 1 || !strings.Contains(js, "function remember(key, value) {\n  try { localStorage.setItem(key, value); }") {
		t.Errorf("localStorage.setItem appears %d times, or outside remember()", n)
	}
	remembers := regexp.MustCompile(`remember\(([^)]*)\)`).FindAllStringSubmatch(js, -1)
	for _, m := range remembers {
		if m[1] != "key, value" && m[1] != "KEY_RELAY, relay" && m[1] != "KEY_NAME, session.name" && m[1] != "KEY_TRUSTED, JSON.stringify(all" {
			t.Errorf("remember(%s): only the relay address, the display name and the confirmed relays are remembered", m[1])
		}
	}
	// Session storage holds the sign-in's state and verifier across GitHub's
	// redirect, and the rejoin token across a reload, and nothing else.
	if n := strings.Count(js, "sessionStorage.setItem("); n != 2 || !strings.Contains(js, "const pending = { state, verifier, relay: session.relay, code: session.code, name: session.name };") {
		t.Errorf("sessionStorage holds something besides the sign-in's state and verifier and the rejoin token (%d writes)", n)
	}
	if !strings.Contains(functionBody(t, js, "keepRejoin"), "sessionStorage.setItem(KEY_REJOIN, JSON.stringify({ relay: session.relay, repo: route.slug, token }))") {
		t.Error("keepRejoin stores something besides the relay, the repository and the rejoin token")
	}
	// The token lives in one variable (and, until its relay is cleared, in a
	// second that holds it) and goes only into verify and the Authorization
	// header: never into storage, a URL, the page or a log.
	lines := strings.Split(js, "\n")
	uses := 0
	for i, line := range lines {
		if !strings.Contains(line, "githubToken") && !strings.Contains(line, "heldToken") {
			continue
		}
		uses++
		for _, bad := range []string{"Storage", "remember(", "console.", "location", "history.", "textContent", "setAttribute", "URLSearchParams", "stash", "+ \"#", "+ \"?"} {
			if strings.Contains(line, bad) {
				t.Errorf("session.js:%d puts the token near %q: %s", i+1, bad, strings.TrimSpace(line))
			}
		}
	}
	if uses < 4 {
		t.Errorf("found the token variable on %d lines; the check is not looking at it", uses)
	}
	// The UI's words sit in one table at the top.
	if !strings.Contains(js, "const WORDS = Object.freeze({") {
		t.Error("session.js has no WORDS table")
	}
	// Step 4's hook is in place and named.
	if !strings.Contains(js, "function submitSession() {") {
		t.Error("session.js has no submitSession hook")
	}
}

// pageScript is session.js as the binary embeds it.
func pageScript(t *testing.T) string {
	t.Helper()
	raw, err := assets.ReadFile("session.js")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// functionBody returns the source of the named top-level function, from its
// declaration to the closing brace at the start of a line.
func functionBody(t *testing.T, js, name string) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^(?:async )?function ` + regexp.QuoteMeta(name) + `\(`).FindStringIndex(js)
	if m == nil {
		t.Fatalf("session.js has no function %s", name)
	}
	rest := js[m[0]:]
	end := strings.Index(rest, "\n}\n")
	if end < 0 {
		t.Fatalf("function %s does not end", name)
	}
	return rest[:end+2]
}

func TestIssueLinksAreMeasuredAgainstOneNamedLimit(t *testing.T) {
	js := pageScript(t)
	m := regexp.MustCompile(`(?m)((?://.*\n)+)const ISSUE_URL_LIMIT = (\d+);`).FindStringSubmatch(js)
	if m == nil {
		t.Fatal("session.js has no ISSUE_URL_LIMIT constant with a comment above it")
	}
	var limit int
	if _, err := fmt.Sscan(m[2], &limit); err != nil || limit > 8000 || limit < 4000 {
		t.Errorf("ISSUE_URL_LIMIT = %s, want a margin under GitHub's roughly 8 KB", m[2])
	}
	if !strings.Contains(m[1], "2026-10-14-github-roles-admission-and-issue-links.md") {
		t.Errorf("the limit's comment does not cite the research note:\n%s", m[1])
	}
	// The link is built with encodeURIComponent and measured in bytes, whole.
	link := functionBody(t, js, "issueLink")
	for _, want := range []string{`"/issues/new?title=" + encodeURIComponent(`, `"&body=" + encodeURIComponent(`, "utf8Length(url) <= ISSUE_URL_LIMIT"} {
		if !strings.Contains(link, want) {
			t.Errorf("issueLink does not carry %q", want)
		}
	}
	// No labels in the link: GitHub answers 404 to a visitor who may not set them.
	for _, bad := range []string{"labels=", "&labels", `"labels"`, "labels:"} {
		if strings.Contains(js, bad) {
			t.Errorf("session.js carries %q", bad)
		}
	}
	// The issue body names the file, the base commit, both versions and the credit.
	body := functionBody(t, js, "issueBody")
	for _, want := range []string{"p.path", "effectiveMeta().base_sha", "fenced(p.before)", "fenced(p.after)", "creditLines(p)"} {
		if !strings.Contains(body, want) {
			t.Errorf("issueBody does not carry %s", want)
		}
	}
	credit := functionBody(t, js, "creditLines")
	if !strings.Contains(credit, `"Suggested by " + a.guest + " (guest), approved by @" + a.approver`) {
		t.Error("creditLines does not credit a guest's change to its approver")
	}
}

func TestPullRequestCommitCreditsEveryone(t *testing.T) {
	js := pageScript(t)
	trailers := functionBody(t, js, "composeTrailers")
	for _, want := range []string{
		`"Co-authored-by: " + noreply(`,
		`lines.push("Suggested-by: " + p.guest + " (guest)", "Approved-by: " + noreply(`,
		"!== submitter.toLowerCase()",
	} {
		if !strings.Contains(trailers, want) {
			t.Errorf("composeTrailers does not carry %q", want)
		}
	}
	if !strings.Contains(functionBody(t, js, "noreply"), `"+" + login + "@users.noreply.github.com>"`) {
		t.Error("noreply does not use GitHub's id+login noreply address")
	}
	pr := functionBody(t, js, "openPullRequest")
	for _, want := range []string{
		`"/git/refs"`, `ref: "refs/heads/" + branch, sha: baseSha`,
		`"/graphql"`, "createCommitOnBranch", "expectedHeadOid: baseSha", "repositoryNameWithOwner",
		`"/pulls"`, "head: branch, base: target",
		`"/users/" + encodeURIComponent(`,
		"headline: COMMIT_HEADLINE",
	} {
		if !strings.Contains(pr, want) {
			t.Errorf("openPullRequest does not carry %q", want)
		}
	}
	if !strings.Contains(js, `const COMMIT_HEADLINE = "Edit the brief in a live session";`) {
		t.Error("the commit headline is not the spec's")
	}
	if !regexp.MustCompile(`"abcdesign/" \+ `).MatchString(functionBody(t, js, "branchName")) {
		t.Error("branchName does not name an abcdesign/ branch")
	}
}

func TestFileContentsAreUTF8Base64(t *testing.T) {
	js := pageScript(t)
	enc := functionBody(t, js, "utf8Base64")
	if !strings.Contains(enc, "new TextEncoder().encode(text)") || !strings.Contains(enc, "btoa(binary)") {
		t.Errorf("utf8Base64 does not encode UTF-8 bytes:\n%s", enc)
	}
	if !strings.Contains(functionBody(t, js, "openPullRequest"), "contents: utf8Base64(") {
		t.Error("the commit's file contents do not go through utf8Base64")
	}
	// btoa sees a binary string built from bytes, never text.
	calls := regexp.MustCompile(`btoa\(([^)]*)\)`).FindAllStringSubmatch(js, -1)
	if len(calls) == 0 {
		t.Fatal("no btoa call found; the check is not looking")
	}
	for _, c := range calls {
		if c[1] != "binary" && c[1] != "s" {
			t.Errorf("btoa(%s): btoa is applied only to a binary string built from bytes", c[1])
		}
	}
	for _, fn := range []string{"b64url", "utf8Base64"} {
		body := functionBody(t, js, fn)
		if !strings.Contains(body, "String.fromCharCode(") {
			t.Errorf("%s does not build its binary string from bytes", fn)
		}
	}
}

func TestSuggestionsLiveInTheDocument(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(js, `const suggestions = doc.getMap("suggestions");`) {
		t.Fatal("suggestions are not a map in the shared document")
	}
	on := functionBody(t, js, "onSuggest")
	if !strings.Contains(on, "!suggestions.has(id)") || !strings.Contains(on, "suggestions.set(id, ") {
		t.Errorf("onSuggest does not record a suggestion in the document once:\n%s", on)
	}
	if strings.Contains(js, "incoming") {
		t.Error("session.js still keeps suggestions in page memory (incoming)")
	}
	if !strings.Contains(functionBody(t, js, "renderSide"), "suggestions.values()") {
		t.Error("the members' panel does not read the suggestions from the document")
	}
}

func TestPageRejoinsWithItsTokenBeforeTheCode(t *testing.T) {
	js := pageScript(t)
	// The token comes with hosted and joined, and is kept in memory and in
	// session storage, never in local storage.
	for _, fn := range []string{"onHosted", "onJoined"} {
		if !strings.Contains(functionBody(t, js, fn), "keepRejoin(msg.rejoin)") {
			t.Errorf("%s does not keep the rejoin token", fn)
		}
	}
	if strings.Contains(js, "remember(KEY_REJOIN") || strings.Contains(js, "localStorage.setItem(KEY_REJOIN") {
		t.Error("the rejoin token reaches local storage")
	}
	if !strings.Contains(js, `const REJOIN_RE = /^[A-Za-z0-9_-]{22}$/;`) {
		t.Error("the page does not check a rejoin token's shape")
	}
	// A reconnect presents the token first, before any sign-in or code.
	open := functionBody(t, js, "afterOpen")
	at := strings.Index(open, `sendJSON({ type: "rejoin", token: rejoinToken });`)
	if at < 0 || at > strings.Index(open, `type: "verify"`) {
		t.Errorf("afterOpen does not rejoin before it verifies or joins:\n%s", open)
	}
	// A refused token is dropped and the page falls back to the code.
	if !strings.Contains(functionBody(t, js, "onError"), `code === "bad-rejoin"`) {
		t.Error("onError does not fall back from a refused rejoin token")
	}
	// A reload with a kept token for this relay and repository rejoins.
	if !strings.Contains(functionBody(t, js, "main"), "if (takeRejoin()) openSocket();") {
		t.Error("a reloaded page does not rejoin with its kept token")
	}
	// So does a page coming back from GitHub's sign-in.
	if !strings.Contains(functionBody(t, js, "finishSignIn"), "takeRejoin();\n  await presentHeldToken(false);") {
		t.Error("a page back from sign-in does not rejoin with its kept token")
	}
	if !strings.Contains(functionBody(t, js, "endSession"), "forgetRejoin()") {
		t.Error("an ended session's rejoin token is kept")
	}
}

func TestPageRedirectsTheBriefPathToItsSlash(t *testing.T) {
	js := pageScript(t)
	fn := functionBody(t, js, "addSlash")
	for _, want := range []string{`/\/[^/]+\/[^/]+\/brief$/.test(location.pathname)`, `location.replace(location.pathname + "/" + location.search + location.hash)`} {
		if !strings.Contains(fn, want) {
			t.Errorf("addSlash does not carry %q:\n%s", want, fn)
		}
	}
	if !strings.Contains(functionBody(t, js, "main"), "if (addSlash()) return;") {
		t.Error("main does not redirect /<owner>/<repo>/brief to its slash first")
	}
}

func TestPageUnderABasePath(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", "/"}, {"/", "/"}, {"abcdesign", "/abcdesign/"}, {"/abcdesign", "/abcdesign/"},
		{"abcdesign/", "/abcdesign/"}, {"/a/b.c_d~e-f/", "/a/b.c_d~e-f/"},
	} {
		if got, err := NormaliseBasePath(c.in); err != nil || got != c.want {
			t.Errorf("NormaliseBasePath(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"..", "/a/../b", "a//b", "a b", "https://x.example", "a?b", "a#b", "a%20b", `a"b`, "a<b", "./a", "//"} {
		if got, err := NormaliseBasePath(bad); err == nil {
			t.Errorf("NormaliseBasePath(%q) = %q, want a refusal", bad, got)
		}
	}

	byPath := func(files []File) map[string]string {
		m := map[string]string{}
		for _, f := range files {
			m[f.Path] = string(f.Data)
		}
		return m
	}
	root := byPath(Files("/"))
	pref := byPath(Files("/abcdesign/"))
	// The default output names the assets from the root; the prefixed one
	// from under the base path, and nothing else changes.
	for _, want := range []string{`src="/session.js"`, `href="/session.css"`, `href="/vendor/collab.js"`} {
		if !strings.Contains(root["index.html"], want) {
			t.Errorf("the default index.html does not carry %s", want)
		}
		pw := strings.Replace(want, `"/`, `"/abcdesign/`, 1)
		if !strings.Contains(pref["index.html"], pw) {
			t.Errorf("the prefixed index.html does not carry %s", pw)
		}
	}
	if strings.Contains(pref["index.html"], `="/session`) || strings.Contains(pref["index.html"], `="/vendor`) {
		t.Error("the prefixed index.html still names an asset from the root")
	}
	if pref["404.html"] != pref["index.html"] {
		t.Error("the prefixed 404.html is not its index.html")
	}
	if strings.Replace(strings.Replace(strings.Replace(pref["index.html"], "/abcdesign/session.js", "/session.js", 1), "/abcdesign/session.css", "/session.css", 1), "/abcdesign/vendor/collab.js", "/vendor/collab.js", 1) != root["index.html"] {
		t.Error("the prefixed index.html differs from the default beyond its asset paths")
	}
	for path, data := range root {
		if path != "index.html" && path != "404.html" && pref[path] != data {
			t.Errorf("%s differs under a base path", path)
		}
	}

	out := filepath.Join(t.TempDir(), "page")
	if _, err := Write(out, "abcdesign"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil || string(got) != pref["index.html"] {
		t.Fatalf("Write under a base path wrote %q, %v", got, err)
	}
	// An earlier copy written under another base path is still the page's own.
	if res, err := Write(out, ""); err != nil || !res.Replaced {
		t.Fatalf("rewrite at the root: %+v, %v", res, err)
	}
	fresh := filepath.Join(t.TempDir(), "page")
	if _, err := Write(fresh, "../up"); !errors.As(err, new(*Refusal)) {
		t.Fatalf("Write with a bad base path = %v, want a refusal", err)
	}
	if _, err := os.Stat(fresh); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("a refused base path still created the directory")
	}
}

func TestIssueBodyEndsWithTheSessionMarker(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(js, `const ISSUE_MARKER = "Sent from an abcdesign.app brief session.";`) {
		t.Fatal("session.js does not name the issue marker")
	}
	if !strings.HasSuffix(functionBody(t, js, "issueBody"), "    \"\",\n    ISSUE_MARKER,\n  ].join(\"\\n\");\n}") {
		t.Errorf("issueBody does not end with the marker on its own line:\n%s", functionBody(t, js, "issueBody"))
	}
	if !strings.Contains(functionBody(t, js, "issueTitle"), `return "Brief: " + `) {
		t.Error("an issue's title does not start with Brief:")
	}
}

func TestEveryApprovedGuestChangeGetsItsOwnPair(t *testing.T) {
	js := pageScript(t)
	// Approvals are told apart by the approval, not by who made them: two
	// approvals by one guest and one approver are two pairs.
	for fn, want := range map[string]string{
		"makePassage":     "approved.set(approvalKey(a), { key: approvalKey(a), guest, approver });",
		"openPullRequest": "pairs.set(a.key, { guest, approver: a.approver });",
	} {
		body := functionBody(t, js, fn)
		if strings.Contains(body, `guest + "\u0000" + approver`) || strings.Contains(body, `guest + "\u0000" + a.approver`) {
			t.Errorf("%s merges approvals by guest and approver", fn)
		}
		if !strings.Contains(body, want) {
			t.Errorf("%s does not key approvals by the approval: want %s", fn, want)
		}
	}
	if !strings.Contains(functionBody(t, js, "approvalKey"), `typeof a.id === "string" && a.id ? a.id : a`) {
		t.Error("approvalKey does not name an approval by its own id")
	}
	if !strings.Contains(functionBody(t, js, "composeTrailers"), "for (const p of pairs) lines.push(") {
		t.Error("composeTrailers does not write one pair per approval")
	}
}

func TestGuestNamesMentionNobody(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(functionBody(t, js, "trailerText"), `.replace(/@/g, "(at)")`) {
		t.Error("trailerText leaves an @ in a guest's name")
	}
	// Every guest name in a trailer, an issue body or a pull request body
	// comes through trailerText.
	if !strings.Contains(functionBody(t, js, "makePassage"), "const guest = trailerText(a.guest);") {
		t.Error("a passage's guest name does not go through trailerText")
	}
	pr := functionBody(t, js, "prBody")
	if !strings.Contains(pr, "people.add(trailerText(r.name)") {
		t.Error("prBody names a guest without trailerText")
	}
	if !strings.Contains(functionBody(t, js, "openPullRequest"), "const guest = trailerText(a.guest);") {
		t.Error("the pull request's pairs name a guest without trailerText")
	}
}

// The commit's paths are the shared files map's keys, which any team
// member's page can write, so each is confined to the brief the session
// started from before it is shown or sent (security review F9).
func TestCommitPathsAreConfinedToTheBrief(t *testing.T) {
	js := pageScript(t)
	brief := functionBody(t, js, "briefFile")
	for _, want := range []string{
		"path.startsWith(BRIEF_PREFIX)", `path.endsWith(".md")`, `!path.includes("..")`, `!path.includes("\\")`, "baseFiles.has(path)",
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("briefFile does not check %s:\n%s", want, brief)
		}
	}
	if !strings.Contains(functionBody(t, js, "baseFilePaths"), "Y.typeMapGetAllSnapshot(files, base)") {
		t.Error("the base file set is not the files map as the base snapshot saw it")
	}
	an := functionBody(t, js, "analyse")
	check, changes := strings.Index(an, "if (!briefFile(path, baseFiles)) {"), strings.Index(an, "changesOf(")
	if check < 0 || changes < 0 || check > changes {
		t.Errorf("analyse does not refuse a path outside the brief before reading its changes:\n%s", an)
	}
	if !strings.Contains(an, "refused.push(path)") || !strings.Contains(functionBody(t, js, "renderSend"), "analysis.refused") {
		t.Error("a refused path is not named in the Send panel")
	}
	// Immediately before every additions.push, the path is checked again.
	pr := functionBody(t, js, "openPullRequest")
	pushes := strings.Count(pr, "additions.push(")
	guarded := regexp.MustCompile(`if \(!briefFile\(f\.path, baseFiles\)\) throw new GitHubTrouble\([^\n]*\n\s*additions\.push\(`).FindAllString(pr, -1)
	if pushes == 0 || len(guarded) != pushes {
		t.Errorf("openPullRequest has %d additions.push, %d guarded by briefFile", pushes, len(guarded))
	}
	if !strings.Contains(js, "  pathRefused: ") {
		t.Error("the page has no words for a refused path")
	}
}

// The vendored editor writes each participant's awareness colour into a
// style attribute and their name into the page, so a remote awareness
// update is decoded and cleaned before it is applied (security review F7).
func TestAwarenessIsCleanedBeforeItIsApplied(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(js, "const HEX_COLOUR_RE = /^#[0-9a-fA-F]{6}$/;") {
		t.Fatal("session.js has no six-digit hex colour pattern")
	}
	// The one call that applies a remote awareness update applies a cleaned one.
	if n := strings.Count(js, "applyAwarenessUpdate("); n != 1 {
		t.Errorf("applyAwarenessUpdate is called %d times, want once", n)
	}
	if !strings.Contains(functionBody(t, js, "admitAwareness"), "applyAwarenessUpdate(awareness, cleanAwarenessUpdate(") {
		t.Error("admitAwareness applies a remote awareness update without cleaning it first")
	}
	clean := functionBody(t, js, "cleanAwarenessUpdate")
	for _, want := range []string{"readVarUint(", "writeVarUint(", "JSON.parse(", "cleanAwarenessState(", "JSON.stringify("} {
		if !strings.Contains(clean, want) {
			t.Errorf("cleanAwarenessUpdate does not carry %s", want)
		}
	}
	state := functionBody(t, js, "cleanAwarenessState")
	for _, want := range []string{"HEX_COLOUR_RE.test(u.color)", "HEX_COLOUR_RE.test(u.colorLight)", "name: label"} {
		if !strings.Contains(state, want) {
			t.Errorf("cleanAwarenessState does not carry %s:\n%s", want, state)
		}
	}
	// The editor sends a relative position as Yjs builds it, with null for
	// a field it does not use (tname, or item at the end of a text): null is
	// such a field's absence, not a malformed position, or no cursor shows.
	pos := functionBody(t, js, "cleanPosition")
	for _, want := range []string{"p.item != null", "p.type != null", "p.tname != null", "p.assoc != null"} {
		if !strings.Contains(pos, want) {
			t.Errorf("cleanPosition does not take a null field as absent (%s):\n%s", want, pos)
		}
	}
	name := functionBody(t, js, "cleanName")
	if !strings.Contains(name, `\u0000-\u001f\u007f-\u009f`) || !strings.Contains(name, ".slice(0, 40)") {
		t.Errorf("cleanName does not remove control characters and clamp to 40:\n%s", name)
	}
}

// The relay sends each member's update on as tag 2 with the sender's
// participant number, and the roster names each number's login, so a page
// applies an update only when every client id in it is credited to its
// sender, and credits an approval or a deletion record only to the login
// whose client wrote it (security review F6).
func TestAttributedUpdatesMustBeTheirSendersOwn(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(js, "const TAG_ATTRIBUTED = 2;") {
		t.Fatal("session.js does not know the relay's attributed update (tag 2)")
	}
	recv := functionBody(t, js, "receive")
	for _, want := range []string{"bytes[0] === TAG_ATTRIBUTED", "getUint32(1)", "admitUpdate(", "noteUnattributed("} {
		if !strings.Contains(recv, want) {
			t.Errorf("receive does not carry %s", want)
		}
	}
	// Until the roster that names the updates' senders arrives, they wait.
	if !strings.Contains(recv, "if (!rosterSeen) held.push(") {
		t.Error("receive judges an attributed update before a roster names its sender")
	}
	roster := functionBody(t, js, "onRoster")
	if !strings.Contains(roster, "session.authors = ") || !strings.Contains(roster, "rosterSeen = true;") || !strings.Contains(roster, "admitUpdate(") {
		t.Errorf("onRoster does not take the authors and judge the waiting updates:\n%s", roster)
	}
	forged := functionBody(t, js, "forgedUpdate")
	for _, want := range []string{"Y.decodeUpdate(update)", "session.authors[String(num)]", "creditedTo(s.id.client) !== login"} {
		if !strings.Contains(forged, want) {
			t.Errorf("forgedUpdate does not carry %s:\n%s", want, forged)
		}
	}
	admit := functionBody(t, js, "admitUpdate")
	at, apply := strings.Index(admit, "forgedUpdate("), strings.Index(admit, "Y.applyUpdate(")
	if at < 0 || apply < 0 || at > apply || !strings.Contains(admit, "WORDS.forgedUpdate") {
		t.Errorf("admitUpdate does not drop a forged update, with a warning, before applying:\n%s", admit)
	}
	// A tag-0 update is applied but credits nobody.
	if !strings.Contains(functionBody(t, js, "creditedTo"), "unattributed.has(") {
		t.Error("a client id from an unattributed update is still credited")
	}
	// Approvals and deletion records count only when their writer is theirs.
	if strings.Contains(functionBody(t, js, "analyse"), "approvals.toArray()") || !strings.Contains(functionBody(t, js, "analyse"), "verifiedApprovals()") {
		t.Error("analyse reads approvals without checking who wrote them")
	}
	va := functionBody(t, js, "verifiedApprovals")
	for _, want := range []string{"item.id.client", "a.client !== writer", "loginOf(writer) !== a.approver) continue;", "a.ins[0] === writer"} {
		if !strings.Contains(va, want) {
			t.Errorf("verifiedApprovals does not carry %s", want)
		}
	}
	del := functionBody(t, js, "deletersOf")
	if strings.Contains(del, "deletions.dss") || !strings.Contains(del, "verifiedDeleters()") {
		t.Error("deletersOf reads deletion records without checking who wrote them")
	}
	vd := functionBody(t, js, "verifiedDeleters")
	for _, want := range []string{"item.id.client", "loginOf(desc)"} {
		if !strings.Contains(vd, want) {
			t.Errorf("verifiedDeleters does not carry %s", want)
		}
	}
	if !strings.Contains(js, "  forgedUpdate: ") {
		t.Error("the page has no words for a dropped update")
	}
}

// A link can name any relay in its fragment, so no GitHub token and no
// sign-in reaches a relay until the repository's committed relay list names
// it, or, where that list cannot be read, the person has confirmed it
// (security review F1).
func TestNoTokenReachesARelayTheRepositoryDoesNotName(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(js, `const RELAYS_FILE = ".abcd/relay/relays.json";`) {
		t.Fatal("session.js does not name the repository's relay list")
	}
	read := functionBody(t, js, "readRelayList")
	if !strings.Contains(read, `GITHUB_API + repoPath() + "/contents/" + RELAYS_FILE`) {
		t.Errorf("readRelayList does not read the list from api.github.com:\n%s", read)
	}
	// The anonymous read comes before anything else the page does.
	main := functionBody(t, js, "main")
	anon, finish := strings.Index(main, `anchor = await readRelayList("");`), strings.Index(main, "finishSignIn()")
	if anon < 0 || finish < 0 || anon > finish {
		t.Errorf("main does not read the relay list before finishing a sign-in:\n%s", main)
	}
	// The relay card is hidden only for a relay that is cleared.
	for _, line := range strings.Split(js, "\n") {
		if strings.Contains(line, "relayCard.hidden = true") && !strings.Contains(line, "relayCleared()") {
			t.Errorf("the relay card is hidden for a relay that may not be cleared: %s", strings.TrimSpace(line))
		}
	}
	// Sign-in starts only through a cleared relay, before /config is asked.
	start := functionBody(t, js, "startSignIn")
	gate, cfg := strings.Index(start, `if (!relayClearedFor("signin")) return;`), strings.Index(start, "relayConfig()")
	if gate < 0 || cfg < 0 || gate > cfg {
		t.Errorf("startSignIn asks the relay before clearing it:\n%s", start)
	}
	// GitHub's code goes to the relay's exchange only when it is cleared.
	fin := functionBody(t, js, "finishSignIn")
	gate, post := strings.Index(fin, "if (!relayCleared())"), strings.Index(fin, `"/auth/token"`)
	if gate < 0 || post < 0 || gate > post {
		t.Errorf("finishSignIn posts the code before clearing the relay:\n%s", fin)
	}
	if !strings.Contains(fin, "heldToken = out.access_token;") || !strings.Contains(fin, "await presentHeldToken(false);") {
		t.Error("finishSignIn does not hold the token for the relay check")
	}
	// A token is held, and becomes the token verify sends only once the relay
	// is cleared, after api.github.com was asked with it.
	if strings.Contains(functionBody(t, js, "useToken"), "githubToken =") {
		t.Error("useToken presents the pasted token without the relay check")
	}
	present := functionBody(t, js, "presentHeldToken")
	ask, give := strings.Index(present, "readRelayList(heldToken)"), strings.Index(present, "githubToken = heldToken;")
	if ask < 0 || give < 0 || ask > give || !strings.Contains(present, "forgetRelay(relay)") {
		t.Errorf("presentHeldToken does not ask GitHub first and forget a refused relay:\n%s", present)
	}
	for _, m := range regexp.MustCompile(`githubToken = ([^;]+);`).FindAllStringSubmatch(js, -1) {
		if m[1] != `""` && m[1] != "heldToken" {
			t.Errorf("githubToken is set from %s, not only from a held token", m[1])
		}
	}
	for fn, want := range map[string]string{
		"afterOpen":       "if (githubToken && relayCleared())",
		"connectSignedIn": "if (!githubToken || !relayCleared())",
		"onJoined":        "githubToken && relayCleared()",
		"readRelayInput":  "if (relayCleared()) remember(KEY_RELAY, relay);",
		"openSocket":      `relayStanding(session.relay) === "refused"`,
	} {
		if !strings.Contains(functionBody(t, js, fn), want) {
			t.Errorf("%s does not carry %s", fn, want)
		}
	}
	// The confirmation names the relay in plain words.
	if !strings.Contains(js, `trustOnly: "Only continue if your team runs this relay:"`) {
		t.Error("the confirmation does not say to continue only for the team's own relay")
	}
	if !strings.Contains(functionBody(t, js, "confirmRelay"), "remember(KEY_TRUSTED, JSON.stringify(all));") {
		t.Error("a confirmed relay is not remembered per repository")
	}
}

// Sign-in asks for no OAuth scope when the relay's app is a GitHub App,
// whose user tokens are bounded by the app's installations and permissions.
func TestSignInPrefersAGitHubApp(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(functionBody(t, js, "relayConfig"), `cfg.app_type === "github-app"`) {
		t.Error("relayConfig does not read the relay's app type")
	}
	start := functionBody(t, js, "startSignIn")
	if !strings.Contains(start, `if (cfg.appType !== "github-app") q.set("scope", SIGN_IN_SCOPE);`) {
		t.Errorf("startSignIn asks for a scope whatever the app type:\n%s", start)
	}
	if regexp.MustCompile(`(?s)new URLSearchParams\(\{[^}]*scope`).MatchString(start) {
		t.Error("the authorize request always carries a scope")
	}
	if !strings.Contains(js, "docs.github.com/en/apps/") {
		t.Error("the GitHub App preference does not cite GitHub's documentation")
	}
	if !strings.Contains(js, "  oauthNote: ") {
		t.Error("the page has no note that a GitHub App is narrower")
	}
}

// The relay's errors each have their answer: an idle connection's normal
// close is quiet, a repository the relay does not serve and a pause on
// joining by code are said, and an update dropped for coming too fast is
// sent again.
func TestPageAnswersTheRelaysRefusals(t *testing.T) {
	js := pageScript(t)
	on := functionBody(t, js, "onError")
	for _, want := range []string{`code === "join-timeout"`, `code === "repo-not-served"`, `code === "paused"`, `code === "rate-limited"`, "resendRecent()"} {
		if !strings.Contains(on, want) {
			t.Errorf("onError does not answer %s", want)
		}
	}
	if !strings.Contains(functionBody(t, js, "socketClosed"), "if (quiet && !session.everJoined) {") {
		t.Error("a join-timeout close is reported as a failure to connect")
	}
	resend := functionBody(t, js, "resendRecent")
	if !strings.Contains(resend, "Y.mergeUpdates(recent)") || !strings.Contains(resend, "sendBinary(TAG_UPDATE, merged)") {
		t.Errorf("resendRecent does not send the recent updates again, merged:\n%s", resend)
	}
}

// Any member's page can write the shared meta map, so the page counts a
// setting only when the host's login wrote it: the base, the branch and the
// repository once, at seeding, and the mode as the host last set it. Every
// read of meta goes through that one rule, the pull request's target and the
// commit's expected head among them, and a void write is shown.
func TestSessionSettingsAreTheHostsAlone(t *testing.T) {
	js := pageScript(t)
	if n := strings.Count(js, "meta.get("); n != 0 {
		t.Errorf("session.js reads meta.get( %d times outside the one rule", n)
	}
	if n := strings.Count(js, "meta._map"); n != 1 || !strings.Contains(functionBody(t, js, "effectiveMeta"), "meta._map.get(key)") {
		t.Errorf("meta's items are read %d times, want once, in effectiveMeta", n)
	}
	if !strings.Contains(js, `const META_ONCE = ["base_snapshot", "base_sha", "branch", "repo"];`) {
		t.Error("session.js does not name the settings written once")
	}
	em := functionBody(t, js, "effectiveMeta")
	for _, want := range []string{"session.hostLogin", "item.left", "creditedTo(item.id.client)", "META_ONCE.includes(key)", "voided.push("} {
		if !strings.Contains(em, want) {
			t.Errorf("effectiveMeta does not carry %s:\n%s", want, em)
		}
	}
	for _, fn := range []string{"baseSnapshot", "issueBody", "prBody", "openPullRequest", "renderSend", "renderSide"} {
		if !strings.Contains(functionBody(t, js, fn), "effectiveMeta()") {
			t.Errorf("%s does not read the settings through effectiveMeta", fn)
		}
	}
	pr := functionBody(t, js, "openPullRequest")
	for _, want := range []string{"const baseSha = settings.base_sha;", "const target = settings.branch;", "settings.repo", "expectedHeadOid: baseSha,", "base: target,"} {
		if !strings.Contains(pr, want) {
			t.Errorf("openPullRequest does not carry %s", want)
		}
	}
	roster := functionBody(t, js, "onRoster")
	if !strings.Contains(roster, "msg.host") || !strings.Contains(roster, "session.hostLogin = ") || !strings.Contains(roster, "watchMeta()") {
		t.Errorf("onRoster does not take the host's login and judge the settings again:\n%s", roster)
	}
	if !strings.Contains(js, "meta.observe(") || !strings.Contains(functionBody(t, js, "watchMeta"), "WORDS.metaVoid") {
		t.Error("a void write to the settings is not shown")
	}
	if !strings.Contains(js, "  metaVoid: ") {
		t.Error("the page has no words for a void write to the settings")
	}
}

// The relay sends each awareness message on as tag 3 with its sender's
// participant number, so the page keeps an awareness state only when the
// roster ties its client id to that sender, and labels the cursor from the
// roster, never from the name the message claims.
func TestAwarenessIsTiedToItsSender(t *testing.T) {
	js := pageScript(t)
	if !strings.Contains(js, "const TAG_ATTRIBUTED_AWARENESS = 3;") {
		t.Fatal("session.js does not know the relay's attributed awareness (tag 3)")
	}
	recv := functionBody(t, js, "receive")
	if !strings.Contains(recv, "bytes[0] === TAG_ATTRIBUTED_AWARENESS") || !strings.Contains(recv, "admitAwareness(") {
		t.Error("receive does not judge attributed awareness by its sender")
	}
	if strings.Contains(recv, "bytes[0] === TAG_AWARENESS") {
		t.Error("receive still applies awareness that names no sender")
	}
	admit := functionBody(t, js, "admitAwareness")
	for _, want := range []string{"rosterSeen", "awarenessTied(", "cursorLabel("} {
		if !strings.Contains(admit, want) {
			t.Errorf("admitAwareness does not carry %s:\n%s", want, admit)
		}
	}
	tied := functionBody(t, js, "awarenessTied")
	for _, want := range []string{"creditedTo(client) === sender.login", "guestAwareness.get(num)", "guestAwareness.set(num, client)", `creditedTo(client) !== ""`} {
		if !strings.Contains(tied, want) {
			t.Errorf("awarenessTied does not carry %s:\n%s", want, tied)
		}
	}
	if !strings.Contains(functionBody(t, js, "onRoster"), "guestAwareness.delete(") {
		t.Error("a guest's tie is not released when the guest leaves the roster")
	}
	label := functionBody(t, js, "cursorLabel")
	if !strings.Contains(label, "sender.login") || !strings.Contains(label, "cleanName(sender.name)") || !strings.Contains(label, "WORDS.guestCursor") {
		t.Errorf("cursorLabel does not label from the roster:\n%s", label)
	}
}

// Every page refuses the same forged update, so an update that depends on
// its clocks would wait in Yjs's pending set for ever: the page sets it
// aside, and says so.
func TestRefusedUpdatesLeaveNothingPending(t *testing.T) {
	js := pageScript(t)
	admit := functionBody(t, js, "admitUpdate")
	if !strings.Contains(admit, "noteRefused(update)") || !strings.Contains(admit, "settlePending()") {
		t.Errorf("admitUpdate does not note a refused update and settle what waits on it:\n%s", admit)
	}
	if !strings.Contains(functionBody(t, js, "noteRefused"), "refusedFrom.set(") {
		t.Error("noteRefused does not record the refused clocks")
	}
	settle := functionBody(t, js, "settlePending")
	for _, want := range []string{"store.pendingStructs.missing", "store.pendingStructs = null;", "Y.decodeUpdateV2(store.pendingDs)", "store.pendingDs = null;", "WORDS.setAside"} {
		if !strings.Contains(settle, want) {
			t.Errorf("settlePending does not carry %s:\n%s", want, settle)
		}
	}
	if !strings.Contains(js, "  setAside: ") {
		t.Error("the page has no words for edits set aside")
	}
}
