---
id: adr-2610141949110653
slug: the-brief-session-s-browser-editor-ships
status: accepted
date: 2026-10-14
supersedes: null
superseded_by: null
related_intents: [itd-2610091016262335]
related_rfcs: []
related_adrs: [adr-47, adr-2610091016270711]
---

# ADR-2610141949110653: The brief session's browser editor ships as a vendored bundle the binary embeds

## Context

itd-2610091016262335 has a team editing its brief live in the browser. Live co-editing of raw Markdown needs a CRDT and an editor. The research note 2026-10-14-live-co-editing-to-one-pull-request.md picked Yjs v13 with CodeMirror 6 through y-codemirror.next: the shared text is the Markdown file byte for byte, and these are the most-used bindings. Writing either library from scratch is out of proportion to the intent.

adr-47 keeps a Node toolchain out of this repository. No `package.json`, lockfile or npm step runs in CI or in a build. The site build depends only on the binary. On 2026-10-14 the product thinker signed off on vendoring the two libraries. They also ruled that a technical choice grounded in state-of-the-art practice needs no further sign-off.

## Decision

We will ship the editor's third-party code as one prebuilt ES module, `internal/surface/session/web/vendor/collab.js`. It is committed beside its build entry module (`collab.entry.js`), a `PROVENANCE.json` and a `LICENSES.txt`, and the binary embeds it.

1. **The bundle is built outside the repository.** The build installs exact versions with install scripts off. It resolves only releases published at least two weeks before the build, and bundles with a pinned esbuild. The build is reproducible: two runs give the same bytes.
2. **Provenance is committed and checked.** `PROVENANCE.json` names every bundled package, its version, its npm integrity hash and its tarball URL. It also records the bundler and command, the entry module's SHA-256 and the bundle's SHA-256. A Go test fails when the committed bytes no longer match the recorded hashes.
3. **Licences travel with it.** `LICENSES.txt` carries every bundled package's licence, all MIT. Where a package ships no licence file, its source copyright header is used instead.
4. **A bump repeats the build in full.** It uses a new cut-off date and rewrites all four files in one change. Nothing in the repository runs npm.

## Alternatives Considered

- **A committed `package.json` and lockfile, built in CI.** Lockfile integrity is the stronger supply-chain control, but it brings the Node toolchain adr-47 rules out, and a CI build step on every release. Rejected for now. A later ADR may revisit adr-47 for this.
- **Loading the libraries from a CDN at runtime.** This adds no repository weight, but every session would then send a request to a third party abcd chose. The page's integrity would also depend on the CDN's availability. Rejected.
- **A plain textarea with hand-written merging.** This needs no third-party code, but merging concurrent edits correctly is the CRDT problem itself. Rejected.
- **The vendored, provenance-pinned bundle (chosen).** No toolchain enters the repository. The bytes are auditable and rebuildable. The cost is a committed 720 KB minified file and a manual bump.

## Consequences

- The binary grows by the bundle's size, about 720 KB, or 250 KB gzipped.
- A security fix in a bundled package reaches users only through a deliberate rebuild and a release. Dependabot cannot see the bundle, so the bump is a facilitator's task. The provenance file makes the bump auditable.
- Reviewers cannot read a minified bundle. They review the entry module, the provenance and the hash test instead, and anyone can rebuild the bundle and compare hashes.
