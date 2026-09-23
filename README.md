<div align="center">

  <img src="docs/assets/img/logo.png" alt="abcd logo" width="150">

  <h1>Agent-Based Configuration for Development</h1>

  <p>For people who know what they want to build and need help shipping it.</p>

  <img src="https://img.shields.io/badge/status-experimental-orange" alt="Status: experimental">
  <a href="https://github.com/intentdriven/abcd/releases"><img src="https://img.shields.io/github/v/release/intentdriven/abcd?cacheSeconds=300" alt="Release"></a>
  <img src="https://img.shields.io/github/last-commit/intentdriven/abcd?cacheSeconds=300" alt="Last commit">
  <br />
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26">
  <a href="https://claude.ai/claude-code"><img src="https://img.shields.io/badge/Built_with-Claude_Code-3B5CE7?logo=anthropic&logoColor=white" alt="Built with Claude Code"></a> <!-- docs-lint: allow -->
  <br />
  <img src="https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=white" alt="macOS">
  <img src="https://img.shields.io/badge/Linux-core%20CI--tested-FCC624?logo=linux&logoColor=black" alt="Linux: core CI-tested">

</div>


## What it is for

AI agents are very good at coding but not always at remembering *human intentions* for why the code was written. `abcd` is a host-agnostic configuration layer for intent-driven development, there to help you actually ship what you set out to build, including *what was decided*, *what was rejected*, and on *what evidence*.

In AI-assisted development, this (human) reasoning typically lives in transcripts that are hard to decipher after the fact. `abcd` keeps it as structured records the product thinker, the facilitator, and their agents *do* read: The *intent* that says what shipping looks like, the *decision* that says what was chosen and what was refused, the *specification* that says how to build it, and the *issue ledger* that says what must be revisited. In `abcd`, these structured records are plain files that live *inside* the repository, and they are checked by gates, most of which *refuse* rather than warn, so a record that drifts from the repository is caught rather than noted. Whether what a record claims about the *product* holds is a second judgement: An audit reads a shipped intent's acceptance criteria against the code that shipped and states a verdict on each one. That verdict is recorded, not enforced.


## Built in the open

`abcd` is an *experiment*. First, it is an experiment of *building itself*, which is why its documented development record is public and complete: Every decision, intent, specification, and issue, from the first commit onward, with the reasoning attached. That record *is* the demonstration. Not a claim that the approach works, but the trail of a real product being built this way, including the parts that were wrong: Reversed decisions, abandoned designs, and defects found by the gates and recorded before they were fixed.

`abcd` is also an experiment for the team building it. As self-declared *enthusiastic dilettantes*, we learn best by doing stuff, and `abcd` demonstrates not only what it does, but also how we're learning AI-assisted development while building it.

If you want to know more about `abcd`, you can interrogate its entire development record at [abcdev.app/record/](https://abcdev.app/record/), alongside more details on [who abcd is for](docs/explanation/rationale.md), the different [roles](docs/explanation/roles.md), [artefacts](docs/explanation/artefacts.md), and [process](docs/explanation/process.md). If you want to get involved, watch the repository or open a [discussion](https://github.com/intentdriven/abcd/discussions).


## Key principles

`abcd` is founded on several [principles](https://abcdev.app/record/foundations/), some of which guide its design while others guide how development artefacts are recorded.


### Design principles

What `abcd` is (and, by extension, what it refuses to become):

- **abcd builds abcd**: The framework develops under its own record and gates, so every convention it imposes is one the team lives with *(for better or worse!)*.
- **Prefer the experiment to the inference**: A claim that can be settled by running the system is settled by running it; reading the files yields a working assumption, never a finding.
- **Verifier selects, gates decide**: A model's verdict ranks, flags, and proposes; admission to the record is decided by deterministic gates.
- And: **Less, but better** (Dieter Rams): Reach for the subtraction first, which, in `abcd` translates into fewer verbs, records, and rules.


### Record principles

How intents, decisions, specifications, and issues are made and kept true:

- **Work starts from an intent**: A shipping change opens as a press release for the user it serves; the work exists to make that page true.
- **The record lands with the act**: A record is written by the same commit that makes it true.
- **Enforcement claims are facts**: A gate is described only where it demonstrably runs; a planned check is an intent.
- And: **The record is part of the product**: Intents, decisions, specifications, and issues are plain files in the repository, versioned and reviewed like the code they explain.

## Install

If you wish to experiment with `abcd`, we recommend installing it as a plugin *(but do remember that it's experimental!)*.


### Requirements

- **Git**: Always. `abcd` shells out to the `git` binary and anchors every record it keeps to a repository.
- **A released platform**: macOS or Linux, on amd64 or arm64. *(Windows runs the Linux route inside WSL)*.
- **An agent harness**: The plugin route and the verbs that hand their work to a model, and nothing else.

At this stage, `abcd` supports a single harness. The command-line app runs in any repository without one.


### As a plugin

The easiest route to get started is to install `abcd` as a [Claude Code](https://claude.ai/claude-code) plugin. <!-- docs-lint: allow -->

Run these two from a session within it. The first registers this repository as a plugin marketplace; the second installs `abcd` from it.

```
/plugin marketplace add intentdriven/abcd
```

Wait for the confirmation that the marketplace was added, then:

```
/plugin install abcd@abcd-marketplace
```

Restart the session afterwards so the hooks load, then check what you got:

```
/abcd:version
```

Later, `/plugin update abcd` takes the latest cut release: the marketplace names that release's plugin archive by its SHA-256, and the harness refuses any other bytes. The plugin route needs Claude Code v2.1.224 or later. <!-- docs-lint: allow -->


### As a CLI

Outside a plugin session, `abcd` runs from a terminal in any repository, with no harness involved. A checksum-verified one-liner provisions it, no administrator rights required:

```sh
sh -c 'set -eu; unset HTTPS_PROXY https_proxy HTTP_PROXY http_proxy ALL_PROXY all_proxy CURL_HOME CURL_CA_BUNDLE SSL_CERT_FILE SSL_CERT_DIR; cd "$(mktemp -d)"; os=$(uname -s | tr "[:upper:]" "[:lower:]"); arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64) arch=arm64;; esac; b="abcd-$os-$arch"; curl -q --proto =https --proto-redir =https -fsSLO "https://github.com/intentdriven/abcd/releases/latest/download/$b"; curl -q --proto =https --proto-redir =https -fsSLO "https://github.com/intentdriven/abcd/releases/latest/download/checksums.txt"; l=$(grep " $b$" checksums.txt); printf "%s\n" "$l" | if command -v sha256sum >/dev/null; then sha256sum -c -; else shasum -a 256 -c -; fi; mkdir -p "$HOME/.local/bin"; install -m 0755 "$b" "$HOME/.local/bin/abcd"; mkdir -p "$HOME/.abcd"; printf "path=%s\nbinary_sha256=%s\n" "$HOME/.local/bin/abcd" "${l%% *}" > "$HOME/.abcd/path-entry"; "$HOME/.local/bin/abcd" version'
```

The [install guide](docs/how-to/install.md) covers building from source and what to do when `abcd` isn't found afterwards.


## First run


### Setup

In a plugin session, inside a repository you own, `/abcd:prepare-this-repo` audits the tree and adopts the working conventions: The three-tier `.abcd/` layout, an `AGENTS.md` router, and the commit gates. Bare `/abcd` (or `abcd` from a terminal) then shows where you are; the status board is read-only, so it is safe on any tree:

```text
$ abcd
abcd — /path/to/your-repo
  git repo:   true
  record:     true
  work tiers: [development work work.local]
  presence:   abcd · your-repo · main · itd 0 · iss 0
```


### Recording your first issue

Issues are everything you wish to revisit: An idea, a user-facing intent, a bug, a thought. `/abcd:capture "..."` files a half-formed observation to the issue ledger so it survives the session. Revisit it with `/abcd iss-N` to report what that record is, where it lives, and its next move, such as graduating it into an intent, or close it with a note.

*(The [verb reference](docs/reference/cli/commands.md) lists the rest.)*


## Citation

If you use `abcd` in your work, please cite it. The form below is rendered from
[`CITATION.cff`](CITATION.cff), which is the record GitHub's own *Cite this
repository* button reads:

```bibtex
@software{reppel_abcd,
  author  = {Reppel, Alex},
  title   = {{abcd}: {Agent-Based} {Configuration} for {Development}},
  url     = {https://github.com/intentdriven/abcd},
  license = {MIT}
}
```


## Resources

- [`LICENSE`](LICENSE): MIT.
- [`SECURITY.md`](SECURITY.md): Report a vulnerability privately.
- [`ACKNOWLEDGEMENTS.md`](ACKNOWLEDGEMENTS.md): The ideas, tools, and writing `abcd` stands on.
