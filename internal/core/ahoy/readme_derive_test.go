package ahoy

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// readmeRepo is a hermetic git repository with origin set (none when empty)
// and the files written at their repo-relative paths.
func readmeRepo(t *testing.T, origin string, files map[string]string) string {
	t.Helper()
	setupHermetic(t)
	r := gittest.NewRepo(t)
	if origin != "" {
		r.Git("remote", "add", "origin", origin)
	}
	for rel, body := range files {
		writeFixture(t, r.Root(), rel, body, 0o644)
	}
	return r.Root()
}

// derivedPart is the derivation of the part named name, failing the test when
// the derivation did not report it.
func derivedPart(t *testing.T, got []readmeDerived, name string) readmeDerived {
	t.Helper()
	for _, d := range got {
		if d.Name == name {
			return d
		}
	}
	t.Fatalf("no derivation reported for %s; got %+v", name, got)
	return readmeDerived{}
}

// wantBody asserts the part was derived with exactly body.
func wantBody(t *testing.T, d readmeDerived, body string) {
	t.Helper()
	if d.Reason != "" || d.Body != body {
		t.Errorf("%s:\n got body %q (reason %q)\nwant body %q", d.Name, d.Body, d.Reason, body)
	}
}

// wantOmitted asserts the part was omitted, its reason naming every piece.
func wantOmitted(t *testing.T, d readmeDerived, pieces ...string) {
	t.Helper()
	if d.Body != "" || d.Reason == "" {
		t.Errorf("%s: derived %q, want it omitted naming %q", d.Name, d.Body, pieces)
		return
	}
	for _, p := range pieces {
		if !strings.Contains(d.Reason, p) {
			t.Errorf("%s: reason %q does not name %q", d.Name, d.Reason, p)
		}
	}
}

// TestLicencePartNamesOnlyFromIdentifierOrFirstLine: the licence is named
// only from an SPDX identifier line in the file's first twenty lines or from a
// forge-standard first line in the one table; any other LICENSE is linked
// without being named, and no LICENSE at the root means no licence part.
func TestLicencePartNamesOnlyFromIdentifierOrFirstLine(t *testing.T) {
	const unnamed = "See [LICENSE](LICENSE) for the licence terms."
	padding := strings.Repeat("Some licence prose.\n", 20)
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string // "" means omitted
	}{
		{"MIT first line", map[string]string{"LICENSE": "MIT License\n\nCopyright (c) 2026 Example\n\nPermission is hereby granted, free of charge.\n"},
			"Licensed under the MIT License; see [LICENSE](LICENSE)."},
		{"MIT first line with CRLF and a byte-order mark", map[string]string{"LICENSE": "\ufeffMIT License\r\n\r\nCopyright\r\n"},
			"Licensed under the MIT License; see [LICENSE](LICENSE)."},
		{"Apache's indented two-line heading", map[string]string{"LICENSE": "\n                                 Apache License\n                           Version 2.0, January 2004\n                        http://www.apache.org/licenses/\n"},
			"Licensed under the Apache License 2.0; see [LICENSE](LICENSE)."},
		{"Apache License not followed by Version 2.0", map[string]string{"LICENSE": "Apache License\nVersion 1.1\n"}, unnamed},
		{"BSD 2-Clause", map[string]string{"LICENSE": "BSD 2-Clause License\n\nCopyright\n"},
			"Licensed under the BSD 2-Clause License; see [LICENSE](LICENSE)."},
		{"BSD 3-Clause", map[string]string{"LICENSE": "BSD 3-Clause License\n"},
			"Licensed under the BSD 3-Clause License; see [LICENSE](LICENSE)."},
		{"ISC in LICENSE.txt", map[string]string{"LICENSE.txt": "ISC License\n\nCopyright\n"},
			"Licensed under the ISC License; see [LICENSE.txt](LICENSE.txt)."},
		{"Mozilla", map[string]string{"LICENSE": "Mozilla Public License Version 2.0\n==================================\n"},
			"Licensed under the Mozilla Public License 2.0; see [LICENSE](LICENSE)."},
		{"identifier line in LICENSE.md", map[string]string{"LICENSE.md": "Copyright 2026 Example\n<!-- SPDX-License-Identifier: GPL-3.0-or-later -->\n\nThis program is free software.\n"},
			"Licensed under GPL-3.0-or-later; see [LICENSE.md](LICENSE.md)."},
		{"identifier expression", map[string]string{"LICENSE": "# SPDX-License-Identifier: MIT OR Apache-2.0\n"},
			"Licensed under MIT OR Apache-2.0; see [LICENSE](LICENSE)."},
		{"a known identifier takes the table's name", map[string]string{"LICENSE": "Copyright\nSPDX-License-Identifier: MIT\n"},
			"Licensed under the MIT License; see [LICENSE](LICENSE)."},
		{"the identifier line outranks the first line", map[string]string{"LICENSE": "MIT License\nSPDX-License-Identifier: Apache-2.0\n"},
			"Licensed under the Apache License 2.0; see [LICENSE](LICENSE)."},
		{"identifier past the twentieth line", map[string]string{"LICENSE": padding + "SPDX-License-Identifier: MIT\n"}, unnamed},
		{"malformed identifier is never echoed", map[string]string{"LICENSE": "SPDX-License-Identifier: MIT; see https://example.com\n"}, unnamed},
		{"GNU text alone is not named", map[string]string{"LICENSE": "                    GNU GENERAL PUBLIC LICENSE\n                       Version 3, 29 June 2007\n"}, unnamed},
		{"MIT text without the forge's first line", map[string]string{"LICENSE": "Copyright (c) 2026 Example\n\nPermission is hereby granted, free of charge.\n"}, unnamed},
		{"LICENSE outranks LICENSE.md", map[string]string{"LICENSE": "ISC License\n", "LICENSE.md": "MIT License\n"},
			"Licensed under the ISC License; see [LICENSE](LICENSE)."},
		{"no LICENSE at the root", map[string]string{"docs/LICENSE": "MIT License\n", "COPYING": "MIT License\n"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := readmeRepo(t, "", tc.files)
			d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"licence"}}), "licence")
			if tc.want == "" {
				wantOmitted(t, d, "no LICENSE file")
				return
			}
			wantBody(t, d, tc.want)
		})
	}
}

// TestLinksPartListsOnlyFilesThatExist: each of the four files is looked for
// at the root, then .github/, then docs/, by its exact name and only as a
// regular file; the first place found is linked, and none found omits the part.
func TestLinksPartListsOnlyFilesThatExist(t *testing.T) {
	root := readmeRepo(t, "", map[string]string{
		"CHANGELOG.md":              "# Changelog\n",
		"docs/CHANGELOG.md":         "# Older changelog\n", // the root one wins
		".github/CONTRIBUTING.md":   "# Contributing\n",
		"docs/CONTRIBUTING.md":      "# Also contributing\n", // .github/ wins over docs/
		"docs/SECURITY.md":          "# Security\n",
		"CODE_OF_CONDUCT.md/README": "a directory, not a file\n",
	})
	// A symlink is not a file the project has at that place.
	if err := os.Symlink("docs/SECURITY.md", filepath.Join(root, "SECURITY.md")); err != nil {
		t.Fatal(err)
	}
	d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"links"}}), "links")
	wantBody(t, d, "- [Changelog](CHANGELOG.md)\n- [Contributing guide](.github/CONTRIBUTING.md)\n- [Security policy](docs/SECURITY.md)")

	t.Run("names are exact", func(t *testing.T) {
		// A case-insensitive file system finds changelog.md as CHANGELOG.md, and
		// the forge would then serve a link to a file that is not there.
		root := readmeRepo(t, "", map[string]string{"changelog.md": "x\n", ".github/code_of_conduct.md": "x\n"})
		d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"links"}}), "links")
		wantOmitted(t, d, "none of CHANGELOG.md, CONTRIBUTING.md, SECURITY.md or CODE_OF_CONDUCT.md")
	})
	t.Run("none found", func(t *testing.T) {
		root := readmeRepo(t, "", nil)
		d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"links"}}), "links")
		wantOmitted(t, d, "none of CHANGELOG.md, CONTRIBUTING.md, SECURITY.md or CODE_OF_CONDUCT.md")
	})
}

// TestContentsPartListsLiveHeadings: the contents part lists the README's live
// level-two and level-three headings outside every part, anchored by the
// forge's slug rule, and is omitted when no level-two heading is live.
func TestContentsPartListsLiveHeadings(t *testing.T) {
	readme := strings.Join([]string{
		"---",
		"## a YAML comment, not a heading",
		"---",
		"# Project",
		"",
		readmePart("links", "## Inside a part"),
		"",
		"## Install",
		"",
		"```sh",
		"## a shell comment",
		"```",
		"",
		"### From source",
		"",
		"<!--",
		"## commented out",
		"-->",
		"",
		"## Usage & notes ##",
		"",
		"#### Too deep",
		"",
		"## Install",
		"",
		"## See [the docs](docs/guide.md)",
		"",
		"## `abcd` and_more",
		"",
		"##NoSpace is not a heading",
		"",
	}, "\n")
	root := readmeRepo(t, "", map[string]string{"README.md": readme})
	d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
	wantBody(t, d, strings.Join([]string{
		"- [Install](#install)",
		"  - [From source](#from-source)",
		"- [Usage & notes](#usage--notes)",
		"- [Install](#install-1)",
		"- [See the docs](#see-the-docs)",
		"- [`abcd` and_more](#abcd-and_more)",
	}, "\n"))

	// The forge numbers a repeated slug from the slug's own count and moves
	// past a number a heading already owns: the third heading here is
	// install-2, not a second install-1.
	t.Run("a numbered slug a heading already owns", func(t *testing.T) {
		root := readmeRepo(t, "", map[string]string{"README.md": "# P\n\n## Install\n\n## Install-1\n\n## Install\n\n## Install\n"})
		d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
		wantBody(t, d, "- [Install](#install)\n- [Install-1](#install-1)\n- [Install](#install-2)\n- [Install](#install-3)")
	})

	// A setext heading is a heading: one underlined with hyphens is level two
	// and listed; one underlined with equals signs is level one, not listed,
	// and both are counted when the forge numbers a repeated anchor. A line
	// of hyphens after a blank line, a list item or indented code is a rule,
	// not an underline.
	t.Run("setext headings", func(t *testing.T) {
		readme := strings.Join([]string{
			"Project",
			"=======",
			"",
			"Install",
			"-------",
			"",
			"## Install",
			"",
			"Usage",
			"===",
			"",
			"## Usage",
			"",
			"A heading over",
			"  two lines",
			"---",
			"",
			"a paragraph",
			"",
			"---",
			"",
			"- a list item",
			"---",
			"",
			"    indented code",
			"---",
			"",
			"> a quotation",
			"---",
			"",
			readmePart("links", "Inside a part\n---"),
			"",
			"   Indented, still a heading   ",
			"   ---   ",
		}, "\n")
		root := readmeRepo(t, "", map[string]string{"README.md": readme})
		d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
		wantBody(t, d, strings.Join([]string{
			"- [Install](#install)",
			"- [Install](#install-1)",
			"- [Usage](#usage-1)",
			"- [A heading over two lines](#a-heading-overtwo-lines)",
			"- [Indented, still a heading](#indented-still-a-heading)",
		}, "\n"))

		only := readmeRepo(t, "", map[string]string{"README.md": "Project\n=======\n\nInstall\n-------\n"})
		wantBody(t, derivedPart(t, deriveReadmeParts(only, readmeSettings{Parts: []string{"contents"}}), "contents"), "- [Install](#install)")
	})

	// A bare carriage return ends a line, as it does on the forge's page: the
	// text after one is not part of the heading before it, and a heading
	// after one is a heading. CRLF endings read as one line ending.
	t.Run("a bare carriage return", func(t *testing.T) {
		readme := "# P\r\n\r\n## Install\rnot a heading\r\n\r\nintro\r## Usage\n"
		root := readmeRepo(t, "", map[string]string{"README.md": readme})
		d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
		wantBody(t, d, "- [Install](#install)\n- [Usage](#usage)")
	})

	// A README abcd cannot read is reported as unread, never as one with no
	// level-two heading: each of these holds one.
	unreadable := map[string]struct {
		lay  func(t *testing.T, root string)
		want string
	}{
		"a symlinked README": {func(t *testing.T, root string) {
			writeFixture(t, root, "real.md", "# P\n\n## Install\n", 0o644)
			if err := os.Symlink("real.md", filepath.Join(root, "README.md")); err != nil {
				t.Fatal(err)
			}
		}, "README.md is a symlink or not a regular file"},
		"a README that is a directory": {func(t *testing.T, root string) {
			writeFixture(t, root, "README.md/x.md", "## Install\n", 0o644)
		}, "README.md is a symlink or not a regular file"},
		"a README past the cap": {func(t *testing.T, root string) {
			writeFixture(t, root, "README.md", "# P\n\n## Install\n"+strings.Repeat("x", maxAhoyFileBytes), 0o644)
		}, "README.md is past abcd's " + strconv.Itoa(maxAhoyFileBytes) + "-byte cap"},
	}
	for name, tc := range unreadable {
		t.Run(name, func(t *testing.T) {
			root := readmeRepo(t, "", nil)
			tc.lay(t, root)
			d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
			wantOmitted(t, d, "contents omitted", tc.want)
			if strings.Contains(d.Reason, noLevelTwoHeading) {
				t.Errorf("reason %q says the README has no level-two heading; it was not read", d.Reason)
			}
		})
	}

	for name, readme := range map[string]string{
		"no README.md":                 "",
		"level-three headings only":    "# Project\n\n### Detail\n",
		"level-two heading in a fence": "# Project\n\n```\n## Install\n```\n",
		"level-two heading in a part":  readmePart("links", "## Install") + "\n",
	} {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{}
			if readme != "" {
				files["README.md"] = readme
			}
			root := readmeRepo(t, "", files)
			d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
			wantOmitted(t, d, "the README has no level-two heading")
		})
	}
}

// TestContentsAnchorsSlugTheRenderedText: the forge makes a heading's anchor
// from the text it renders, not from the markdown source
// (iss-2610111850237083). An HTML entity is decoded before the slug is made,
// so `## A &amp; B` links #a--b, as the forge has it; the list shows the
// entity as written, which renders the same. An emoji shortcode renders as
// an emoji only when the forge knows its name, which abcd cannot tell without
// the forge's list, so a heading carrying one is listed without a link, and
// so is every later heading whose anchor it could have taken or numbered
// (here `Rocket Launch` and `-Launch`, never the plain `Launch`), each named
// in a note.
func TestContentsAnchorsSlugTheRenderedText(t *testing.T) {
	readme := strings.Join([]string{
		"# P",
		"",
		"## A &amp; B",
		"",
		"## Caf&eacute; &#35;1",
		"",
		"## Not &notanentity; here",
		"",
		"## :rocket: Launch",
		"",
		"## Launch",
		"",
		"## Rocket Launch",
		"",
		"## -Launch",
		"",
		"## Usage",
		"",
	}, "\n")
	root := readmeRepo(t, "", map[string]string{"README.md": readme})
	d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
	wantBody(t, d, strings.Join([]string{
		"- [A &amp; B](#a--b)",
		"- [Caf&eacute; &#35;1](#café-1)",
		"- [Not &notanentity; here](#not-notanentity-here)",
		"- :rocket: Launch",
		"- [Launch](#launch)",
		"- Rocket Launch",
		"- -Launch",
		"- [Usage](#usage)",
	}, "\n"))
	notes := strings.Join(d.Notes, "\n")
	for _, line := range []string{"line 9", "line 13", "line 15"} {
		if !strings.Contains(notes, line) {
			t.Errorf("no note names the unlinked heading at %s; notes %q", line, d.Notes)
		}
	}
	if !strings.Contains(notes, "emoji shortcode") {
		t.Errorf("the notes do not say why a heading is unlinked: %q", d.Notes)
	}
}

// TestContentsStripsInlineHTMLFromHeadings: inline HTML in a heading renders
// as markup, not text (iss-2610111850405447), so its tags are carried neither
// into the list's text nor into the anchor. The space the tags leave before
// the text is the forge's too: `## <a name="install"></a> Install` anchors
// #-install. An autolink is not a tag, and a lone angle bracket is text.
func TestContentsStripsInlineHTMLFromHeadings(t *testing.T) {
	readme := strings.Join([]string{
		"# P",
		"",
		`## <a name="install"></a> Install`,
		"",
		"## Usage <sup>beta</sup>",
		"",
		"## Old<br/>New",
		"",
		"## a < b > c",
		"",
	}, "\n")
	root := readmeRepo(t, "", map[string]string{"README.md": readme})
	d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
	wantBody(t, d, strings.Join([]string{
		"- [Install](#-install)",
		"- [Usage beta](#usage-beta)",
		"- [OldNew](#oldnew)",
		"- [a < b > c](#a--b--c)",
	}, "\n"))
}

// TestBadgePartsNeedPublicAndForge: the release and build badges are derived
// only where the owner declared the project public and the origin is on
// github.com; the build badge also needs one workflow it can name. The
// managed-with-abcd badge reads nothing from the project and is never omitted.
func TestBadgePartsNeedPublicAndForge(t *testing.T) {
	all := []string{"release-badge", "build-badge", "abcd-badge"}
	const (
		release = "[![Latest release](https://img.shields.io/github/v/release/owner/repo)](https://github.com/owner/repo/releases/latest)"
		abcd    = "[![Managed with abcd](https://img.shields.io/badge/managed%20with-abcd-blue)](https://github.com/intentdriven/abcd)"
	)
	build := func(wf string) string {
		return "[![Build](https://github.com/owner/repo/actions/workflows/" + wf + "/badge.svg)](https://github.com/owner/repo/actions/workflows/" + wf + ")"
	}
	ci := map[string]string{".github/workflows/test.yml": "on: push\n"}

	t.Run("not declared public", func(t *testing.T) {
		for _, vis := range []string{"private", ""} {
			root := readmeRepo(t, "https://github.com/owner/repo.git", ci)
			got := deriveReadmeParts(root, readmeSettings{Parts: all, Visibility: vis})
			wantOmitted(t, derivedPart(t, got, "release-badge"), "not declared public")
			wantOmitted(t, derivedPart(t, got, "build-badge"), "not declared public")
			wantBody(t, derivedPart(t, got, "abcd-badge"), abcd)
		}
	})
	t.Run("public on github.com", func(t *testing.T) {
		for _, origin := range []string{
			"https://github.com/owner/repo.git",
			"https://github.com/owner/repo",
			"git@github.com:owner/repo.git",
			"ssh://git@github.com/owner/repo.git",
		} {
			root := readmeRepo(t, origin, ci)
			got := deriveReadmeParts(root, readmeSettings{Parts: all, Visibility: "public"})
			wantBody(t, derivedPart(t, got, "release-badge"), release)
			wantBody(t, derivedPart(t, got, "build-badge"), build("test.yml"))
			wantBody(t, derivedPart(t, got, "abcd-badge"), abcd)
		}
	})
	t.Run("no origin remote", func(t *testing.T) {
		root := readmeRepo(t, "", ci)
		got := deriveReadmeParts(root, readmeSettings{Parts: all, Visibility: "public"})
		wantOmitted(t, derivedPart(t, got, "release-badge"), "no origin remote")
		wantOmitted(t, derivedPart(t, got, "build-badge"), "no origin remote")
	})
	t.Run("origin not on github.com", func(t *testing.T) {
		root := readmeRepo(t, "git@gitlab.com:owner/repo.git", ci)
		got := deriveReadmeParts(root, readmeSettings{Parts: all, Visibility: "public"})
		wantOmitted(t, derivedPart(t, got, "release-badge"), "not github.com", "gitlab.com")
		wantOmitted(t, derivedPart(t, got, "build-badge"), "not github.com", "gitlab.com")
	})
	t.Run("which workflow", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			files   map[string]string
			want    string
			reasons []string
		}{
			{"ci.yml among several", map[string]string{".github/workflows/a.yml": "x", ".github/workflows/ci.yml": "x"}, build("ci.yml"), nil},
			{"ci.yaml among several", map[string]string{".github/workflows/a.yml": "x", ".github/workflows/ci.yaml": "x"}, build("ci.yaml"), nil},
			{"several, none ci", map[string]string{".github/workflows/a.yml": "x", ".github/workflows/b.yaml": "x"}, "", []string{"a.yml", "b.yaml", "ci.yml"}},
			{"no workflow", map[string]string{".github/workflows/README.md": "x"}, "", []string{"no workflow file in .github/workflows/"}},
			{"no workflows folder", nil, "", []string{"no workflow file in .github/workflows/"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				root := readmeRepo(t, "https://github.com/owner/repo.git", tc.files)
				d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"build-badge"}, Visibility: "public"}), "build-badge")
				if tc.want != "" {
					wantBody(t, d, tc.want)
					return
				}
				wantOmitted(t, d, tc.reasons...)
			})
		}
	})
	t.Run("owner or repository not a plain name", func(t *testing.T) {
		root := readmeRepo(t, "https://github.com/owner/re%22po.git", ci)
		d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"release-badge"}, Visibility: "public"}), "release-badge")
		wantOmitted(t, d, "not a plain")
		if strings.Contains(d.Reason, `re"po`) || strings.Contains(d.Reason, "re%22po") {
			t.Errorf("reason %q echoes the remote's path", d.Reason)
		}
	})
}

// TestBadgeAddressRefusesForeignQueryOrHost (R6): a badge address carrying a
// query key the forge does not use, or a host that is not the forge's, is
// refused naming the piece, and nothing is written.
func TestBadgeAddressRefusesForeignQueryOrHost(t *testing.T) {
	ci := map[string]string{".github/workflows/ci.yml": "on: push\n"}
	for _, tc := range []struct {
		origin, names string
	}{
		{"https://github.com/owner/repo.git?token=abc", "token"},
		{"https://example.com/owner/repo.git", "example.com"},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			root := readmeRepo(t, tc.origin, ci)
			got := deriveReadmeParts(root, readmeSettings{Parts: []string{"release-badge", "build-badge"}, Visibility: "public"})
			for _, name := range []string{"release-badge", "build-badge"} {
				d := derivedPart(t, got, name)
				wantOmitted(t, d, name+" refused", tc.names)
				if strings.Contains(d.Reason, "abc") {
					t.Errorf("%s: reason %q echoes the query's value", name, d.Reason)
				}
			}
			if _, err := os.Lstat(filepath.Join(root, "README.md")); !os.IsNotExist(err) {
				t.Errorf("README.md was written (Lstat: %v)", err)
			}
		})
	}

	const wf = "https://github.com/o/r/actions/workflows/ci.yml/badge.svg"
	for _, tc := range []struct {
		part, role, addr string
		refusal          string // "" means allowed; else a piece the refusal names
	}{
		{"build-badge", readmeAddrImage, wf, ""},
		{"build-badge", readmeAddrImage, wf + "?branch=main", ""},
		{"build-badge", readmeAddrImage, wf + "?event=push", ""},
		{"build-badge", readmeAddrImage, wf + "?branch=main&event=push", ""},
		{"build-badge", readmeAddrLink, "https://github.com/o/r/actions/workflows/ci.yml", ""},
		{"release-badge", readmeAddrImage, "https://img.shields.io/github/v/release/o/r", ""},
		{"release-badge", readmeAddrLink, "https://github.com/o/r/releases/latest", ""},
		{"abcd-badge", readmeAddrImage, "https://img.shields.io/badge/managed%20with-abcd-blue", ""},
		{"abcd-badge", readmeAddrLink, "https://github.com/intentdriven/abcd", ""},

		{"build-badge", readmeAddrImage, wf + "?token=abc", "query key the forge does not use: token"},
		{"build-badge", readmeAddrImage, wf + "?branch=main&token=abc", "token"},
		{"build-badge", readmeAddrLink, "https://github.com/o/r?branch=main", "query key the forge does not use: branch"},
		{"release-badge", readmeAddrImage, "https://img.shields.io/github/v/release/o/r?branch=main", "branch"},
		{"build-badge", readmeAddrImage, "https://img.shields.io/x", "host is not github.com: img.shields.io"},
		{"release-badge", readmeAddrImage, "https://github.com/o/r/badge.svg", "host is not img.shields.io: github.com"},
		{"release-badge", readmeAddrLink, "https://github.com.example.net/o/r", "host is not github.com: github.com.example.net"},
		{"build-badge", readmeAddrLink, "https://github.com:8443/o/r", "host is not github.com: github.com:8443"},
		{"build-badge", readmeAddrLink, "https://user@github.com/o/r", "credentials"},
		{"build-badge", readmeAddrLink, "https://github.com/o/r#readme", "fragment"},
		{"build-badge", readmeAddrLink, "http://github.com/o/r", "scheme is not https: http"},
		{"build-badge", readmeAddrLink, "javascript:alert(1)", "scheme is not https: javascript"},
		{"build-badge", readmeAddrLink, "ftp://github.com/o/r", "scheme is not https: ftp"},
		{"build-badge", readmeAddrLink, "data:text/html,x", "scheme is not https: data"},
		{"build-badge", readmeAddrLink, "//github.com/o/r", "scheme is not https"},
		{"build-badge", readmeAddrLink, "https://github.com/o/r\n", "not an address"},
		{"licence", readmeAddrLink, "https://github.com/o/r", "not a badge part"},
	} {
		err := checkBadgeAddress(tc.part, readmeAddr{Role: tc.role, URL: tc.addr})
		switch {
		case tc.refusal == "" && err != nil:
			t.Errorf("%s %s %q refused: %v", tc.part, tc.role, tc.addr, err)
		case tc.refusal != "" && err == nil:
			t.Errorf("%s %s %q allowed, want refused naming %q", tc.part, tc.role, tc.addr, tc.refusal)
		case tc.refusal != "" && !strings.Contains(err.Error(), tc.refusal):
			t.Errorf("%s %s %q refused as %q, want it to name %q", tc.part, tc.role, tc.addr, err, tc.refusal)
		case tc.refusal != "" && !strings.HasPrefix(err.Error(), tc.part+" refused: "):
			t.Errorf("refusal %q does not lead with %q", err, tc.part+" refused: ")
		}
	}
}

// TestReadmePartRefusedByOutboundScrub: every derived body passes through the
// outbound scrub; any finding refuses the part naming the finding's rule, and a
// scanner configuration that refuses to load refuses every part with its reason.
func TestReadmePartRefusedByOutboundScrub(t *testing.T) {
	token := "ghp_" + testsecret.Synthetic(7, 36)
	root := readmeRepo(t, "", map[string]string{
		"LICENSE":   "MIT License\n",
		"README.md": "# Project\n\n## Use " + token + " to sign in\n",
	})
	got := deriveReadmeParts(root, readmeSettings{Parts: []string{"licence", "contents"}})
	wantBody(t, derivedPart(t, got, "licence"), "Licensed under the MIT License; see [LICENSE](LICENSE).")
	d := derivedPart(t, got, "contents")
	wantOmitted(t, d, "contents refused", "outbound scrub", "token:github_pat")
	if strings.Contains(d.Reason, token) {
		t.Errorf("reason %q carries the token it refused", d.Reason)
	}

	t.Run("a scanner configuration that refuses to load", func(t *testing.T) {
		root := readmeRepo(t, "", map[string]string{
			"LICENSE":               "MIT License\n",
			"CHANGELOG.md":          "x\n",
			".abcd/config/pii.json": "{ not json",
		})
		got := deriveReadmeParts(root, readmeSettings{Parts: []string{"licence", "links"}})
		for _, name := range []string{"licence", "links"} {
			wantOmitted(t, derivedPart(t, got, name), name+" refused", "outbound scrub", "degraded scanner config")
		}
	})

	// The project's own pii.json names its patterns' kinds, so a kind is the
	// repository's text: it reaches the reason with no control character in
	// it, as the reason's other untrusted pieces do.
	t.Run("a hostile pattern kind", func(t *testing.T) {
		kind := "house:\x1b[31mKIND\x1b[0m\nFORGED part written"
		cfg, err := json.Marshal(map[string]any{"patterns": map[string]any{
			"house_word": map[string]any{"regex": `\bZebrafish\b`, "kind": kind, "severity": "hard_fail"},
		}})
		if err != nil {
			t.Fatal(err)
		}
		root := readmeRepo(t, "", map[string]string{
			"README.md":             "# Project\n\n## Zebrafish\n",
			".abcd/config/pii.json": string(cfg),
		})
		d := derivedPart(t, deriveReadmeParts(root, readmeSettings{Parts: []string{"contents"}}), "contents")
		wantOmitted(t, d, "contents refused", "outbound scrub", "house:?[31mKIND?[0m?FORGED part written")
		wantTerminalSafe(t, d)
	})
}

// wantTerminalSafe asserts nothing a part reports carries a control character.
func wantTerminalSafe(t *testing.T, d readmeDerived) {
	t.Helper()
	for _, s := range append([]string{d.Reason}, d.Notes...) {
		if strings.IndexFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
			t.Errorf("%s: %q carries a control character", d.Name, s)
		}
	}
}

// TestReadmePartRefusedByDocsLint: where the project keeps .abcd/docs-lint.json,
// its banned tokens run over every derived body; a blocker refuses the part
// naming the token's id, a warning is noted and the part kept, a configuration
// that does not load refuses every part, and a project with none skips it.
func TestReadmePartRefusedByDocsLint(t *testing.T) {
	quote := func(s string) string {
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	ban := func(id, pattern, severity string) string {
		return `{"id": ` + quote(id) + `, "pattern": ` + quote(pattern) + `, "severity": ` + quote(severity) +
			`, "successor": "other words", "allow_context": ["docs-lint: allow"], "message": "house style"}`
	}
	config := `{"roots": ["README.md"], "banned_tokens": [` +
		ban("house/guide", `(?i)\bguide\b`, "blocker") + ", " +
		ban("house/licensed", `\bLicensed\b`, "warn") + `]}`
	files := map[string]string{
		"LICENSE":                  "MIT License\n",
		"CONTRIBUTING.md":          "x\n",
		".abcd/docs-lint.json":     config,
		".github/workflows/ci.yml": "x",
	}
	root := readmeRepo(t, "", files)
	got := deriveReadmeParts(root, readmeSettings{Parts: []string{"licence", "links", "abcd-badge"}})
	wantOmitted(t, derivedPart(t, got, "links"), "links refused", "docs-lint", "house/guide")
	lic := derivedPart(t, got, "licence")
	wantBody(t, lic, "Licensed under the MIT License; see [LICENSE](LICENSE).")
	if len(lic.Notes) != 1 || !strings.Contains(lic.Notes[0], "house/licensed") {
		t.Errorf("licence notes = %q, want one naming the warning house/licensed", lic.Notes)
	}
	wantBody(t, derivedPart(t, got, "abcd-badge"), "[![Managed with abcd](https://img.shields.io/badge/managed%20with-abcd-blue)](https://github.com/intentdriven/abcd)")

	t.Run("a configuration that does not load", func(t *testing.T) {
		broken := map[string]string{"LICENSE": "MIT License\n", "CONTRIBUTING.md": "x\n", ".abcd/docs-lint.json": "{"}
		got := deriveReadmeParts(readmeRepo(t, "", broken), readmeSettings{Parts: []string{"licence", "links"}})
		for _, name := range []string{"licence", "links"} {
			wantOmitted(t, derivedPart(t, got, name), name+" refused", ".abcd/docs-lint.json")
		}
	})
	// A banned token's id is the repository's text: a blocker's and a
	// warning's reach the reason and the notes with no control character.
	t.Run("hostile ids", func(t *testing.T) {
		hostile := `{"roots": ["README.md"], "banned_tokens": [` +
			ban("house/\x1b[31mRED\x1b[0m\nFORGED LINE", `(?i)\bguide\b`, "blocker") + ", " +
			ban("warn/\x1b[32mG\x1b[0m\nFORGED", `\bLicensed\b`, "warn") + `]}`
		files := map[string]string{"LICENSE": "MIT License\n", "CONTRIBUTING.md": "x\n", ".abcd/docs-lint.json": hostile}
		got := deriveReadmeParts(readmeRepo(t, "", files), readmeSettings{Parts: []string{"licence", "links"}})
		links := derivedPart(t, got, "links")
		wantOmitted(t, links, "links refused", "docs-lint", "house/?[31mRED?[0m?FORGED LINE")
		wantTerminalSafe(t, links)
		lic := derivedPart(t, got, "licence")
		if len(lic.Notes) != 1 || !strings.Contains(lic.Notes[0], "warn/?[32mG?[0m?FORGED") {
			t.Errorf("licence notes = %q, want one naming the sanitised warning id", lic.Notes)
		}
		wantTerminalSafe(t, lic)
	})
	t.Run("no configuration", func(t *testing.T) {
		plain := map[string]string{"LICENSE": "MIT License\n", "CONTRIBUTING.md": "x\n"}
		got := deriveReadmeParts(readmeRepo(t, "", plain), readmeSettings{Parts: []string{"links"}})
		wantBody(t, derivedPart(t, got, "links"), "- [Contributing guide](CONTRIBUTING.md)")
	})
}

// countingTransport counts every request http.DefaultTransport is handed.
type countingTransport struct{ n *atomic.Int64 }

func (c countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	c.n.Add(1)
	return nil, http.ErrNotSupported
}

// TestReadmeDerivationMakesNoNetworkCall (R3): every part is read from the
// project on this computer. With no LICENSE and the project not declared
// public, the licence and both forge badges are omitted with their reasons;
// in a public project on github.com every part is derived. Neither run hands
// a request to http.DefaultTransport or opens a connection through the proxy
// every HTTP client in the process would use, and the derivation's files
// import no network package.
func TestReadmeDerivationMakesNoNetworkCall(t *testing.T) {
	var requests, dials atomic.Int64
	saved := http.DefaultTransport
	http.DefaultTransport = countingTransport{&requests}
	t.Cleanup(func() { http.DefaultTransport = saved })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			dials.Add(1)
			c.Close()
		}
	}()
	proxy := "http://" + ln.Addr().String()

	root := readmeRepo(t, "https://github.com/owner/repo.git", map[string]string{".github/workflows/ci.yml": "x"})
	for _, k := range []string{"HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY", "https_proxy", "http_proxy", "all_proxy"} {
		t.Setenv(k, proxy)
	}
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")

	got := deriveReadmeParts(root, readmeSettings{Parts: []string{"licence", "release-badge", "build-badge"}, Visibility: "private"})
	wantOmitted(t, derivedPart(t, got, "licence"), "no LICENSE file")
	wantOmitted(t, derivedPart(t, got, "release-badge"), "not declared public")
	wantOmitted(t, derivedPart(t, got, "build-badge"), "not declared public")

	writeFixture(t, root, "LICENSE", "MIT License\n", 0o644)
	writeFixture(t, root, "CHANGELOG.md", "x\n", 0o644)
	writeFixture(t, root, "README.md", "# P\n\n## Install\n", 0o644)
	got = deriveReadmeParts(root, readmeSettings{Parts: readmePartOrder, Visibility: "public"})
	for _, d := range got {
		if d.Body == "" {
			t.Errorf("%s omitted in a public project on github.com: %s", d.Name, d.Reason)
		}
	}

	if n := requests.Load(); n != 0 {
		t.Errorf("derivation handed %d request(s) to http.DefaultTransport", n)
	}
	if n := dials.Load(); n != 0 {
		t.Errorf("derivation opened %d connection(s) through the proxy", n)
	}

	for _, file := range []string{"readme_derive.go", "readme_check.go"} {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if p == "net" || (strings.HasPrefix(p, "net/") && p != "net/url") || strings.Contains(p, "/adapter/gh") {
				t.Errorf("%s imports %s; derivation imports no network package (net/url only, as a parser)", file, p)
			}
		}
	}
}
