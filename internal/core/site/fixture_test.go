package site

// A miniature managed repository, built in-process, that exercises every path
// the landing page and record.json take: all four layouts, both asset kinds, a
// shipped intent with a MET audit, a dangling supersession, a two-release
// changelog, and a history with dated commits and attribution trailers.
//
// It is built rather than committed because half of what the build reads is git
// history, and a fixture whose history is a fixture is the only way to pin
// dates. Every commit carries a fixed date and a fixed author, so the whole
// export is a function of this file.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// onePixelPNG is a 1×1 PNG. The build reads its IHDR for the width and height it
// puts on the rendered <img>, so the bytes have to be a real PNG.
var onePixelPNG = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89,
	0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01,
	0x0d, 0x0a, 0x2d, 0xb4,
	0x00, 0x00, 0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

// fixture is the built repository.
type fixture struct {
	t    *testing.T
	repo *gittest.Repo
	root string
}

// fixtureRootDate dates the first commit; the fixture is deterministic, so two
// fixtures built with the same date are the SAME repository as far as the
// root-commit SHA can tell.
const fixtureRootDate = "2026-01-05T09:00:00+00:00"

// newFixture builds the whole repository and returns it at HEAD.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	return newFixtureRootedAt(t, fixtureRootDate)
}

// newFixtureRootedAt is newFixture with the first commit at rootDate, for a
// test that needs a second repository with a different identity.
func newFixtureRootedAt(t *testing.T, rootDate string) *fixture {
	t.Helper()
	r := gittest.NewRepo(t)
	f := &fixture{t: t, repo: r, root: r.Root()}
	f.writeSources()
	f.commitAt(rootDate, "feat: the record and the pages", "")
	f.shipTheIntent()
	f.writeChangelog()
	f.commitAt("2026-03-02T09:00:00+00:00", "docs: two releases", "None")
	f.mergeInARecord()
	// Shipped LAST, so a date-only derivation would pick it.
	f.shipTheStub()
	return f
}

// shipTheStub files the placeholder intent as shipped, after everything else.
func (f *fixture) shipTheStub() {
	f.t.Helper()
	const date = "2026-03-06T09:00:00+00:00"
	f.git(date, "mv",
		".abcd/development/intents/planned/itd-3-the-stubbed-one.md",
		".abcd/development/intents/shipped/itd-3-the-stubbed-one.md")
	f.git(date, "mv",
		".abcd/development/intents/planned/itd-4-the-promoted-stub.md",
		".abcd/development/intents/shipped/itd-4-the-promoted-stub.md")
	f.commitAt(date, "feat: ship the stubs", "None")
}

// mergeInARecord gives the fixture the history shape a linear one cannot have:
// a record whose content enters the trunk ONLY as part of a merge commit, the
// way a conflict resolution or a rename settled while merging does.
//
// It is here because a strictly linear fixture cannot see the defect it guards.
// `git log --name-status` prints no file lines for a merge, so such a record is
// invisible to the whole history walk and comes out with no dates at all —
// which then seats it at the centre of a chronological chart.
func (f *fixture) mergeInARecord() {
	f.t.Helper()
	const date = "2026-03-05T09:00:00+00:00"
	trunk := f.gitOut("rev-parse", "--abbrev-ref", "HEAD")

	f.git(date, "checkout", "-b", "side")
	f.write("docs/explanation/aside.md", "# An aside\n\nWritten on a branch.\n")
	f.commitAt(date, "docs: an aside on a branch", "None")

	f.git(date, "checkout", trunk)
	// Merge without committing, then add the record as part of the resolution,
	// so its only appearance in history is inside the merge commit itself.
	f.git(date, "merge", "--no-ff", "--no-commit", "side")
	f.write(".abcd/development/decisions/adrs/0003-settled-while-merging.md", `---
id: adr-3
slug: settled-while-merging
status: accepted
date: 2026-03-05
supersedes: null
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: []
---

# ADR-3: Settled while merging

This decision's file entered the trunk as part of a merge commit.
`)
	f.commitAt(date, "merge: side", "None")
}

// gitOut runs one git command and returns its trimmed stdout.
func (f *fixture) gitOut(args ...string) string {
	f.t.Helper()
	full := append([]string{"-C", f.root}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = f.repo.Env()
	out, err := cmd.Output()
	if err != nil {
		f.t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// Root is the repository root.
func (f *fixture) Root() string { return f.root }

// write puts one file in the tree.
func (f *fixture) write(rel, body string) {
	f.t.Helper()
	abs := filepath.Join(f.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) writeBytes(rel string, body []byte) {
	f.t.Helper()
	abs := filepath.Join(f.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(abs, body, 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// git runs one git command at a fixed date, so the history the build reads is
// the same on every run.
func (f *fixture) git(date string, args ...string) {
	f.t.Helper()
	env := append(append([]string{}, f.repo.Env()...),
		"GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	full := append([]string{
		"-C", f.root,
		"-c", "user.email=fixture@example.invalid",
		"-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// commitAt stages everything and commits it, optionally with an attribution
// trailer.
func (f *fixture) commitAt(date, subject, assistedBy string) {
	f.t.Helper()
	msg := subject
	if assistedBy != "" {
		msg += "\n\nAssisted-by: " + assistedBy
	}
	f.git(date, "add", "-A")
	f.git(date, "commit", "--allow-empty", "-m", msg)
}

// shipTheIntent moves an intent from planned/ into shipped/ in its own commit,
// which is how the record says a promise was delivered — and the only way the
// build can date it.
func (f *fixture) shipTheIntent() {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Join(f.root, ".abcd", "development", "intents", "shipped"), 0o755); err != nil {
		f.t.Fatal(err)
	}
	f.git("2026-02-10T09:00:00+00:00",
		"mv",
		".abcd/development/intents/planned/itd-2-the-shipped-one.md",
		".abcd/development/intents/shipped/itd-2-the-shipped-one.md")
	f.commitAt("2026-02-10T09:00:00+00:00", "feat: ship itd-2", "Assistant:assistant-model-1")
}

func (f *fixture) writeChangelog() {
	f.write("CHANGELOG.md", strings.Join([]string{
		"# Changelog",
		"",
		"## [Unreleased]",
		"",
		"## [0.2.0] - 2026-02-11",
		"",
		"### Added",
		"",
		"- The shipped one. (itd-2)",
		"",
		"## [0.1.0] - 2026-01-06",
		"",
		"### Added",
		"",
		"- The first cut.",
		"",
	}, "\n"))
}

// writeSources writes every committed input the build reads.
func (f *fixture) writeSources() {
	f.write(".claude-plugin/plugin.json", `{
  "name": "fixture",
  "description": "A fixture package.",
  "author": {"name": "Fixture Author"},
  "homepage": "https://example.invalid/fixture/repo",
  "repository": "https://example.invalid/fixture/repo",
  "license": "MIT"
}
`)

	f.write(".abcd/record-lint.json", `{
  "roots": [".abcd/development"],
  "banned_tokens": [],
  "rules": {
    "record_schema": {"enabled": true, "severity": "blocker", "record_stores": {
      "adr": ".abcd/development/decisions/adrs",
      "itd": ".abcd/development/intents",
      "spc": ".abcd/development/specs",
      "iss": ".abcd/work/issues"
    }}
  }
}
`)

	f.write(".abcd/site.json", `{
  "schema_version": 1,
  "purpose": "Composition manifest for the fixture.",
  "identity": {"file": ".abcd/development/brief/01-product/README.md", "heading": "Identity (canonical)"},
  "ui_strings": "site-src/ui.json",
  "home": {
    "hero": {"page": "docs/explanation/rationale.md", "figure": "first-image"},
    "chapters": [
      {"letter": "a", "page": "docs/explanation/roles.md", "layout": "cards-from-h2"},
      {"letter": "b", "page": "docs/explanation/artefacts.md", "layout": "lead-in-cards",
       "feature": {"kind": "shipped-intent-press-release", "pick": "newest-with-audit-MET",
                   "parts": ["press-release", "first-acceptance-criterion"]},
       "icons": "image-before-lead-in", "table_portraits": "roles"},
      {"letter": "c", "page": "docs/explanation/process.md", "layout": "prose",
       "figure": {"kind": "first-image", "labels-from-page": true}},
      {"letter": "d", "page": "docs/how-to/install.md", "layout": "install",
       "lead": "CLI", "after": "Afterwards", "left": ["Plugin"],
       "release": {"from": "CHANGELOG.md", "assets": "release.yml"},
       "tabs": "left-h2s, then lead-h3s and remaining-h2s as a labelled group"}
    ]
  },
  "record": {"issue_ledger": true},
  "docs": {"index": "docs/README.md", "cli": "docs/reference/cli/commands.md"},
  "record_pages": {"contributors": {"policy": {"file": "CONTRIBUTING.md", "heading": "Attribution", "part": "first-bullet"}}},
  "checks": {
    "every_text_node_has_source": true,
    "docs_lint_on_rendered_text": true,
    "command_snippets_pinned_to_cli_reference": true,
    "unresolved_reference_baseline": ".abcd/site-baseline.json"
  }
}
`)

	f.write("site-src/ui.json", `{
  "_purpose": "The complete list of words the fixture generator may add.",
  "nav_story": "Story",
  "nav_install": "Install",
  "nav_docs": "Docs",
  "nav_record": "Record",
  "nav_references": "References",
  "cta_roles": "Roles",
  "cta_install": "Install",
  "cta_docs": "Documentation",
  "record_link": "built in the open",
  "from_the_record": "from the record",
  "latest_release": "Latest release",
  "all_releases": "all releases",
  "copy": "copy",
  "copied": "copied",
  "search_docs": "Search the docs",
  "platform": {
    "darwin-arm64": "macOS · Apple silicon",
    "darwin-amd64": "macOS · Intel",
    "linux-arm64": "Linux · arm64",
    "linux-amd64": "Linux · amd64"
  },
  "tiles": {"releases": "releases", "adr": "decisions", "intent": "intents", "spec": "specs",
            "issue": "issues", "principle": "principles",
            "discipline": "disciplines", "commits": "commits"},
  "record_nav": {"dashboard": "Dashboard", "graph": "Relationships", "timeline": "Genealogy",
                 "foundations": "Foundation", "development": "Work", "health": "Health",
                 "glossary": "Glossary", "contributors": "Contributors"},
  "health": {"unresolved": "References a record the tree does not hold",
             "isolated": "Linked to nothing, and nothing links to it",
             "same_author": "Two author names, one contributor",
             "undeclared": "Authored commits declaring nothing",
             "multi_trailer": "Commits declaring more than one model",
             "not_a_defect": "Not a fault: it is why the trailer count and the commit count differ",
             "supersedes_lead": "Not a fault: the record on the left replaced the one on the right", "clean": "Nothing to report", "suggestion": "Suggested"},
  "panels": {"latest": "Latest decisions", "health": "Record health",
             "unresolved": "unresolved", "baseline": "baseline", "isolated": "isolated"},
  "graph": {"arrange": "Arrange", "by_date": "by date", "by_links": "by links", "filters": "Filters",
            "find": "Find a record", "mentions": "include body mentions", "browse_list": "Browse as a list",
            "zoom_in": "Zoom in", "zoom_out": "Zoom out", "reset_view": "Reset view",
            "full_screen": "Full screen", "exit_full_screen": "Exit full screen", "close": "Close",
            "back": "Back", "forward": "Forward", "history": "History of records looked at",
            "linked": "linked records", "no_links": "no typed cross-references"},
  "record": {"frontmatter": "Frontmatter", "inbound": "Referenced by", "outbound": "References",
             "mentions": "mentions", "not_in_tree": "not in the tree", "open_on_forge": "open on GitHub",
             "commit_history": "commit history"},
  "relations": {"blocked_by": "blocks", "supersedes": "superseded by",
                "implements": "implemented by", "builds_on": "built on by"},
  "contributors": {"authors": "Authors of record", "tools": "Bots and tools",
                   "assisted": "of authored commits disclose AI assistance",
                   "trailers": "Assisted-by trailers", "merges_excluded": "merge commits excluded",
                   "declared_none": "declared no assistance", "undeclared": "no declaration"},
  "more": "more",
  "standby": "Stand by…",
  "cli_group": "CLI",
  "matches_system": "matches this computer",
  "read_script": "read the script",
  "unreleased": "unreleased",
  "beta": "Beta"
}
`)

	f.write("site-src/redirects", "/old/  /docs/old/  301\n")
	// A miniature installer, structured like the one this repository ships: a
	// shebang, a header comment, definitions, and a single final call. What the
	// build does with it is copy it and stamp it, so the fixture only has to be
	// real shell.
	f.write("site-src/install.sh.tmpl", `#!/bin/sh
# fixture installer. The build renders this to /install.sh.

set -eu

main() {
	printf 'fixture install\n'
}

main "$@"
`)
	// The same block shape the repository ships, so the coverage check exercises
	// the mechanism rather than a stub. The policies are abbreviated; what is
	// under test is that every emitted file matches a block setting the headers
	// its kind can carry, and that the chart's route may fetch.
	f.write("site-src/headers", `# fixture headers
/install.sh
  Content-Type: text/plain; charset=utf-8
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/
  Content-Security-Policy: default-src 'none'; script-src 'self'; connect-src 'none'
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/record/*
  Content-Security-Policy: default-src 'none'; script-src 'self'; connect-src 'self'
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/contributors/*
  Content-Security-Policy: default-src 'none'; script-src 'self'; connect-src 'self'
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/references/*
  Content-Security-Policy: default-src 'none'; script-src 'self'; connect-src 'self'
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/record.json
  Content-Type: application/json; charset=utf-8
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/site.css
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/site.js
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/record.js
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin

/assets/*
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin
`)
	f.write("site-src/site.css", ":root{--ink:#000}\n")
	f.write("site-src/site.js", "/* fixture */\n")

	f.write(".abcd/site-baseline.json", `{
  "schema_version": 1,
  "unresolved_references": [
    {"from": "adr-2", "to": "adr-9"}
  ]
}
`)

	f.write(".abcd/development/brief/01-product/README.md", `# Product

## Identity (canonical)

- **Title:** fixture — A Fixture Product
- **Tagline:** A fixture tagline for a fixture product.
- **Pitch:** A fixture pitch, two clauses long, so the hero has something to
  carry in its aside.
`)

	// --- assets -------------------------------------------------------------
	f.writeBytes("docs/assets/img/intro.png", onePixelPNG)
	f.writeBytes("docs/assets/img/logo.png", onePixelPNG)
	f.writeBytes("docs/assets/img/role-thinker.png", onePixelPNG)
	f.writeBytes("docs/assets/img/role-facilitator.png", onePixelPNG)
	f.write("docs/assets/img/artefact-brief.svg",
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" width="10" height="10"><rect width="10" height="10" fill="var(--ink, #000)"/></svg>`)
	f.write("docs/assets/img/loop.svg",
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" width="10" height="10"><circle cx="5" cy="5" r="4" fill="var(--accent, #06c)"/></svg>`)

	// --- composed pages -----------------------------------------------------
	f.write("docs/explanation/rationale.md", `# Who this is for

A fixture lede with a [link](https://example.invalid/), some *emphasis*, some
**strength**, and a `+"`code span`"+`.

![A fixture cartoon](../assets/img/intro.png)

A second paragraph that the hero does not use.
`)

	f.write("docs/explanation/roles.md", `# Roles

The fixture has two roles, and this paragraph introduces them.

## Product thinker

![The product thinker](../assets/img/role-thinker.png)

The one who holds the *why*.

## Technical facilitator

![The facilitator](../assets/img/role-facilitator.png)

The one who translates it.
`)

	f.write("docs/explanation/artefacts.md", `# Artefacts

A fixture opening paragraph about the artefacts.

| | Product thinker | Facilitator |
|--|--|--|
| The brief | bring the substance | shape it |
| An intent | write the press release | sharpen the criteria |

![The brief](../assets/img/artefact-brief.svg)

The **brief** *(owned jointly)* answers what this is about.

**Intents** answer why each change matters.

A closing paragraph that is not a card.
`)

	f.write("docs/explanation/process.md", `# Process

![The loop](../assets/img/loop.svg)

**It starts with the brief.** A fixture paragraph opening the process.

## Capturing intents

One line, deliberately shaped like a capture:

`+"```sh"+`
fixture intent "<one-line idea>"
`+"```"+`

| | What it pins down |
|------|-------------------|
| **Given** | The starting state. |
| **When** | The trigger. |
| **Then** | The observable outcome. |

## Reading the verdict

The verdict is written back onto the intent.
`)

	f.write("docs/how-to/install.md", `# Install

You can use it as a [plugin](#plugin), a [binary](#cli), or by [building](#build) it.

## Plugin

Add the marketplace, then install the plugin:

`+"```text"+`
/plugin marketplace add fixture/repo
`+"```"+`

A second paragraph about the plugin.

A third paragraph about the plugin.

A fourth paragraph about the plugin.

A fifth paragraph, which folds away behind the disclosure.

## CLI

One line, checksum-verified, no administrator rights.

### macOS

`+"```sh"+`
sh -c 'echo fixture-darwin'
`+"```"+`

### Linux

`+"```sh"+`
sh -c 'echo fixture-linux'
`+"```"+`

### Windows

Not yet. This page will carry its command when it ships.

### Afterwards

Add this line to your shell profile:

`+"```sh"+`
export PATH="$HOME/.local/bin:$PATH"
`+"```"+`

A closing paragraph about what to do next.

## Build

`+"```bash"+`
go build ./cmd/fixture
`+"```"+`
`)

	f.write("docs/README.md", "# Documentation\n\nThe fixture's documentation index.\n")

	// --- the record ---------------------------------------------------------
	f.write(".abcd/development/decisions/adrs/0001-the-first-decision.md", `---
id: adr-1
slug: the-first-decision
status: accepted
date: 2026-01-02
supersedes: null
superseded_by: null
related_intents: [itd-1]
related_rfcs: []
related_adrs: [adr-2]
---

# ADR-1: The first decision

The first decision's body, which mentions iss-1 in prose.
`)

	f.write(".abcd/development/decisions/adrs/0002-the-second-decision.md", `---
id: adr-2
slug: the-second-decision
status: accepted
date: 2026-01-03
supersedes:
- adr-9
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-1]
---

# ADR-2: The second decision

The second decision's body.
`)

	// itd-1 is BLOCKED BY itd-2, so the pair exercises a directed relation whose
	// two ends are named by different words: itd-1 is "blocked by" itd-2, and
	// itd-2 "blocks" itd-1.
	f.write(".abcd/development/intents/drafts/itd-1-the-drafted-one.md", `---
id: itd-1
slug: the-drafted-one
spec_id: null
kind: standalone
builds_on: []
blocked_by: [itd-2]
severity: minor
impact: additive
---

# The Drafted One

## Press Release

> **A fixture user gets a fixture thing.** It has not been built.

## Acceptance Criteria

- Given nothing, when nothing, then nothing.
`)

	f.write(".abcd/development/intents/planned/itd-2-the-shipped-one.md", `---
id: itd-2
slug: the-shipped-one
spec_id: spc-1
kind: standalone
builds_on: []
severity: minor
impact: additive
---

# The Shipped One

## Press Release

> **A fixture user opens the page and sees the record.** They had been reading
> frontmatter on a forge; now one page says what the project decided and why.
> "I stopped guessing," said the fixture user.

## Acceptance Criteria

- Given the fixture repository, when the build runs, then the landing page
  renders every chapter from its own page.
- Given a second criterion, then it does not appear in the quote.

## Audit Notes

Acceptance rollup: MET 2 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0
`)

	// A shipped intent with a met audit whose press release is still the mint
	// placeholder. It is NEWER than itd-2, so a derivation that only sorts by
	// date picks it — and quotes a template at the reader as a testimonial.
	f.write(".abcd/development/intents/planned/itd-3-the-stubbed-one.md", `---
id: itd-3
slug: the-stubbed-one
spec_id: null
kind: standalone
builds_on: []
severity: minor
impact: additive
---

# The Stubbed One

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Acceptance Criteria

- Given a stub, when it is picked, then the page quotes a template.

## Audit Notes

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0
`)

	// The other minted placeholder: an intent promoted from a ledger issue.
	// Same template, different opening clause and a variable source id.
	f.write(".abcd/development/intents/planned/itd-4-the-promoted-stub.md", `---
id: itd-4
slug: the-promoted-stub
spec_id: null
kind: standalone
builds_on: []
severity: minor
impact: additive
---

# The Promoted Stub

## Press Release

> _Seeded by promotion from iss-1. Expand into the full press-release narrative before planning._

## Acceptance Criteria

- Given a promoted stub, when it is picked, then the page quotes a template.

## Audit Notes

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0
`)

	f.write(".abcd/development/specs/open/spc-1-the-spec.md", `---
id: spc-1
slug: the-spec
intent: itd-2
---
# The Spec

The spec's body, which mentions adr-1.
`)

	f.write(".abcd/work/issues/open/iss-1-a-fixture-issue.md", `---
schema_version: 1
id: "iss-1"
slug: "a-fixture-issue"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
---

a fixture issue with no heading, mentioning adr-2 in its text
`)

	// A discipline: an intent that states a rule rather than shipping a change.
	// It is what the foundations page lists beside the principles.
	f.write(".abcd/development/intents/disciplines/itd-5-a-fixture-discipline.md", `---
id: itd-5
slug: a-fixture-discipline
spec_id: null
kind: discipline
builds_on: []
severity: minor
impact: internal
---

# A Fixture Discipline

## Rule

Every fixture record carries a body, so the explorer has something to render.
`)

	// A superseded intent, so a set-aside lifecycle has a bubble and a pill.
	f.write(".abcd/development/intents/superseded/itd-6-the-superseded-one.md", `---
id: itd-6
slug: the-superseded-one
spec_id: null
kind: standalone
builds_on: [itd-1]
severity: minor
impact: additive
---

# The Superseded One

## Press Release

> **A fixture user reads a promise that was withdrawn.** The record keeps it.

## Acceptance Criteria

- Given a withdrawn promise, when the site builds, then it still has a page.
`)

	// The principle store: no frontmatter, the file name is the handle, and its
	// README is the store's index rather than one of its records.
	f.write(".abcd/development/principles/README.md", "# principles/\n\nThe store's own index.\n")
	// The links here are the two shapes a record writes at a relative path that
	// is NOT markdown: a directory that exists in the tree, and a file that does
	// not. The first has a forge view to point at; the second has nothing, and
	// the record's own text is what stands.
	f.write(".abcd/development/principles/one-fixture-principle.md", `# One fixture principle

**The rule.** A principle carries no frontmatter, so the lint scan cannot see
it and the site reads the store directly. It sits beside
[the decisions](../decisions) and is not [a missing thing](../nowhere).

**Why.** It exercises the frontmatter-free path, mentioning adr-1 in passing.
`)
	f.write(".abcd/development/principles/a-second-fixture-principle.md", `# A second fixture principle

**The rule.** Two entries make a deck rather than a card.
`)

	// The bibliography, and the acknowledgement file whose numbering it must
	// agree with. Both are invented for this fixture: no line here is copied
	// from the repository's own sources.
	f.write(".abcd/development/research/references.csl.json", fixtureCSL)

	// The attribution policy the contributors page quotes beside its numbers.
	// The section opens with its own preamble before the bullets, which is the
	// shape a policy section usually has: the manifest asks for the first
	// BULLET, and a selector that took the first block would publish the
	// lead-in and leave the rule off the page.
	f.write("CONTRIBUTING.md", `# Contributing

## Attribution

A fixture preamble about disclosure. The rules:

- **Human author of record.** The human contributor is the author of record and
  is responsible for all assisted output. A trailer is disclosure, never an
  authorship claim.
- A change no tool touched declares itself.
`)
	f.write("SECURITY.md", "# Security\n")
	f.write("ACKNOWLEDGEMENTS.md", fixtureAcknowledgements)
}

// fixtureCSL is a synthetic bibliography: two invented sources, one with a DOI
// and one with a URL, exercising both link forms and both title styles.
const fixtureCSL = `[
  {
    "id": "quill2019fixtures",
    "type": "article-journal",
    "author": [{"family": "Quill", "given": "Fenella"}],
    "title": "On the making of fixtures",
    "container-title": "Journal of Invented Sources",
    "volume": "4",
    "issue": "2",
    "page": "11-19",
    "issued": {"date-parts": [[2019]]},
    "DOI": "10.5555/fixture.2019.4"
  },
  {
    "id": "tamsin2021rendering",
    "type": "book",
    "author": [{"family": "Tamsin", "given": "Orrin"}, {"family": "Vole", "given": "Iris"}],
    "title": "Rendering the record",
    "publisher": "Invented Press",
    "publisher-place": "Nowhere",
    "issued": {"date-parts": [[2021]]},
    "URL": "https://example.invalid/rendering-the-record"
  }
]
`

// fixtureAcknowledgements numbers the same two sources in the same order, and
// credits two inspirations beside them.
const fixtureAcknowledgements = `# Acknowledgements

The fixture's credits, in the three parts the convention keeps.

## Inspirations

Ideas that shaped the fixture — not code it depends on.

- **A first invented inspiration** — the shape of the fixture's record.
- **A second invented inspiration** — the shape of its pages.

## References & sources

CSL-JSON: ` + "`.abcd/development/research/references.csl.json`" + `

1. Fenella Quill. 2019. On the making of fixtures. *Journal of Invented Sources*
   4, 2 (2019), 11-19. [doi:10.5555/fixture.2019.4](https://doi.org/10.5555/fixture.2019.4)
2. Orrin Tamsin and Iris Vole. 2021. *Rendering the record*. Invented Press,
   Nowhere. <https://example.invalid/rendering-the-record>
`
