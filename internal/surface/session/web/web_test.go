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
	// localStorage is written in one place, which remembers the relay and
	// the display name and nothing else.
	if n := strings.Count(js, "localStorage.setItem("); n != 1 || !strings.Contains(js, "function remember(key, value) {\n  try { localStorage.setItem(key, value); }") {
		t.Errorf("localStorage.setItem appears %d times, or outside remember()", n)
	}
	remembers := regexp.MustCompile(`remember\(([^)]*)\)`).FindAllStringSubmatch(js, -1)
	for _, m := range remembers {
		if m[1] != "key, value" && m[1] != "KEY_RELAY, relay" && m[1] != "KEY_NAME, session.name" {
			t.Errorf("remember(%s): only the relay address and the display name are remembered", m[1])
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
	// The token lives in one variable and goes only into verify and the
	// Authorization header: never into storage, a URL, the page or a log.
	lines := strings.Split(js, "\n")
	uses := 0
	for i, line := range lines {
		if !strings.Contains(line, "githubToken") {
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
	for _, want := range []string{"p.path", `meta.get("base_sha")`, "fenced(p.before)", "fenced(p.after)", "creditLines(p)"} {
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
	if !strings.Contains(functionBody(t, js, "finishSignIn"), "takeRejoin();\n  connectSignedIn();") {
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
