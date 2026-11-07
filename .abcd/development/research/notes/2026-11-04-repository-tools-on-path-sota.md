# How a tool finds what a repository needs when it is not on PATH

A state-of-the-art review run on 2026-11-04 for the draft on checking a
repository's tools before a run starts. Recency was assessed against that date.
The evidence is primary documentation, source and advisories; no comparative
evaluation exists, and nothing was run. Tiers: EVIDENCE (documentation, source
or an advisory says so), CONSENSUS (several independent projects agree),
INFERENCE (this review's own design step, built from the sources named).

## Findings

1. **Asking a tool its version can run or fetch code the repository chose.**
   `go version` runs the selected toolchain, and rustup installs on demand by
   default. A probe therefore needs an absolute path, a neutral working
   directory, a minimal environment and the no-download switches
   (`GOTOOLCHAIN=local`, `RUSTUP_AUTO_INSTALL=0`, `COREPACK_ENABLE_NETWORK=0`,
   `--no-python-downloads`). EVIDENCE:
   [Go toolchains](https://go.dev/doc/toolchain),
   [rustup variables](https://rust-lang.github.io/rustup/environment-variables.html),
   [Corepack](https://github.com/nodejs/corepack),
   [uv](https://docs.astral.sh/uv/concepts/python-versions/).
2. **Declarations can be read as plain data.** Go: `go.work`, else `go.mod`,
   whose `go` line is the required minimum and whose `toolchain` line is only a
   suggestion. Rust: `rust-toolchain`, which wins over `rust-toolchain.toml`.
   Node: `.nvmrc`, `.node-version`, `package.json` `engines` (advisory),
   `devEngines` and `packageManager`. Python: `.python-version`,
   `requires-python`. Across tools: `.tool-versions` and plain-version
   `mise.toml` `[tools]`. Version strings only, which is mise's own no-trust
   line. EVIDENCE:
   [Go](https://go.dev/doc/toolchain),
   [rustup overrides](https://rust-lang.github.io/rustup/overrides.html),
   [package.json](https://docs.npmjs.com/cli/v11/configuring-npm/package-json),
   [uv](https://docs.astral.sh/uv/concepts/python-versions/),
   [mise security](https://mise.jdx.dev/security.html). Node files have no
   standard precedence.
3. **Some selectors name a binary, not a version.** asdf `path:` and `ref:`,
   rustup path toolchains, a `packageManager` URL and a Go `-suffix` toolchain
   name all choose what executes. rustup's own source says path toolchains
   "permit arbitrary code execution"; CVE-2023-39320 did the same through Go's
   `toolchain` line. EVIDENCE:
   [asdf](https://asdf-vm.com/manage/configuration.html),
   [rustup source](https://raw.githubusercontent.com/rust-lang/rustup/master/src/config.rs),
   [GO-2023-2042](https://pkg.go.dev/vuln/GO-2023-2042).
4. **Trust controls read from a repository get bypassed.** CVE-2026-35533 let a
   repository's `trusted_config_paths` trust itself, and CVE-2026-55448 ran a
   repository's `credential_command` before any trust decision. direnv and mise
   both key a person's approval on the file's contents, so an edit blocks it
   again. EVIDENCE:
   [GHSA-436v-8fw5-4mj8](https://osv.dev/vulnerability/GHSA-436v-8fw5-4mj8),
   [GHSA-29hf-rm4x-xxph](https://osv.dev/vulnerability/GHSA-29hf-rm4x-xxph),
   [direnv source](https://raw.githubusercontent.com/direnv/direnv/master/internal/cmd/rc.go),
   [mise](https://mise.jdx.dev/paranoid.html).
5. **Shims are not real binaries.** A mise shim re-reads the current directory's
   configuration on each call, Volta is shim-only and unmaintained, and nvm is a
   shell function. Real binaries can be read from the stores as files. EVIDENCE:
   [mise shims](https://mise.jdx.dev/dev-tools/shims.html),
   [mise where](https://mise.jdx.dev/cli/where.html),
   [pyenv](https://github.com/pyenv/pyenv),
   [uv storage](https://docs.astral.sh/uv/reference/storage/),
   [Volta](https://github.com/volta-cli/volta),
   [nvm](https://github.com/nvm-sh/nvm).
6. **Capturing a login shell is fragile and runs all of a person's startup
   code.** VS Code runs `-i -l -c` with a 10-second timeout and then launches
   without the environment, and its FAQ names nvm as a common slow command. Run
   from a repository directory, activation can parse repository configuration;
   the first CVE above was shown through `mise hook-env`. CONSENSUS on the
   fragility:
   [VS Code source](https://raw.githubusercontent.com/microsoft/vscode/main/src/vs/platform/shell/node/shellEnv.ts),
   [VS Code FAQ](https://code.visualstudio.com/docs/supporting/faq),
   [Dev Containers](https://containers.dev/implementors/json_reference/).
7. **Doctor commands that work separate a failure from a partial result.**
   Flutter separates a crash or missing tool from a partial report and gives
   each check a timeout, mise adds `--json` with a suggested fix per problem,
   rustup uses a distinct exit code, and `brew doctor` is the shape to avoid: it
   exits non-zero on any problem and then says to ignore it if things work.
   EVIDENCE:
   [Flutter source](https://raw.githubusercontent.com/flutter/flutter/master/packages/flutter_tools/lib/src/doctor.dart),
   [mise doctor](https://mise.jdx.dev/cli/doctor.html),
   [rustup changelog](https://raw.githubusercontent.com/rust-lang/rustup/master/CHANGELOG.md),
   [brew](https://docs.brew.sh/Manpage).
8. **Agent products run setup in environments they manage, which does not
   transfer to a person's own account.** Cursor's `install`, Codex setup scripts
   and `copilot-setup-steps.yml` run in product-managed machines. On failure
   Copilot begins work anyway, while Claude's cloud environments fail the
   session start and Cursor keeps the last good build. Copilot and Cursor read
   setup from the default branch. EVIDENCE:
   [Copilot](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/coding-agent/customize-the-agent-environment),
   [Cursor](https://cursor.com/docs/cloud-agent/setup),
   [Claude cloud environments](https://code.claude.com/docs/en/cloud-environments),
   [Codex](https://learn.chatgpt.com/docs/environments/cloud-environment).
9. **Go cannot bootstrap from nothing.** Toolchain switching needs a `go` of 1.21
   or later already present, and its downloads are checked against the checksum
   database. With no Go at all, an archive's `sha256` from go.dev's JSON comes
   from the same origin as the file. rustup does not validate signatures.
   EVIDENCE: [Go](https://go.dev/doc/toolchain),
   [go.dev/dl](https://go.dev/dl/?mode=json),
   [rustup security](https://rust-lang.github.io/rustup/security.html).

## A second review: keeping a list of tools up to date

Run the same day, when the product thinker asked for a list of each project's tools,
added to while building, that abcd could keep up to date.

10. **New versions reach a repository as a reviewed change from a bot.** Renovate has
    managers for `.tool-versions`, `mise.toml`, `.nvmrc`, `.python-version`,
    `rust-toolchain` and go.mod's `toolchain` line, bumping `go` only with
    `rangeStrategy: bump`. Dependabot lists `rust-toolchain`, `devcontainers` and `nix`;
    asdf, mise and Go's lines are open requests. CONSENSUS:
    [Renovate asdf](https://docs.renovatebot.com/modules/manager/asdf/),
    [Renovate mise](https://docs.renovatebot.com/modules/manager/mise/),
    [Renovate gomod](https://docs.renovatebot.com/modules/manager/gomod/),
    [Dependabot options](https://docs.github.com/en/code-security/dependabot/working-with-dependabot/dependabot-options-reference).
11. **Those bots wait before proposing a release.** Dependabot has defaulted to three
    days since July 2026, and mise and pnpm default to about a day. The value is contested:
    one review counts 8 of 10 attacks with windows under a week, and another calls waiting
    free-riding. Neither is controlled. EVIDENCE for the features:
    [Dependabot cooldown](https://github.blog/changelog/2026-07-14-dependabot-version-updates-introduce-default-package-cooldown/),
    [Renovate](https://docs.renovatebot.com/key-concepts/minimum-release-age/),
    [Woodruff](https://blog.yossarian.net/2025/11/21/We-should-all-be-using-dependency-cooldowns),
    [Paterson](https://calpaterson.com/deps.html).
12. **Upgrading a developer's tools is something a person asks for.** `brew upgrade`,
    `rustup update`, `mise upgrade` and `uv python upgrade` are all invoked, and no primary
    document found endorses silent toolchain upgrades. CONSENSUS:
    [brew](https://docs.brew.sh/Manpage),
    [rustup](https://rust-lang.github.io/rustup/basics.html),
    [mise](https://mise.jdx.dev/cli/upgrade.html).
13. **No existing list format holds a tool with no version and a reason.** Brewfile has no
    versions and keeps reasons as comments, aqua and devcontainers treat a missing version
    differently, and checksums always live in a separate lockfile. EVIDENCE:
    [Brewfile](https://docs.brew.sh/Brew-Bundle-and-Brewfile),
    [aqua](https://aquaproj.github.io/docs/reference/config/),
    [mise.lock](https://mise.jdx.dev/dev-tools/mise-lock.html). No survey of what projects
    usually do was found.
14. **Recording a tool at the moment of use has a shape, and agents doing it is new.**
    `mise use`, `brew bundle add` and `devbox add` record at use. No product was found where
    an agent records a tool with a reason. EVIDENCE:
    [mise use](https://mise.jdx.dev/cli/use.html),
    [devbox add](https://www.jetify.com/docs/devbox/cli-reference/devbox-add). Detecting an
    undeclared tool has no known tool; scanning command words and reading
    [exit 127](https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html) is
    this review's inference, and it will produce false positives.

## A third review: keeping a tricked agent from getting a tool approved

Run the same day, when the product thinker asked what prevents a malicious issue from
talking an agent into requesting a harmful program. Most of this rests on vendor
statements and single studies, and no study of this exact approve-then-persist chain was
found.

15. **Filters of an agent's words are not a boundary.** One 2025 paper bypassed 12 published
    defences, most above 90%, and human attackers won every scenario. Cursor's documentation
    says of its own checks that they are "not a security boundary". Vendor recall figures (OpenAI 99.3% against its own
    attacker; Anthropic 5.7% misses on synthetic exfiltration, 17% on real overeager actions,
    n=52) are unreplicated. EVIDENCE:
    [The Attacker Moves Second](https://arxiv.org/abs/2510.09023),
    [Cursor run modes](https://cursor.com/docs/agent/security/run-modes),
    [OpenAI auto-review](https://alignment.openai.com/auto-review/),
    [Anthropic auto mode](https://anthropic.com/engineering/claude-code-auto-mode).
16. **The working designs limit what a fooled agent can do.** Meta requires "human-in-the-loop
    approval or another reliable means of validation" when an agent has all three risky
    properties. CaMeL and FIDES enforce this deterministically, and both assume owning the
    agent loop. Benchmarks only: CaMeL 77% against 84% of AgentDojo tasks; Progent 25.8% to
    2.6%. EVIDENCE: [Meta](https://ai.meta.com/blog/practical-ai-agent-security/),
    [CaMeL](https://arxiv.org/abs/2503.18813),
    [FIDES](https://devblogs.microsoft.com/agent-framework/fides/),
    [Design Patterns](https://arxiv.org/html/2506.08837),
    [lethal trifecta](https://simonwillison.net/2025/Jun/16/the-lethal-trifecta/).
17. **First-on-PATH is itself the attack surface.** Checkmarx showed shadowed commands
    trojanising a command line, and Codex's execution policy can name an exact host
    executable. EVIDENCE:
    [Checkmarx](https://checkmarx.com/blog/this-new-supply-chain-attack-technique-can-trojanize-all-your-cli-commands/),
    [Codex execpolicy](https://raw.githubusercontent.com/openai/codex/main/codex-rs/execpolicy/README.md).
18. **An approval tied to a name lets the thing behind it change.** Cursor's CVE-2025-54136 is
    that failure, and Invariant's tool pinning and Santa's hash rules are the shape of the fix.
    No shipping agent was found binding approval to a file's hash. It pins bad bytes too, and
    Ona's write-up sidesteps hash enforcement through the dynamic linker. EVIDENCE:
    [MCPoison](https://research.checkpoint.com/2025/cursor-vulnerability-mcpoison/),
    [Invariant](https://invariantlabs.ai/blog/mcp-security-notification-tool-poisoning-attacks),
    [Santa](https://northpole.dev/features/binary-authorization/),
    [Ona](https://ona.com/stories/how-claude-code-escapes-its-own-denylist-and-sandbox).
19. **Reviewers see less, and that helps only so far.** Chrome's critic reads metadata and
    Anthropic's classifier strips tool results, while Codex's reviewer sees outputs. CONTESTED:
    one author's test had a stripped classifier allow untrusted code in 6 of 10 trials, and no
    adaptive evaluation of metadata-only reviewers was found. EVIDENCE:
    [Google](https://blog.google/security/architecting-security-for-agentic/),
    [Codex auto-review](https://learn.chatgpt.com/docs/sandboxing/auto-review),
    [the 6-of-10 test](https://veganmosfet.codeberg.page/posts/2026-08-12-opus5_automode/).
20. **Repetition wears down a filter.** Attack success is cited as rising from 4.7% to 63%
    over 1 to 100 attempts ([arXiv:2606.26479](https://arxiv.org/html/2606.26479v1); the primary
    was not opened), and resampling amplifies injection
    ([Terekhov](https://arxiv.org/abs/2510.09462)). Structured fields can carry attacks too:
    [data injection](https://arxiv.org/abs/2607.05120) disguises payloads as resource
    identifiers (abstract only).
21. **People stop reading frequent prompts.** Anthropic states users "accept 93%", with no
    method given. EVIDENCE as a vendor statement:
    [Anthropic auto mode](https://anthropic.com/engineering/claude-code-auto-mode).
22. **Who may request and who may approve can be separated.** Copilot never presents comments
    from users without write access, and the requester cannot approve its own pull request.
    EVIDENCE:
    [Copilot risks](https://docs.github.com/en/copilot/concepts/agents/cloud-agent/risks-and-mitigations).
23. **Fences and firewalls raise cost and do not settle it.** Copilot says its firewall "should
    not be considered a comprehensive security solution", and Clinejection shows what an
    injected agent reaches on an open network. Provenance is necessary and never sufficient:
    TanStack's attestations were "indistinguishable". EVIDENCE:
    [Claude sandboxing](https://code.claude.com/docs/en/sandboxing),
    [Copilot firewall](https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/customize-the-agent-firewall),
    [Clinejection](https://simonwillison.net/2026/Mar/6/clinejection/),
    [SLSA](https://slsa.dev/blog/2026/05/mini-shai-hulud-what-slsa-can-and-cannot-do).

The review favoured letting automatic approval accept only what fixed checks prove was
already on the person's path, and new locations going to a person. The product thinker
weighed that and took the stricter rule: a person says where, every time. A security review
then rejected an autonomous facilitator approving even a name, so no AI reviewer decides
anything in this design, and findings 19 and 20 are recorded as why not.

Not adopted: a guard model as a boundary (finding 15); rebuilding CaMeL or FIDES, which assume
owning the loop; tracking taint through a session; Gatekeeper, since `curl` sets no quarantine
([Unit 42](https://unit42.paloaltonetworks.com/gatekeeper-bypass-macos)); requiring Santa.

Unsolved: an agent on the same account can edit PATH or the approvals file, so in host sessions
all of this catches mistakes only; a harmful file approved the first time; and a swap between
the check and the start, without a daemon.

## What abcd takes from this

- Read the list and version files as data and never execute them; refuse a selector that
  names a binary or a remote. Findings 1, 2, 3.
- Take a copy's location from a person, at a terminal, every time. This is the product thinker's
  design, and it replaces the search of known stores and the login-shell capture the first
  review weighed, and the automatic acceptance the third review favoured. Findings 4, 5, 6, 15.
- Bind each approval to the file's SHA-256, and put only those programs first on PATH for each
  command, checked immediately before. Findings 17, 18.
- Make a request a pull request changing only the list, approved by a person, so asking and
  approving stay apart, and a name approved runs nothing until a person registers it. Finding 22.
- Keep person prompts rare and generated from the file's facts. Finding 21.
- Depend on no version manager. Findings 4, 5.
- Test before work starts, and separate a refusal from a warning. Findings 3, 7, 8.
- Keep versions in the files that already hold them, so a bot can bump them, and report a
  machine's copy against them from disk. adr-38 stands: abcd looks nothing up online.
  Findings 10, 11, 12.
- Keep the list as an own versionless file of names and reasons. Findings 13, 14.
- Installing is a separate step behind a per-tool yes, and abcd's own registry cannot serve
  it as it stands: that registry fetches only fixed data compiled into the binary
  (adr-2610241057124151). Finding 9.

## One recommendation not taken

The review advised keeping the no-download switches set while a stage runs. abcd
sets them for its own probes only. A stage runs the repository's own commands,
and forcing `GOTOOLCHAIN=local` on them would change how a repository works, which
abcd does not do. The trade-off is that a stage may still fetch what the person's
own setup would fetch.

## Not verified

- launchd's default PATH: Apple's pages state none, and one
  [2023 blog](https://lucaspin.github.io/launchd/macos/2023/01/08/where-is-my-path-launchd.html)
  observed `/usr/bin:/bin:/usr/sbin:/sbin`. Measure it rather than hardcode it.
- cron's default PATH: [FreeBSD](https://man.freebsd.org/cgi/man.cgi?query=crontab&sektion=5)
  documents one; the [Debian](https://manpages.debian.org/bookworm/cron/crontab.5.en.html)
  and macOS pages do not.
- Go's module-cache toolchain directory was seen on one machine, not read from
  documentation. With registration it is a path a person gives, so abcd does not rely
  on it.
- What environment a host's own hooks and agents receive.
- Setup-failure behaviour for Devin and Codex.
- A claimed gap in CaMeL's taint tracking; whether Anthropic's auto mode became the default;
  how macOS quarantine behaves now; Clinejection's download count.
