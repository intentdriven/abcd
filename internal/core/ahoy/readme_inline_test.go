package ahoy

import (
	"strings"
	"testing"
)

// The inline form of the README parts (iss-2610311544004033): a part whose
// opening mark continues an open HTML block is written as HTML, since the
// forge renders no Markdown there, and stays on its own lines with no blank
// line around it, so the block it sits in stays one block.

// wantInline asserts the part was derived with exactly inline as its inline
// form.
func wantInline(t *testing.T, d readmeDerived, inline string) {
	t.Helper()
	if d.Reason != "" || d.Inline != inline {
		t.Errorf("%s inline:\n got %q (reason %q)\nwant %q", d.Name, d.Inline, d.Reason, inline)
	}
}

const (
	inlineRelease = `<a href="https://github.com/owner/repo/releases/latest"><img src="https://img.shields.io/github/v/release/owner/repo" alt="Latest release"></a>`
	inlineBuild   = `<a href="https://github.com/owner/repo/actions/workflows/ci.yml"><img src="https://github.com/owner/repo/actions/workflows/ci.yml/badge.svg" alt="Build"></a>`
	inlineAbcd    = `<a href="https://github.com/intentdriven/abcd"><img src="https://img.shields.io/badge/managed%20with-abcd-blue" alt="Managed with abcd"></a>`
)

// TestReadmeInlineBodiesDerived: every part is derived in both forms. A badge
// inline is one <a> around one <img>; the licence, the links and the contents
// are each one line of <a> links joined by a spaced middle dot, with short
// labels; the inline contents lists level-two headings only, a heading whose
// anchor abcd cannot be sure of shown as plain text, and every text and
// attribute HTML-escaped.
func TestReadmeInlineBodiesDerived(t *testing.T) {
	root := readmeRepo(t, "https://github.com/owner/repo.git", map[string]string{
		".github/workflows/ci.yml": "on: push\n",
		"LICENSE":                  "MIT License\n\nCopyright\n",
		"CHANGELOG.md":             "# Changelog\n",
		".github/CONTRIBUTING.md":  "# Contributing\n",
		"SECURITY.md":              "# Security\n",
		"docs/CODE_OF_CONDUCT.md":  "# Code of conduct\n",
		"README.md":                "# Project\n\n## Install\n\n### A sub-heading\n\n## A & B\n\n## Party :tada:\n\n## Quote \"q\" <b>x</b>\n\n## The `abcd` tool\n",
	})
	got := deriveReadmeParts(root, readmeSettings{Parts: readmePartOrder, Visibility: "public"})
	wantInline(t, derivedPart(t, got, "release-badge"), inlineRelease)
	wantInline(t, derivedPart(t, got, "build-badge"), inlineBuild)
	wantInline(t, derivedPart(t, got, "abcd-badge"), inlineAbcd)
	wantInline(t, derivedPart(t, got, "licence"), `<a href="LICENSE">MIT licence</a>`)
	wantInline(t, derivedPart(t, got, "links"), `<a href="CHANGELOG.md">Changelog</a> · <a href=".github/CONTRIBUTING.md">Contributing</a> · <a href="SECURITY.md">Security</a> · <a href="docs/CODE_OF_CONDUCT.md">Code of conduct</a>`)
	wantInline(t, derivedPart(t, got, "contents"), `<a href="#install">Install</a> · <a href="#a--b">A &amp; B</a> · Party :tada: · <a href="#quote-q-x">Quote &#34;q&#34; x</a> · <a href="#the-abcd-tool">The abcd tool</a>`)

	// The block forms are what they were.
	wantBody(t, derivedPart(t, got, "licence"), "Licensed under the MIT License; see [LICENSE](LICENSE).")

	t.Run("the licence unnamed, or named by its identifier", func(t *testing.T) {
		for _, tc := range []struct{ licence, want string }{
			{"Some licence prose.\n", `<a href="LICENSE">Licence</a>`},
			{"SPDX-License-Identifier: GPL-3.0-or-later\n", `<a href="LICENSE">GPL-3.0-or-later licence</a>`},
			{"Apache License\nVersion 2.0, January 2004\n", `<a href="LICENSE">Apache-2.0 licence</a>`},
		} {
			root := readmeRepo(t, "", map[string]string{"LICENSE": tc.licence})
			wantInline(t, derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"licence"}}), "licence"), tc.want)
		}
	})
}

// inlineWant is a want whose two forms tell apart which one was written.
func inlineWant(name string) readmePartWant {
	return readmePartWant{Name: name, Body: "BLOCK " + name, Inline: "<i>" + name + "</i>"}
}

func blockPart(name string) string  { return readmePart(name, "BLOCK "+name) }
func inlinePart(name string) string { return readmePart(name, "<i>"+name+"</i>") }
func emptyPart(name string) string  { return readmePart(name, "") }

// TestReadmeInlinePlacement holds which form a part is written in and where an
// absent one goes: a part whose opening mark continues an open block-level
// HTML block is inline, anywhere else block; an absent part beside an inline
// one joins its block with no blank line; with no part present an absent part
// goes after an HTML <h1> as after a Markdown title; and removing an inline
// part leaves no blank line behind.
func TestReadmeInlinePlacement(t *testing.T) {
	rb, bb, lic := inlineWant("release-badge"), inlineWant("build-badge"), inlineWant("licence")
	cases := []struct {
		name, content string
		want          []readmePartWant
		result        string
	}{
		{"a mark continuing a <p> block is written inline",
			"<div align=\"center\">\n\n<p>\n" + emptyPart("release-badge") + "\n<img src=\"x\" alt=\"x\">\n</p>\n\n</div>\n",
			[]readmePartWant{rb},
			"<div align=\"center\">\n\n<p>\n" + inlinePart("release-badge") + "\n<img src=\"x\" alt=\"x\">\n</p>\n\n</div>\n"},
		{"a mark on its own is written as a block",
			"# Project\n\n" + emptyPart("release-badge") + "\n\nprose\n",
			[]readmePartWant{rb},
			"# Project\n\n" + blockPart("release-badge") + "\n\nprose\n"},
		{"a blank line ends the HTML block, so a mark after it is a block",
			"<div>\n\n" + emptyPart("licence") + "\n\n</div>\n",
			[]readmePartWant{lic},
			"<div>\n\n" + blockPart("licence") + "\n\n</div>\n"},
		{"a lone tag outside a paragraph opens a block too",
			"<span>\n" + emptyPart("licence") + "\n</span>\n",
			[]readmePartWant{lic},
			"<span>\n" + inlinePart("licence") + "\n</span>\n"},
		{"a lone tag cannot interrupt a paragraph",
			"text\n<span>\n" + emptyPart("licence") + "\n</span>\n",
			[]readmePartWant{lic},
			"text\n<span>\n" + blockPart("licence") + "\n</span>\n"},
		{"a block-form part moved into an HTML block is rewritten inline",
			"<p>\n" + blockPart("licence") + "\n</p>\n",
			[]readmePartWant{lic},
			"<p>\n" + inlinePart("licence") + "\n</p>\n"},
		{"an absent part after an inline one joins its block",
			"<p>\n" + inlinePart("release-badge") + "\n<br />\n</p>\n",
			[]readmePartWant{rb, bb},
			"<p>\n" + inlinePart("release-badge") + "\n" + inlinePart("build-badge") + "\n<br />\n</p>\n"},
		{"an absent part before an inline one joins its block",
			"<p>\n<img a>\n" + inlinePart("build-badge") + "\n</p>\n",
			[]readmePartWant{rb, bb},
			"<p>\n<img a>\n" + inlinePart("release-badge") + "\n" + inlinePart("build-badge") + "\n</p>\n"},
		{"with no part present, after an HTML h1",
			"<div align=\"center\">\n\n  <h1>Title</h1>\n\n  <p>x</p>\n\n</div>\n",
			[]readmePartWant{lic},
			"<div align=\"center\">\n\n  <h1>Title</h1>\n\n" + blockPart("licence") + "\n\n  <p>x</p>\n\n</div>\n"},
		{"after an HTML h1 that spans lines, past its closing tag",
			"<h1>\nTitle\n</h1>\nprose\n",
			[]readmePartWant{lic},
			"<h1>\nTitle\n</h1>\n\n" + blockPart("licence") + "\n\nprose\n"},
		{"an HTML h1 in a code block is not the title",
			"```\n<h1>x</h1>\n```\n\n# Real\n\nprose\n",
			[]readmePartWant{lic},
			"```\n<h1>x</h1>\n```\n\n# Real\n\n" + blockPart("licence") + "\n\nprose\n"},
		{"the first title of either kind",
			"<h1>A</h1>\n\n# B\n",
			[]readmePartWant{lic},
			"<h1>A</h1>\n\n" + blockPart("licence") + "\n\n# B\n"},
		{"removing an inline part leaves no blank line",
			"<p>\n<img a>\n" + inlinePart("release-badge") + "\n<img b>\n</p>\n",
			nil,
			"<p>\n<img a>\n<img b>\n</p>\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeReadme(t, c.content)
			if _, err := syncReadmeParts(path, c.want); err != nil {
				t.Fatal(err)
			}
			got := readReadme(t, path)
			if got != c.result {
				t.Fatalf("README:\n%s\nwant\n%s", got, c.result)
			}
			// A second run finds every part current and writes nothing.
			res, err := syncReadmeParts(path, c.want)
			if err != nil || res.Wrote {
				t.Errorf("second run: wrote=%v err=%v (notes %q)", res.Wrote, err, notes(res))
			}
		})
	}
}

// TestReadmeLayoutCHeaderFromParts: a centred header written in HTML, with the
// parts' marks placed in its badge rows and its link line, is filled in place
// by setup's derivation and write, every row kept in its order and no blank
// line added inside an HTML block; a second run writes nothing.
func TestReadmeLayoutCHeaderFromParts(t *testing.T) {
	header := func(part func(string) string) string {
		return "<div align=\"center\">\n\n" +
			"  <h1>Project</h1>\n\n" +
			"  <p>\n" +
			part("release-badge") + "\n" +
			"  <img src=\"https://img.shields.io/badge/Go-1.26-00ADD8\" alt=\"Go 1.26\">\n" +
			"  <br />\n" +
			part("build-badge") + "\n" +
			"  <br />\n" +
			part("abcd-badge") + "\n" +
			"  </p>\n\n" +
			"  <p>\n" +
			"    <a href=\"#install\">Install</a> ·\n" +
			part("links") + "\n" +
			"    ·\n" +
			part("licence") + "\n" +
			"  </p>\n\n" +
			"</div>\n\n## Install\n\ntext\n"
	}
	inline := map[string]string{
		"release-badge": inlineRelease,
		"build-badge":   inlineBuild,
		"abcd-badge":    inlineAbcd,
		"links":         `<a href="CHANGELOG.md">Changelog</a> · <a href=".github/CONTRIBUTING.md">Contributing</a> · <a href=".github/SECURITY.md">Security</a>`,
		"licence":       `<a href="LICENSE">MIT licence</a>`,
	}
	root := readmeRepo(t, "https://github.com/owner/repo.git", map[string]string{
		".github/workflows/ci.yml": "on: push\n",
		"LICENSE":                  "MIT License\n\nCopyright\n",
		"CHANGELOG.md":             "# Changelog\n",
		".github/CONTRIBUTING.md":  "# Contributing\n",
		".github/SECURITY.md":      "# Security\n",
		"README.md":                header(emptyPart),
	})
	parts := []string{"release-badge", "build-badge", "abcd-badge", "licence", "links"}
	path := root + "/README.md"
	sync := func() readmePartsResult {
		derived, want := readmeDerive(root, parts, "public")
		for _, d := range derived {
			if d.Reason != "" {
				t.Fatalf("%s omitted: %s", d.Name, d.Reason)
			}
		}
		res, err := syncReadmeParts(path, want)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	sync()
	want := header(func(name string) string { return readmePart(name, inline[name]) })
	if got := readReadme(t, path); got != want {
		t.Fatalf("README:\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(want[:strings.Index(want, "</div>")], "\n\n\n") {
		t.Fatal("the fixture itself holds a double blank line")
	}
	if res := sync(); res.Wrote {
		t.Errorf("the second run wrote: %q", notes(res))
	}
}
