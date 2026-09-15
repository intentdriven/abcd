# internal/

abcd's Go implementation. The organising rule is a **transport-agnostic core**:
all behaviour lives in `core/` as functions that take a structured request and
return a structured result, and the front doors under `surface/` only marshal
those results for their transport. This is what lets the CLI, the markdown
plugin surface, and a future MCP server share one engine.

## Package map

- **`core/`** — the engine. One package per capability; no stdout, prompt text,
  or transport coupling. The capability set is the subdirectory set — browse
  [`core/`](core/) for it rather than a list kept here, which drifts the moment a
  phase lands (adr-5, derive don't store). The bullets below cover the packages
  whose boundary is not obvious from the name.
- **`core/changelog/`** — release derivation. Holds `impact`, the one product
  judgement a record declares, and derives the next SemVer from the records that
  entered the terminal folders since the anchor tag. It owns the enum so the
  lints that GATE the judgement (`core/lint`), the ledger reader that VALIDATES
  it (`core/capture`), and the derivation that CONSUMES it cannot drift apart.
- **`core/issueschema/`** — the issue record's required frontmatter properties,
  and nothing else. It is a leaf for the same reason `core/changelog` owns the
  impact enum: the ledger reader (`core/capture`) and the lint that gates the
  committed ledger (`core/lint`) must agree about what a well-formed record
  carries, and a record they disagree about is one that sits in the ledger unread
  by every surface. It is not inside `core/capture` because that package's own
  tests import `core/lint`, so a lint importing capture back is an import cycle.
- **`core/grounds/`** — the recorded-grounds vocabulary and its record form: the
  three values (`pursued`, `deferred`, `declined`), the `<token>: <text>` grammar,
  the substance floor that refuses a degenerate text, and the append-only
  `## Grounds` section both record families accumulate entries in. A leaf on the
  `core/issueschema` precedent, because three sites record grounds — the intent
  record writer (`core/intent`), the ledger writer (`core/capture`), and the
  committed-record gate that reads them back (`core/lint`) — and a vocabulary
  spelled three times is one the three can disagree about. It holds a FLOOR, not
  a judgement: whether a text names a conjecture rather than restating the
  decision is a review property, carried by the interview prompts on the plugin
  surface, and this package claims nothing about it.
- **`core/mdrecord/`** — the Markdown machinery a record BODY is read and written
  through: which lines are live markdown and which lie inside a fence or an HTML
  comment, where a section starts and stops, what a top-level bullet is, and
  where a trailing run of link-reference definitions ends. A leaf on the
  `core/grounds` precedent, because two record families carry the same
  constructs — the intent record's scope conditions and audit notes
  (`core/intent`), the issue record's grounds (`core/capture`) — and a body
  reader spelled twice is one the two can disagree about, which is how a bullet
  one writer appends becomes a bullet the other cannot find. It owns no heading's
  meaning: a caller supplies the pattern it is looking for.
- **`core/provenance/`** — the record's disclosure vocabulary: where an item came
  from (`origin`) and how its text was produced (`production_mode`), plus the one
  parser that reads and renders them. It is a leaf for the same reason
  `core/issueschema` and `core/changelog` are: the WRITERS that stamp the pair
  (`core/intent`, `core/spec`, `core/capture`) and the GATE that judges it
  (`core/lint`) must agree about what a legal value is, and two hand-kept copies
  of a closed set drift the moment one side gains a member. Both keys are
  single-line scalars — the reading pointer rides inside the `origin` value —
  because a nested mapping is invisible to `core/frontmatter`'s same-line scanner
  and would need a second record parser. It reads `core/issueschema` for the
  reading families' own spelling of their id prefixes and imports nothing else
  beyond the standard library; the arrow points one way, so the issue schema's
  allow-list carries the two key names as literals, pinned to this package's
  constants by a test here.
- **`core/surface/`** — the compatibility surface as DATA: the snapshot of every
  command, flag, and manifest entry a consumer binds to, and the diff that names
  what a release narrowed. It shares a word with the `surface/` front-door tier
  and nothing else — that tier is about transports, this package is about what
  those transports expose — so it is cobra-free like the rest of `core/`. The
  walk that reads the live command tree needs cobra and therefore lives in
  `surface/cli`, which hands its result in; the dependency never points back.
- **`core/site/`** — the website as a rendering of this repository (adr-47). It
  shares a word with nothing else here and is easy to mistake for a front door,
  so: it is a pure producer of files, transport-free like the rest of `core/`,
  and it writes only inside the output directory it is handed. It carries LAYOUT
  and no prose — every sentence it emits is a span of a repository file selected
  by `.abcd/site.json`, and the only words it may add are the closed allowlist in
  `site-src/ui.json`. The record graph it renders comes from `core/lint`'s own
  record scan (`LoadRecordGraph`), never a second parser; the dates come from one
  `git log` pass; the chart arrangements are computed here so the published pages
  need no layout engine and make no requests.
- **`core/cite/`** — the live half of the citation gate: the bounded fetcher and
  the refresh that writes the committed baseline `core/lint` then enforces with
  zero network. It is the only place abcd dials out on behalf of documentation,
  and it holds the seam a specialist link checker would later slot into — the
  baseline schema and the lint rules are the contract, the fetcher is a
  replaceable producer.
- **`core/lifeboat/`** — the brief↔lifeboat contract. `mapping.go` is the single
  source of truth for which brief section a lifeboat fills from which source
  tier, and it is rendered into the brief's `00-meta.md` with a test asserting
  the two agree. The table is a *hypothesis*: `abcd disembark probe` measures the
  same sections against real repositories in the same `grounded`/`partial`/`blank`
  vocabulary, and the evidence is expected to revise it (adr-35, itd-88).
- **`core/reading/`** — the cold-reading input assembler (itd-183, spc-61). The name is
  about what a READING may see, not about file I/O: it holds the positive include
  table that decides, at field granularity, what travels into a reading's context,
  the projection that takes named fields out of a record rather than copying the
  file, and the hashed manifest that lets a reader check the result instead of
  trusting a disclosure. The bundle it emits carries no repository path — the key is
  an ordinal and the kind is a material class — and only the manifest maps a key back
  to a path, which is what makes the blindness structural rather than instructed
  (brief invariant 15). Its deny is `core/launch`'s shape, measured from each include
  row's own source downward, so a record family added later is excluded by
  construction. Record enumeration is `core/lint`'s `LoadRecordGraph`, never a second
  parser. The package assembles input and never runs a reading.
- **`core/decide/`** — the decision record's WRITE side: `abcd decide "<title>"`
  mints an `adr-<stamp>` through `core/recordid` and lays the ADR skeleton under
  `.abcd/development/decisions/adrs/`. It is the last record family to reach that
  seam (the 2026-09-01 ruling, the turn adr-45 ruling 3 deferred), and it adopts it
  the way every family before it did — by holding a `recordid.Minter` and naming its
  family tag, never by carrying an allocator of its own. The presence check that
  redraws a taken draw calls `recordid.ADRFileID`, the SAME derivation the read-side
  resolver reads a filename with, so the mint cannot re-issue an id a citation
  already resolves. It owns the id, the date, the filename and the four sections,
  and states nothing: the decision is a human's to write, so the record lands
  `proposed` and the skeleton carries questions rather than answers.
- **`core/glossary/`** — the brief glossary's index, derived. The term files under
  `.abcd/development/brief/glossary/` are the source of truth for what terms exist
  and in which bounded context; this package renders the directory tree and the
  term index its README carries, and a test in the same package holds the
  committed README to that render. It is a generator and a gate, not a validator:
  the frontmatter shape is specified by the glossary README, and the rule that
  READS a term's frontmatter for enforcement is `GL002` in `core/lint`.
- **`core/guard/`** — the shell-hazard registry. Bundled hazard entries (id,
  command-position pattern over shell tokens, blocker/warn tier, safe successor,
  plain-language why) plus the deterministic allow/warn/block decision a harness
  hook calls before a command runs. Matching is token-aware and command-position
  only, so a hazard named inside a quoted argument never fires; every bundled
  entry ships known-bad and known-good fixtures, and the admission gate test
  holds each one to its own corpus (100% true-negative floor, at least 40%
  known-good). An allow means no entry matched, never that a command is safe: a
  hazard reached another way — a string handed to an interpreter (`eval`,
  `sh -c`), a launcher outside the `wrappers` set, a launcher inside it carrying
  a value-taking flag `wrapperValueFlags` does not name, an `ArgPaths` root
  segment the host serves under a prefix (a GitHub Enterprise Server `/api/v3/`
  mount; `pathOf` normalises a `scheme://host` URL, not a mount point), a
  backtick substitution, or a form no entry describes — is not seen. Each limit
  is stated here and in
  the `abcd guard check` reference doc, never left implicit. `Load` reads the
  working tree, so a `disabled: true` takes effect before it is reviewed; the
  front door compensates by making a disabled registry loud rather than silent.
  Fail-open-loud on a broken guard belongs to the hook shim (`hooks/hooks.json`)
  and the `abcd guard hook` adapter, not here.
- **`core/banlist/`** — the two banned-names stores (itd-74, spc-20). The public
  layer is managed IN the docs-lint `banned_tokens` family under a `names/` id
  prefix: one banned-token primitive, and the prefix is the ownership boundary a
  removal respects. The private layer is the gitignored per-machine store, whose
  FIRST line declares its format — keyed (`KEY<space-or-tab>PATTERN`) or legacy (one
  whole-line pattern per line) — which the committed `.githooks/pre-commit` guard
  parses identically. Three shared fixture corpora under `testdata/` are read by both
  parsers, so their agreement on the keyed format, the legacy format, and every
  unusable line class is checked rather than assumed. Enforcement is NOT here: the
  hook is the private layer's enforcement point and `core/lint` the public one, so
  this package owns the stores, their formats, and their editing discipline. It does
  VALIDATE against each layer's own engine, though — a private pattern goes to `grep`
  on stdin, a public one through the linter's compile path — because a pattern checked
  against a third engine is stored as healthy while it matches nothing. Redaction is
  structural — the exported private entry type carries no pattern field, so no
  rendering can leak a value — and edits are surgical (a line for the private store,
  byte surgery on the located array for the config), never a whole-file re-marshal.
  Scaffolding is NOT here either: `core/ahoy` writes the five artefacts (the two
  committed guard hooks, the `.gitattributes` line keeping them at LF, the public
  family, and the gitignored stub) into a repo it configures, importing this package
  for the paths and for the one canonical statement of the private layer's reach.
- **`adapter/scanner/`** — the secret/PII seam, and the first adapter seam to
  ship. Its native implementation is pure Go with no external dependency: a
  bundled secret-pattern set plus the probed machine identity, layered with an
  optional per-repo `.abcd/config/pii.json` override that may raise a severity
  but never lower one past the floor. It fails closed — an override that cannot
  be read, parsed, or compiled marks the scanner unavailable rather than letting
  a caller sanitise with a silently weakened pattern set — and redaction masks by
  byte span, so two secrets on one line cannot leak each other through the
  snippet they share. `core/launch` consumes it before a bundle is published;
  external scanners stay config-selected plug-ins behind the same seam.
- **`surface/cli/`** — the default front door: a Cobra command tree that calls
  `core` and formats results as text or `--json`. Holds no business logic.
- **`surface/mcp/`** *(later)* — an additive front door exposing the same core
  verbs as `mcp:abcd:*` tools. Added once a surface is worth exposing; no core
  rework required because the core is transport-agnostic.

## Planned seams (added when a phase consumes them, never as dead scaffolding)

Per the project rule "wired or it isn't done," the remaining pluggable adapter
seams are introduced by the phase that first uses them, not pre-emptively. The
list is gated rather than trusted: the `index_drift` record-lint rule holds every
path in the marked region to being absent from the tree, so a seam that ships
cannot go on being described here as planned.

<!-- index: planned-seams -->
- **`adapter/oracle/`** — LLM review. Native default = host-delegated (the host
  runs subagents); opt-in plug-ins: claude CLI, Anthropic API, MCP oracle.
- **`adapter/history/`** — transcripts. Native default = redacted local store;
  opt-in: private companion/remote, specstory cloud.
- **`adapter/spec/`** — spec/task store. Native minimal default; opt-in: the companion harness
  `ccpm` at the convention level.
- **`adapter/run/`** — autonomous run. Native thin loop fallback; host backends:
  Claude Workflows, the companion harness's agent loop.
- **`registry/`, `config/`** — declarative wiring of chosen adapters.
<!-- /index -->

The full rationale is in the plan and the design record under
`.abcd/development/`.
