---
id: spc-2611042207343783
slug: a-person-who-starts-a-run-finds-out-at
intent: itd-2611041815429986
origin: researcher-authored
production_mode: hand-written
---
# A person tells abcd where a project's tools live, and abcd checks them before a run starts

## Summary

Delivers itd-2611041815429986 under the trust rule adr-2611041815445203: a committed list
of the tools a project needs, a person's registration of where each lives on this computer
with its fingerprint, a test before a build or a drain spends anything, a PATH made for each
command abcd starts, and a wait that keeps an item's work when its agent needs an unlisted
tool. Nothing here downloads or looks online. An agent's request as a pull request and the
wrapper are itd-2611042148411751; other languages are itd-2611042148424105; installing is
itd-2611041815433220.

This is a filter against mistakes under adr-42. Tests prove the refusals; none of them can
prove an agent on the person's account is stopped, and no text here or in the surfaces may
say it is.

## Approach, by acceptance criterion

The criteria are read by their headings in the intent.

**Adding and registering.** A new core package, `internal/core/needs`, owns the list, the
home file and the judge. The list is `.abcd/needs.json`, read through the guarded read from
the commit a lane is based on. The home file holds, per program, the path, the resolved
target and the SHA-256, under the ownership test `rules.json` uses, and sits apart from the
installed-tools record `internal/core/tools` keeps. The judge is extracted from the runner's
admit rule (`internal/core/runner/proc.go`) rather than copied, and gains ancestor, owner and
file-name checks and the same judgement for a link's target. Adding, registering and waiving
use `internal/term.IsTerminal` on input and output, as the drain rule's offer does, and join
the guard's person acts (`internal/core/guard/personacts.go`) with `abcd land --approve`. A
confirmation shows contents and metadata change times from the file, and no repository or
agent text.

**The test.** One function answers for the report, a start, a drain move and a lane's base
commit, in the order `internal/core/needs` defines: list read, then each tool's registration,
fingerprint and, for Go, version. A waived tool is skipped and reported. A PATH copy is only
described, and no output joins it into a command. The caller's PATH is compared with the
registration only where a host session drives the lane. A build calls it after the claim and
peers checks and before the budget check and any reservation (`internal/core/implement/loop/loop.go`,
`start`), never in the per-candidate check `abcd build next` calls. A refused build has its
own exit code. A drain calls it once per move, and a failure records a new wait kind, `tools`,
beside `cap` and `held` (`drain_wait.go`), with the same look-again.

**Running a command.** Before a command abcd starts, one helper makes a uniquely named
directory under the person's abcd home, links each program this repository lists and this computer has registered, checks each
fingerprint, prepends the directory and removes it when the command ends. The landing
preflight (`internal/core/landing/land.go`) and a runner's launch (`internal/core/runner/proc.go`)
call it, and the preflight resolves `make` through the made directory rather than abcd's own PATH. Links are made only for listed, registered, unwaived programs, so a copy only another repository lists stays out. A mismatch stops the command
and, in a lane, leaves the item in the tool wait.

**An agent needing a tool.** `abcd needs ask <tool> --why` is the one sub-verb an agent may
run. Its reason passes the same allowlist as the list's. It records the request on the item, and the lane stops with its work kept. A hand-back
(`internal/core/claim/handback.go`) is not used, because it discards work and holds the item
until its text changes. A lane's brief names the command. The next build or drain re-tests
and resumes from the kept work once the tool is listed and registered. Landing refuses a
lane's change that touches the list; a diff-range check beside the issue-resolution gate
refuses a list change mixed with anything else; and `mergeGate`
(`internal/core/landing/facilitator.go`) gains a case that a list-touching head needs
`approvedBy` under any mode, so a review receipt does not arm it. A forge approval counts here as it does elsewhere, a limit rule 11 of adr-2611041815445203 states.

**Go.** A strict reader of `go.mod` and `go.work` takes the `go` and `toolchain` lines and
refuses a suffixed toolchain. Version text never reaches a path. The probe runs `go env
GOVERSION`-style fixed arguments by absolute path, from a directory outside the repository,
with a minimal environment, a timeout and `GOTOOLCHAIN=local`. A script, or a link to a
program of another name, is never probed.

**The report.** `abcd needs` bare writes nothing and asks no network, in text and JSON through
the existing envelope. It proposes tools from version files and changes nothing until a person
adds them. `abcd ahoy doctor` names it.

**The verb's surfaces.** `commands/needs.md` and the brief's surface chapter arrive with the
verb, per wired-or-it-isn't-done. Brief invariant 26 is written already and says nothing
enforces it yet; its wording changes to present tense in the change that closes this spec.

## Out of scope

- An agent's request as a pull request, and the wrapper (itd-2611042148411751).
- Version checks beyond Go, the `node` reservation's lifting, two copies of one tool, and
  naming a tool from exit 127 (itd-2611042148424105).
- Installing anything (itd-2611041815433220).
- Changing a repository's branch protection or adding a CODEOWNERS file. A direct forge merge
  stays outside abcd's gate, which is that repository's own setting.

## Footprint

- packages: internal/core/needs, internal/core/runner, internal/core/landing, internal/core/implement/loop, internal/core/guard, internal/surface/cli
- tests: internal/core/needs/needs_test.go, internal/core/guard/personacts_test.go, internal/core/landing/facilitator_test.go, internal/core/implement/loop/drain_wait_test.go, internal/surface/cli/needs_test.go

## Steps

1. The list, its grammar and the judge extracted from the runner's admit rule
   - packages: internal/core/needs, internal/core/runner
   - tests: list grammar, reserved names with `go` and `make` accepted, closed version-file sets, a reason's limit and allowlist refusing a path; the judge over links, targets, ancestors and owners; the runner's admit tests unchanged
   - needs: none
2. Registering, fingerprints, waivers and the person's acts, with the verb's surfaces
   - packages: internal/core/needs, internal/surface/cli, internal/core/guard
   - tests: off-terminal refusal for each act, the guard refusing each under any program name, the home file's ownership test, the confirmation's facts, `gofmt` recorded beside `go`
   - needs: 1
3. The test and the report, with Go's reader and the sealed probe
   - packages: internal/core/needs, internal/surface/cli
   - tests: each outcome of the test, a changed file never run, no pasteable PATH copy, a suffixed toolchain refused, nothing downloaded, nothing written, JSON and text agreeing
   - needs: 2
4. A PATH made for each command abcd starts
   - packages: internal/core/needs, internal/core/landing, internal/core/runner
   - tests: only listed, registered programs linked, a listed `make` resolved through the made directory, a fingerprint checked immediately before, the directory removed on exit, a system-directories PATH finding every copy
   - needs: 2
5. The gate at build and drain, with the tool wait
   - packages: internal/core/implement/loop, internal/surface/cli
   - tests: nothing reserved, created or spent on a refused build, the exit code, a drain's wait and its look-again, a re-test on a changed base commit, a host session's caller PATH missing the copy, no list refusing nothing
   - needs: 3, 4
6. An agent asking, with the work kept
   - packages: internal/core/implement/loop, internal/core/needs, internal/surface/cli
   - tests: the item waits and others carry on, the list unchanged, the item resuming from kept work once registered, a mismatch mid-drain routed to the same wait
   - needs: 5
7. The gates on a change to the list
   - packages: internal/core/landing, scripts
   - tests: a lane's change refused at landing, a mixed change refused by the check, `mergeGate` waiting for an approval under `auto` with a receipt present
   - needs: 1

## Delivery

Each step lands on its own. Real runs are the proof: register a real Go on a checkout whose
PATH misses it, then run a build, a drain and `make preflight` through abcd. Closing the spec
turns invariant 26 to the present tense, and the change carries `Delivers: itd-2611041815429986`.
