---
id: adr-2610091150447054
slug: the-guard-reads-a-script-the-command-names-before-it-judges
status: accepted
date: 2026-10-09
supersedes: null
superseded_by: null
related_intents: [itd-103]
related_rfcs: []
related_adrs: [adr-42, adr-25]
---

# ADR-2610091150447054: The guard reads a script the command names before it judges the command

## Context

The guard judges the text of a command line ([adr-42](0042-guard-parse-layer-is-a-mistake-filter.md):
a mistake filter for a cooperating agent, not a security boundary; Tier 1 blocks,
Tier 2 warns). Since iss-2609251640462464 it refuses a shell that reads its script
from a stream (`printf '<blocker>' | sh`, `bash <(curl …)`), under
`interpreter-reads-stream`, because the stream is text the guard read as data. That
refusal's successor tells the agent: "to run a script, save it and run it as a
file after reading it" (`internal/core/guard/payload.go:1970-1971`). The
over-long-command refusal says the same: "put the long text in a file and pass
the file" (`internal/core/guard/guard.go:752-753`).

The same text can reach a shell through a file, and today the guard allows every
form of that, including the one its own successors recommend:

| command | today |
| --- | --- |
| `printf '<blocker>' > /tmp/s.sh; bash /tmp/s.sh` | allow |
| `source /tmp/s.sh`, `. /tmp/s.sh` | allow |
| `bash < /tmp/s.sh`, `bash -s < /tmp/s.sh` (while `cat /tmp/s.sh \| bash` blocks) | allow |
| `BASH_ENV=/tmp/e bash -c true` | allow |
| `bash --rcfile /tmp/e -i -c true` | allow |
| `SHELLOPTS=xtrace PS4='$(<blocker>)' bash -c true` | allow |
| `env 'BASH_FUNC_true%%=() { <blocker>; }' bash -c true` | allow |
| `ZDOTDIR=/tmp/zd zsh -c true`, `HOME=/tmp/h zsh -c true` | allow |
| `./deploy.sh` | allow |
| `eval "$(cat /tmp/s.sh)"`, `bash -c "$(</tmp/s.sh)"` | warn |

So the guard's successor is the route around the guard: an agent refused
`printf … | sh` that follows the advice runs the same blocker one step later. A
mistake filter can leave a determined author out of scope. It cannot be the one
pointing at the door.

`Registry.Check(command string)` (`internal/core/guard/guard.go:457`) reads no
file content today and receives no directory: the `Registry` holds only a
worktree counter (`guard.go:168-176`), which a bare `git stash` consults
(`internal/core/guard/stash.go:34-73`). The hook resolves the session directory
and the host workdir (`internal/surface/cli/guard.go:376-420`) and then runs two
`Check`s, the repository registry's and the workdir registry's, folded with
`guard.Strictest` (`:422`, `:448`). The guard package's one file-content
precedent is the registry load, which refuses a symlinked `.abcd` and reads
`.abcd/guard.json` through `fsutil.ReadGuarded` under a 256 KiB cap
(`internal/core/guard/config.go:22-64`). The command text itself is capped at
64 KiB (`guard.go:698`). This record moves file reading into the verdict.

Two shapes were weighed, and the product thinker ruled on 2026-10-09 for the
first ("rule A: read, then judge"):

- A: the guard reads a script the command names and judges its content;
- B: the guard blocks only the hidden spellings (`BASH_ENV`, `ENV`), stops
  recommending the file form, and names script files a known limit.

## Decision

We will have the guard read a shell script that the command line points a shell
at, and judge the script with the registry's Tier 1 rules. Where the guard
cannot be sure which bytes the shell will run, it warns (Tier 2), and it blocks
only for a matched hazard or for text it knows will run unread.

**1. What the guard reads.** A shell-family shell (`shellFamily`,
`payload.go:974`) is pointed at a file when the command line, or a payload
segment inside it (a `-c` string, a script being read), has:

- a script operand (`bash f`, `sh f`, `zsh f`, after the options
  `shellReadsStream` walks, `payload.go:1860`), or `source f` / `. f`;
- a stdin redirect into a shell with no `-c` string and no script operand, in
  any spelling: `bash < f`, `bash -s < f`, `0< f`, a descriptor duplicated onto
  stdin from a file opened earlier on the line, or an `exec` earlier on the line
  that redirects the shell's own stdin from a file. Today `stdinStream`
  (`internal/core/guard/tokenize.go:49-54`) is set only for a pipe, a
  here-document, a here-string and a pipe into a group [beyond the ruling's words];
- a startup file the command line selects: `BASH_ENV=f`; `ENV=f` on an
  interactive shell (`-i`) in any mode; `--rcfile f` / `--init-file f` on bash;
  for zsh, the startup files under an assigned `ZDOTDIR` or `HOME` (`.zshenv`,
  and `.zprofile`, `.zshrc`, `.zlogin` for a login or interactive zsh); and for
  a login or interactive bash under an assigned `HOME`, `.bash_profile`, else
  `.bash_login`, else `.profile`, and `.bashrc` [zsh and HOME forms are beyond
  the ruling's words].

An assignment counts in any form the guard already resolves: a prefix, an `env`
wrapper, or an `export` earlier in the same line.

**2. What the guard judges inline, without a file.** Startup text the command
line carries in a variable is not a file and needs no read: `SHELLOPTS` or
`BASHOPTS` with a prompt variable bash expands (`PS0`, `PS1`, `PS2`, `PS4`,
`PROMPT_COMMAND`), and an exported function (`BASH_FUNC_<name>%%`). The guard
judges each command substitution in such a prompt, and each function body, as a
payload segment [beyond the ruling's words].

**3. A direct run is classified, not read.** A path run directly (`./deploy.sh`,
`bin/tool`) is stat'ed and its first 8 KiB sniffed only to classify it. A
shell-family shebang, or no shebang and no NUL byte, makes it a shell script,
read as in decision 5. A non-shell shebang, a NUL in the sniff, or a size over
the cap makes it a program, allowed unread, as every program is (decision 8).
A direct run whose path the guard cannot resolve is a program
[direct runs are beyond the ruling's words].

**4. How the file is found.** `Check` gains the directory the command runs in:
the host workdir when it exists, else the session directory, set on the
`Registry` by `Load`/`LoadRepo` beside the worktree counter (the precedent at
`guard.go:168-176`). Both of the hook's registries resolve against that same
directory; each file is read once per hook call and its verdict folded into
both. `abcd guard check` resolves against the process working directory, and
`commands/guard.md` says so. Resolution follows the shell:

- a `cd`/`pushd` earlier on the line is followed only when its target exists
  at check time and the script run is chained after it with `&&`; otherwise the
  operand is unresolved, never resolved as if the `cd` had failed
  (`internal/core/guard/match.go:190-193`, `internal/core/guard/workdir.go:28-33`);
- `~` follows the shell's own rule: an `export HOME=` earlier on the line
  changes it, a prefix assignment on the same simple command does not;
- a `source` operand with no slash is searched on `PATH`, then the working
  directory, as bash does;
- symlinks in the path resolve to their target, which is read with
  `fsutil.ReadGuarded` (`internal/fsutil/fsutil.go:62`: regular files only,
  no symlinked leaf, capped).

Tilde and path expansion come from one primitive in `internal/fsutil`. Two
local copies exist today (`internal/core/ahoy/harness_strays.go:422`,
`internal/core/lab/lab.go:366`); the implementing change moves them there
rather than adding a third.

A shell or `source` operand that cannot be resolved to a path (a variable, a
command substitution, a glob) is class (2) of iss-2609281134544802, whose
verdict is a product ruling not yet made. This record does not rule it: the
implementing change allows it, as today, until that ruling sets the verdict.

**5. Reading and judging a script.** Only Tier 1 verdicts propagate out of a
script: a registry entry matched at command position, and the synthetic
blockers (`interpreter-reads-stream`, an unread payload, the depth block). A
Tier 2 speculative hit inside a script does not propagate. A script is a
reviewed artefact rather than a line being typed, and the rationale for Tier 2
(an unknown command may exec its arguments, adr-42) is weaker still for text the
agent did not write on this line. A propagated block names the script, the line
and the entry.

The guard's verdicts on what it finds:

| finding | verdict |
| --- | --- |
| a registry blocker at command position in the script | block |
| the script was written earlier on the same line (decision 6) | block |
| the script does not exist | allow, with a diagnostic naming it [differs from the pitch] |
| a startup file the line selects does not exist | allow, silently (the shell skips it) |
| exists but cannot be read (permission, not a regular file) | warn |
| a shell-pointed file that is binary (NUL in the first 8 KiB) | block: a shell will run bytes the guard cannot judge |
| a shell script over 256 KiB | warn: not read, size named [differs from the pitch] |
| the read budget is spent (decision 7) | warn, through the fail-loud path |

**6. Written, then run, on one line.** A script is written-then-run when an
earlier segment on the line, payloads included, writes a path that resolves
(decision 4) to the script or to a directory above it: a redirect target, or the
target operand of a listed writer (`tee`, `cp`, `mv`, `install`, `dd of=`,
`curl -o`/`-O`, `wget -O`, `sed -i`, `patch`, `git checkout`/`restore`/`clone`,
`tar -x`, `unzip`). The file the guard reads at check time is not the file that
runs, so this blocks. A write target the guard cannot resolve warns. Reading the
script first (`cat s.sh && bash s.sh`) is not a write and stays allowed. The
writer list is an enumeration in the sense of adr-42 decision 5: incomplete by
design, extended over time, and named in decision 8.

**7. Cost is bounded by depth and by an aggregate budget.** File reads happen
only from Tier 1 command positions, never from a speculative start
(`internal/core/guard/speculate.go:50`). A script that points a shell at a
script is read in turn, sharing one depth counter with the payload families'
`maxPayloadDepth` (2 today, `payload.go:28`); past it the guard blocks, as it
does for nested payloads. A file already on the current read stack is a cycle
and is skipped, not counted, since everything in it was already read. One
budget per `Check` caps the total at 16 files and 1 MiB read; past it the guard
warns. The implementing change extends the `workTally` cost guards
(`internal/core/guard/work.go`) to bytes read and adds the case to
`BenchmarkCheck`, keeping the bound adr-42 decision 3 makes load-bearing.

**8. The check-to-run race is accepted, and the limits are named.** The file
can change between the guard's read and the shell's. Closing that needs the
host to run a frozen copy, which is host-specific and outside abcd (adr-25).
Under adr-42's threat model the race is accepted; decision 6 closes the one form
a single line can express. These stay unseen, and `commands/guard.md` and the
`SHELL` rules domain say so:

- other interpreters' files: `python3 f.py`, `node f.js`, `ruby`, `perl`,
  `make` (a Makefile), package-manager scripts (`npm run`);
- a program run directly or found through `PATH`, including a `PATH`
  assignment on the line that changes which file a bare name finds;
- the files a shell loads implicitly from the account's real `HOME`
  (`~/.bashrc`, `~/.zshenv`, `/etc/profile`). A `source` operand naming one
  is read like any other;
- file text substituted into a command string (`eval "$(cat f)"`,
  `bash -c "$(<f)"`), which keeps today's warn;
- the check-to-run race, and any writer missing from decision 6's list;
- any spelling of the class not listed in decisions 1 and 2.

**9. The successors change.** The stream block's successor reads: "Run the
commands directly, or pass them with `sh -c '<commands>'` so the guard reads
them. A script file is read and judged when it is run, so write it in one
command and run it in the next." The over-long-command successor
(`guard.go:752-753`) stops recommending a file without a size: "split the
command, or put the text in a script under 256 KiB and run it in a separate
command." Each new block names its own fix: for write-then-run, split the line;
for a binary file handed to a shell, run the program directly.

**10. The sibling sweep and the warn-rate check.** The implementing change
sweeps every startup file, startup-expanded variable and option a shell-family
shell honours, against the installed bash (3.2 and 5.x), sh and zsh manuals:
at least everything in decisions 1 and 2. Each is pinned by a guard test watched
fail first, or recorded under decision 8 with its reason. It also adds a corpus
test that runs the repository's own `scripts/*.sh`, `.githooks/*` and
`hooks/*.sh` through decision 5, the way the `Makefile` runs them, with a pinned
warn ceiling in the `corpus_test.go` shape, so the warn-rate STOP of adr-42
decision 8 is measured rather than assumed.

**11. Two refinements the product thinker ruled while it was built
(2026-10-09).** Decision 5 carries out of a script only block-level verdicts:
a registry entry that only warns, met inside a script, stays inside it with the
Tier 2 hits, so a script that runs `git clean` on its own scratch does not make
every run of it warn; the repository's own scripts then warn on none. Decision
6 applies to a direct run too: a file the line writes and then runs by path
blocks whatever the file is, closing the gap decision 3 left for a file that is
absent, or a program, at check time.

## Alternatives Considered

- **B: name script files a known limit, block only `BASH_ENV`/`ENV`.** Small and
  certain, but the successors keep pointing at an unguarded route, and the other
  startup spellings stay open one by one. Rejected by the product thinker on
  2026-10-09.
- **Refuse every script run the guard has not read** (block `bash f` outright).
  Closes the class, but breaks the ordinary `bash ./build.sh` a cooperating agent
  runs all day, which turns a mistake filter into an obstacle and invites the
  registry-disable escape. Rejected.
- **Block a missing, oversized or binary file in every position** (the pitch's
  wording). Simple to state, but measured as false blocks: a compiled binary run
  by path (`bin/abcd-darwin-arm64` is 14 MB), `./configure` scripts over 256 KiB,
  the conditional-source idiom `[ -f .env ] && source .env`, and `HOME=<tmp>`
  test lines whose startup files do not exist. Replaced by decisions 3 and 5.
- **Propagate every verdict from a script, Tier 2 included.** Measured: this
  repository's own `scripts/check-issue-resolution.sh`, run as the `Makefile`
  runs it, warns on Tier 2 speculation with no hazard in it. Rejected in favour
  of decision 5.
- **Run a frozen copy** (the guard copies the script and the host runs the copy).
  Closes the race, but needs the host to rewrite the command it runs, which no
  host abcd supports does today and adr-25 keeps out of the core. Rejected for
  now; decision 8 names the race instead.
- **Warn, never block, on the file forms.** Keeps the guard textual, but a warn
  is advice the agent can step past, and it leaves the successor's route open at
  Tier 2. Rejected.

## Consequences

- The verdict depends on file content as well as command text. `Check` gains a
  working directory on the `Registry` and a filesystem read bounded by decision
  7, through `fsutil.ReadGuarded`, the primitive the registry load already uses.
- A few commands allowed today block: a script with a registry blocker in it,
  write-then-run on one line, a binary handed to a shell. Each block names the
  script, the line and the fix.
- Guard tests grow a fixture tree of scripts; every row of the Context table is
  pinned by a test watched fail first, and the repository's own scripts are a
  corpus with a warn ceiling.
- `commands/guard.md`, the `SHELL` rules domain (generated from the registry),
  and both successor texts change in the same change.
- Tilde and path expansion gain one home in `internal/fsutil`.
- The class (2) ruling owed on iss-2609281134544802 now also decides how an
  unresolved script operand is judged.
