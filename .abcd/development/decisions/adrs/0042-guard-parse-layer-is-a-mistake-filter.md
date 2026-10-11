---
id: adr-42
slug: guard-parse-layer-is-a-mistake-filter
status: accepted
date: 2026-08-18
supersedes: null
superseded_by: null
related_intents: [itd-103]
related_rfcs: []
related_adrs: [adr-25]
---

# ADR-42: The guard's parse layer is a mistake filter, not a security boundary — matching is two-tier, and completeness is abandoned as a goal

## Context

`abcd guard` reads a proposed shell command, walks past "wrappers" to find the
real command, and matches it against a registry of hazard patterns
([itd-103](../../intents/shipped/itd-103-abcd-teaches-repo-agents-the-shell.md),
spc-16). Three separate defects have now been filed against the same shape:

| defect | the enumeration that was incomplete |
|---|---|
| gh-297 | the interpreter set (`sh bash dash …`) |
| gh-299 | git's global value flags (`-c`, `--config-env`, `--shallow-file`, …) |
| iss-272 | the wrapper set (`sudo doas command env nohup time xargs timeout exec`) |

Each was fixed by extending a list. A third instance of one defect predicts a
fourth, so iss-272 was investigated design-first rather than patched: a plan
drafted from the issue text went to adversarial review before any code and came
back with three blockers and four factual errors — a count the note revises
upward as it verifies the rest of the table. The evidence is in
[`2026-08-18-guard-wrapper-family-and-parse-layer-limits.md`](../../research/notes/2026-08-18-guard-wrapper-family-and-parse-layer-limits.md);
this record holds the decision it informed.

Three findings forced the framing.

**The defect is an asymmetry, not a short list.** The interpreter path already
fails loud on what it cannot resolve — `shellUnresolved` yields a warn, so a
payload the guard cannot read is never mistaken for clearance. The wrapper path
has no equivalent: an unrecognised wrapper simply *becomes* the command, nothing
matches, and the verdict is a confident `allow`. Ten wrapper bypasses were
verified live on the merged tree, `nice git push --force` among them, each
exiting 0.

**The set is unbounded in principle, and the tractable part cannot be read off
the documentation.** Any program that execs another is a wrapper. `nsenter --help`
renders `-S/--setuid` as optional-argument and `unshare --help` renders the same
letter in the same package at the same version as required-argument — yet both
binaries consume the following token, so nsenter's help contradicts its own
parser. A per-wrapper flag table can therefore only be derived by probing the
installed binary, and gh-299 is the in-repo proof, recorded in
`gitglobals_test.go`: the first cut of git's value-flag list was taken from the
bug report and "was wrong three ways" — it omitted `--shallow-file`, a live
force-push bypass present in git since 1.9, and counted `--exec-path` and
`--super-prefix` as value-taking when neither is. **It totalled nine either way,
so a size assertion certified the wrong list as complete.** A completeness check
that passes on a list wrong in both directions is the strongest available
argument that only probing the installed binary settles this. The grammar differs
again on macOS.

**Every vendor shipping this control documents it as steering, not enforcement.**
[Claude Code][cc-permissions] calls argument-constraining patterns "fragile" and
refuses to implement `Bash(command:rm *)` because a compound command would bypass
it; [Cursor][cursor-security] shipped a command denylist, watched it fall to `bash
-c`, subshells and base64 — the gh-297 gap verbatim — replaced it with an
allowlist, and then had *that* bypassed by poisoning an allowed command's
environment (GHSA-82wg-qcm4-fp2w / CVE-2026-22708). The posture is [CWE-184 (Incomplete List of Disallowed
Inputs)][cwe-184] and the literature behind it is settled —
[Ranum's "Enumerating Badness"][ranum-dumbest],
[Garfinkel (NDSS 2003)][garfinkel-traps].
Meanwhile the repo rates this class two ways at once: the GitHub-filed siblings
gh-297 and gh-299 both carry `severity:critical` — labelled "silent
security/data-loss at a trust boundary" — with adversarial framing
("agent-proposed commands are steerable by untrusted content the agent has
read"), while the ledger entry for the same defect, iss-272, is `minor`. Both
cannot be right, and the critical framing is a claim no denylist can honour.

## Decision

**1. The guard's parse layer is a mistake filter.** It catches accidents and
casual evasion by a cooperating agent. It does not withstand a determined author.
Anything needing an actual boundary needs an execution-layer control behind it;
the parse layer never claims to be one. **The guard does not say this today** —
its help text gets as far as "an allow means no registry entry matched — it is
never a statement that a command is safe", and the surface brief says "coverage is
what the registry names", but neither states the scope in these terms. Saying it
is part D, the first thing built. The split rating is resolved with it: a missing
wrapper or interpreter name is a real defect in a mistake filter, not a silent
failure of a trust boundary, so `severity:critical` is the wrong label for the
class.

**2. Enumeration-completeness is abandoned as the matching strategy, and
replaced by two tiers:**

- **Tier 1 — position-anchored, blocks.** A registry entry matched at command
  position after stepping known wrappers. Precise, and the only tier that blocks.
- **Tier 2 — position-agnostic, warns.** When no entry matched the segment, the
  matcher re-runs speculatively from each later command-position token, and each
  speculative suffix is run through `expandPayloads`. A hit is a loud **warn**,
  never a block, because an unknown command is not proof that it execs its
  arguments.
- **A synthetic verdict raised from a speculative suffix is demoted to warn, and
  never dropped.** `expandPayloads` does not only match — it raises its own
  signals, and two of them are blockers (`envSpecialBlockSignal` for an `env -S`
  value it cannot prove plain, `depthBlockSignal` past `maxPayloadDepth`). Tier 2
  reaches them: `env -S 'git push --force $X'` raises the synthetic blocker today
  while `myrunner env -S 'git push --force $X'` is allowed, because
  `splitStringValue` walks only the leading wrapper chain. Honouring such a signal
  at its own tier would make Tier 2 block on two layers of uncertainty at once —
  an unknown command that may not exec its arguments, carrying a payload the
  guard could not read. Discarding it would invent a second silent suppression
  path inside the fail-safe. Demotion is the only option that keeps both
  invariants.

  The `$` has to sit **inside** the quoted `-S` value, where `isPlainCommand`
  tests it. The unquoted `env -S git push --force origin main` also blocks, but on
  the ordinary `git-push-force` entry — `envInspect` decodes the value and
  rebuilds the command — so it looks like this phenomenon and is not it.

  "Never dropped" binds the **verdict**, which is all the safety argument needs:
  a demoted signal keeps the line at warn. It cannot bind the reported *message*
  as `Check` is shaped — the merge keeps only the first synthetic per pool, so a
  demoted signal either loses the message to an existing `shellUnresolved` or
  displaces one. Settling that precedence is the implementation's, with one
  requirement this ADR does set: **a Tier 2 verdict must be distinguishable from
  a Tier 1 one in what the user is told.** Decision 2's whole rationale is that an
  unknown command is not proof it execs its arguments, and a reader who cannot
  tell a speculative warn from a literal one has not been given that.

Tier 2 converts the unbounded *wrapper* class — a command that execs its own
arguments — from *silent allow* to *loud warn* without enumerating anything, and
gives the wrapper path the fail-safe the interpreter path already had. It does
**not** reach the *exec-string* class: `su -c`, `runuser -c` and `script -c` keep
their payload as one opaque token that `isShellFamily` will not open. Six of the
ten verified bypasses are covered by speculation, seven once each speculative
suffix is run through `expandPayloads`; the remaining three are part C's whole
reason for existing.

**Shipped state.** All four parts are built. Of the ten bypasses, part B turned
seven into precise Tier 1 blocks and part C reached the other three, so none is
an allow and the fail-safe now stands underneath the enumeration rather than
behind it. The gate condition acquired a third clause in the building: a segment
whose payload the guard successfully READ is not an unrecognised launcher either,
or `busybox sh -c "<hazard>"` reports the same hazard twice — once as the entry
its payload matched and once as a guess about the parent.

**3. Tier 2 is bounded against the denial of service it would otherwise
reintroduce.** The obvious form of speculation is N starts × E entries — exactly
the per-entry splice-and-restart `payload.go` forbids by name. Prototyped, it
is strictly quadratic: 14.9 s at 6,000 tokens, and roughly **eight hours of CPU
inside a `PreToolUse` hook** extrapolated to the 1 MiB stdin cap. The required
shape is therefore binding: command-position starts computed once per segment in
an O(N) pass, a hard cap of ~64 starts **per segment** past which the warn is
emitted unconditionally rather than skipped, short-circuit on first hit, pinned by
a benchmark test.

**Amended when built: the start cap alone does not bound the cost.** Every start
pays `commandOf`, which walks the whole leading wrapper chain, so 64 starts on a
1 MiB line of `env env env …` measured **14.2 s** — linear in line length, and
still a hang inside the hook. A second bound was added: the tokens one start
reads are capped (`maxSpeculativeWindow`, 512), taking the same input to ~120 ms.
That is a coverage trade, and a small one — every pattern reads its command, the
operand at position 0, and flags, all of which sit near the command in any hazard
the Pattern language can express — but it is a trade, so a truncated window warns
through the same fail-loud path as an exhausted start cap rather than falling
quiet.

Both granularities bound the cost: per segment gives Σ 64·len(seg)·E for entry
matching, plus the `expandPayloads` call each speculative suffix now makes, itself
O(line × `maxPayloadDepth`) and including a re-tokenize — so the total stays
linear in line length at a constant of 64·2. The benchmark must exercise the
payload path, or it pins only the cheaper half. What the two granularities do
*not* share is warn behaviour, which is why the cap is specified here rather than
left to the implementation: a per-line cap spent early would fire the
unconditional warn on every remaining segment of a long line, and decision 8
makes warn rate the STOP.

**4. The Tier 2 gate is "no entry matched *this segment*" — never "argv[0] is
unknown", and never "no entry matched the command line".** Two separate
corrections, each with a counterexample:

- *Not argv[0].* An argv[0] gate switches speculation off for `git`, and git
  launches things — `git bisect run`, `git submodule foreach`,
  `git rebase --exec` are silent allows today and would stay silent under it.
- *Per segment, not per `Check`.* `Registry.Check` collects matches across every
  segment of the command line and merges them, so a whole-command gate hands an
  author a one-token suppression: `git clean -fd ; nice git push --force origin
  main` matches `git-clean`, which would switch speculation off for the *whole
  line* and leave the wrapped force-push silent. Prefixing any warn-tier command
  would disarm Tier 2 entirely. The gate is therefore evaluated per segment, and
  a segment that matched nothing is speculated on regardless of what any other
  segment did.

**"Unknown command" means `commandOf`'s output, never the literal first token.**
`commandOf` steps wrappers, environment-assignment prefixes and reserved words,
then takes the basename, and `matchSegment`'s first test is `cmd != p.Command`
against exactly that value — so `sudo git push --force`, `FOO=bar git push
--force` and `/usr/bin/git push --force` all match entries despite a first token
no entry names. With the term defined that way the nesting the floor claim rests on is **by
construction**: if `commandOf` resolves to something no entry names, no entry can
match that segment — `matchSegment` tests that first, before every other pattern
field — so every fire of a gate so defined is also an adopted-gate fire. One half
of the claim is therefore checkable from `match.go`; the other half, that the
prototype gated on `commandOf`'s output rather than the first token, is
**testimony**, since the prototype lived in the gitignored local tier. It becomes
checkable when the corpora land as committed test data, and not before. Under the literal
reading the nesting is simply false — `env git clean -fd gh repo delete .`
resolves to `git` and matches `git-clean`, while a literal-token gate would see
`env`, speculate, and fire on `git clean`'s own pathspecs. The term is defined
here because that ambiguity makes the difference between a floor and no floor.

**5. Enumeration continues, demoted to an upgrade.** Adding wrapper names lifts a
case from "loud warn" to "precise block" — better UX and a stronger verdict —
and it is safe to be incomplete precisely because Tier 2 backs it. It is no
longer the fix.

**6. The exec-string family gets a small table, not a generalisation of
`shellCPayload`.** `su -c`, `runuser -c`, `script -c`, `flock -c` take a command
string a shell executes, but none is a shell. Generalising the existing helper
resolves every *short* spelling free and returns `shellNone` for every long one
(`su --command`, `--session-command`, …) — six new **silent** allows, not even
the `shellUnresolved` fail-safe. A command → payload-flag table is the correct
shape and leaves that contract untouched.

**7. Sequencing is D → A → B → C**, the four parts being:

| part | what it is | decisions above |
|---|---|---|
| **D** | the scope statement in the guard's own documentation | 1 |
| **A** | the bounded Tier 2 fail-safe | 2, 3, 4 |
| **B** | wrapper names added to `wrappers` / `wrapperOperands` | 5 |
| **C** | the exec-string command → payload-flag table | 6 |

Decisions 7 and 8 are not parts: 7 is this table, and 8 is a gate that binds all
four.

D leads because shipping enumeration without it manufactures exactly the false
confidence that makes the next gap dangerous. C is last but not optional: until
it lands, three of the ten verified bypasses are still silent.

**8. The warn-storm STOP binds this work.** Per
[`2026-08-15-guard-execute-string-family-design.md`](../../research/notes/2026-08-15-guard-execute-string-family-design.md):
a warn rate that trains users to ignore warns is a STOP, measured on real agent
commands before merge.

## Alternatives Considered

**Add the missing wrapper names (the issue's own proposal).** Cheapest, and it
closes the ten verified bypasses. Rejected as the *primary* fix: it is the third
instance of the identical defect, the class is unbounded, and it leaves the
silent-allow verdict intact for everything not listed. Retained as decision 5,
below the fail-safe.

**Speculative re-match in its drafted form.** Rejected as drafted for the
quadratic DoS above, and because the coverage claim behind it was wrong —
speculation alone catches 6 of the 10 verified bypasses, not all of them, since
`matchSegment` never descends into a payload. Adopted only in the bounded,
`expandPayloads`-composed form of decisions 2–4, which recovers `busybox sh -c`
at zero additional false positives.

**Descend into any argument token containing a space.** Catches everything,
including `su -c` and `su bob -c`, at +2 false positives. Rejected with evidence:
it breaks the shipped known-good gate — `abcd capture "git clean -fd wiped my
scratch notes"` fires, and that fixture is an incident capture, the case spc-16
says the design turns on. The true-negative floor is 100%.

**Speculate only when every token before the suffix is flag-shaped.** Silences
the `rg -F git push --force docs/` false positive. Rejected: it also silences
`find . -exec`, `chroot /jail`, `flock /tmp/l`, `chrt -f 99`, `taskset -c 0`,
`docker run`, `ssh host` — most of the true positives.

**Invert to an allowlist.** The only shape that is complete by construction, and
what Claude Code's read-only fast path does. Rejected for this surface: the guard
sits on `PreToolUse` for arbitrary developer work, where an allowlist either
blocks ordinary commands or grows into the same unbounded list from the other
side. Recorded rather than dismissed — if the guard ever gains a fail-closed
mode, this is the shape it takes.

**Demote the parse layer to advisory and put the boundary elsewhere.** Partly
adopted: decision 1 *is* the demotion of the claim. Not adopted in full, because
removing the block tier would discard precise, correct verdicts on the cases the
registry does match, and abcd is host-delegated by default ([adr-25](0025-host-delegated-llm-default.md))
— the execution-layer boundary belongs to the host, not to abcd.

## Consequences

- **The guard stops lying about the wrapper cases it cannot see.** An
  unrecognised *wrapper* becomes a loud warn instead of a confident allow, and
  that is not hostage to a list staying complete. The *exec-string* family is not
  covered by the fail-safe — part C covers it by enumeration, and until part C
  lands `su -c`, `runuser -c` and `script -c` are still silent.
- **False positives become the accepted cost, and the measured figures are a
  floor rather than an estimate.** The corpora — 0 fires across 1,144 repo-mined
  lines, 21 of 79 on an adversarial corpus, 2 of those true positives — were run
  against a prototype gated on argv[0], which is the gate decision 4 **rejects**.
  Because decision 4 evaluates the gate per segment, every prototype fire is also
  an adopted-gate fire — the figures are a floor and not merely a different
  measurement — and the margin above it is unmeasured: `git grep git push --force`
  matches no entry today, so the argv[0] gate silenced it while the adopted gate
  speculates and warns. A whole-command gate would *not* have this property, and
  is rejected in decision 4 for a worse reason than noise. The shape is unquoted hazard-shaped text in a
  command line where nothing matched; quoting silences it. This repo's subject
  *is* the hazard registry, so expect interactive hits here specifically.
- **Discharged when built.** Both corpora were re-measured against the shipped
  implementation and now live in `internal/core/guard/testdata/corpus/`: 961
  command lines mined from this repository's prose, Makefile, scripts and
  workflows, on which Tier 2 fires **zero** times, and a labelled adversarial
  corpus carrying all four directions. The counts move as parts B and C upgrade
  warns to precise blocks, so the corpus file and its test's own log line are the
  figure — not a number copied into this record, which is how the first version of
  this paragraph came to disagree with the data it was discharging. The real implementation is
  substantially quieter than the argv[0]-gated prototype predicted, because a
  hazard inside a quoted argument or a path never reaches command position at
  all. The ceiling is enforced by a test, and raising it is the visible cost of
  making the guard noisier.
- **A new standing obligation: the DoS bound is load-bearing, not an
  optimisation.** The start cap and the O(N) pass carry a benchmark test, and any
  future change to matching re-runs it. The guard gates the hook; a slow guard is
  an outage.
- **The evidence is committed, not recounted.** Both corpora and the DoS bound
  ship as test data and tests (`corpus_test.go`, `speculate_bound_test.go`,
  `BenchmarkCheck`), so every figure above is now a claim the next change has to
  keep true rather than a run the reader has to take on trust.
- **Wrapper and payload-flag tables are documented as one platform's grammar.**
  `nsenter unshare chrt taskset ionice setsid flock stdbuf runuser` do not exist
  on macOS, and `su`, `script`, `chroot`, `nice` carry BSD grammars there. CI runs
  both platforms, so the table says which platform it encodes.
- **`su -c` is not guaranteed to be shell grammar** — it runs the target user's
  login shell, overridable with `-s`. This costs false negatives only (a mis-parse
  yields a non-match, never a false block), and it belongs in the scope statement
  rather than in a claim that the family parses uniformly.
- **Residuals stay open and named**, so no reader mistakes the fix for a boundary:
  top-level `curl … | bash` is a silent allow today (`pipesIntoInterpreter` is
  consulted only inside a payload — separate defect, found while testing), and
  `echo <b64> | base64 -d | sh`, `make push-force`, `npm run deploy`, shell
  functions, aliases and variables are permanently invisible to any parser.
- **The next gap in the *wrapper* enumeration is no longer an incident — the
  other two enumerations are untouched.** A missing wrapper name degrades a block
  to a warn instead of to silence, which is the whole point of spending the false
  positives. The fail-safe does not extend to the sibling enumerations this ADR's
  own Context table names: a missing *interpreter* name (`fish -c "git push
  --force …"`) and a missing *git value flag* (`git --newglobal X push --force …`)
  are both still silent allows under Tier 2, for two *different* reasons. The
  interpreter case hides the hazard inside one opaque token: a start does land on
  that token, but it resolves to a command name no entry knows and
  `expandPayloads` will not open it, because `fish` is not in `isShellFamily`.
  That one *is* repairable by enumeration — it is gh-297's shape. The git case is worse: a start *does* land
  on `push --force origin main`, and it resolves to the command `push`, which no
  entry names — the gh-299 class displaces the operand **after** the
  command-position boundary, and speculation only ever moves that boundary, so no
  number of extra starts can reach it. Stating it as "no start lands on it" would
  invite the repair that cannot work. The thesis is that a third instance predicts
  a fourth; this record covers the fourth only if it lands in the wrapper set.
- **Two labels are now wrong and are corrected with part D.** gh-297 and gh-299
  still carry `severity:critical` for a class decision 1 rules out of that
  category. Relabelling them is part of the scope statement's landing, not an
  optional tidy-up: leaving them is the same false-confidence signal from the
  other direction.

## References

[cc-permissions]: https://code.claude.com/docs/en/permissions "Claude Code — permissions and Bash command-pattern matching (Anthropic docs)"
[cursor-security]: https://cursor.com/security "Cursor security overview — run modes and terminal allowlisting; the GHSA-82wg-qcm4-fp2w / CVE-2026-22708 advisory is published separately"
[cwe-184]: https://cwe.mitre.org/data/definitions/184.html "CWE-184: Incomplete List of Disallowed Inputs (under CWE-693 Protection Mechanism Failure)"
[ranum-dumbest]: https://www.ranum.com/security/computer_security/editorials/dumb/ "The Six Dumbest Ideas in Computer Security (Ranum, 2005) — 'Enumerating Badness'"
[garfinkel-traps]: https://www.ndss-symposium.org/ndss2003/ "NDSS 2003 proceedings index — Garfinkel, Traps and Pitfalls: Practical Problems in System Call Interposition Based Security Tools (link is the proceedings, not the paper)"
