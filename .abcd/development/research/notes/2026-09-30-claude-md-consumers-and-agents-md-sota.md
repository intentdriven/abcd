# SOTA survey and inventory — can `CLAUDE.md` go now that the first harness reads `AGENTS.md`?

Dated 2026-09-30. Evaluates iss-2609291925136841, captured by the product
thinker on 2026-09-29: "Evaluate whether CLAUDE.md can be removed safely now
that Claude Code reads AGENTS.md natively ... Only remove it if every
consumer reads AGENTS.md." This note evaluates and removes nothing. One
host-run research pass over the harnesses' own documentation (read on
2026-09-30) and a search of this repository at the lane base, challenged for
fit per [`prefer-sota`](../../principles/prefer-sota.md).

**What is there today.** In this repository `CLAUDE.md` is not a copy: it is
a committed symlink to `AGENTS.md` (git mode `120000`), and so is
`GEMINI.md`. Nothing is duplicated on disk; the question is whether the link
is still needed, and whether abcd should keep scaffolding one in the
repositories it adopts.

## What the harness documents (vendor tier, read 2026-09-30)

- **Native reading, with conditions.** "Claude Code can read `AGENTS.md` as
  your project instructions, so a repository already set up for other coding
  agents works without adding a `CLAUDE.md`, an import, or a setting." It
  "requires Claude Code v2.1.277 or later".
- **The default reads one or the other, not both.** Under the default
  (`claude-md-or-agents-md`), `AGENTS.md` is read only when there is no
  `CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md` "in your working
  directory or any directory above it". The user's `~/.claude/CLAUDE.md` and
  a managed policy file do not count.
- **Sessions that cannot read it.** A version before v2.1.277; the built-in
  `agents-md` plugin disabled; "in some cases, it's your first session after
  you upgrade from v2.1.276 or earlier"; and, before v2.1.281, some sessions
  "such as those on Amazon Bedrock or with telemetry disabled". The setting
  can also be set to `claude-md` or `managed-only`, both of which drop
  `AGENTS.md`.
- **Where `AGENTS.md` read natively differs from `CLAUDE.md`:**
  `InstructionsLoaded` hooks "don't fire"; directories added with `--add-dir`
  under `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD` load their `CLAUDE.md`
  but "their `AGENTS.md` doesn't load"; an external `@path` import in
  `AGENTS.md` loads only if external imports were already approved.
- **Sub-agents.** A non-fork sub-agent loads "every level of the CLAUDE.md
  hierarchy the main conversation loads ... and any `AGENTS.md` files loaded
  as project instructions"; "The built-in Explore and Plan agents skip
  this", as does any agent whose definition sets `omitClaudeMd: true`. A fork
  inherits the parent's context. Plugin-provided agents "load the same
  CLAUDE.md hierarchy as custom subagents". Agent-team teammates load
  "CLAUDE.md, MCP servers, and skills" like a regular session. So a
  sub-agent sees `AGENTS.md` exactly when the main session does.
- **The harness's own advice on a symlink.** "A `CLAUDE.md` symlinked to
  `AGENTS.md`: nothing, or delete the symlink. Either way Claude reads the
  content once." Its Windows caveat: a committed symlink checks out as "a
  one-line `CLAUDE.md`" unless `core.symlinks` is enabled; the `@AGENTS.md`
  import is the portable form.

**The second harness.** The Gemini CLI loads `GEMINI.md` by default and reads
`AGENTS.md` only when `context.fileName` names it (vendor tier). `GEMINI.md`
is out of this capture's scope but has the same shape.

**The standard.** `AGENTS.md` is "stewarded by the Agentic AI Foundation
under the Linux Foundation" and read by "20+ coding agents and IDEs" (project
tier).

## Every consumer of `CLAUDE.md` in this repository

Found by `git grep` at 2b9d52fbb, excluding the ledger and the durable
record; 21 non-test files and 27 test files name it.

| Consumer | What it does with `CLAUDE.md` | Reads `AGENTS.md` too? |
| --- | --- | --- |
| The harness, main session, in this checkout | loads the symlink as project instructions | yes, natively, when no `CLAUDE.md` is present and the session is on v2.1.277+ with the default setting |
| The harness, sub-agents, forks, plugin agents, teammates | inherit the main session's project instructions | as the main session; Explore and Plan read neither |
| `internal/core/ahoy/detect.go` (`classify`), `managed.go` (`Managed`) | treat a marker block in `CLAUDE.md` **or** `AGENTS.md` as "this repository is managed" | yes |
| `internal/core/ahoy/marker.go` (`markerTargets`), `prompt_help.go` | `docs.target` = `claude_md` \| `agents_md` \| `both` \| `skip`; the default writes neither | yes (`agents_md`) |
| `internal/core/ahoy/apply.go` (`Uninstall`) | removes the marker from both files | yes |
| `internal/core/lifeboat/embark.go` (`EnsureMarker` call) and `commands/embark.md` | re-injects the marker block into the target's `CLAUDE.md` **only**; a symlinked or unwritable `CLAUDE.md` is skipped | **no** — the one hard-coded consumer |
| `internal/core/lifeboat/sources_native.go` | packs `AGENTS.md` and `CLAUDE.md` as the router into a lifeboat | yes |
| `commands/prepare-this-repo.md` (Phase 3, step 3) | creates `AGENTS.md` "with `CLAUDE.md` as a symlink to it", and lists `CLAUDE.md`/`GEMINI.md` bridges in the audit | yes; the bridge is what it scaffolds |
| `internal/core/repolint/rule_router.go` (`conventions-router`) | requires `AGENTS.md`; names `CLAUDE.md` as an optional bridge | yes; passes without `CLAUDE.md` |
| `internal/core/lint/lint.go` (`checkStrayRootDocs`) | exempts a root symlink by its target's stem, which is what lets the bridge pass | n/a; passes without it |
| `internal/core/lint/citations.go` | exempts the in-repo bridge from the leaf rule | n/a |
| `.github/workflows/ci.yml` (inert-path classifier) | names the three routers as deliberately not inert | n/a; removing one changes nothing |
| `docs/how-to/install.md` | tells adopters the managed block goes into `CLAUDE.md` or `AGENTS.md` only with `--docs-target` | yes |
| `.abcd/docs-lint.json` | bans harness tokens in user-facing prose; does not reference the file | n/a |
| Tests (27 files) | build `CLAUDE.md` fixtures in temporary directories | none reads this repository's own `CLAUDE.md` |
| Comments and history: `ahoy/embark_marker.go`, `ahoy/rewritelock.go`, `ahoy/store.go`, `lifeboat/embark_types.go`, `surface/cli/cli.go`, `CHANGELOG.md`, a guard test-corpus line | name the file in prose; no behaviour | n/a |

Outside this repository: the user-level `~/.claude/CLAUDE.md` is a personal
file that does not count against `AGENTS.md`, and is untouched by the
question. No `CLAUDE.md` or `CLAUDE.local.md` sits above the primary
checkout or the machine-scoped worktree store on the machine this was
checked on, so removing the link would let `AGENTS.md` load there — but
that holds per machine, not by construction.

## Adversary filter

- **Host-agnostic.** `AGENTS.md` is the host-agnostic router the
  conventions-router rule already requires; a harness-named bridge is the
  host-specific part. Removal moves toward the preference.
- **The symlink costs almost nothing.** It duplicates no content and the
  harness reads it once. The case for removal is tidiness and fewer
  host-named files, not correctness.
- **What removal would lose, today.** Sessions on older harness versions,
  with the `agents-md` plugin disabled, with the setting at `claude-md`, or on
  the first session after an upgrade from v2.1.276 or earlier, would load no
  project instructions at all — silently, for an agent that would then work
  without this repository's rules. The `InstructionsLoaded` hook would stop
  firing for the router (abcd registers none today; `hooks/hooks.json`
  declares UserPromptSubmit, SessionStart, PreToolUse, PreCompact, SessionEnd
  and SubagentStop).
- **A `CLAUDE.local.md` would silently switch the router off.** Under the
  default setting, a contributor who adds one for private notes stops
  `AGENTS.md` loading. With the symlink in place that cannot happen.
- **Managed repositories are not this repository.** An adopted repository may
  hold a real `CLAUDE.md` with its own content; abcd cannot remove that, only
  stop creating a bridge where none exists.

## Verdict

**Do not remove `CLAUDE.md` yet.** Two consumers still need it, and the
harness's own documentation names sessions in which `AGENTS.md` does not
load. Removal would need all of the following, each checkable:

1. **Embark reads `AGENTS.md`.** `lifeboat/embark.go` hard-codes
   `CLAUDE.md` as the marker target; it must follow `docs.target` (or prefer
   `AGENTS.md`) the way ahoy's `markerTargets` does, with a test that embarks
   into a target that has only `AGENTS.md`.
2. **prepare-this-repo stops scaffolding the bridge**, or scaffolds the
   portable `@AGENTS.md` import instead of a symlink where the product thinker
   wants one kept for older sessions.
3. **A floor on the harness version** in abcd's own install checks
   (v2.1.281, the version from which every session type the documentation
   names reads `AGENTS.md`), so a session that would load no instructions is
   refused or warned rather than silent. This step couples abcd to one
   vendor's version string, against the host-agnostic preference the
   adversary filter above applies elsewhere; its cost is weighed with the
   other steps.
4. **A detector for the silent switch-off**: `conventions-router` warns
   when `AGENTS.md` exists beside a `CLAUDE.local.md` and no `CLAUDE.md`, the
   one state in which the default setting stops reading `AGENTS.md`. The
   vendor documentation also names a setting, `claude-md-and-agents-md`,
   that keeps `CLAUDE.local.md` and `AGENTS.md` loading together; it is the
   alternative the detector is weighed against.
5. **Then** remove the committed symlink here, in a change that shows a fresh
   session loading `AGENTS.md` (the harness prints
   `no CLAUDE.md found; AGENTS.md loaded: …`).

The capture's own remedy asks for "a test that fails if a consumer still
needs it". Item 1's test is that test for the one code consumer; the
harness-side consumers can only be checked by a session, not by `go test`.
What would show this verdict wrong: a supported session type that loads
`CLAUDE.md` but not `AGENTS.md` after v2.1.281 (the floor is then too low),
or a consumer this inventory missed.

## Review record

2026-09-30: authored in one pass by an implementer in autonomous run A (lane
drainResearch), with the fit-challenge run in-pass by the author. No
independent adversarial reviewer has read this note yet; per the
[research protocol](2026-08-22-sota-research-protocol.md) that review is owed
before a removal relies on it.

## Sources (all accessed 2026-09-30)

- [Claude Code — How Claude remembers your project (CLAUDE.md, AGENTS.md)](https://code.claude.com/docs/en/memory)
- [Claude Code — Subagents (what loads at startup)](https://code.claude.com/docs/en/sub-agents)
- [Claude Code — Agent teams (context and communication)](https://code.claude.com/docs/en/agent-teams)
- [Gemini CLI — GEMINI.md context files](https://geminicli.com/docs/cli/gemini-md/)
- [AGENTS.md](https://agents.md/)
