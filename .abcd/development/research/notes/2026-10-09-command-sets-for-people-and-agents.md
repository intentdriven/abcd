# How well-designed developer tools divide, name and size their commands

Research note. Date: 2026-10-09. Written for the verb audit (iss-2610090831317531) behind itd-2610090831227812.

Scope: how mature CLIs and agent plugin systems split person-facing commands
from machine-facing ones, how many commands they show, how they name them,
and whether one source feeds both surfaces. Primary sources only: official
docs, specifications, design guides, upstream source files, and the real help
output of installed tools.

**Method and limits.** Web pages were read with WebFetch, which summarises
through a small model. Where its summary disagreed with the source, the
source won, and the disagreement is recorded below as falsified. Three sources
were downloaded raw so they could be quoted exactly: the clig.dev Markdown, the
Primer CLI foundations `.mdx`, and the `root.go` / `alpha.go` / `command-list.txt`
files from gh, kubectl and git. Help-output counts come from the tools
installed on this machine. The captures are kept next to this note
(`git-help.txt`, `git-help-a.txt`, `gh-help.txt`, `kubectl-help.txt`,
`docker-help.txt`, `docker-help-hidelegacy.txt`, `go-help.txt`,
`npm-help.txt`). Versions: git 2.52.0, gh 2.100.0 (2026-09-03), kubectl client
v1.34.1, Docker 28.5.1, go1.27.1, npm 11.17.0. Counts depend on version and
on which plugins are installed. For example, no docker CLI plugins (buildx,
compose) appear in this docker help.

Status key: **V** verified against the primary source; **F** falsified (a
secondary or summarised claim the source contradicts); **U** unverifiable
within budget.

---

## Q1. Separating a person's commands from machine-facing ones

### Finding 1.1: git splits commands by stability as well as by audience (V)

git sorts commands into "porcelain", for people, and "plumbing", for scripts
and alternative front ends. It also gives plumbing a stronger stability
promise:

> "The interface (input, output, set of options and the semantics) to these
> low-level commands are meant to be a lot more stable than Porcelain level
> commands, because these commands are primarily for scripted use. The
> interface to Porcelain commands on the other hand are subject to change in
> order to improve the end user experience."

Source: https://git-scm.com/docs/git

### Finding 1.2: git's help is tiered, and all tiers come from one classification file (V)

There are three tiers:

- Bare `git` / `git --help` shows 23 "common" commands in 5 task-shaped
  groups: "start a working area", "work on the current change", "examine the
  history and state", "grow, mark and tweak your common history",
  "collaborate".
- `git help -a` lists every command under 11 headings: Main Porcelain,
  Ancillary Manipulators/Interrogators, Interacting with Others, Low-level
  Manipulators/Interrogators/Syncing/Internal Helpers, two interface-guide
  headings, and External commands.
- `git help -g` lists the concept guides.

`command-list.txt` drives all of these. Its header says the list "is used to
list commands in various places". Categories include `mainporcelain`,
`plumbingmanipulators` and others. `mainporcelain` entries can also carry an
`init`, `worktree`, `info`, `history` or `remote` tag, which places them in a
group of the common list. A `complete` attribute feeds shell completion.

Sources: local `git --help` and `git help -a` (git 2.52.0);
https://raw.githubusercontent.com/git/git/master/command-list.txt

### Finding 1.3: kubectl uses ordered task groups, hides by filtering, and parks alpha features under a separate root (V)

**Groups.** The help shows 7 named groups in task order: "Basic Commands
(Beginner)", "Basic Commands (Intermediate)", "Deploy", "Cluster Management",
"Troubleshooting and Debugging", "Advanced", "Settings". It then shows
"Subcommands provided by plugins" and "Other Commands".

**Hiding.** Hiding is done by filtering. `filters := []string{"options"}`
drops `options` from the list, although the help footer still points at
`kubectl options`. `alpha` is filtered out when it has no subcommands.

**Alpha.** `alpha` is "an alternate root command for features in alpha", and
its long text reads: "These commands correspond to alpha features that are not
enabled in Kubernetes clusters by default." Typed with no alpha features
present, it prints "No alpha commands are available in this version of
kubectl". The local client printed exactly that.

**Scripting.** The machine contract is a set of conventions, not separate
commands: "Request one of the machine-oriented output forms, such as
`-o name`, `-o json` …", "Fully-qualify the version", "Don't rely on context,
preferences, or other implicit states."

Sources: https://raw.githubusercontent.com/kubernetes/kubectl/master/pkg/cmd/cmd.go;
https://raw.githubusercontent.com/kubernetes/kubectl/master/pkg/cmd/alpha.go;
https://kubernetes.io/docs/reference/kubectl/conventions/; local `kubectl --help`.

### Finding 1.4: gh uses cobra groups, hidden stubs, help topics, and one raw-API escape hatch (V)

**Groups.** `root.go` declares groups with `AddGroup`: "Core commands",
"GitHub Actions commands", "Extension commands", plus an "Alias commands"
group added when aliases exist. Ungrouped commands print under "ADDITIONAL
COMMANDS".

**Hidden commands.**

> "Official extension stubs: hidden commands that suggest installing
> GitHub-owned extensions when invoked."

**Help topics.** Help topics (`environment`, `exit-codes`, `formatting`,
`reference` …) are registered as commands and listed under HELP TOPICS.

**Machine surface.** `api` ("Make an authenticated GitHub API request") is the
raw machine-facing surface, and the help topic `formatting` covers JSON
output.

**Design guide.** GitHub's CLI design guide (Primer) has a "Scriptability"
section:

> "Create flags for anything interactive … Consider what should be different
> for terminal vs machine output".

It lists what changes in machine output: "No color or styling … Tabs between
columns … No truncation … Exact date format … No header".

Sources: https://raw.githubusercontent.com/cli/cli/trunk/pkg/cmd/root/root.go;
https://raw.githubusercontent.com/primer/design/main/content/native/cli/foundations.mdx
(the rendered page at primer.style/native/cli/foundations now redirects to a
404); local `gh --help`.

### Finding 1.5: docker groups commands and offers an opt-in to hide its legacy verbs (V)

Help groups: "Common Commands" (12), "Management Commands" (11), "Swarm
Commands" (6) and a plain "Commands" group (28) of legacy top-level verbs. The
docs say:

> "When set, Docker hides "legacy" top-level commands (such as `docker rm`,
> and `docker pull`) in `docker help` output, and only `Management commands`
> per object-type (e.g., `docker container`) are printed. This may become the
> default in a future release."

Locally, setting `DOCKER_HIDE_LEGACY_COMMANDS=1` removed the 28-command group,
leaving 29 listed. The docs also say experimental CLI features have been
enabled by default since Docker 20.10.

Sources: https://docs.docker.com/reference/cli/docker/; local `docker --help`
with and without the variable.

### Finding 1.6: clig.dev puts people first, and machines get flags, not a second command set (V)

Quotes:

- "Human-readable output is paramount. Humans come first, machines second."
- "Have machine-readable output where it does not impact usability."
- `--json` and `--plain` flags.
- "Changing output for humans is usually OK … Encourage your users to use
  `--plain` or `--json` in scripts to keep output stable."

The guide has no rule about hidden commands, no command-count rule, and no
mention of AI agents. A grep of the source for "hidden", "undocumented",
"agent" and "LLM" finds nothing relevant; the only "agent" hit is the tool
name yubikey-agent.

Source: https://clig.dev/ (raw:
https://raw.githubusercontent.com/cli-guidelines/cli-guidelines/main/content/_index.md)

### Finding 1.7: cobra provides the hiding and grouping mechanics (V)

Field comments in `command.go`:

- `Hidden`: "defines, if this command is hidden and should NOT show up in the
  list of available commands"
- `Deprecated`: "should print this string when used"
- `GroupID`: "The group id under which this subcommand is grouped in the
  'help' output of its parent"
- `type Group struct { ID, Title }`

`IsAvailableCommand()` returns false when a command is hidden or deprecated,
so such a command drops out of help (and, by the same check, out of
completion). Groups are declared with `AddGroup()` and appear "in the order
defined".

Sources: https://raw.githubusercontent.com/spf13/cobra/main/command.go;
https://raw.githubusercontent.com/spf13/cobra/main/site/content/user_guide.md

### Finding 1.8: How these tools show the split in help (V, synthesis of 1.2 to 1.7)

| Technique | Tools |
|---|---|
| Short curated list by default, full list behind a flag | git (`git help -a`), npm (4 usage lines, then "All commands" with 71 names, and `npm -l` for full usage) |
| Named groups in task order | kubectl, gh, docker, git |
| Hidden-but-callable commands | gh extension stubs, kubectl `options` / empty `alpha`, cobra `Hidden` |
| Opt-in to hide legacy verbs | docker `DOCKER_HIDE_LEGACY_COMMANDS` |
| Help topics as pseudo-commands | gh HELP TOPICS (8), go "Additional help topics" (20), git guides (`-g`) |
| Separate root for unstable features | kubectl `alpha` |

None of the five ships a second binary for machines. The machine contract is
expressed in one of three ways:

- stable plumbing commands (git);
- output flags (`-o json`, `--json`, `--plain`);
- a raw API command (`gh api`).

### Finding 1.9: Falsified claims met along the way (F)

- WebFetch's summary of git-scm.com/docs/git said there are 57 "Main
  porcelain commands". **F.** Upstream `command-list.txt` has 46
  `mainporcelain` entries, and the local `git help -a` lists 45 under Main
  Porcelain Commands. A second summary count of 48 is also **F**.
- WebFetch's summary of the docker reference page said the commands there
  form "a flat, alphabetical list … no hierarchical grouping". This is true of
  that web page only. The CLI's own help is grouped (1.5), so the two
  surfaces differ.

---

## Q2. How many top-level commands people are shown, and whether any guideline sets a number

### Finding 2.1: Counts from real help output (V, local captures, versions above)

| Tool | Shown by default | Full set reachable from help |
|---|---|---|
| git 2.52.0 | 23 common, in 5 groups | `git help -a`: 45 main porcelain; 173 entries across 11 headings, including plumbing, guides and external commands |
| gh 2.100.0 | 34 commands: core 12 + Actions 3 + alias 1 + additional 18; plus 8 help topics | same; extension stubs hidden |
| kubectl 1.34.1 | 42 commands in 8 groups (7 named + "Other") | `options`, and `alpha` when empty, are filtered out |
| docker 28.5.1 | 57: common 12 + management 11 + swarm 6 + legacy 28 | 29 with `DOCKER_HIDE_LEGACY_COMMANDS` |
| go 1.27.1 | 19 commands + 20 help topics | same |
| npm 11.17.0 | 4 usage examples on top | 71 commands in "All commands" |

The default view ranges from about 20 to 57 commands. The two tools with the
largest sets either curate a short top list (git: 23 of 45+) or offer a way to
shrink the list (docker: 57 to 29).

### Finding 2.2: No human-CLI guideline found sets a number (V for the sources read, U for the field as a whole)

clig.dev gives ordering advice, not a count: "Display the most common flags
and commands at the start of the help text", and it cites git's grouped help
as its example. Primer's CLI foundations give no count either. Cobra's docs
give mechanics only.

Sources: https://clig.dev/#help; Primer foundations (above); cobra user guide
(above).

### Finding 2.3: The only numeric limit found is on the model side (V)

**OpenAI.**

> "Aim for fewer than 20 functions available at the start of a turn at any one
> time, though this is just a soft suggestion."

> "Combine functions that are always called in sequence."

Source: https://developers.openai.com/api/docs/guides/function-calling
(redirected from platform.openai.com).

**Anthropic.** No count:

> "More tools don't always lead to better outcomes. A common error we've
> observed is tools that merely wrap existing software functionality or API
> endpoints."

It recommends "a few thoughtful tools targeting specific high-impact
workflows", and namespacing by prefix or suffix. Source:
https://www.anthropic.com/engineering/writing-tools-for-agents (11 Sep 2025).

**Claude Code** budgets by size, not count:

- MCP tool search defers tool definitions. In `auto` mode they load upfront
  only "while their definitions total less than 10% of the context window".
- Each MCP tool description and server instruction is truncated at 2,048
  characters by default.
- Each skill's `description` plus `when_to_use` is truncated at 1,536
  characters "in the skill listing to reduce context usage".

Sources: https://code.claude.com/docs/en/mcp; https://code.claude.com/docs/en/skills

Whether there is a total budget for the whole skill listing (as opposed to the
per-skill limit): **U**. The fetched part of the skills page does not state
one.

---

## Q3. How agent and assistant plugin systems expose commands

### Finding 3.1: Claude Code skills and commands have two independent switches, one per surface (V)

Custom commands are merged into skills:

> "A file at `.claude/commands/deploy.md` and a skill at
> `.claude/skills/deploy/SKILL.md` both create `/deploy` and work the same
> way."

The invocation matrix:

| Frontmatter | You can invoke | Claude can invoke | Context |
|---|---|---|---|
| default | Yes | Yes | description always in context |
| `disable-model-invocation: true` | Yes | Not on its own | description NOT in context |
| `user-invocable: false` | No | Yes | description always in context |

On `user-invocable: false`: "Claude Code hides it from the `/` menu and
doesn't run it when you type `/name`. Use for background knowledge users
shouldn't invoke directly." Also: "With `user-invocable: false`, you can't
invoke the skill, but Claude still can. To keep Claude from invoking it
through the Skill tool, set `disable-model-invocation: true`."

Permission rules `Skill(name)` and `Skill(name *)` allow or deny model
invocation.

Plugin skills are namespaced `/plugin:skill`. "The bare `/fancy` also invokes
the skill unless another command already uses that name."

So an author can hide a skill from the person while keeping it for the model
(`user-invocable: false`), or the reverse (`disable-model-invocation: true`).
The reverse also saves context, because the description is dropped.

Source: https://code.claude.com/docs/en/skills

### Finding 3.2: MCP assigns each primitive a controller (V)

Specification overview table:

| Primitive | Control | Description | Example |
|---|---|---|---|
| Prompts | User-controlled | Interactive templates invoked by user choice | Slash commands, menu options |
| Resources | Application-controlled | Contextual data attached and managed by the client | File contents, git history |
| Tools | Model-controlled | Functions exposed to the LLM to take actions | API POST requests, file writing |

**Prompts.** "designed to be **user-controlled** … For example, as slash
commands". "the protocol itself does not mandate any specific user
interaction model."

**Tools.** "designed to be **model-controlled** … there **SHOULD** always be a
human in the loop with the ability to deny tool invocations."

**Tool names.** "SHOULD be between 1 and 128 characters", may use only
`A-Za-z0-9_-.`, and "SHOULD be unique within a server". A tool also has a
separate display `title`.

**Annotations.** Clients "MUST consider tool annotations to be untrusted
unless they come from trusted servers."

MCP has no per-item "hide from user" or "hide from model" flag. The choice of
primitive is the hiding mechanism.

Sources: https://modelcontextprotocol.io/specification/2025-06-18/server;
https://modelcontextprotocol.io/specification/2025-11-25/server/prompts;
https://modelcontextprotocol.io/specification/2025-11-25/server/tools

### Finding 3.3: Claude Code as an MCP host (V)

- MCP prompts appear in the `/` menu as `/servername:promptname (MCP)` and
  can also be run as `/mcp__servername__promptname`.
- Resources are `@`-mentioned.
- A server author can mark a tool `_meta["anthropic/alwaysLoad"]` (load
  upfront) or `_meta["anthropic/requiresUserInteraction"]` (prompt the person
  on every call).

Source: https://code.claude.com/docs/en/mcp

### Finding 3.4: VS Code hides from the palette, but the command stays public (V, with a caveat)

Commands declared in `contributes.commands` appear in the Command Palette, and
"The Command Palette … prefixes commands with their `category`".
`menus.commandPalette` takes a `when` clause "to control if a command should
be visible in the Command Palette or not". Registered commands remain callable
via `vscode.commands.executeCommand`.

Using `"when": "false"` to hide a command is not stated in the official API
pages read. A VS Code team member said in issue #45119: "Using `false` should
also work", and "every command is public and when-clauses are
recommendations only."

Sources: https://code.visualstudio.com/api/references/contribution-points;
https://code.visualstudio.com/api/extension-guides/command;
https://github.com/microsoft/vscode/issues/45119

### Finding 3.5: Gemini CLI has no visibility switch in its docs (V for what the page says)

Custom commands are TOML files under `~/.gemini/commands/` or
`<project>/.gemini/commands/`. Subdirectories namespace them with colons
(`git/commit.toml` becomes `/git:commit`), and `description` shows in
`/help`. The page names no way to hide a command from the person or the
model.

Source: https://geminicli.com/docs/cli/custom-commands/

### Synthesis (V, across 3.1 to 3.5)

Only Claude Code skills give two independent switches, one per audience.
Elsewhere:

- MCP decides by which primitive the author picks (prompt for the person,
  tool for the model).
- VS Code hides from the palette but keeps everything callable.
- cobra's `Hidden` hides from help but keeps everything callable.

---

## Q4. Naming: verb versus noun-verb, and consistency rules

### Finding 4.1: clig.dev rules (V)

- "Use consistent names for multiple levels of subcommand … one is a noun and
  one is a verb. For example, `docker container create`. Be consistent with
  the verbs you use across different types of objects."
- "Either `noun verb` or `verb noun` ordering works, but `noun verb` seems to
  be more common."
- "Don't have ambiguous or similarly-named commands … 'update' and 'upgrade'
  is quite confusing."
- "Don't have a catch-all subcommand."
- "Don't allow arbitrary abbreviations of subcommands … aliases … should be
  explicit and remain stable."

Source: https://clig.dev/#subcommands and https://clig.dev/#future-proofing

### Finding 4.2: gh is noun then verb (V)

Primer: "**Command:** The object you want to interact with. **Subcommand:**
The action you want to take on that object". Examples: `gh issue view`,
`gh pr create`, `gh repo fork`. Guidance: "Use a flag for modifiers of
actions" and "Avoid making modifiers their own commands"; "Use unambiguous
language that can't be confused for something else"; "Use shorter phrases if
possible and appropriate".

Source: Primer foundations `.mdx` (above).

A web-search summary attributed "Prefer noun verb naming; avoid confusing
near-duplicates" to Primer. **F as worded.** The source gives the
object/action structure and the "unambiguous language" line, but contains no
such sentence.

### Finding 4.3: kubectl is verb first, with the resource as argument (V)

Top-level commands are verbs (`get`, `create`, `delete`, `apply`, `describe`),
and the resource follows as an argument (`kubectl get pods`), with the version
fully qualified in scripts. Source: local help; kubectl conventions page.

### Finding 4.4: docker runs both forms side by side (V)

Legacy verbs (`docker rm`) live alongside noun-verb management commands
(`docker container rm`), and the legacy set can be hidden (1.5).

### Finding 4.5: cobra's own guidance is verb first (V)

> "The pattern to follow is APPNAME VERB NOUN --ADJECTIVE or APPNAME COMMAND
> ARG --FLAG"

This is the opposite order to gh's, which is built on cobra.

Source: https://raw.githubusercontent.com/spf13/cobra/main/README.md

### Finding 4.6: Naming on the model side (V)

- Anthropic recommends namespacing tools by service or resource
  (`asana_search`, `asana_projects_search`) and reports that prefix versus
  suffix "has non-trivial effects on our tool-use evaluations".
- MCP constrains tool-name characters and length (3.2).
- OpenAI asks for "clear and detailed function names".

---

## Q5. One list for both surfaces, kept in sync

### Finding 5.1: Common within one tool, generated from its command tree (V)

**git.** `command-list.txt` feeds `git help`, the git(1) "GIT COMMANDS"
section and bash completion (1.2).

**cobra.** Cobra "can generate documentation based on subcommands, flags,
etc." (user guide). `IsAvailableCommand` is the one predicate that keeps
hidden and deprecated commands out of the listings (1.7).

**gh.** The Makefile's `site-docs` target runs `go run ./cmd/gen-docs
--website --doc-path site/manual` to build the online manual from the command
tree. The doc generator skips `Hidden` commands unless they are annotated for
generation (`internal/docs/markdown.go`). The in-CLI `gh help reference` is
built from the same tree. A `root.go` comment: "doc generation makes use of
the Long text".

**docker.** `docker/cli-docs-tool` generates "Markdown, YAML, and man page
documentation from a Cobra command tree" for Docker's CLIs.

Sources: https://raw.githubusercontent.com/cli/cli/trunk/Makefile;
https://raw.githubusercontent.com/cli/cli/trunk/internal/docs/markdown.go;
https://github.com/docker/cli-docs-tool

### Finding 5.2: In editors and agents, one declaration feeds palette, menus and model (V)

- **VS Code.** A command is declared once in `contributes.commands`;
  "Presentation of contributed commands depends on the containing menu"
  (palette, context menus, keybindings), and visibility is set per menu with
  `when` clauses.
- **Claude Code.** One `SKILL.md` feeds both the `/` menu and the model's
  skill listing, and two frontmatter fields set its visibility on each.
- **MCP.** A host may surface prompts as slash commands from the same
  `prompts/list`, and `listChanged` notifications keep that list in sync.

### Finding 5.3: CLI help and an in-app palette from one source (U)

No primary source was found, within budget, for a product that generates both
its terminal `--help` and a GUI or agent command palette from one registry.
The examples above each sync one product's own surfaces: help, docs, man pages
and completion in a CLI; palette and menus in an editor; menu and model in an
agent host. Whether a shared registry feeds the CLI and an agent plugin at the
same time is unverifiable here.

---

## Implications for a tool exposing one command set to people and to agents

These are inferences from the findings, not findings.

- The mature pattern is one command tree with tiered visibility: a curated
  default list, the full list behind a flag, and hidden or alpha commands
  that stay callable. Splitting into two products is not the pattern.
- Machine stability comes from stable output flags (`--json`, `-o json`,
  `--plain`), and in git also from a stability promise on plumbing. A
  separate machine verb set is used only where the semantics really differ
  (git plumbing, `gh api`).
- Human help keeps to roughly 20 to 40 commands by default, by curating
  (git), grouping (kubectl, gh) or opting out of legacy commands (docker).
  The agent side is budgeted in tokens and characters (Claude Code) or
  softly by count (OpenAI, under 20). The two limits are different in kind.
- Per-audience visibility exists natively only in Claude Code skill
  frontmatter. MCP gets the same effect through the primitive the author
  picks.
- Naming: choose one order, noun-verb (gh, docker management) or verb-first
  (kubectl, cobra's README), and keep the verbs consistent across nouns. No
  near-synonyms, no catch-all default subcommand, no implicit prefix
  abbreviations.
