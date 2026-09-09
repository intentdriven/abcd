# CLI command reference

This page is generated from the abcd command tree by `GenerateReference` in
`internal/surface/cli`. It is a derived artefact: do not edit it by hand. A
drift test regenerates the tree and fails the build whenever this page and the
tree disagree, so the reference can never silently go stale. Regenerate it with
`go generate ./internal/surface/cli`.

Every user-facing command is listed with its usage line, summary, and flags;
the operator-internal hook entrypoints are omitted.

## `abcd`

Agent-based configuration for development

**Usage:** `abcd [<record-id>] [flags]`

Agent-based configuration for development.

Bare `abcd` renders the read-only status board — what can I do. A single
positional matching a record id (`iss-N`, `itd-N`, `spc-N`, `adr-N`) instead
reports what that record is, where it lives, and the next move for its
lifecycle state — what is this. Both forms are strictly read-only; any other
positional is refused as an unknown command.

**Flags:**

```
      --json       emit machine-readable JSON
      --no-color   render the banner without color
```

### `abcd ahoy`

Install/update abcd in this repo; bare invocation is read-only status

**Usage:** `abcd ahoy`

#### `abcd ahoy doctor`

Report every gap read-only, including user-scope state (never mutates)

**Usage:** `abcd ahoy doctor`

#### `abcd ahoy dry-run`

Render the detection-result JSON envelope; never mutates

**Usage:** `abcd ahoy dry-run`

#### `abcd ahoy identity-check`

Exit non-zero if the git commit identity does not match .abcd/config/identity.json

**Usage:** `abcd ahoy identity-check`

#### `abcd ahoy install`

Install or update abcd in this repo (idempotent)

**Usage:** `abcd ahoy install [flags]`

**Flags:**

```
      --adopt                   adopt an unmanaged repo without prompting
      --allow-stale-binary      proceed even when the running binary is stale against its source tip or its vintage cannot be determined; the default is to refuse before any write and name the rebuild fix
      --attribution             opt this repo into the committed prepare-commit-msg prompt asking every commit to declare whether a tool assisted it; the choice is recorded, so a later install without the flag keeps the hook
      --bin-dir string          directory for the PATH entry (default ~/.local/bin, or an existing abcd install adopted in place); fails when it is not writable — abcd never escalates privileges
      --dev                     track-latest dogfood mode: the PATH entry rebuilds from the source tip on every call instead of pinning the built binary
      --docs-target string      marker target: claude_md | agents_md | both | skip
      --oracle-backend string   oracle backend: host-delegated | native | cli | api | mcp
      --refuse-adopt            decline to adopt an unmanaged repo
      --scan-deep string        enable deep scan: true | false
      --visibility string       repo visibility: private | public
      --yes                     approve every resolvable change category without prompting; excludes the optional git-identity pin, which needs an answered prompt (run without --yes, or answer every prompt with: yes | abcd ahoy install)
```

#### `abcd ahoy remote`

Report the managed repo's GitHub secret-scanning settings (read-only); apply enables them

**Usage:** `abcd ahoy remote`

##### `abcd ahoy remote apply`

Enable GitHub secret scanning and push protection on this managed repo, and mirror the desired state

**Usage:** `abcd ahoy remote apply [flags]`

**Flags:**

```
      --yes   confirm the remote change without being asked; without it an unanswered run declines and changes nothing
```

#### `abcd ahoy uninstall`

Remove the marker block, abcd's owned PATH copy, and its provenance record (leaves .abcd/ intact)

**Usage:** `abcd ahoy uninstall [flags]`

**Flags:**

```
      --bin-dir string   directory holding the PATH entry to remove; needed only when it was installed with --bin-dir into a directory that is not on PATH
```

### `abcd banlist`

Banned-names layers (bare renders both, read-only); add/remove maintain them

**Usage:** `abcd banlist`

#### `abcd banlist add`

Add one banned-name entry to the named layer (pattern `-` reads one line from stdin)

**Usage:** `abcd banlist add --private|--public <key> <pattern|-> [flags]`

**Flags:**

```
      --private            the gitignored per-machine layer (.abcd/.work.local/private-names.txt)
      --public             the committed, CI-enforced layer (.abcd/docs-lint.json)
      --severity string    public entry severity: blocker (default) | warn
      --successor string   public entry's replacement, cited in the finding (default "a generic term")
```

#### `abcd banlist list`

Render the banlist layers; private entries render by key only

**Usage:** `abcd banlist list [--private | --public] [flags]`

**Flags:**

```
      --private   the gitignored per-machine layer (.abcd/.work.local/private-names.txt)
      --public    the committed, CI-enforced layer (.abcd/docs-lint.json)
```

#### `abcd banlist remove`

Remove one banned-name entry from the named layer

**Usage:** `abcd banlist remove --private|--public <key> [flags]`

**Flags:**

```
      --private   the gitignored per-machine layer (.abcd/.work.local/private-names.txt)
      --public    the committed, CI-enforced layer (.abcd/docs-lint.json)
```

### `abcd capture`

Capture issues to the ledger; bare invocation is read-only status

**Usage:** `abcd capture [text] [flags]`

**Flags:**

```
      --blocked-by string        comma-separated iss-ids this issue is blocked by
      --category string          issue category (default observation)
      --found-at string          optional repo-relative path or conceptual location
      --found-during string      session/command context (default manual-capture)
      --lapsed-at string         RFC 3339 instant a discipline gave way (the lapse, not the write-up)
      --production-mode string   how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
      --severity string          severity: nitpick | minor | major | critical (default minor)
      --slug string              override the slug derived from the text
      --source string            surfacing channel (default user-observation)
```

#### `abcd capture disposition`

Answer one reading item (a separate record, keyed to the item)

**Usage:** `abcd capture disposition <rdi-N> --state <accepted|rejected|declined|held> [--grounds <text>] [--exit-condition <text>] [--supersedes <dsp-N>] [--recurs <rdi-N,...>] [flags]`

**Flags:**

```
      --exit-condition string        what would end a held disposition (required on held; a hold exits only through a superseding disposition that cites it)
      --grounds string               disposition_grounds: why this answer (free text; required on every state except held)
      --hold-frame-location string   RESERVED (dormant): the frame element a hold sits at; a populated value is refused until activation is ruled
      --hold-moscow string           RESERVED (dormant): must | should | could | wont; a populated value is refused until activation is ruled
      --recurs string                comma-separated prior rdi-ids this item recurs from — the recorded form of a warm recognition, never a mechanical join
      --state string                 the answer: accepted | rejected | declined | held (availability varies by the item's position)
      --supersedes string            the standing dsp-N this answer replaces; required once an item already carries one
```

#### `abcd capture list`

List issues by state (one of --open/--resolved/--wontfix/--all required)

**Usage:** `abcd capture list [flags]`

**Flags:**

```
      --all        issues across all three states
      --open       issues currently in open/
      --resolved   issues currently in resolved/
      --wontfix    issues currently in wontfix/
```

#### `abcd capture promote`

Graduate an issue or a dispositioned reading item into an intent draft (mints + stamps promoted_to)

**Usage:** `abcd capture promote <iss-N> [--grounds "<token>: <text>"] | promote <rdi-N> [flags]`

**Flags:**

```
      --grounds string           optional; recorded when given — the conjecture being acted on, not the route taken: "<pursued|deferred|declined>: <what is expected, and what would show it wrong>"
      --intent string            stamp-only mode: link this existing itd-N instead of minting a draft
      --production-mode string   how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
```

#### `abcd capture resolve`

Mark an open issue resolved (open/ -> resolved/), optionally naming what fixed it

**Usage:** `abcd capture resolve <iss-N> <note> --impact <additive|breaking|fix|internal> [--grounds "<token>: <text>"] [--intent itd-N] [--spec spc-N] [--commit sha] [--shipped-in vX.Y.Z] [flags]`

**Flags:**

```
      --commit string            resolved_by provenance: the fixing commit sha (7-64 hex chars, shape-checked only)
      --grounds string           optional; recorded when given — the conjecture being acted on, not the route taken: "<pursued|deferred|declined>: <what is expected, and what would show it wrong>"
      --impact string            product impact: additive|breaking|fix|internal (required)
      --intent string            resolved_by provenance: the itd-N that fixed it (must exist)
      --production-mode string   restamp how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: leave the record's existing stamp alone; refused on a record that predates disclosure)
      --shipped-in string        MIGRATION USE: the release that already carried this work (vX.Y.Z), leaving the record out of the current cut; unnecessary in a repo abcd managed from the start
      --spec string              resolved_by provenance: the spc-N that fixed it (must exist)
```

#### `abcd capture wontfix`

Record an explicit non-action decision (open/ -> wontfix/)

**Usage:** `abcd capture wontfix <iss-N> <reason> [--grounds "declined: <text>"] [flags]`

**Flags:**

```
      --grounds string           override the recorded grounds text (the token stays declined — a wontfix IS that non-action)
      --production-mode string   restamp how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: leave the record's existing stamp alone; refused on a record that predates disclosure)
```

### `abcd changelog`

Preview the next release cut — derived version, records, guardrail (read-only, no prose)

**Usage:** `abcd changelog`

### `abcd decide`

Mint a decision record (ADR) and lay its skeleton

**Usage:** `abcd decide "<title>"`

Mint an architecture decision record: allocate its id through the shared record-id
seam and write the store's skeleton under .abcd/development/decisions/adrs/.

The id is `adr-<yymmddHHMMSS><rrrr>` and the filename is ordered by that stamp, so two
branches deciding on the same day cannot allocate the same number — the collision a
hand-numbered ordinal has by construction. The hand-numbered records 0001-0058 keep
their ids and their filenames; nothing is renumbered, and every reader admits both.

The verb writes an EMPTY record: it owns the id, the date, the filename and the four
sections, and states nothing. The decision is the author's to write, and the status it
lands with is `proposed` until the author sets `accepted`.

### `abcd disembark`

Lifeboat tooling: coverage probe, pack dry-run, and out-of-tree pack

**Usage:** `abcd disembark`

#### `abcd disembark coverage`

Aggregate probe reports into the cross-repo section×repo coverage table

**Usage:** `abcd disembark coverage <report.json>...`

#### `abcd disembark graveyard`

Validate host-produced lesson JSON against a packed lifeboat and write the survivors (cite-or-be-dropped)

**Usage:** `abcd disembark graveyard <lifeboat-dir> --lessons-json <file|-> [flags]`

**Flags:**

```
      --lessons-json string   path to the host-produced lesson JSON (or - for stdin)
```

#### `abcd disembark pack`

Pack a lifeboat from a repository into a destination directory (writes <dest>, never the source)

**Usage:** `abcd disembark pack <repo> <dest> [flags]`

**Flags:**

```
      --include-ignored   also read files git ignores (widens the scan; the report says so)
```

#### `abcd disembark plan`

Show the full lifeboat file set a pack would write, without writing anything (dry run)

**Usage:** `abcd disembark plan [repo] [flags]`

**Flags:**

```
      --include-ignored   also read files git ignores (widens the scan; the report says so)
```

#### `abcd disembark press-release`

Compose the lifeboat's press release (deterministic from the brief/spine, or validate host-produced press-release JSON)

**Usage:** `abcd disembark press-release <lifeboat-dir> [--press-release-json <file|->] [flags]`

**Flags:**

```
      --press-release-json string   path to host-produced press-release JSON (or - for stdin); absent runs deterministic mode
```

#### `abcd disembark principles`

Distil principles from a packed lifeboat (deterministic from the ADRs, or validate host-produced principle JSON)

**Usage:** `abcd disembark principles <lifeboat-dir> [--principles-json <file|->] [flags]`

**Flags:**

```
      --principles-json string   path to host-produced principle JSON (or - for stdin); absent runs deterministic mode
```

#### `abcd disembark probe`

Report which brief sections a lifeboat could ground from a repository (read-only)

**Usage:** `abcd disembark probe [repo] [flags]`

**Flags:**

```
      --include-ignored   also read files git ignores (widens the scan; the report says so)
```

#### `abcd disembark review`

Review a packed lifeboat against its source repo — a registered verdict and cited findings (deterministic, or validate a host-produced verdict JSON)

**Usage:** `abcd disembark review <lifeboat-dir> <source-repo> [--review-json <file|->] [flags]`

**Flags:**

```
      --review-json string   path to the host-produced review verdict JSON (or - for stdin); absent runs deterministic mode
```

### `abcd docs`

Documentation-currency checks for this repo

**Usage:** `abcd docs`

#### `abcd docs cite`

Maintain the citation baseline the docs lint enforces offline

**Usage:** `abcd docs cite`

##### `abcd docs cite confirm`

Record that a human verified a cited URL the fetcher could not read

**Usage:** `abcd docs cite confirm [url...] [flags]`

Record that a human verified a cited URL the fetcher could not read.

Name the URLs directly, or pass --receipt with a receipt file. Both write the same dated manual entry: the baseline records THAT a human confirmed the citation and WHEN, never how. Only URLs the documentation actually cites can be confirmed.

**Flags:**

```
      --config string    path to docs-lint.json (default: <root>/.abcd/docs-lint.json)
      --receipt string   path to a receipt file listing the confirmed citations (the format the generated checklist page emits)
      --root string      repo root (default: current working directory)
```

##### `abcd docs cite refresh`

Fetch every cited URL once and rewrite the committed citation baseline

**Usage:** `abcd docs cite refresh [flags]`

Fetch every cited URL once and rewrite the committed citation baseline.

This is the only abcd verb that reaches the network on behalf of documentation. Each URL gets exactly one bounded attempt; a failure is recorded as an outcome, never retried. Sources that refuse automated fetchers are printed as a manual checklist for `abcd docs cite confirm` rather than recorded as broken.

**Flags:**

```
      --config string   path to docs-lint.json (default: <root>/.abcd/docs-lint.json)
      --root string     repo root (default: current working directory)
```

#### `abcd docs lint`

Lint docs for change-narration, broken links, and stray root markdown

**Usage:** `abcd docs lint [flags]`

**Flags:**

```
      --config string   path to docs-lint.json (default: <root>/.abcd/docs-lint.json)
      --release-gate    run as the release gate: a citation past its staleness threshold blocks instead of warning (release-time only)
      --root string     repo root to lint (default: current working directory)
```

### `abcd embark`

Unpack a lifeboat's record families back into a target repo (probe read-only; from writes)

**Usage:** `abcd embark`

#### `abcd embark from`

Write a lifeboat's record families into a target repo; refuses on any conflict

**Usage:** `abcd embark from <lifeboat-dir> [target-dir]`

#### `abcd embark probe`

Report what a lifeboat would write into a target, read-only (coverage blanks first)

**Usage:** `abcd embark probe <lifeboat-dir> [target-dir]`

### `abcd guard`

Check a shell command against the hazard registry before it runs

**Usage:** `abcd guard`

#### `abcd guard check`

Decide whether a candidate shell command is safe to run

**Usage:** `abcd guard check [flags]`

Evaluates one candidate command line against the hazard registry — the
bundled defaults merged with this repo's `.abcd/guard.json` — and reports
allow, warn, or block. A blocker exits 1 and names the safe successor; a
warn exits 0 with the warning rendered; an allow exits 0. A guard that
cannot be evaluated at all (an unparsable command line, a malformed
registry) exits 2, so a caller never reads silence as clearance. Unparsable
means an unterminated quote in COMMAND text, which no shell runs either;
an unterminated quote inside a here-document body is document text and is
not one. Grammar a shell does run gets a verdict instead: a trailing
backslash is read as bash reads it, and a here-document whose delimiter
line never comes is a block, because the rest of the input may be commands
the guard did not check.

Matching is shell-token-aware and applies in command position only, so a
hazard named inside a quoted argument never fires.

The guard is a MISTAKE FILTER, not a security boundary. It catches a hazard
typed by accident or reached through an ordinary wrapper — the cases that
actually cost people work. It does not withstand an author trying to get a
command past it, and it does not claim to: the set of programs that launch
another program is open-ended, so no list inside this binary can enumerate
it, and a repository extends that set with one line in a Makefile. Anything
that needs an enforced boundary needs a control at the execution layer — a
sandbox, a permission system, a restricted shell — with this guard in front
of it to teach, never in place of it.

An allow means no registry entry matched — it is never a statement that a
command is safe. A hazard behind a launcher the guard does not recognise is
a WARN naming the entry it matched, rather than an allow, because the guard
cannot tell whether that program runs the rest of the line. What an
allow still does not see is a hazard that never reaches command position at
all: one launched through a known
wrapper carrying a value-taking flag the guard does not name (`sudo -u bob
<hazard>` is seen; the bundled short form `sudo -Hu bob <hazard>` reaches
only the warn, not the entry that names it),
one whose API path an entry names by its ROOT segment but the host serves
under a prefix (a GitHub Enterprise Server install mounts the same endpoints
under `/api/v3/`; the api.github.com URL form IS read), a bare `$VAR` inside
an interpreter payload (an execute-a-string payload IS read — `sh -c`,
`env -S`; one the guard cannot read is warned or, for `env -S`, blocked),
a hazard inside a top-level command substitution (`$(…)` and
backticks are both followed into command position),
a hazard inside a NON-shell interpreter's payload (`python -c`, `perl -e`) —
one opaque token the tokenizer cannot read, today a silent allow (a warn for
it is a recorded design target, not yet raised),
or a dangerous form no entry describes. Coverage is what the registry
names.

The candidate comes from --command, or from stdin when the flag is absent.
Prefer stdin for a command line you did not type yourself: the shell expands
a double-quoted --command argument before this verb starts, so a candidate
containing a command substitution would run at check time. A quoted-delimiter
heredoc (`abcd guard check <<'EOF'` ... `EOF`) passes it through untouched.

**Flags:**

```
      --command string   the candidate command line (default: read from stdin)
```

#### `abcd guard hook`

Host pre-tool-use adapter: decide a shell command from a hook payload

**Usage:** `abcd guard hook`

Reads a host pre-tool-use hook payload on stdin and evaluates its shell
command against the hazard registry. A blocker exits with the host's
blocking status and puts the safe successor and the plain-language why on
stderr, which is the channel the host replays to the agent. A warn and an
allow both let the command run.

Anything the adapter cannot turn into a decision — an unreadable payload, a
tool call that is not a shell command, an unparsable command line, a
registry that will not load — allows the command and warns loudly on
stderr. A guard that cannot answer never stops a session, and is never
silently absent. Unparsable means an unterminated quote in COMMAND text,
which no shell runs either — a quote inside a here-document body is
document text and is not one. A trailing backslash and a here-document with
no delimiter line are grammar a shell does run, so each gets a verdict —
the backslash is read as bash reads it, the unterminated document blocks.

### `abcd history`

Manage the native session-transcript store

**Usage:** `abcd history`

#### `abcd history capture`

Redact and store a raw session transcript (reads a file or stdin)

**Usage:** `abcd history capture [<transcript-file>|-] [flags]`

**Flags:**

```
      --kind string      source kind: native | specstory-import (default native)
      --session string   session id for the record (default: transcript filename; required for stdin)
```

#### `abcd history drain`

Redact and store every staged transcript for this repo

**Usage:** `abcd history drain`

#### `abcd history list`

List stored transcripts for this repo, newest first

**Usage:** `abcd history list`

#### `abcd history show`

Show one stored transcript's metadata and redacted body

**Usage:** `abcd history show <session-id-or-filename>`

#### `abcd history staged`

List transcripts that ended but are not yet redacted into the store

**Usage:** `abcd history staged`

### `abcd ideate`

Idea-admission protocol: record the verdict of the three-leg gauntlet

**Usage:** `abcd ideate`

Record the verdict of abcd's idea-admission protocol — primary-source research, a grill
against the existing record, and an independent adversarial review.

The legs are host work; `/abcd:ideate` orchestrates them. This verb validates what they
produced and writes the durable verdict. Ideate is OPTIONAL and never a gate: no other
verb requires it, and skipping it is never warned about.

#### `abcd ideate record`

Validate a host-composed verdict and write the dated research record

**Usage:** `abcd ideate record <idea-slug> --verdict-json <file|-> [flags]`

**Flags:**

```
      --verdict-json string   path to the host-composed verdict JSON (or - for stdin)
```

### `abcd identity`

Show this repo's canonical identity block and every surface held to it (read-only)

**Usage:** `abcd identity`

#### `abcd identity init`

Record this repo's identity block and the pointer to it (adopts an existing block)

**Usage:** `abcd identity init [flags]`

**Flags:**

```
      --file string      repo-relative file the identity block lives in (default .abcd/development/IDENTITY.md)
      --heading string   heading the identity block sits under (default "Identity (canonical)")
      --pitch string     the project's short elevator pitch (optional)
      --tagline string   the project's one-line tagline (required unless a block already exists)
      --title string     the project's title (required unless a block already exists)
```

#### `abcd identity render`

Print the proposed correction for every drifted surface as a unified diff (writes nothing)

**Usage:** `abcd identity render`

### `abcd intent`

Intent lifecycle; bare invocation is read-only status, quoted text files a draft

**Usage:** `abcd intent [text] [flags]`

**Flags:**

```
      --impact string            stamp the draft's product impact: additive|breaking|fix (optional)
      --production-mode string   how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
```

#### `abcd intent audit`

Intent audit (promise vs delivered): re-emit a shipped intent's request, or ingest a verdict

**Usage:** `abcd intent audit [<itd-N>]`

##### `abcd intent audit ingest`

Ingest an intent-audit verdict JSON into the shipped intent's Audit Notes

**Usage:** `abcd intent audit ingest --verdict-json <path> [flags]`

**Flags:**

```
      --verdict-json string   path to the intent-audit verdict JSON
```

#### `abcd intent link`

Link a planned intent to an existing spec (writes the intent's spec_id)

**Usage:** `abcd intent link <itd-N> <spc-N>`

#### `abcd intent new`

Deprecated alias for `abcd intent "<text>"` (files a draft from the text)

**Usage:** `abcd intent new <text>`

#### `abcd intent plan`

Plan a draft intent (mint its spec, link both sides, move drafts -> planned); on an already-planned intent, stamp its unmarked scope conditions

**Usage:** `abcd intent plan <itd-N> [flags]`

**Flags:**

```
      --production-mode string   how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
```

#### `abcd intent ready`

Report whether an intent is ready to implement (planned + AC + written spec; claims and grounds reported, never refused); exit 1 when not

**Usage:** `abcd intent ready <itd-N> [--grounds "<pursued|deferred|declined>: <conjecture>"] [flags]`

**Flags:**

```
      --grounds string   record the conjecture behind this gate decision: "<pursued|deferred|declined>: <what is expected, and what would show it wrong>"
```

### `abcd launch`

Preview the public launch bundle and release gates (--dry-run required; read-only)

**Usage:** `abcd launch [flags]`

**Flags:**

```
      --dry-run   preview the launch bundle and gates without publishing
```

#### `abcd launch scaffold`

Scaffold the changelog-driven release gate (release.yml, auto-release.yml, runbook) into this repo

**Usage:** `abcd launch scaffold [--confirm] [flags]`

**Flags:**

```
      --confirm   overwrite a hand-edited scaffolded file with the current machinery
```

#### `abcd launch ship`

Cut a release: derive the version and the record set from what shipped (exit 1 when the cut refuses)

**Usage:** `abcd launch ship [--changelog-json <file|->] [--payload-dir <dir>] [flags]`

**Flags:**

```
      --changelog-json string   path to the host-composed changelog JSON (or - for stdin); absent runs the deterministic emit step
      --payload-dir string      stage the versioned release payload in this directory (must be empty and outside the repository)
```

### `abcd lint`

Check this repo against the working conventions (read-only)

**Usage:** `abcd lint [flags]`

**Flags:**

```
      --root string   repo root to lint (default: current working directory)
```

### `abcd memory`

Curated knowledge substrate; bare invocation is read-only status

**Usage:** `abcd memory`

#### `abcd memory ask`

Query memory and synthesise a cited answer

**Usage:** `abcd memory ask <question> [flags]`

**Flags:**

```
      --file-back          file the synthesised answer back as a new memory page
      --page-json string   the answer page dict as JSON (file path, or - for stdin)
      --top-n int          retrieval depth (0 uses the pinned default)
```

#### `abcd memory ingest`

Distil an external source into cited memory pages (https URLs only)

**Usage:** `abcd memory ingest <path-or-https-url> [flags]`

**Flags:**

```
      --keep-original       store the original at .abcd/memory/sources/<sha256>.<ext>
      --pages-json string   DistilledPage JSON array (file path, or - for stdin)
```

#### `abcd memory lint`

Curator health-check over the whole memory store

**Usage:** `abcd memory lint`

### `abcd reading`

Cold-reading input assembler: what a reading sees, and the manifest proving it

**Usage:** `abcd reading`

Assemble the input a cold reading is handed.

Blindness is a property of the input, not a promise the reader makes: a positive include
table names what may travel, fields are projected out of records rather than files copied
whole, and a hashed manifest records what was passed so a reader can judge contamination
rather than accept a disclosure on trust.

Bare `abcd reading` renders the assembler's state and writes nothing.

#### `abcd reading assemble`

Assemble one reading's input and its manifest

**Usage:** `abcd reading assemble --position <position> --target <HEAD|sha> [flags]`

Walk the repository under the include table at one reading position and write two
artefacts: the assembled input, which carries no repository path, and the manifest,
which maps every passed item back to its path, its field and its hash.

The invocation is a position and a target state, and nothing else. --position takes
one of four closed tokens; --target takes HEAD or a hexadecimal commit sha of 7 to 40
digits, because a branch or a tag moves and the manifest's re-runnability rests on a
reference that cannot. Both are required.

What the reading is handed comes from the committed preset entry for the position, in
.abcd/config/reading-presets.json, applied with no operand. Changing it is a commit to
that file, reviewed and inside the dirty gate; the manifest records the entry applied
and its hash, so a run is reproducible from the commit it names.

**Flags:**

```
      --dry-run           write nothing; with --out the two artefacts still land in that directory
      --out string        an empty or absent directory the assembled input and the manifest are written to
                          (default: the local-tier run directory)
      --position string   the reading position: widening, entailment, comparative, detection
                          (comparative derives its candidate set from the record: the one committed
                          widening run at the target whose items carry no disposition and no
                          admission — at the target, or at an ancestor of it across which only the
                          readings store and the issue ledger changed, so a run's own records can be
                          committed between its ingest and this assembly. None, or more than one,
                          refuses and lists the runs)
      --target string     the commit the assembly describes: HEAD, or a hexadecimal sha of 7 to 40 digits
```

**Example:**

```
abcd reading assemble --position widening --target HEAD --dry-run
  abcd reading assemble --position entailment --target HEAD \
    --out .abcd/.work.local/scratch/reading-runs/manual --json
```

#### `abcd reading ingest`

Validate one reading's returned output and write its records

**Usage:** `abcd reading ingest --reading-json <path> [flags]`

Validate the JSON a cold reading returned and write its reading records.

The verb checks what the reading was LICENSED to produce, not only what it saw: the
supply regime is read from the position's definition and compared with the output's own
claim, and an item carrying a reserved name as one of its own fields is refused with the
licence stated. The reserved-name table is read at the run's own regime, one row per
regime, and the generative regime has no row: no name is reserved at the generative
position.

Item identifiers are minted here. The payload carries none, so a supplied one is refused
as an unknown field. A refusal becomes DURABLE once the run's identity is proven — the run
id resolving to a parked manifest whose content hash matches — and from there a list-level
refusal writes refusal.json under the run's directory; before that point nothing durable is
written anywhere. No OTHER run's durable state is touched until the whole payload validates:
a refusal after the run is proven writes its refusal record and nothing else, and the one
delete it makes is on its OWN run id — the records of an earlier attempt at it that never
committed. The reading records land as one batch and the run metadata is written last as the
commit marker: a run without one never happened.

An ingest interrupted before that marker leaves an orphaned stage, and every invocation names
it. Only the next one whose payload validates sweeps it: where the run reached no commit
marker the sweep ROLLS THAT RUN'S READING RECORDS OUT OF THE COMMITTED LEDGER, because the
run never happened; where the marker is there the run stands and only the stage goes. A
refused run reports the orphans it left in place, and the ids a sweep removed are reported as
rolled_back_records on every exit, including a failing one.

**Flags:**

```
      --reading-json string   path to the JSON the cold reading returned
```

**Example:**

```
abcd reading ingest --reading-json ./reading-output.json --json
```

### `abcd rules`

Render the active rule set; a positional DOMAIN scopes to one (read-only)

**Usage:** `abcd rules [domain]`

Render the rule set the modular-rules loader injects: the bundled default
domains merged with this repo's .abcd/rules.json. Bare, it renders every active
domain; a positional DOMAIN (case-insensitive) renders that one domain regardless
of its state or the kill switch, so a dormant domain is still inspectable.

Every domain says which layer it came from. A domain the repo override names —
its rules replaced, its state changed, or a custom domain declared — renders as
"## NAME (repo override)" here, in the injected block and in the hook's
diagnostic, and carries "source": "repo" in --json; an untouched bundled domain
renders bare and carries "source": "bundled". Read-only.

### `abcd site`

The website rendered from this repository: what is declared, and what was built (read-only)

**Usage:** `abcd site [flags]`

**Flags:**

```
      --out string   output directory to report on (default "site")
```

#### `abcd site build`

Render the site into the output directory (writes nothing outside it)

**Usage:** `abcd site build [flags]`

**Flags:**

```
      --commit string    commit for the footer and the build stamp (default: git HEAD)
      --date string      date for the build stamp (default: the newest release's date)
      --out string       directory to render into (default "site")
      --preview          stamp the build as unreleased at this commit, for a preview deployment of an untagged tree
      --version string   version for the footer and the build stamp (default: the newest dated CHANGELOG heading)
```

#### `abcd site check`

Gate the built site: provenance, hero drift, banned tokens, snippets, the reference ratchet, mobile and figure labels

**Usage:** `abcd site check [flags]`

**Flags:**

```
      --out string   built output directory to check (rendered first if absent) (default "site")
```

### `abcd spec`

Native spec store; bare invocation is read-only status

**Usage:** `abcd spec`

#### `abcd spec close`

Close a spec (open/ -> closed/) and ship its linked intent (planned/ -> shipped/)

**Usage:** `abcd spec close <spc-N>`

### `abcd update`

Complete a chosen update: fetch, verify, and swap the PATH-installed binary

**Usage:** `abcd update [tag] [flags]`

Fetches the named release (or resolves the latest, naming it before acting),
verifies the platform binary against the same release's checksums.txt, and
swaps the PATH-installed copy atomically. The verb is the only ask: abcd
never checks for or applies updates on its own (adr-38). A plugin-root
binary, the dev shim, and package-manager installs are refused with the
command that owns them.

**Flags:**

```
      --yes   skip the TTY confirmation of a freshly resolved tag
```

### `abcd version`

Print abcd's version, install mode, and vintage

**Usage:** `abcd version [flags]`

**Flags:**

```
      --check   fetch the latest release once and compare (this command's only network touch; abcd never fetches implicitly — adr-38); names its source
```
