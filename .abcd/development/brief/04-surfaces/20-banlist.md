# `/abcd:banlist` — Banned Names, Two Layers

`/abcd:banlist` maintains the names a repo must not publish. Bare invocation and
`list` are **strictly read-only**; `add` and `remove` are the write paths, and each
names its layer explicitly.

Two failure modes share one root. A tool that claims to be host-agnostic
undermines itself the moment its published docs name a specific harness. Worse, a
private collaborator's, project's, or machine's name leaking into a public repo is
a confidentiality breach a history rewrite cannot fully undo — merged-PR diffs and
cached views persist server-side. Both are cheap to prevent at authoring time and
expensive to remediate afterwards.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `add` | — | shipped |
| `list` | — | shipped |
| `remove` | — | shipped |

Every sub-verb names its layer with `--private` or `--public`, and neither
defaults: a write that guessed the layer would be a private pattern published,
or a public ban nobody can see. `add` takes a `<key>` and a `<pattern>`, and the
pattern `-` reads one line from **stdin** instead, so a private pattern never
has to sit in a shell history or a process list. Two further flags shape a
public entry only: `--severity` (`blocker`, the default, or `warn`) and
`--successor`, the replacement the docs-lint finding cites when it fires
(default "a generic term"). The private write path takes neither and ignores
both, because a private entry has one severity and its refusal names nothing but
its key.


## Why two layers

A deterministic CI gate is the right tool for a *public* banned name and the wrong
place for a *private* one, because the rule would have to contain the very string
it forbids. Splitting enforcement by sensitivity resolves the tension without
compromise.

| | public layer | private layer |
|---|---|---|
| store | `.abcd/docs-lint.json`, the `banned_tokens` family | `.abcd/.work.local/private-names.txt`, gitignored; inside a linked worktree the primary checkout's copy is read as well (below) |
| enforced by | `abcd docs lint` in CI, per-line escape | the committed `.githooks/pre-commit` and `pre-merge-commit` guards |
| reach | every clone and pull request — **when the config is tracked** (see below) | only machines that have opted in, and only the commits git runs a hook for |
| visibility | entries render in full | entries render **by key only** |

The public layer is not a new mechanism. It is the `banned_tokens` family that
already gates this repo's harness names — one canonical primitive, so an entry a
verb writes and an entry a human hand-curated are enforced by the same engine with
the same escape hatch. Verb-written entries carry the `names/` id prefix, which is
the ownership boundary: `list` shows the whole family, and a removal is refused for
anything outside that namespace.

## The private entry format

**The store's first line declares its format**, and that one line decides the whole
file. A store whose first line is exactly `# abcd-banlist: keyed` is a *keyed*
store: every non-comment, non-blank line must parse as `KEY<space-or-tab>PATTERN`.

```text
# abcd-banlist: keyed
lab-host   alice-laptop\.example\.com
lab-ip     192\.0\.2\.17
```

`KEY` is a stable, non-sensitive handle (`[A-Za-z0-9][A-Za-z0-9._/-]*`) and the
only part of an entry that reaches any output. `PATTERN` is a POSIX extended
regular expression matched case-insensitively — the engine is the guard's `grep
-iE`, so `(?i)` is never needed and Perl escapes such as `\d`, `\w` and `\b` are
not available. Machine identifiers — hostnames, IPv4/IPv6 addresses, CIDR prefixes,
MAC addresses, device names — are ordinary entries, matched exactly as a name is
(the fixture values above are RFC 5737 and persona-derived, per
[`examples-use-reserved-identifiers`](../../principles/examples-use-reserved-identifiers.md)).

A store **without** that first line is a *legacy* store, the format the guard
shipped with: every non-comment, non-blank line is one whole-line pattern under the
synthetic key `entry-<line-number>`, and no line is ever split. An old store
therefore keeps matching exactly what it always matched, and no part of any line can
be printed. That is the whole reason the declaration exists. Deciding per line
whether a first field "looks like a key" did two harmful things at once: it printed
part of a legacy line as a key — and on this layer a pattern *is* the secret — and it
narrowed an old whole-line pattern to the remainder after its first field. The
declaration makes both unrepresentable, at the cost of one line a user adds by hand.
`add` and `remove` refuse a non-empty legacy store for exactly that reason: writing a
keyed line into it would change what every *other* line means.

**The private store has a second writer**, and the format declaration is what
lets the two share it. `~/.abcd/sources/bin/sync-banlist <repo-root>` derives
patterns from the confidential entries of the sources corpus and maintains them
inside a fenced generated block in the same file; it refuses a target whose
first line is not the declaration, and lines outside its block, whether a verb
wrote them or a human did, survive untouched. So `add --private` and the corpus
sync write into one store without either clobbering the other. See
[`13-consult.md`](13-consult.md) for the corpus side of that contract.

Leading and trailing ASCII spaces and tabs are stripped, and nothing else is — the
Go parser and the shell hook strip the same set, byte for byte. A whitespace class
that differs between the two readers is a line one of them silently ignores while
the other reports it as live.

## The guard's output contract

The guard checks the **content of every staged file**, read out of the index
(`git show ":0:<path>"`, stage-explicit so a path shaped like a revision cannot
redirect the read), not the text of a diff. That is the question it is actually
asking — is this name in what I am about to commit? — and unlike diff text it has no
shape to route around: a content line beginning `++`, a blob containing a NUL, a
committed `.gitattributes` carrying `-diff`, and a rename all hide a name from a
diff-text reading. Binary blobs are scanned like anything else, because a name in a
binary file is in history just the same.

On a match the guard refuses the commit and names **the key alone**. The matched
string and the pattern value never reach stdout, stderr, or a log — a refusal that
echoed the string would defeat the layer at the moment it worked. The pattern reaches
grep on stdin, never in argv, for the same reason. A line that does not parse, and a
pattern the engine refuses, are each themselves a refusal naming a line number and
nothing else: an unusable entry is never skipped, because a banlist that cannot be
read must not look like a banlist that found nothing. **Any** git step that fails
refuses the commit too — a check that could not run must never be indistinguishable
from a check that passed.

An absent store prints a loud `INACTIVE` warning and lets the commit through, and a
store that exists but yields no entries prints an equally loud `NO ENTRIES` warning:
it checks exactly as much. The layer protects machines that opted in, and silence
must never impersonate protection — which is why the read surface reports `present`
as a distinct state rather than rendering an empty list.

### Inside a linked worktree, the primary checkout's store is read too

A checkout is not always the only store in play. Inside a **linked git
worktree** both the guard and the read surface resolve the PRIMARY checkout's
`.abcd/.work.local/private-names.txt` as a read-side fallback (itd-150): the
guard reads it alongside the local store, and `abcd banlist` renders it under an
`inherited from the primary checkout` heading, entries by key as ever. So an
`INACTIVE` local store in a linked worktree does **not** mean the commit goes
through unchecked, and the render says so on the same screen it says `INACTIVE`.

The fallback is **read-side only**, and that asymmetry is the thing to hold on
to. `add --private` in a linked worktree writes to that worktree's own store,
which the primary checkout's guard does not read back: a name that must be
enforced in a given checkout is declared in that checkout. The store the
per-worktree tier gives each lane its own copy of is exactly why the read side
has to reach across and the write side must not.

## Two ways an entry fails, reported apart

`abcd banlist list --private` distinguishes a line the guard's engine **cannot use**
from one it **accepts and reads differently**, because the two need opposite
responses. An unusable line stops every commit until it is fixed. An inert line — a
Perl-style escape, an inline flag group — stops nothing: grep may read it
differently than written, so the name can go unguarded while the store looks healthy. `add --private` refuses both up
front, screening the constructs POSIX ERE does not implement and then asking grep
itself, with the pattern on stdin, whether the expression is usable. A private
pattern is therefore checked against the engine that enforces it rather than
against Go's, which accepted `\d` and `(?i)` as healthy (grep reads them
differently) and accepted `[a-z-.]`, which grep refuses (its fail-safe branch then
blocks every commit).

Because the store's safety rests entirely on its being untracked, `add --private`
refuses outright when git does not ignore the store's path: the guard cannot catch
its own source.

## Scaffolded, not hand-wired

`abcd ahoy` scaffolds five artefacts, so a repo becomes name-safe by being
abcd-managed. Four of the five carry a condition, stated in the table: abcd
writes an artefact where writing it is safe and reports the state where it is
not, because a scaffolded file abcd would immediately declare unenforceable is
worse than an absent one.

| artefact | where | written when |
|---|---|---|
| guard hook | `.githooks/pre-commit` | unconditionally, when absent. Committed, so every clone inherits it; a clone arms it once with `git config core.hooksPath .githooks` |
| merge guard | `.githooks/pre-merge-commit` | **only** beside abcd's own guard (`guardOwned`). git runs no `pre-commit` for a merge commit, so the same guard runs from a second entry point; the shim delegates to whatever occupies `pre-commit`, so beside a foreign hook it would both claim coverage it has not got and silently start running the maintainer's hook on merges |
| EOL pin | `.gitattributes` | **only** beside abcd's own guard, the same key: the pin belongs with the hooks and with nothing else. One appended line keeps them at LF, because a `core.autocrlf` checkout rewrites a script git EXECUTES and its shebang stops resolving |
| public family | `.abcd/docs-lint.json` | **only** where `publicPathIsWritable` says the path would be tracked: withheld where git would ignore it, and withheld where git cannot be asked, since a config written into a path nobody could check is the same wager either way. Seeded with an **empty** `banned_tokens` array, because abcd cannot know which names a repo may not publish and a ban nobody declared would fail a build over a word the maintainer never chose |
| private stub | `.abcd/.work.local/private-names.txt` | **only** where git itself reports the path as ignored (below). Inside the gitignored local tier |

Every one of the five is create-if-absent and keyed on the gap detection
actually raised, so nothing above overwrites a file the maintainer owns.

Presence is not identity. Each hook carries an `# abcd-name-guard: v1` line —
matched as a whole line, so a hook that merely mentions it in a comment is not
claimed — and a hook without it is a **foreign** hook: abcd reports it and never
replaces it, because a maintainer's own `pre-commit` is legitimate and calling it
"the abcd guard" would mean nothing checks the banlist while the status board says
something does.

Identity is not integrity, and abcd asserts only the first. ANY file carrying that
line is treated as abcd's guard — including one that quotes the template inside a
heredoc, and including a hook edited to check nothing — and it earns the merge shim
that delegates to it. What the marker answers is "did abcd put this here", well
enough to keep abcd from claiming a maintainer's hook; it is not a signature, and a
repo whose committed hooks a contributor can change is a repo where the guard's
behaviour is whatever that contributor last committed. "Committed" is likewise not "armed" — git runs the hook the clone's hooks
path selects, which abcd neither sets nor fully observes, so every surface prints
the arming instruction rather than claiming the guard is running.

Every write is create-if-absent, and every one is **contained**: paths resolve
through an `os.Root` opened at the repo, so a symlink committed at `.githooks` or
at the local tier cannot land a hook or a stub outside the repo while the surfaces
report the in-repo path. The reads detection makes are resolved through the same
root, so a report can never describe a file apply could not act on. Containment is
about the repo boundary and nothing finer: a symlink that redirects *within* the
repo (`.githooks -> docs/`) is followed, and a repo that commits one has chosen
where its own files live. A hook, a CI-gating config, and above all a populated
private store are the maintainer's, and re-seeding one would delete work abcd
cannot see.

The stub is written only where **git itself** reports the store's path as ignored —
`git check-ignore`, the primitive the `add --private` write path already uses, not a
comparison of `.gitignore` text. The two answer different questions: a repo can
carry a byte-perfect abcd block and still track the store (a negation after it, a
tracked tier), and a repo whose config step never ran has no block at all. A stub
git would track is the hazard, not the remedy, so the gap that asks for it is
advertised as resolvable only when git says the path is safe.

The stub's worked examples are **all commented out**, so a fresh scaffold parses to
zero entries and the guard says so loudly at commit time rather than looking like
protection. Every illustrative value in it is a reserved documentation value (RFC
5737, RFC 3849, RFC 2606, RFC 7042) or a persona-derived fixture host, per
[`examples-use-reserved-identifiers`](../../principles/examples-use-reserved-identifiers.md);
the scaffold test judges that with the repo's own network-identifier detector, and
proves the detector is armed with a control value first.

## What the copy refusals reach, and what they do not

The store is protected from being committed by three tests, and it is worth being
exact about each, because a guard described more broadly than it works is a guard
people stop checking behind.

| test | catches | escapable |
|---|---|---|
| tier path | any staged path inside `.abcd/.work.local/`, including a rename's **source** path | no |
| first line | a staged blob whose first line is the format declaration | yes — see below |
| basename | a staged path whose filename is `private-names.txt` | yes — see below |

These are **shape tests on a mistake, not a net against an adversary.** A copy with
the declaration stripped (`tail -n +2`), altered by one byte, or pushed below a
preamble passes the first-line test; a legacy store — which declares no format at
all — is invisible to it, and passes the basename test the moment it is renamed. The
rename-source test only fires for a store git already tracks, which is itself the
accident it exists to help clean up. What all three catch is the accident that
actually happens: a `cp`, a `.bak`, a stray editor duplicate. Widening them further
means matching against the store's key list or scanning further into every staged
blob, which puts the secret into more code paths to protect it in fewer.

The escape is published because the tests are shape tests: a repo that legitimately
commits a store-shaped file — this repo's own fixture corpora, a document quoting
the declaration, a file that happens to be called `private-names.txt` — needs a way
to say so, and `--no-verify` is an off switch for the whole guard rather than a
per-file escape. A blob whose **second line** reads `# abcd-banlist-example` — matched after the same
byte-order-mark and surrounding-blank normalisation the format declaration gets — is
exempt from the first-line and basename tests, and from nothing else.

Be exact about what that buys. The exempt blob is still scanned against every entry,
so it cannot smuggle a **plaintext** banned name past the guard. It is not, and
cannot be, protection for a copy of the store itself: a store's entries are escaped
regular expressions and do not match their own text, which is the whole reason the
copy refusals exist. The trap that follows is second-order and worth naming — a LIVE
store carrying the marker on its own second line exempts every copy of itself, so
the marker belongs in fixtures and documents, never in a store holding real
patterns.

## What the public layer does not reach

The public layer's claim is that it is committed and enforced for everyone, and
that claim depends on the config being tracked. `.abcd/docs-lint.json` sits
inside the namespace `visibility: public` fences, so the question is a live one.

On a repo that commits the documented three-tier layout it is settled by the
fence itself. `effectiveVisibilityEntries`
(`internal/core/ahoy/gitignore.go`, iss-255) narrows the anchored `/.abcd/`
entry — and only that entry — to `.abcd/.work.local/` whenever git reports any
tracked file under `.abcd/`, because an ignore rule cannot untrack committed
records and a wholesale fence there would only hide new ones. So in a normal
abcd-managed public repo, this repository included, `.abcd/docs-lint.json` is
tracked, CI reads it, and the public layer reaches every clone. The narrowing is
the mechanical boundary of what an ignore rule can do, not a per-subdirectory
carve-out of the one-switch decision: see
[`../05-internals/03-configuration.md`](../05-internals/03-configuration.md).

The reach that remains unmet is the repo that commits nothing under `.abcd/`,
and the repo whose git cannot be asked. Narrowing needs positive evidence, so
both keep the wholesale fence, and a public family written there would reach no
CI run. abcd does not write one: `publicPathIsWritable` withholds the scaffolded
config in exactly those two cases, and where a config is already sitting in an
ignored path detection reports `banlist.public_family_ignored` while the status
board reads "public family NOT ENFORCEABLE". Not claiming enforcement is the
whole remedy, and it is the shipped one.

[`iss-176`](../../../work/issues/resolved/iss-176-public-banlist-family-unenforceable-under-public-visibility.md)
asked whether the file should move, and is **resolved**: the premise it rested
on no longer holds, the relocation was built and abandoned as unnecessary, and
no maintainer decision is outstanding.

## Honest reach

`abcd banlist` states plainly that CI cannot enforce the private layer, and so does
every other surface that describes it: `abcd ahoy`'s status board and the detection
envelope both carry the sentence beside the state it qualifies. It lives once, as
`banlist.PrivateReachNote`, because a human line, a verb, and a machine consumer
reading the envelope's fields must not be able to disagree about what "hook
committed" beside a present store means — and the reading a machine consumer takes
on its own is the wrong one.

The sentence names the second limit too, because "machines that have opted in" is
necessary and not sufficient. A hook sees the commits git asks it about, and the
list is explicitly non-exhaustive: a fast-forward `git pull` creates no commit at
all, git runs no hook for a rebase, a `git am`, a `git revert` or a cherry-pick,
`--no-verify` switches it off, and a merge commit needs the `pre-merge-commit`
half. A reader who stopped at opt-in would believe an opted-in machine is fully
covered, which is the belief that gets a name committed.

None of that is a limitation to be fixed. A pattern in CI config is a published
pattern, and a guard is only ever asked about what git asks it about.

## References

- Plugin command: [`commands/banlist.md`](../../../../commands/banlist.md)
- Spec: [`spc-20`](../../specs/closed/spc-20-name-banlist.md)
- Intent: [`itd-74`](../../intents/shipped/itd-74-name-banlist.md)
- Public-layer gate: [`10-docs.md`](10-docs.md)
- Install surface: [`01-ahoy.md`](01-ahoy.md)
- The corpus writer that shares the private store: [`13-consult.md`](13-consult.md)
- The visibility fence and its tracked-tier narrowing:
  [`../05-internals/03-configuration.md`](../05-internals/03-configuration.md)
