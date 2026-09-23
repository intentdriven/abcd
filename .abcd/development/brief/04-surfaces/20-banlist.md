# `/abcd:banlist` — Banned Names, Two Layers

Some names must never be published: a private collaborator, a client project, a
lab machine. Once one reaches a public repository it cannot be fully taken back —
merged-PR diffs and cached views persist server-side long after a history
rewrite. A second class is cheaper but still costly: a tool that claims to be
host-agnostic undermines itself the moment its published docs name one specific
harness.

`/abcd:banlist` declares both classes as banned names and stops them at authoring
time, where prevention is cheap. Bare invocation and `list` are **strictly
read-only**; `add` and `remove` are the write paths, and each names its layer
explicitly.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `add` | — | shipped |
| `list` | — | shipped |
| `remove` | — | shipped |

`add` and `remove` each name their layer with `--private` or `--public`, and
neither defaults: a write that guessed the layer would be a private pattern
published, or a public ban nobody can see. On `list` the same two flags are
optional and narrow the render to one layer; naming neither renders both, which
is what bare invocation already does. `add` takes a key and a pattern, and the
pattern `-` reads one line from **stdin** instead, so a private pattern never has
to sit in a shell history or a process list. `--severity` and `--successor` shape a public
entry only, because a private entry has one severity and its refusal names
nothing but its key. A private add still accepts both and silently does nothing
with them, which is a rough edge rather than a guard: nothing tells the user the
flag did not apply.

## Why two layers

A deterministic CI gate is the right tool for a *public* banned name and the
wrong place for a *private* one, because the rule would have to contain the very
string it forbids. Splitting enforcement by sensitivity resolves that without
compromise.

| | public layer | private layer |
|---|---|---|
| store | `.abcd/docs-lint.json`, the `banned_tokens` family | `.abcd/.work.local/private-names.txt`, gitignored; inside a linked worktree the primary checkout's copy is read as well (below) |
| enforced by | `abcd docs lint` in CI, per-line escape | the committed `.githooks/pre-commit` and `pre-merge-commit` guards |
| reach | every clone and pull request, when the config is tracked (see below) | only machines that have opted in, and only the commits git runs a hook for |
| visibility | entries render in full | entries render **by key only** |

The public layer is not a new mechanism. It is the `banned_tokens` family that
already gates this repo's harness names, so an entry a verb writes and an entry a
human hand-curated are enforced by the same engine with the same escape hatch.
Verb-written entries carry the `names/` id prefix, which is the ownership
boundary — and it holds in one direction only. `list` shows the whole family, and
a removal is refused for anything outside that namespace, so the verb cannot
delete an entry a person curated by hand elsewhere in the family. Nothing stops a
person writing an entry *into* `names/` by hand, and this repository's own config
already carries one: it renders as verb-managed, and a removal would take it.
Read the prefix as the place the verb writes, not as proof of what wrote an
entry.

## The private store's format is declared, not guessed

The store's first line decides how the whole file is read. A store whose first
line is exactly `# abcd-banlist: keyed` is a *keyed* store: every non-comment,
non-blank line parses as `KEY<space-or-tab>PATTERN`.

```text
# abcd-banlist: keyed
lab-host   alice-laptop\.example\.com
lab-ip     192\.0\.2\.17
```

`KEY` is a stable, non-sensitive handle and the only part of an entry that ever
reaches any output. `PATTERN` is a POSIX extended regular expression matched
case-insensitively: the engine is the guard's `grep -iE`, so `(?i)` is never
needed and Perl escapes such as `\d`, `\w` and `\b` are unavailable. Machine
identifiers — hostnames, IP addresses, CIDR prefixes, MAC addresses, device names
— are ordinary entries, matched exactly as a name is (the fixture values above are
reserved documentation values, per
[`examples-use-reserved-identifiers`](../../principles/examples-use-reserved-identifiers.md)).

A store **without** that first line is a *legacy* store, the format the guard
shipped with: every non-comment, non-blank line is one whole-line pattern under a
synthetic key, and no line is ever split. An old store keeps matching exactly what
it always matched, and no part of any line can be printed. That is the whole
reason the declaration exists. Deciding per line whether a first field "looks like
a key" did two harmful things at once: it printed part of a legacy line as a key —
and on this layer a pattern *is* the secret — and it narrowed an old whole-line
pattern to the remainder after its first field. `add` and `remove` refuse a
non-empty legacy store for the same reason: writing a keyed line into it would
change what every *other* line means.

The store has a second writer, and the format declaration is what lets the two
share it. The sources corpus derives patterns from its confidential entries and
maintains them inside a fenced generated block in the same file, refusing a target
that does not carry the declaration and leaving every line outside its block
untouched. So a hand-added private entry and the corpus sync write into one store
without either clobbering the other. See [`13-consult.md`](13-consult.md) for the
corpus side of that contract.

Leading and trailing ASCII spaces and tabs are stripped, and so are a trailing
carriage return on any line and a byte-order mark at the very start of the file,
so a store saved by a Windows editor reads the same as one saved anywhere else;
nothing else is stripped, and the Go parser and the shell hook strip the same
set, byte for byte. A whitespace class that differed between the two readers
would be a line one of them silently ignores while the other reports it as live.

## What the guard does at commit time

The guard checks the **content of every staged file**, read out of the index
rather than out of diff text. That is the question it is actually asking — is this
name in what I am about to commit? — and unlike diff text it has no shape to route
around: a content line beginning `++`, a blob containing a NUL, a committed
`.gitattributes` carrying `-diff`, and a rename all hide a name from a diff-text
reading. Binary blobs are scanned like anything else, because a name in a binary
file is in history just the same.

On a match the guard refuses the commit and names **the key alone**. The matched
string and the pattern never reach stdout, stderr, or a log — a refusal that
echoed the string would defeat the layer at the moment it worked — and the pattern
reaches grep on stdin rather than in argv for the same reason.

Everything that could go wrong fails closed and loud. A line that does not parse,
and a pattern the engine refuses, are each themselves a refusal naming a line
number and nothing else: an unusable entry is never skipped, because a banlist
that cannot be read must not look like a banlist that found nothing. Any git step
that fails refuses the commit too. An absent store prints a loud `INACTIVE`
warning and lets the commit through, and a store that exists but yields no entries
prints an equally loud `NO ENTRIES` warning: the layer protects machines that
opted in, and silence must never impersonate protection.

### Inside a linked worktree, the primary checkout's store is read too

A checkout is not always the only store in play. Inside a linked git worktree both
the guard and the read surface resolve the primary checkout's store as a read-side
fallback (itd-150): the guard reads it alongside the local store, and `abcd
banlist` renders it under an `inherited from the primary checkout` heading,
entries by key as ever. So an `INACTIVE` local store in a linked worktree does not
mean the commit goes through unchecked, and the render says so on the same screen.

The fallback is **read-side only**, and that asymmetry is the thing to hold on to.
`add --private` in a linked worktree writes to that worktree's own store, which
the primary checkout's guard does not read back: a name that must be enforced in a
given checkout is declared in that checkout.

## Two ways an entry fails, reported apart

`abcd banlist list --private` distinguishes a line the guard's engine **cannot
use** from one it **accepts and reads differently**, because the two need opposite
responses. An unusable line stops every commit until it is fixed. An inert line —
a Perl-style escape, an inline flag group — stops nothing: grep may read it
differently than written, so the name goes unguarded while the store looks
healthy.

`add --private` refuses both up front, screening the constructs POSIX ERE does not
implement and then asking grep itself, with the pattern on stdin, whether the
expression is usable. A private pattern is therefore checked against the engine
that enforces it rather than against Go's, which accepted `\d` and `(?i)` as
healthy and rejected `[a-z-.]`, which grep refuses.

Because the store's safety rests entirely on its being untracked, `add --private`
refuses outright when git does not ignore the store's path: the guard cannot catch
its own source.

## Scaffolded, not hand-wired

A repo becomes name-safe by being abcd-managed. `abcd ahoy` lays down the guard
artefacts, and four of the five carry a condition, because a scaffolded file abcd
would immediately declare unenforceable is worse than an absent one.

| artefact | where | written when |
|---|---|---|
| guard hook | `.githooks/pre-commit` | when absent. Committed, so every clone inherits it; a clone arms it once with `git config core.hooksPath .githooks` |
| merge guard | `.githooks/pre-merge-commit` | only beside abcd's own guard. git runs no `pre-commit` for a merge commit, so the same guard needs a second entry point; the shim delegates to whatever occupies `pre-commit`, so beside a foreign hook it would both claim coverage it has not got and silently start running the maintainer's hook on merges |
| EOL pin | `.gitattributes` | only beside abcd's own guard. One appended line keeps the hooks at LF, because a `core.autocrlf` checkout rewrites a script git executes and its shebang stops resolving |
| public family | `.abcd/docs-lint.json` | only where git says the path would be tracked. Seeded with abcd's own Writing-Guide rules armed (the `present_tense`, `punctuation` and `spelling` token families, held to the set abcd runs on itself, and the `links_resolve`, `harness_leak` and `stray_root_docs` rules) and with **no** banned names: abcd cannot know which names a repo may not publish, and a ban nobody declared would fail a build over a word the repository never chose. The `harness` token family is left for the repository to declare, since refusing to name a specific agent tool is wrong for a repository whose content teaches those tools (iss-2609150805167646) |
| private stub | `.abcd/.work.local/private-names.txt` | only where git itself reports the path as ignored. A stub git would track is the hazard, not the remedy |

Every write is create-if-absent and keyed on the gap actually detected, so nothing
overwrites a file the maintainer owns. Every write is also **contained**: paths
resolve through an `os.Root` opened at the repo, so a symlink committed at
`.githooks` or at the local tier cannot land a hook or a stub outside the repo
while the surfaces report the in-repo path.

Presence is not identity, and identity is not integrity. Each hook carries an
`# abcd-name-guard: v1` line, matched as a whole line; a hook without it is a
**foreign** hook that abcd reports and never replaces, because a maintainer's own
`pre-commit` is legitimate and calling it "the abcd guard" would mean nothing
checks the banlist while the status board says something does. But any file
carrying that line is treated as abcd's, including one edited to check nothing.
The marker answers "did abcd put this here", well enough to keep abcd off a
maintainer's hook; it is not a signature. "Committed" is likewise not "armed" —
git runs the hook the clone's hooks path selects, which abcd neither sets nor
fully observes, so every surface prints the arming instruction rather than
claiming the guard is running.

The stub's worked examples are all commented out, so a fresh scaffold parses to
zero entries and the guard says so loudly at commit time rather than looking like
protection. Every illustrative value in it is a reserved documentation value or a
persona-derived fixture host, per
[`examples-use-reserved-identifiers`](../../principles/examples-use-reserved-identifiers.md).

## What the copy refusals reach, and what they do not

Three tests stop the private store itself from being committed: any staged path
inside the local tier, including a rename's source path; a staged blob whose first
line is the format declaration; and a staged path whose filename is that of the
store.

These are **shape tests on a mistake, not a net against an adversary**, and it is
worth being exact, because a guard described more broadly than it works is a guard
people stop checking behind. A copy with the declaration stripped or altered by
one byte passes the first-line test; a legacy store declares no format at all and
is invisible to it, and passes the basename test the moment it is renamed. What
all three catch is the accident that actually happens: a `cp`, a `.bak`, a stray
editor duplicate. Widening them means scanning further into every staged blob,
which puts the secret into more code paths to protect it in fewer.

The escape is published because the tests are shape tests: a repo that
legitimately commits a store-shaped file — a fixture corpus, a document quoting
the declaration — needs a way to say so, and `--no-verify` is an off switch for
the whole guard rather than a per-file escape. A blob whose **second line** reads
`# abcd-banlist-example` is exempt from the first-line and basename tests, and
from nothing else. The exempt blob is still scanned against every entry, so it
cannot smuggle a plaintext banned name past the guard; and it is not protection
for a copy of the store itself, since a store's entries are escaped regular
expressions that do not match their own text. The trap worth naming is
second-order: a live store carrying the marker on its own second line exempts
every copy of itself, so the marker belongs in fixtures and documents, never in a
store holding real patterns.

## What the public layer does not reach

The public layer's claim is that it is committed and enforced for everyone, and
that claim depends on the config being tracked. `.abcd/docs-lint.json` sits inside
the namespace `visibility: public` fences, so the question is a live one.

On a repo that commits the documented three-tier layout the fence settles it: an
anchored `/.abcd/` ignore entry is narrowed to the local tier whenever git reports
any tracked file under `.abcd/`, because an ignore rule cannot untrack committed
records and a wholesale fence there would only hide new ones. So in a normal
abcd-managed public repo, this repository included, the config is tracked, CI reads
it, and the public layer reaches every clone.

The reach that remains unmet is the repo that commits nothing under `.abcd/`, and
the repo whose git cannot be asked. Narrowing needs positive evidence, so both keep
the wholesale fence, and a public family written there would reach no CI run. abcd
does not write one: the scaffold withholds the config in exactly those two cases,
and where a config is already sitting in an ignored path, detection reports it and
the status board reads "public family NOT ENFORCEABLE". Not claiming enforcement is
the whole remedy, and it is the shipped one.

## Honest reach

`abcd banlist` states plainly that CI cannot enforce the private layer, and so does
every other surface that describes it: `abcd ahoy`'s status board and the detection
envelope both carry the sentence beside the state it qualifies. It lives once, as
one exported string, because a human line, a verb, and a machine consumer reading
the envelope must not be able to disagree about what "hook committed" beside a
present store means.

The sentence names the second limit too, because "machines that have opted in" is
necessary and not sufficient. A hook sees the commits git asks it about, and that
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
