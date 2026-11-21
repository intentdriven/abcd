# site-src

Build inputs for abcdev.app. Composition comes from `.abcd/site.json`; the text
comes from the repository. This directory holds:

- `ui.json` — the interface-string allowlist: every word the generator may add.
  It is decoded against a closed struct with unknown fields refused, so a word
  added here that no field reads fails the build.
- `site.css` — the stylesheet: light is the base and dark redefines tokens only.
  The landing page follows the reader's system preference; a page that sets an
  explicit `data-theme` attribute is honoured too, though no page sets one yet —
  those blocks are carried deferred for a future theme control.
- `site.js` — the landing page's only script: install tabs, copy buttons, and
  nothing else. It adds no words of its own — every string it shows is read back
  from the markup, where the build put it from `ui.json`. No analytics, no
  trackers, no network requests.
- `install.sh.tmpl` — the committed source of `/install.sh`.
- `redirects` — the `_redirects` source.
- `headers` — the `_headers` source. Every file the build emits matches a block
  in it: a document carries a content policy, `nosniff` and a referrer policy;
  an asset carries the two of those that mean anything for a non-document.
- `audit/` — CI-only tooling, never part of the site. `overflow-audit.js` drives
  a headless browser over every route at four viewport widths and fails on
  horizontal overflow; it runs in the optional `site-screenshots` workflow and
  in no build or deploy path. `routes.json` names the routes it loads for each
  build it measures: the documentation site's own pages on the composed build,
  and the record's pages on a build of the project route. The build reaches
  it by no path: `copiedSources` in `internal/core/site/build.go` is an
  explicit five-file allowlist and nothing walks this directory, so files
  added here cannot reach the output.

`abcd site build` copies `site.css`, `site.js`, `record.js`, `redirects` and
`headers` into the output tree (the last two under their served names), renders
`install.sh.tmpl` to `install.sh` with one build-stamp comment added, and reads
`ui.json`. A site narrowed to its documentation (`record.elsewhere.narrow` in
`.abcd/site.json`) copies no `record.js`, and its `_headers` keeps no block for
a record path.
