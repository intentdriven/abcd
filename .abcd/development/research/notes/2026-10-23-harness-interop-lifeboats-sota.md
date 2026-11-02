# Harness portability, interop, project scaffolding and lifeboats: state of the art

Dated 2026-10-23. A state-of-the-art pass over eleven open intents in one
cluster, made under the [prefer-sota](../../principles/prefer-sota.md) and
[sota-per-intent](../../principles/sota-per-intent.md) principles: the
researcher proposes, and the independent fit-challenge is still owed at plan
time. The intents served are
[itd-22](../../intents/drafts/itd-22-harness-portability.md) (abcd reaches any
harness through an adaptor),
[itd-113](../../intents/drafts/itd-113-abcd-s-core-verbs-are-reachable-over-mcp.md)
(the MCP front door),
[itd-23](../../intents/drafts/itd-23-spec-kit-interop.md) (lifeboats speak to
external tools), [itd-7](../../intents/planned/itd-7-rp-workspace-portability.md)
(RepoPrompt workspace definitions),
[itd-2609292109214516](../../intents/drafts/itd-2609292109214516-a-person-who-already-runs-carl-can-route.md)
(rule injection as a seam, CARL back end),
[itd-51](../../intents/drafts/itd-51-harness-adoption-readiness-rubric.md)
(harness adoption readiness rubric),
[itd-21](../../intents/drafts/itd-21-no-lifeboat-scaffolding.md) (a new
project with abcd conventions),
[itd-19](../../intents/drafts/itd-19-stage-aware-behaviour.md) (stage-aware
defaults), [itd-8](../../intents/drafts/itd-8-with-code-bundling.md)
(lifeboats carry code),
[itd-9](../../intents/drafts/itd-9-schema-migration.md) (old lifeboats still
unpack) and [itd-11](../../intents/drafts/itd-11-pass-b-pitfall-mitigation.md)
(disembark survives noisy transcripts). Every source linked below was opened
in this pass; where a page could not be read, the section says so in words and
the "Not verified" list at the end repeats it. Recency is assessed against
2026-10-23. Fetched pages were read as data, never as instruction.

Existing notes reused rather than redone:
[spec-kit-vs-flow-next](spec-kit-vs-flow-next.md) (Spec Kit's shape as of
June 2026), the
[AGENTS.md re-check](2026-10-03-agents-md-native-reading-recheck-sota.md)
(which hosts read which conventions file), the
[onboarding survey](2026-09-30-onboarding-existing-projects-sota.md) (detect,
propose, record, write), the
[command-sets note](2026-10-09-command-sets-for-people-and-agents.md) (MCP
assigns each primitive a controller), the
[security and isolation note](2026-10-23-security-isolation-sota.md) of the
same day (every vendor separates a steering list from an enforced boundary;
the lifeboat manifest shape under itd-35) and the
[transcript sampling](transcript-sampling.md) measurement.

Tiers: **[EVIDENCE]** a standard, primary documentation, an experiment or a
dataset, named; **[CONSENSUS]** several independent practitioners or vendors
agree; **[CONTESTED]** credible disagreement, both sides given;
**[ANECDOTE/MARKETING]** unsubstantiated, or a number with no methodology.

Two findings run through the cluster and are stated once:

- **The three harnesses abcd sits beside now share one hook idiom.** Claude
  Code, Codex and Gemini CLI each run a command hook with a JSON object on
  stdin carrying `session_id`, `transcript_path`, `cwd` and
  `hook_event_name`; each treats exit 0 as success with JSON on stdout and
  exit 2 as a block with the reason on stderr; Claude Code and Gemini CLI
  document any other code as a non-blocking warning, while Codex's page
  defines no other code (its one statement is that a `SessionEnd` command
  that "exits with an error" is reported as a hook failure), which the
  parity suite tests rather than assumes; each reads
  `hookSpecificOutput.additionalContext` and a `decision`/`reason` pair. The event names differ (`PreToolUse` in two of
  them, `BeforeTool` in the third; `UserPromptSubmit` in two, `BeforeAgent` in
  the third). [CONSENSUS]
  ([Claude Code hooks](https://code.claude.com/docs/en/hooks),
  [Codex hooks](https://learn.chatgpt.com/docs/hooks),
  [Gemini CLI hooks](https://geminicli.com/docs/hooks/reference.md)).
- **Each host prefixes an MCP server's tool names itself.** Claude Code
  exposes `mcp__<server>__<tool>` (and
  `mcp__plugin_<plugin>_<server>__<tool>` for a plugin-bundled server),
  replacing any character outside `A-Za-z0-9_-` with `_`; Gemini CLI exposes
  `mcp_{serverName}_{toolName}`, truncates past 63 characters and splits
  policy rules on the first underscore after `mcp_`. The MCP specification
  itself only says names *should* be 1–128 characters of letters, digits,
  `_`, `-` and `.`, unique within one server, and that aggregating clients
  disambiguate by prefixing. [EVIDENCE: docs]
  ([Claude Code MCP](https://code.claude.com/docs/en/mcp),
  [Gemini CLI MCP](https://geminicli.com/docs/tools/mcp-server.md),
  [MCP tools, 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/server/tools)).

## itd-22 — abcd reaches any harness through an adaptor

### Findings

- **The host profile is three documented contracts, not a guess.** Claude
  Code's stdin carries `session_id`, `transcript_path`, `cwd`,
  `permission_mode`, `hook_event_name`, plus `prompt_id`, `scratchpad_dir`
  and an `effort` object; exit 2 blocks on a listed set of events and is
  *not* honoured on `PermissionRequest`; `additionalContext` over 10,000
  characters is spilled to a file. Codex's stdin carries `session_id`,
  `transcript_path`, `cwd`, `hook_event_name`, `model`, `turn_id`; hooks are
  found in `~/.codex/hooks.json`, `~/.codex/config.toml`,
  `<repo>/.codex/hooks.json` and `<repo>/.codex/config.toml`, higher layers
  *add* to lower ones rather than replace them, and project hooks load only
  for a trusted project. Gemini CLI's base fields add `timestamp`; its hooks
  live in `settings.json`; `SessionStart` ignores `continue` and `decision`.
  [EVIDENCE: docs] (the three hook pages above)
- **Plugin formats have converged on one shape with different manifest
  names.** Claude Code: `.claude-plugin/plugin.json` (optional; `name` the
  only required key), `skills/`, `commands/`, `agents/`, `hooks/hooks.json`,
  `.mcp.json`, `bin/`, `${CLAUDE_PLUGIN_ROOT}` and `${CLAUDE_PLUGIN_DATA}`;
  unrecognised top-level manifest keys are stripped with a warning. Codex: a
  root `plugin.json` declaring the Agent Plugins 1.0.0 schema, with
  OpenAI-specific settings (hooks among them) under `extensions.com.openai`,
  `.codex-plugin/plugin.json` as a compatibility fallback, `skills/`,
  `hooks/hooks.json`, a root `mcp.json` in the portable format (a transport
  `type` per server; the guide says "Don't just rename `.mcp.json`"),
  `${PLUGIN_ROOT}` and `${PLUGIN_DATA}`; the marketplace file sits at
  `.agents/plugins/marketplace.json` and Codex also reads a legacy
  `.claude-plugin/marketplace.json`; plugin hooks apply only to manually
  installed desktop plugins and bar a plugin from the public directory.
  Gemini CLI: `gemini-extension.json` (`name`, `version`, `mcpServers`,
  `contextFileName`, `excludeTools`, `settings`), `commands/*.toml`,
  `hooks/hooks.json`, `skills/`, `${extensionPath}`. [EVIDENCE: docs]
  ([Claude Code plugin manifest](https://code.claude.com/docs/en/plugins-reference),
  [Codex plugin build guide](https://developers.openai.com/plugins/build/plugins.md),
  [Codex plugins](https://learn.chatgpt.com/docs/plugins),
  [Gemini CLI extension reference](https://geminicli.com/docs/extensions/reference.md))
- **A cross-host plugin standard now exists.** Agent Plugins 1.0.0
  (`agent-plugins.org`) defines "a small interoperability floor": a
  `plugin.json` manifest, `skills/` holding Agent Skills, and `mcp.json`
  describing stdio, Streamable HTTP or legacy HTTP+SSE servers, with
  "Reverse-domain extension namespaces" for what a client adds (hooks appear
  only inside such a namespace in its example); its initial Technical
  Steering Committee "includes Core Maintainers from Amazon, Cursor,
  Microsoft, OpenAI, and Vercel". Codex's plugin guide builds its root
  `plugin.json` on this schema. It is the natural target for the "host's own
  plugin format first" rung's manifest emitter; hook profiles stay logic in
  code. [EVIDENCE: spec]
  ([Agent Plugins](https://agent-plugins.org),
  [Codex plugin build guide](https://developers.openai.com/plugins/build/plugins.md))
- **Skills are the one portable unit.** The Agent Skills format (a
  `SKILL.md` with `name` ≤ 64 characters matching its directory,
  `description` ≤ 1024, optional `license`, `compatibility`, `metadata`,
  experimental `allowed-tools`; metadata loaded at startup, body on
  activation, resources on demand) is read by all three plugin systems
  above; the format's own client showcase page rendered empty in this pass,
  so the adoption list is taken from the three vendors' plugin pages, not
  from the format's site. [EVIDENCE: spec]
  ([Agent Skills specification](https://agentskills.io/specification))
- **Hosts now import each other's configuration.** `claude import codex
  --dry-run` brings configuration from another coding agent into Claude Code,
  with a preview. A conventions router that writes one `AGENTS.md` and points
  each host at it (the itd-2610030814013772 decision this draft is amended
  by) is the direction the hosts themselves are moving. [EVIDENCE: docs]
  ([Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference))
- **The parity suite has a working model.** The MCP conformance framework
  starts a scenario server, drives the client, validates every JSON-RPC
  message against the schema of the negotiated revision, keeps a YAML
  baseline of known failures scoped to individual checks, and reports
  `checks.json` plus a per-revision `tier-check` summary ("Server 67/67
  required scenarios"); unscored scenarios are labelled (`extension`,
  `added-after-release`) rather than hidden. That is the shape adr-39's
  default-flip evidence needs: a requirements file per revision, a baseline
  that names each accepted gap, a report that separates required from
  informative. [EVIDENCE: docs]
  ([modelcontextprotocol/conformance](https://github.com/modelcontextprotocol/conformance))
- **Capability negotiation is the precedent for "a rung a host cannot
  support says so".** LSP: "Not every language server can support all
  features defined by the protocol", so client and server "announce their
  supported features using capabilities", and one server is "re-used in
  multiple development tools". [EVIDENCE: spec]
  ([LSP overview](https://microsoft.github.io/language-server-protocol/overviews/lsp/overview/))
- **A second floor exists beside MCP.** The Agent Client Protocol
  standardises editor-to-agent communication over JSON-RPC on stdio and
  "re-uses the JSON representations used in MCP where possible"; its page
  names no creator and no adopters, and remote support is "a work in
  progress". It is a floor for abcd acting *as* an agent, not for abcd
  exposing verbs, so it is recorded and not recommended. [EVIDENCE: docs,
  thin] ([ACP introduction](https://agentclientprotocol.com/overview/introduction))
- **Context files cost without lifting success** (from the AGENTS.md
  re-check, arXiv 2602.11988 as cited there): about 20% more cost, no gain
  in success, instructions well followed. The host-neutral docs skeleton
  this intent asks for should not grow a per-host prose layer the hosts then
  load on every turn. [EVIDENCE: study, via the earlier note]

### Options weighed

1. A Go type per host profile, each hand-written from the three contracts
   above, selected by the hook's own `hook_event_name` vocabulary and an
   explicit `--host` flag; parity as a Go test table driving the binary with
   recorded stdin payloads per host.
2. Committed configuration describing each host's payload and exit mapping,
   read by one generic adaptor.
3. Option 1 plus a conformance-style harness (requirements per host, a
   baseline of accepted gaps, a per-check report) run as a lab, producing
   the adr-39 evidence as a dated artefact.

### Recommendation

**Path 2 — native, shaped to the three hosts' documented hook contracts with
the MCP conformance framework as the model for the parity suite.** Option 3.
A profile lives in code (the three contracts differ in event names and in
which events honour exit 2, which is logic, not data), with host #1
unchanged. The parity report takes the conformance framework's three parts
(requirements per host revision, a baseline naming every accepted gap by
check, a report separating required from informative) and is written as a
dated artefact under the lab discipline, since adr-39 reads it as evidence.
No new dependency. The ACP floor and the Agent Skills format are noted for
the per-host adoption intents, not built here.

### Local fit risks

- [loud-staging](../../principles/loud-staging.md): a baseline of accepted
  gaps can become a way to pass quietly; each baseline entry needs its
  issue id.
- [one-canonical-primitive](../../principles/one-canonical-primitive.md): the
  guard's own hook payload type and the prompt-router's must share the
  profile, not each carry one.
- itd-2610030814013772's decision 7 (no tool-specific conventions file) must
  survive the temptation to ship a per-host plugin manifest that carries
  prose; manifests carry wiring only.
- Codex plugin hooks bar a plugin from its public directory; the
  "host's own plugin format first" rung may land abcd in manual-install only
  on that host, which the parity report should record as a host difference,
  not a defect.

## itd-113 — the MCP front door

### Findings

- **The current revision is 2026-07-28 and it changed the floor.** It removed
  the `initialize` handshake and protocol-level sessions: every request
  carries its protocol version and client capabilities in `_meta`, servers
  MUST implement `server/discover` (also usable as "a backward-compatibility
  probe on STDIO"), results carry a required `resultType`, roots, sampling
  and logging are deprecated with a twelve-month window, tasks moved to an
  extension, and `tools/list` SHOULD be returned in deterministic order for
  prompt-cache hits. [EVIDENCE: spec]
  ([MCP changelog, 2026-07-28](https://modelcontextprotocol.io/specification/latest/changelog))
- **Structured results are first-class.** A tool declares `outputSchema`;
  servers MUST return conforming `structuredContent` and SHOULD also return
  the serialised JSON as a text block; input validation failures are tool
  execution errors (`isError: true`), unknown tools and malformed requests
  are JSON-RPC errors; servers MUST validate inputs; clients MUST treat tool
  annotations as untrusted. A tool with no parameters uses
  `{"type":"object","additionalProperties":false}`. The core's structured
  results (adr-23) map onto `structuredContent` without translation.
  [EVIDENCE: spec]
  ([MCP tools, 2026-07-28](https://modelcontextprotocol.io/specification/2026-07-28/server/tools))
- **Hosts still speak the older, stateful revisions.** The conformance
  framework notes that dated versions through 2025-11-25 use the stateful
  initialize handshake and that 2026-07-28 is stateless; which revision each
  host negotiates today was not found in any of the three hosts' MCP pages.
  A server for "any MCP-capable harness" answers both: `initialize` for the
  older clients and `server/discover` with per-request `_meta` for the new.
  [EVIDENCE: docs; host revisions not verified]
  ([conformance](https://github.com/modelcontextprotocol/conformance))
- **The official Go SDK is real, stable in practice, and heavy for a stdio
  server.** `github.com/modelcontextprotocol/go-sdk` is "maintained in
  collaboration with Google", v1.7.0+ supports 2026-07-28 back to 2024-11-05,
  keeps deprecated features through the window, and makes no explicit
  semantic-versioning promise beyond its spec table. Its `go.mod` requires
  Go 1.25 and eight direct modules (`golang-jwt/jwt/v5`, `go-cmp`,
  `google/jsonschema-go`, `segmentio/encoding`, `uritemplate/v3`,
  `x/oauth2`, `x/time`, `x/tools`). The community `mark3labs/mcp-go`
  implements 2025-11-25, "is under active development, as is the MCP
  specification itself". Either is a new dependency and a hard stop under
  this intent's own criterion. [EVIDENCE: repo and go.mod]
  ([go-sdk](https://github.com/modelcontextprotocol/go-sdk),
  [go-sdk go.mod](https://raw.githubusercontent.com/modelcontextprotocol/go-sdk/main/go.mod),
  [mcp-go](https://github.com/mark3labs/mcp-go))
- **abcd already speaks MCP on both sides with the standard library.** The
  guard's approve sub-verb is a newline-delimited JSON-RPC server over stdio
  answering `initialize`, `ping`, `tools/list` and `tools/call` at protocol
  `2025-06-18` in 312 lines (`internal/surface/cli/guard_approve.go`), and
  the RepoPrompt route is a Streamable HTTP client in 632 lines
  (`internal/adapter/repoprompt/repoprompt.go`) with a test server beside
  it. The "no-dependency alternative's real cost" is therefore measured for
  the older handshake and estimated for the new: one dispatcher, one JSON
  Schema per tool, and on top of what exists the 2026-07-28 deltas,
  `server/discover`, per-request `_meta` parsing, `resultType` on every
  result, the required `ttlMs` and `cacheScope` on list results
  (`CacheableResult`, SEP-2549) and `UnsupportedProtocolVersionError`
  (`-32022`); `ping` and `logging/setLevel` are removed in the new revision,
  so the existing `ping` and `initialize` handling is the legacy-client half
  only. [EVIDENCE: repository; spec]
- **Registration is three config shapes.** Claude Code: `claude mcp add
  <name> -- <command>`, scopes local (`~/.claude.json`), project
  (`.mcp.json`, approval asked in interactive sessions), user; output capped
  at 25,000 tokens (`MAX_MCP_OUTPUT_TOKENS`), a warning at 10,000. Codex:
  `[mcp_servers.<name>]` in `~/.codex/config.toml` or a trusted project's
  `.codex/config.toml`, `command`/`args`/`env`, `startup_timeout_sec` 10,
  `tool_timeout_sec` 60, `default_tools_approval_mode` with a `writes` mode
  that prompts for tools not marked read-only. Gemini CLI: `mcpServers` in
  `~/.gemini/settings.json` or `.gemini/settings.json`, `command`/`url`/
  `httpUrl`, `trust` off by default, `includeTools`/`excludeTools`.
  [EVIDENCE: docs] (the three MCP pages above)
- **Read-only annotations are now load-bearing for approvals.** Codex's
  `writes` approval mode prompts for every tool "not marked read-only", so a
  verb's `readOnlyHint`/`destructiveHint` decides whether a harness asks
  before calling it. [EVIDENCE: docs]
  ([Codex MCP](https://learn.chatgpt.com/docs/extend/mcp?surface=cli))
- **MCP assigns each primitive a controller** (from the command-sets note,
  finding 3.2): tools are model-controlled, prompts user-controlled,
  resources application-controlled. The floor exposes verbs as tools; the
  bare status renders could be resources later, which is a separate
  decision. [EVIDENCE: spec, via the earlier note]

### Options weighed

1. Adopt the official Go SDK (eight direct dependencies, `x/tools` among
   them; explicit sign-off required).
2. Extend the hand-rolled stdio server: one dispatcher that answers
   `initialize` and `server/discover`, a per-verb tool table generated from
   the core's result types, `outputSchema` and `structuredContent`, tool
   annotations from the verb's read/write class; the conformance harness run
   as a lab lane (it needs Node, a tool dependency for a test, not a Go
   dependency).
3. A thin seam over option 2 whose transport and framing could be swapped
   for the SDK later, keeping the tool table and the verb mapping.

### Recommendation

**Path 2 — a native stdio server shaped to the 2026-07-28 specification, with
the official Go SDK as the named later replacement for the framing layer.**
Option 3. Grounds: the binary already carries a working server and client on
the standard library; the floor needs stdio only; the SDK's dependency weight
buys Streamable HTTP and OAuth the floor does not use; and the spec's
stateless redesign makes the hand-rolled server *simpler*, not harder (no
session state, a `server/discover` answer and per-request `_meta`). Answers
to the draft's open questions: the v1 tool list is every read verb plus the
mutating verbs whose input is a single record id or quoted text and whose
refusals are already structured (`capture`, `decide`, `reserve`), each
carrying `readOnlyHint`/`destructiveHint`; the repository is resolved per
call from a `cwd` argument defaulting to the server's launch directory
(hosts register one server per project scope, so launch cwd is the common
case and the argument is the escape); tool names use only `[a-z0-9_]` so
every host's prefixing leaves them legible. If the planning interview picks
the SDK instead, that is a `go get` needing the maintainer's explicit
sign-off before anything is fetched.

### Local fit risks

- [spec-moves-with-the-surface](../../principles/spec-moves-with-the-surface.md):
  the brief's surface registry and `internal/README.md` both name
  `surface/mcp/` as "later"; the row lands in the same change.
- [unrecognized-input-never-writes](../../principles/unrecognized-input-never-writes.md):
  MCP says input validation failures are tool errors the model may retry; a
  mutating verb must validate before any write, and the error text must not
  echo the rejected value.
- [one-canonical-primitive](../../principles/one-canonical-primitive.md): the
  guard's approve server and the floor must become one server package, not
  two dispatchers.
- Conformance is a Node tool; running it in a lab is a tool dependency the
  lab discipline must name, never a build step.

## itd-23 — lifeboats speak to external tools (Spec Kit interop)

### Findings

- **Spec Kit reached 1.0 and grew machine-readable state.** v1.0.0 was
  released on 2026-08-21 and v1.2.3 was current on 2026-10-22 (the
  repository's releases; the 2026-10-16 date in its news is a video about
  1.0); the quickstart now records the active feature in
  `.specify/feature.json` ("Commands resolve the feature from that state,
  not from the checked-out Git branch"), overridable by
  `SPECIFY_FEATURE_DIRECTORY`; the artefacts are `spec.md`, `plan.md`,
  `tasks.md` and `checklists/requirements.md`; the sequence is
  `/speckit-constitution` → `specify` → `clarify` → `plan` → `checklist` →
  `tasks` → `analyze` → `implement` → `converge`, and `converge` "checks the
  codebase against the spec, plan, and tasks", appending gap tasks to
  `tasks.md` until it reports "Converged". The site claims 38 integrations
  and advertises presets and extensions that tune the core templates, so the
  template shape is overridable per project. This supersedes the June note's "no machine-readable state" in one respect
  (the feature pointer) and confirms it in every other (artefacts are
  Markdown). [EVIDENCE: docs]
  ([github/spec-kit](https://github.com/github/spec-kit),
  [releases](https://api.github.com/repos/github/spec-kit/releases),
  [quickstart](https://github.github.io/spec-kit/quickstart.html),
  [site](https://github.github.io/spec-kit/))
- **The spec template is the mapping target, overridable by presets.** Header (feature name,
  branch `###-feature-name`, created, status, input); user stories with
  priority P1…, each with acceptance scenarios in Given/When/Then; edge
  cases; functional requirements `FR-001…` with inline
  `[NEEDS CLARIFICATION: …]`; key entities; measurable, technology-agnostic
  success criteria `SC-001…`; assumptions. The tasks template is one
  checkbox line per task, `- [ ] T010 [P] [US1] Description with exact file
  paths`, in phases (Setup, Foundational, one per story, Polish), with
  ordering rules in prose. [EVIDENCE: template source]
  ([spec-template.md](https://raw.githubusercontent.com/github/spec-kit/main/templates/spec-template.md),
  [tasks-template.md](https://raw.githubusercontent.com/github/spec-kit/main/templates/tasks-template.md))
- **The page the earlier note cited as the layout authority has moved on.**
  The SDD concepts page now says Spec Kit "does not prescribe how teams
  preserve or mutate" `spec.md`, `plan.md` and `tasks.md` after requirements
  change and points to a separate persistence-models page; the feature
  folder's parent (`specs/` in the earlier note) is not stated on the pages
  read. [EVIDENCE: docs]
  ([SDD concepts](https://github.github.io/spec-kit/concepts/sdd.html))
- **Kiro is the same three-file shape under another name.** Every Kiro spec
  "generates three key files": `requirements.md` (or `bugfix.md`),
  `design.md`, `tasks.md`; tasks are "updated as in-progress or completed"
  and run in dependency "waves". The page does not name EARS or a directory,
  and the concepts page returned 404. [EVIDENCE: docs, partial]
  ([Kiro specs](https://kiro.dev/docs/specs/))
- **BMAD's v7 preview ships as skills; its stable release does not.** The
  stable release is v6.12, "installed by the npm installer and frozen aside
  from bug fixes" (`npx bmad-method install`); v7 is "in preview alpha" with
  "Expect breaking changes between updates", installed through the Skills
  CLI, a Claude Code plugin or a Codex plugin; MIT. Its artefact names are
  not on the repository page. [EVIDENCE: repo]
  ([BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD))
- **OpenSpec is a second documented spec-driven layout.** Fission-AI's
  OpenSpec (MIT, "works with 30+ AI assistants via slash commands") keeps
  live specs under `openspec/specs/`, gives each change its own
  `openspec/changes/<name>/` with `proposal.md`, a `specs/` directory,
  `design.md` and `tasks.md`, archives completed changes under
  `openspec/changes/archive/`, and writes requirement deltas as plain
  Markdown: `## ADDED Requirements`, `### Requirement: ...`, and
  `#### Scenario:` blocks with `**WHEN**` and `**THEN**` bullets. A natural
  second test of whether the normalised exchange record is format-neutral.
  [EVIDENCE: repo] ([OpenSpec](https://github.com/Fission-AI/OpenSpec))
- **Tessl's format is in closed beta with three variants in its own
  registry** (search snippets only; its reference page was not opened), so
  it is not a target this cycle. [ANECDOTE/MARKETING]
- **Interop at the record level follows the onboarding shape**: detect,
  propose one disposition per item, record the answers, write once; no
  flow surveyed auto-merges drift. [CONSENSUS, via the onboarding note]

### Options weighed

1. Adapter per external tool, each mapping straight to intents (the draft's
   shape, with Linear/Jira/Notion stubs).
2. One normalised exchange record (title, narrative, prioritised scenarios
   in Given/When/Then, numbered requirements, measurable criteria, tasks as
   an ordered list with story tags), with Spec Kit's 1.0 templates as the
   first and only adapter; intent ↔ normalised ↔ spec-kit, tasks landing in
   abcd's spec store, not in the intent.
3. Import only (Spec Kit → draft intents), export deferred until a shipped
   intent has been asked for in Spec Kit form.

### Recommendation

**Path 2 — a native normalised exchange record shaped to Spec Kit 1.0's spec
and tasks templates, pinned to the template shape rather than to a CLI
version.** Option 2, import first. Answers to the draft's open questions:
tasks map to abcd's spec store (a planned intent's spec), round-tripped as
`T###` lines with `[P]` and `[US#]` preserved verbatim in the spec's task
list; personas go out as `metadata` the template ignores and come in as the
default persona; versioning pins the adapter to the fields it reads (`FR-`,
`SC-`, Given/When/Then, `T###`) and declines a document missing them with
the supported-shape list, not to a release number, which also covers a
project whose presets have replaced the stock template. The Linear/Jira/Notion
stubs found no documented import format in this pass and are left to the
planning interview with that fact on the table.

### Local fit risks

- [less-but-better](../../principles/less-but-better.md): three stub
  adapters with no caller are dead scaffolding under "wired or it isn't
  done".
- [host-delegated LLM](../../decisions/adrs/0025-host-delegated-llm-default.md):
  "press releases auto-drafted from the spec content" is model work; the
  core maps fields, the host drafts prose.
- [the-users-directory-is-theirs](../../principles/the-users-directory-is-theirs.md):
  an export writes under the destination the person names, never into a
  `.specify/` the person did not create.

## itd-7 — lifeboats carry RepoPrompt workspace definitions forward

### Findings

- **RepoPrompt's documentation is readable as plain text, and publishes no
  workspace file schema.** `repoprompt.com/docs`, `/docs/workspaces` and
  `/docs/mcp` returned only a page title (the content is script-rendered),
  but the same documentation is served whole at `repoprompt.com/llms.txt`
  (read in the 2026-10-23 adjudication). It documents a `manage_workspaces`
  MCP tool with the actions "list | switch | create | delete | add_folder |
  remove_folder | list_tabs | select_tab | create_tab | close_tab", of which
  "Create, delete, and folder operations modify workspace state"; it
  describes no on-disk workspace file or its fields; and it says "Presets
  save and restore your complete context configuration - file selection,
  prompt, and app settings", so selections and prompts persist through
  presets, not workspaces. A third-party write-up (mid-2026) adds that the
  app "runs entirely locally" and that MCP access needs the Pro plan.
  [EVIDENCE: vendor docs for the tool; the file format stays unverified]
  ([RepoPrompt documentation](https://repoprompt.com/llms.txt),
  [third-party write-up](https://rywalker.com/research/repo-prompt))
- **Apple's guidance makes another app's Application Support a private
  surface.** Application Support "contains all app-specific data and support
  files … that your app creates and manages on behalf of the user", under a
  subdirectory named by the app's bundle identifier; the guide says nothing
  about other apps reading or writing there. Writing a workspace entry into
  RepoPrompt's directory on embark is writing into a store whose format and
  invariants abcd does not own. [EVIDENCE: vendor guide]
  ([Apple file system guide](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/MacOSXDirectories/MacOSXDirectories.html))
- **Reading an undocumented file calls for the tolerant reader, not a
  schema.** Fowler: take only the elements you use, make minimal structural
  assumptions, confine parsing to one place; schema-bound readers "tend to
  break when the provider adds a field". [EVIDENCE: pattern]
  ([Tolerant Reader](https://martinfowler.com/bliki/TolerantReader.html))
- **The route abcd already has to RepoPrompt is MCP, loopback only**
  (`internal/adapter/repoprompt`): it connects and nothing else, starts no
  process and reads no file. [EVIDENCE: repository]

### Options weighed

1. As planned: parse the workspace file against a JSON Schema abcd ships,
   normalise paths, write it back into Application Support on embark.
2. Carry the file as an opaque, provenance-stamped artefact (source app,
   app version if readable, capture time), rewrite only path strings it can
   prove are home-rooted, and on embark *offer* it to the person (and to
   RepoPrompt through its MCP tools where a registering tool exists), never
   writing into another app's directory.
3. Drop the record until RepoPrompt publishes a workspace format.

### Recommendation

**Path 2 — opaque carriage with a tolerant reader, shaped to Fowler's
pattern, leaving registration to RepoPrompt's own interface.** Option 2,
with that interface now concrete: `manage_workspaces` can create a workspace
by name and add folders to it, and cannot restore a selection or a prompt.
This contradicts the planned record's in-scope line that a JSON Schema ships
with the adapter and that embark writes into `~/Library/Application
Support/RepoPrompt/Workspaces/`; the record stands as ruled (waiting on
itd-6), and the SOTA section says where the research disagrees so the
spec-writing interview can weigh it. The `_source` provenance field the
record proposes is supported; the UUID-clash policy becomes moot if abcd
does not write the store.

### Local fit risks

- [the-users-directory-is-theirs](../../principles/the-users-directory-is-theirs.md)
  and Apple's guidance pull the same way: abcd never writes another app's
  Application Support.
- [a-remedy-is-filed-whole](../../principles/a-remedy-is-filed-whole.md):
  the record's two missing prerequisites (dev-sync, a non-record embark
  family) are the real blockers and this research does not remove them.
- A schema abcd ships for a format it does not own fails on the vendor's
  next release and reads as abcd's defect.

## itd-2609292109214516 — rule injection as a seam; a CARL back end

### Findings

- **CARL is identifiable and documented.** "Context Augmentation &
  Reinforcement Layer" (ChristopherKahler/carl, MIT): a hook at
  `~/.claude/hooks/carl-hook.py` registered in `~/.claude/settings.json`
  scans each prompt against each domain's `recall` keywords and injects the
  matching rules; star-commands (`*brief`) are explicit triggers under
  `config.commands`; configuration is `carl.json` with `version: 1`,
  `config` (`devmode`, `context_brackets`, `commands`), `domains` (each with
  `state`, `always_on`, `recall`, `exclude`, `rules` of `{id,text,added}`,
  `decisions` of `{id,decision,rationale,date,recall}`) and `staging`;
  global config in `~/.carl/`, project in `./.carl/`, merged up the tree
  with project domains overriding global; re-injection skipped while the
  signature is unchanged and forced every five prompts; an MCP server
  `carl-mcp` in `~/.mcp.json` exposes `carl_v2_list_domains`,
  `carl_v2_create_domain`, `carl_v2_toggle_domain`, `carl_v2_add_rule`,
  `carl_v2_replace_rules`, `carl_v2_log_decision`, `carl_v2_stage_proposal`,
  `carl_v2_approve_proposal`, `carl_v2_get_config`, `carl_v2_update_config`
  and others; install `npx carl-core`; no release number on the page; a
  v1→v2 migration script exists. The npm page returned 403. [EVIDENCE:
  repository README] ([ChristopherKahler/carl](https://github.com/ChristopherKahler/carl))
- **CARL's design is the same as abcd's loader, with two extras abcd keeps
  elsewhere.** Domain-keyed keyword recall at prompt submission, explicit
  activation, dedup across a session: shared. `decisions` with their own
  recall and a `staging` queue for proposed rules: CARL carries them in the
  rules file; abcd carries decisions in `abcd decide` records and has no
  staging. The seam therefore covers rules only, as the draft scopes it.
  [EVIDENCE: repository README and `internal/core/rules`]
- **The ecosystem churns.** A separate project (AIOX, SynkraAI) advertises a
  "SYNAPSE" engine that "replaces the legacy CARL system", though its
  repository page mentions neither; whether that is the same CARL is not
  established. A back end selected by name should be checked for presence
  and version at session start, as the draft's loud fallback already
  requires. [CONTESTED: one vendor page against silence]
  ([SynkraAI/aiox-core](https://github.com/SynkraAI/aiox-core))
- **Two hooks on one event both run.** Claude Code runs every matching hook
  for `UserPromptSubmit`; Codex "loads all matching hooks" across layers.
  With CARL selected, abcd's own prompt hook still fires and must return
  nothing for the domains CARL carries, or the person gets both.
  [EVIDENCE: docs] (the hook pages under the cluster findings)
- **CARL is Claude Code-only as documented** (its hook path and
  `~/.mcp.json` are that host's); the seam's "native loader stays the floor"
  is what keeps the other hosts whole. [EVIDENCE: repository README]

### Options weighed

1. abcd writes CARL's project layer: renders its resolved domains into
   `./.carl/carl.json` (a derived, marked file regenerated on change) and
   its hook exits silently for those domains while CARL is selected.
2. abcd drives CARL's MCP tools (`carl_v2_create_domain`,
   `carl_v2_replace_rules`) at session start to register its domains.
3. CARL reads abcd's rules files (needs a change in CARL; out of abcd's
   hands).

### Recommendation

**Path 2 — native seam, CARL back end shaped to CARL's documented project
layer.** Option 1. Grounds: a file abcd generates is inspectable,
version-controllable and needs no running MCP server at prompt time; option
2 couples a hook to a tool call per session and to CARL's tool names, which
have already been through one migration. Answers to the draft's open
questions: the back end is selected in the machine-wide file only (it names
a tool installed on this machine; a repository cannot know), with the
repository file able to force `native`; domains reach CARL as rendered
`carl.json` files carrying abcd's domain names, `recall` keywords and rules
verbatim and a marker that they are generated, layered as CARL layers them:
the bundled defaults and the machine-wide `~/.abcd.noindex/rules.json`
domains into `~/.carl/`, the repository's own domains into `./.carl/`, so a
machine-wide (possibly private) rule never lands in a working tree where it
can be committed; `abcd rules` renders the same resolved set and adds one
line naming the back end in force and the files it wrote. The kill switch
stops the render and removes the generated files.

### Local fit risks

- [the-users-directory-is-theirs](../../principles/the-users-directory-is-theirs.md):
  `./.carl/carl.json` and `~/.carl/carl.json` may already be the person's own
  files; abcd writes only when a file is absent or carries abcd's marker,
  else refuses and says so.
- itd-2610030814013772's decision 7 ("abcd writes no tool-specific
  conventions file for any tool"): whether a generated `./.carl/carl.json` in
  the repository is such a file is for the fit-challenge; if it is, the
  render may have to stay in `~/.carl/` or be gitignored.
- [one-canonical-primitive](../../principles/one-canonical-primitive.md):
  the rules stay authored once in abcd's files; the CARL file is derived,
  never edited.
- [enforcement-claims-are-facts](../../principles/enforcement-claims-are-facts.md):
  "the domain's rules reach the session through CARL" is CARL's behaviour,
  which abcd cannot prove from inside its own hook; the acceptance criterion
  should be a canary word the way the AGENTS.md re-check proposes.

## itd-51 — harness adoption readiness rubric

### Findings

- **There is now an industry taxonomy to group the rubric by.** OWASP's
  Top 10 for Agentic Applications (published 2025-12-09 by OWASP's GenAI
  Security Project; the vendor summary calls it "peer-reviewed", meaning
  expert-reviewed by its contributors, not refereed) names ASI01 Agent
  Goal Hijack through ASI10 Rogue Agents; the categories nearest this
  intent's 2026-06-02 findings are ASI02 Tool Misuse, ASI03 Identity and
  Privilege Abuse, ASI05 Unexpected Code Execution, ASI08 Cascading Failures
  and ASI09 Human-Agent Trust Exploitation (an agent's summary misleading
  the approver, which is the forged review verdict). The official page gave
  only a download link; the category names were read from a vendor summary
  dated 2026-07-21. [EVIDENCE for the publication; the list itself is
  second-hand]
  ([OWASP resource page](https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/),
  [vendor summary](https://cycode.com/blog/owasp-top-10-agentic-applications/))
- **A runtime-control standard exists and is honest about its state.** The
  Agent Control Standard v0.1.0 (OWASP GenAI, Apache-2.0 code) is "a wire
  specification": an Observed Agent sends hook requests, a Guardian Agent
  returns one of `allow`, `deny`, `modify`, `ask`, `defer`; nineteen
  lifecycle hooks of which the reference implementation evaluates two
  (`steps/toolCallRequest`, `steps/toolCallResult`); an AgBOM (CycloneDX,
  SPDX or SWID); each decision an OpenTelemetry span and an OCSF event;
  conformance by self-declaration ("nobody verifies a self-declared claim in
  v0.1.0"); the reference Guardian "does not yet implement the required
  HMAC-SHA256 envelope signing" and "its default failure posture is
  fail-open". The vocabulary is usable; the implementation is not a gate.
  [EVIDENCE: repository README]
  ([agent-control-standard](https://github.com/GenAI-Security-Project/agent-control-standard),
  [OWASP ACS page](https://genai.owasp.org/resource/agent-control-standard-acs/))
- **Scorecard is the model for a scored checklist that admits its limits.**
  Each check returns 0–10; checks carry a risk level (Critical 10, High
  7.5, Medium 5, Low 2.5) that weights the aggregate; "the checks themselves
  are heuristics; there are false positives and false negatives";
  inconclusive is a result (`?`); output is text or JSON with
  `--show-details`. [EVIDENCE: docs] ([ossf/scorecard](https://github.com/ossf/scorecard))
- **The bounds the rubric asks for exist as host flags, each with a stated
  gap.** Claude Code: `--max-turns` ("No limit by default"),
  `--max-budget-usd` checked against "its client-side cost estimate, which
  can differ from your bill" and which "spend can pass", `--permission-prompts
  none` (denies what nobody can answer), `--restricted` (removes command and
  code tools, confines file tools, refuses `bypassPermissions`), `--bare`,
  and a sandbox that "covers shell commands only" while "file tools, MCP
  servers, and hooks run outside it", with an unsandboxed retry escape hatch
  that an allow rule approves silently. Codex: `read-only`,
  `workspace-write`, `danger-full-access`; approval `on-request`, `never`,
  `granular`; network off by default; Seatbelt on macOS, `bwrap` plus
  `seccomp` on Linux; "safety monitoring doesn't replace sandboxing,
  permissions, or review of the result"; the network proxy does not filter
  MCP connections or the client's own model calls. [EVIDENCE: docs]
  ([Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference),
  [Claude Code sandboxing](https://code.claude.com/docs/en/sandboxing),
  [Codex approvals and security](https://learn.chatgpt.com/docs/agent-approvals-security))
- **Every vendor draws the same line between steering and enforcing** (the
  security note's cross-cluster finding): a rubric item is only "pass" when
  an OS or wrapper boundary enforces it; a permission list is "conditional"
  at best. [CONSENSUS, via the security note]
- **Verdict integrity has a measured ceiling in the literature.** The best
  of thirteen models localised the step responsible for an agent
  hallucination in 41.1% of 693 human-annotated trajectories (AgentHallu,
  January 2026), 11.6% for tool-use faults. A reviewer reading a worker's
  transcript cannot be the sole verdict channel. [EVIDENCE: benchmark]
  ([arXiv 2601.06818](https://arxiv.org/abs/2601.06818))

### Options weighed

1. A prose rubric in a research note, scored by hand.
2. An ADR-shaped standing decision whose items are written as checks
   (claim, how verified, enforcing layer), grouped by the ASI categories
   they answer, scored pass / conditional / fail in Scorecard's spirit with
   "inconclusive" allowed, each conditional naming the wrapper compensation.
3. Option 2 plus machine probes reusing `ahoy`'s detection layer for the
   checks that are flag or version facts (which flags exist, sandbox
   dependencies present), leaving behavioural checks to a lab.

### Recommendation

**Path 2 — a native rubric shaped to OWASP's agentic taxonomy for grouping,
ACS's decision vocabulary for naming what a wrapper can do (allow, deny,
modify, ask, defer), and Scorecard's scoring shape.** Option 2 now, option 3
when a second candidate appears. Answers to the draft's open questions: it
lives as an ADR (a standing decision), with the per-harness instantiation
in that host's adoption intent; checks are manual this cycle, written so
the flag-and-version ones can become `ahoy` probes; scope stays
safety and integrity but adds one non-safety row, "the harness states the
limits of its own boundary", because every vendor page read does and a
candidate that does not is itself a finding; the rubric is the source of
truth a live spec cites, never a summary of one.

### Local fit risks

- [enforcement-claims-are-facts](../../principles/enforcement-claims-are-facts.md):
  a rubric row that reads "pass" on a vendor's prose is a claim; the check
  column must say what was run.
- [guards-prove-themselves](../../principles/guards-prove-themselves.md):
  a conditional pass that names a wrapper compensation obliges abcd to test
  that compensation, which is the spc-37-shaped work the draft keeps out of
  scope.
- ACS's fail-open reference guardian is the cautionary row: abcd's own
  wrapper defaults must be fail-closed, as the guard already is.

## itd-21 — start a new project with abcd conventions

### Findings

- **Language scaffolders are opinionated and current; a copy of their
  output goes stale.** `uv init` writes `.python-version`, `README.md`,
  `pyproject.toml` and `src/<pkg>/__init__.py` with `uv_build` as the
  default backend, `--lib` adds `py.typed`, `--bare` writes `pyproject.toml`
  alone, and it "exits with an error" when a `pyproject.toml` already
  exists. Embedding seven per-language templates means re-deriving each
  language's current layout by hand and shipping it frozen in a binary.
  [EVIDENCE: docs]
  ([uv init](https://docs.astral.sh/uv/concepts/projects/init/))
- **Copier is the scaffolder that handles the day after.** Answers are
  recorded in `.copier-answers.yml` (never edited by hand); `copier update`
  regenerates from the template's latest tag, diffs, reapplies the person's
  changes and marks conflicts inline or as `.rej`; `_skip_if_exists` lists
  files created only if missing; `_tasks` run after copy, `_migrations` only
  on update and gated by version; `copier check-update` has a JSON output
  and exit 2 when an update exists. [EVIDENCE: docs]
  ([Copier updating](https://copier.readthedocs.io/en/stable/updating/),
  [Copier configuring](https://copier.readthedocs.io/en/stable/configuring/))
- **GitHub templates are a one-shot copy without history.** A repository
  created from a template "starts with a single commit"; `gh repo create
  --template` with `--include-all-branches`, `--add-readme`, `--gitignore`,
  `--license`; branches from a template "have unrelated histories".
  [EVIDENCE: docs] ([gh repo create](https://cli.github.com/manual/gh_repo_create),
  [GitHub template repositories](https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-repository-from-a-template))
- **The adoption shape is settled by the earlier survey**: detect, propose a
  default the person accepts or overrides, leave what is not yours alone,
  record the answers so the next run does not re-ask, write once
  reversibly. [CONSENSUS, via the onboarding note]
- **A scaffolded project receives one conventions file, `AGENTS.md`**, read
  natively by Codex, Copilot, Cursor, opencode, Amp, Jules and Zed and by
  configuration in Gemini CLI and Aider; nothing per host. [EVIDENCE, via the
  AGENTS.md re-check]

### Options weighed

1. As drafted: per-language templates embedded in the binary, an interview,
   conventions, a starter intent.
2. Delegate the language layout to the language's own initialiser where one
   is installed (`uv init`, `cargo new`, `go mod init`, `npm init`), fall
   back to a minimal language-agnostic skeleton; abcd writes only its own
   artefacts (`AGENTS.md`, `.abcd/`, gitignore policy, the starter intent)
   and records the interview's answers in `.abcd/` so `ahoy` re-runs read
   them instead of asking.
3. Point people at Copier or `gh repo create --template` and ship an abcd
   template repository instead of a verb.

### Recommendation

**Path 2 — native scaffold shaped to Copier's answers-file discipline, with
the language layout delegated to each language's initialiser.** Option 2.
Answers to the draft's open questions: the unique value over `git init` then
`ahoy install` is the interview, the recorded answers and the starter intent,
not the file tree, so the verb is a thin composition and `ahoy` stays the
one installer; the starter intent is offered, not required, with the decline
recorded as the draft already says; the emptiness rule follows `uv init`:
refuse on an existing marker file (a manifest, a `.abcd/`), ignore `.git/`.
This contradicts the draft's in-scope line "templates committed and
versioned as assets embedded in the Go binary" and the Python criterion's
exact file list; the draft's "CLAUDE.md marker block" criterion is already
overridden by itd-2610030814013772's decision 7 (no tool-specific
conventions file for any tool), and its `.claude/settings.local.json` is one
host's file; the research is recorded and the interview decides.

### Local fit risks

- [one-canonical-primitive](../../principles/one-canonical-primitive.md): a
  second installer beside `ahoy install` is the risk the draft's first open
  question already names.
- [less-but-better](../../principles/less-but-better.md): seven language
  templates nobody maintains are the opposite of the opinion abcd sells.
- Delegating to `uv`, `cargo` or `npm` is a tool presence check at run time,
  never a dependency; absence falls back loudly to the plain skeleton.

## itd-19 — defaults that match where the project lives

### Findings

- **The detection source is documented and small.** `chezmoi data` writes
  "the computed template data to stdout" as JSON or YAML; the configuration
  file's `data` section is typed `object`, "Template data", so a `devPath`
  key is the person's own convention, not chezmoi's. [EVIDENCE: docs]
  ([chezmoi data](https://www.chezmoi.io/reference/commands/data/),
  [chezmoi configuration variables](https://www.chezmoi.io/reference/configuration-file/variables/))
- **The contrarian take is twelve-factor's.** Grouping config into named
  environments "does not scale cleanly: as more deploys of the app are
  created, new environment names are necessary"; settings should be
  "granular controls, each fully orthogonal", "never grouped together as
  'environments'". [CONTESTED: against the intent's premise]
  ([The Twelve-Factor App, config](https://12factor.net/config))
- **The counter-example is GitHub's deployment environments**, where a
  named environment carries its own protection rules (required reviewers,
  wait timer, branch restrictions), secrets and variables, and a job "must
  follow any protection rules for the environment before running". Named
  stages work when the set is small, fixed and each stage gates something.
  [EVIDENCE: docs]
  ([GitHub deployment environments](https://docs.github.com/en/actions/how-tos/deploy/configure-and-manage-deployments/manage-environments))
- **The answers-file pattern resolves the override question** (Copier,
  above): a profile applied once is recorded with its origin, and a later
  run proposes only the diff, asking per key where the person's own value
  differs. [EVIDENCE: docs, via itd-21]

### Options weighed

1. As drafted: four named profiles that set several keys at once, applied
   on confirmation, re-confirmed on a move.
2. Named profiles as a *proposal layer only*: each key stays an independent
   setting with its origin recorded (`profile:autonomous` or `person`), the
   profile is a template of proposed values, and a move shows the per-key
   diff with person-set keys asked about individually.
3. No stages: the four knobs stay independent and documented, and the
   ABCDevelopment convention lives in that one person's own chezmoi data.

### Recommendation

**Path 3 — deliberate divergence from twelve-factor's "no named
environments", bounded by option 2.** Named stages are kept because the set
is four, fixed, and each gates visibility and scanning, which is the GitHub
environments case; but the stage never owns a key: every key stays
orthogonal with its origin recorded, so the twelve-factor objection
(combinatorial growth, silent overwrite) is answered in the data model.
Answers to the draft's open questions: a person-set key is preserved and
asked about by name on a transition; "read-mostly" is a permission concern
outside this intent; the lifeboat carries the stage name in `meta` as the
draft proposes, as a hint the target may ignore.

### Local fit risks

- [the-users-directory-is-theirs](../../principles/the-users-directory-is-theirs.md):
  reading another tool's data (`chezmoi data`) is a presence-checked read,
  never a write, and absence is silent as the draft requires.
- [one-canonical-primitive](../../principles/one-canonical-primitive.md): a
  profile must not become a second config file; it is proposed values
  written into the one `.abcd/config.json` with origins.
- The convention is one person's; host-agnostic docs describe "a stage
  convention supplied by your dotfiles" and never a directory layout.

## itd-8 — lifeboats carry source code

### Findings

- **git has two first-party answers, for two different questions.** `git
  archive` writes "the tree structure for the named tree" (no history, no
  untracked files unless `--add-file`), honours `export-ignore` and
  `export-subst` from the tree's own `.gitattributes` (or the working tree's
  with `--worktree-attributes`), stores the commit id in a pax header, and
  takes a `--prefix`. `git bundle` packs objects *and refs* for offline
  transfer, `verify` checks a bundle "will apply cleanly", a self-contained
  bundle "can be extracted into anywhere, even into an empty repository",
  a range bundle needs its basis, and a bundle carries no index, working
  tree, stash, config or hooks. [EVIDENCE: docs]
  ([git archive](https://git-scm.com/docs/git-archive),
  [git bundle](https://git-scm.com/docs/git-bundle))
- **Scope can be declared in the repository, not invented by a curator.**
  `export-ignore` is the maintained, in-tree statement of "not part of the
  shipped tree"; a curator's per-file decision then starts from that
  baseline and records only its deviations. [EVIDENCE: docs] (git archive)
- **A payload with a per-file manifest has a standard.** BagIt (RFC 8493):
  `bagit.txt`, `data/`, at least one `manifest-<alg>.txt` with one checksum
  per payload file, optional `tagmanifest-<alg>.txt` and `bag-info.txt`;
  "complete" means every listed file exists and every payload file is
  listed, "valid" means every checksum verifies; SHA-512 recommended;
  manifests "are not designed to resist deliberate tampering". The
  `_manifest.json` the draft asks for is BagIt-shaped (one digest per payload
  file, the complete/valid distinction) plus a decision column BagIt does not
  have; a JSON file is not a `manifest-<alg>.txt` BagIt tooling would read.
  [EVIDENCE: standard] ([RFC 8493](https://www.rfc-editor.org/rfc/rfc8493.html))
- **The lifeboat's integrity manifest is already being shaped** (the
  security note, itd-35): a flat, sorted digest-per-file list in the in-toto
  subject shape, verified on extracted files so a directory and an archive
  verify alike. A `code/` family slots into that manifest as more rows, not
  a second mechanism. [EVIDENCE, via the security note]
- **The launch bundler is default-deny by construction**
  (`internal/core/launch/bundle.go`: "no include pattern can promote a
  denied path"); `code/` is a lifeboat family and never a launch payload
  unless named. [EVIDENCE: repository]

### Options weighed

1. Copy files by a curator's judgement into `code/` with a JSON manifest
   (the draft).
2. `git archive HEAD` scoped by `export-ignore` plus a curator override
   list, written as a BagIt-shaped tree under `code/` (`data/` plus a
   manifest row per file carrying digest and decision), size-reported.
3. Option 2 plus an optional `git bundle` of the branch for the fork-and-
   improve case that wants history, verified with `git bundle verify` on
   embark.

### Recommendation

**Path 2 — a native `code/` family shaped to `git archive` scoping and a
BagIt-shaped manifest, with a git bundle as the opt-in history carrier.**
Option 3, with one rule from loud staging: `git archive HEAD` carries the
committed tree only, and a lifeboat is packed when a project is being left
behind, which is when uncommitted and untracked work exists, so pack reports
every file the archive leaves out, or adds it with `--add-file` on the
person's say-so, and never drops one silently.
Answers to the draft's open questions: the scope model is "declared in the
tree, overridden by the person" (`export-ignore` baseline, curator
suggestions, per-file override recorded in the manifest with its reason);
`launch` is untouched because its default-deny already excludes a family it
does not name; size is reported before writing and compression is the
archive's own, no streaming until a lifeboat shows the need.

### Local fit risks

- [host-delegated LLM](../../decisions/adrs/0025-host-delegated-llm-default.md):
  the `code-rescuer` extension is model work; the core applies attributes
  and records decisions, the host suggests.
- [secret-shaped-fixtures-at-runtime](../../principles/secret-shaped-fixtures-at-runtime.md):
  source carries secrets more often than records do; the scanner that gates
  the launch bundle gates `code/` too, fail-closed.
- itd-9: a new family is a schema change; it ships with its revision and
  its reader, not before.

## itd-9 — old lifeboats still unpack

### Findings

- **The repository already has seven independent schema versions and one
  direction of refusal.** `probe.go` `SchemaVersion = 2`,
  `EmbarkSchemaVersion = 1`, `GraveyardSchemaVersion = 1`,
  `LessonsSchemaVersion = 1`, `PrinciplesSchemaVersion = 2`,
  `PressReleaseSchemaVersion = 1`, `ReviewSchemaVersion = 1`; embark refuses
  a `_provenance.json` newer than it knows and the lessons reader refuses
  any version but its own. There is no reader that *accepts* an older
  revision yet, which is the gap the intent names. [EVIDENCE: repository]
- **Upcasters are the pattern for read-time migration without rewriting.**
  Axon stores a revision with each event; an upcaster takes revision *x* and
  emits zero or more events at *x+1*; upcasters chain so "the output of one
  upcaster is sent to the input of the next"; the store is append-only and
  "the complete event history remains intact". A lifeboat is an immutable
  artefact in exactly this sense: migrate on read, never in place.
  [EVIDENCE: docs] ([Axon event versioning](https://docs.axoniq.io/axon-framework-reference/4.11/events/event-versioning/))
- **Kubernetes states the round-trip rule and separates stored from served
  versions.** Conversion is "a star with the internal form at the center";
  "it must be possible to round-trip your change (convert to different API
  versions and back) with no loss of information"; adding a field with a
  sane default needs no new version; objects are "always converted to a
  versioned form before being written to storage"; and a new version should
  not become both preferred and the storage version in one release, so a
  rollback can still decode. [EVIDENCE: docs]
  ([Kubernetes API changes](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md))
- **Protobuf's rules are the field-level discipline.** Never change or reuse
  a field number, reserve removed numbers and names, old readers "simply
  ignore the new field", unknown fields are preserved in binary but can be
  lost through JSON. [EVIDENCE: docs]
  ([protobuf updating a message type](https://protobuf.dev/programming-guides/proto3/#updating))
- **Go's decoder ignores unknown fields by default** ("object keys which
  don't have a corresponding struct field are ignored");
  `DisallowUnknownFields` makes them an error; the package now points new
  code at `encoding/json/v2`. abcd's `jsonstrict` already refuses duplicate
  keys, and the lifeboat readers refuse unknown keys on purpose
  (`graveyard_lessons.go`: `dec.DisallowUnknownFields() // reject smuggled
  extra fields`, and any `schema_version` but its own), so the
  tolerant-reader rule, ignore what you do not need and confine parsing to
  one place, applies here only in bounded form: known fields of an older
  revision accepted, unknown keys still refused where a reader refuses them
  today. [EVIDENCE: docs; repository]
  ([encoding/json](https://pkg.go.dev/encoding/json#Decoder.DisallowUnknownFields),
  [Tolerant Reader](https://martinfowler.com/bliki/TolerantReader.html))
- **Expand and contract is the writer's half.** Add the new shape beside the
  old, migrate readers, then remove the old; skipping the contract leaves
  two shapes forever. [EVIDENCE: pattern]
  ([Parallel Change](https://martinfowler.com/bliki/ParallelChange.html))

### Options weighed

1. A single lifeboat-level version with monolithic `1 → 2` migrators that
   rewrite every artefact.
2. Per-artefact revisions (as the seven constants already are) with a chain
   of single-step upcasters per artefact applied on read, the lifeboat
   envelope version in `_provenance.json` naming the set, the lifeboat on
   disk never rewritten, and the migration log listing each step applied.
3. Tolerant readers only (ignore unknown, default missing), no explicit
   migrators until a renamed or re-shaped field forces one.

### Recommendation

**Path 2 — native per-artefact upcaster chains shaped to Axon's revision
model, with Kubernetes' no-information-loss rule applied to each one-way
up-step.** Kubernetes' round trip serves an API server converting in both
directions; abcd's chain is one-way and newer-than-known stays refused, so
no downcaster is owed. Option 2, with a bounded form of option 3 as the
first rung: an older revision's known fields are accepted and missing ones
defaulted, so an additive change needs no upcaster, while unknown keys stay
refused where a reader refuses them today (the lessons reader's smuggled-field
guard is a deliberate defence, not a strictness to relax). Answers to the draft's open questions: re-disembark is required when
the chain has no step starting at the artefact's revision (not a
major-version rule; the chain is the fact); migrators live beside each
artefact's type in `internal/core/lifeboat` (one file per step, named by
`<artefact>_v<n>_to_v<n+1>`); `_provenance.json` is left as written and the
voyage store's `provenance.json` records `was_schema`, `now_schema` and the
steps, as the acceptance criterion already says. The newer-than-known
refusal stays exactly as it is.

### Local fit risks

- [one-canonical-primitive](../../principles/one-canonical-primitive.md):
  the probe's "is its schema version one this build understands" and
  embark's check must be the same function.
- itd-35's manifest: an upcast artefact has a different digest from the
  packed one; verification runs on the packed bytes, migration on the
  decoded value, and the log records both.
- [ratchet-not-big-bang](../../principles/ratchet-not-big-bang.md): no
  migrator is written before a schema step exists; the chain mechanism and
  the tolerant readers land first, empty.

## itd-11 — disembark survives noisy transcripts

### Findings

- **The local measurement is the baseline.** On 14 stratified transcripts
  from 439, one project's, decision turns were 22.3% of user turns (138/619)
  and 1.4% of all turns; the note rules "VIABLE" against a 15% floor on the
  user-turn denominator it adopted as a recorded measurement deviation, and
  records that on the spec's original all-turn metric the 1.4% was "NEEDS
  REDESIGN", now superseded; the small stub sessions were at zero. The noise
  this intent fears is real and quantified; the filter chain is tuned against
  this corpus or not at all. [EVIDENCE: local study]
  ([transcript sampling](transcript-sampling.md))
- **Compaction practice keeps decisions and drops tool output.** Anthropic's
  guidance: compaction "preserves architectural decisions, unresolved bugs,
  and implementation details" while "discarding redundant tool outputs or
  messages"; tool-result clearing is "the safest, lightest touch"; start by
  maximising recall, then raise precision; "overly aggressive compaction can
  result in the loss of subtle but critical context whose importance only
  becomes apparent later". [EVIDENCE: vendor engineering post, 2025-09-29]
  ([effective context engineering](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents))
- **Agents are less faithful to condensed experience than to raw.** Across
  four self-evolving frameworks, thirteen backbones and nine environments,
  agents "consistently depend on raw experience" and "often disregard or
  misinterpret condensed experience, even when it is the only experience
  provided" (January 2026). The study is about agents acting on their own
  experience, not about a summariser; by analogy, a distiller that reads a
  summary of a transcript rather than the transcript is the weaker design.
  [EVIDENCE: study; the application is an analogy]
  ([arXiv 2601.22436](https://arxiv.org/abs/2601.22436))
- **Verbalised confidence is overconfident; consistency helps.** LLMs
  stating confidence in words "tend to be overconfident"; human-inspired
  prompts, "consistency across multiple sampled responses" and better
  aggregation reduce it; no method wins everywhere; white-box methods lead
  by a narrow margin (ICLR 2024). A `[0.0, 1.0]` score the distiller
  announces about itself is not a calibrated number. [EVIDENCE: study]
  ([arXiv 2306.13063](https://arxiv.org/abs/2306.13063))
- **Locating the step responsible for an agent hallucination is hard even
  for the best models**: 41.1% step-localisation accuracy at best over 693
  annotated trajectories, 11.6% on tool-use hallucinations (January 2026).
  The benchmark measures hallucination attribution, not fault-finding in
  general; by analogy, a finding that cannot cite the transcript range it
  rests on is unreviewable. [EVIDENCE: benchmark; the application is an
  analogy] ([arXiv 2601.06818](https://arxiv.org/abs/2601.06818))
- **abcd's transcript store is already line-delimited and redacted**
  (`internal/core/history`: JSONL records under
  `~/.abcd.noindex/transcripts/<root-sha>/records/`, fail-closed on a
  surviving blocking span), and the hosts' `transcript_path` files are
  JSONL (the hook pages). Filtering by time window, keyword and record id is
  a deterministic pass over lines before any model reads them. [EVIDENCE:
  repository and docs]

### Options weighed

1. The draft's chain: four strategies with weights, a model-reported
   confidence per finding, quarantine below a threshold.
2. Structural confidence: each finding carries the anchors that selected
   its transcript range (time window hit, keyword, record id, semantic
   only) and the range itself; the score is a function of anchors present
   and of agreement across two independent distiller runs, never a
   self-report; semantic-only findings are quarantined by rule; quarantine
   is visible in the report with the reason.
3. Defer until a disembark on a real corpus shows the time window failing
   (the draft's first open question).

### Recommendation

**Path 2 — native structural confidence shaped to the calibration evidence:
anchors and run-to-run agreement, not a verbalised score.** Option 2, with
the regression check the draft already asks for measured against the sampled
corpus. Answers to the draft's open questions: the sampling note says the
window is viable but thin at 1.4% of all turns, so the chain is justified
but the first rung is deterministic; the threshold is a rule over anchors
(semantic-only is quarantined; keyword-or-id-anchored is promoted), not a
tuned number; low-confidence findings stay visible, each with its range and
reason, as the fifth acceptance criterion already says.

### Local fit risks

- [host-delegated LLM](../../decisions/adrs/0025-host-delegated-llm-default.md):
  "semantic relevance" is a model step; it runs through the host, and
  running the distiller twice for agreement doubles that cost.
- [prefer-the-experiment-to-the-inference](../../principles/prefer-the-experiment-to-the-inference.md):
  the threshold is set by the corpus run, not by this note.
- [a-model-run-is-re-runnable-not-reproducible](../../principles/a-model-run-is-re-runnable-not-reproducible.md):
  agreement across two runs is a signal, not a proof; the report says so.

## Not worth adopting

- The official Go MCP SDK for a stdio-only floor: eight direct modules,
  including `x/tools` and OAuth, for framing the binary already does by
  hand; revisit if a Streamable HTTP front door is ever decided.
- The Agent Client Protocol as a floor: it makes abcd an agent inside an
  editor, which is not what the front door is for.
- Per-host conventions files or plugin manifests carrying prose: hosts load
  them every turn and the cited study found cost without gain.
- Copier or `gh repo create --template` *as* the scaffold verb: a one-shot
  copy, no interview, no answers record abcd can read later.
- Monolithic lifeboat migrators that rewrite the artefact on disk: the
  upcaster pattern exists to avoid that.
- A model-reported confidence number on distiller findings: overconfident
  by measurement.
- Writing into RepoPrompt's Application Support on embark: another app's
  private store, undocumented format.
- Linear, Jira and Notion stub adapters this cycle: no documented import
  format was found, and an interface with no caller is dead scaffolding.

## Not verified

- RepoPrompt's workspace file format and its storage path: the plain-text
  documentation (`llms.txt`) names the MCP tools and the `manage_workspaces`
  actions but describes no on-disk workspace file.
- Which MCP revision Claude Code, Codex and Gemini CLI negotiate today; the
  conformance framework's note that dated versions through 2025-11-25 are
  stateful is the nearest fact.
- The Agent Skills client list: the format's showcase page rendered empty;
  adoption is inferred from the three vendors' plugin pages.
- The ten OWASP ASI categories were read from a vendor summary, not from
  the OWASP document, which is a download.
- Whether AIOX's "legacy CARL" is ChristopherKahler/carl; CARL's current
  release number (its npm page returned 403); the hook event CARL
  registers under.
- Spec Kit's feature-folder parent directory on 1.0 and the full content of
  `.specify/feature.json`; Kiro's spec directory and requirement notation
  (concepts page 404); Tessl's reference page (search snippets only).
- The exact Codex plugin manifest fields beyond the build guide's list; the
  Codex MCP page was read for the CLI surface only.
- The AgentHallu and faithfulness figures are taken from the arXiv
  abstracts, not from the papers' bodies.
- Web search was exhausted part-way through the pass (a shared session
  budget), so the later angles (calibration surveys, RepoPrompt, Tessl)
  were covered by direct page fetches only.
