# Changelog

All notable changes to abcd are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and abcd
uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html) with a
leading `v`.

Before v1.0.0, a minor release may make breaking changes: a record
declaring `impact: breaking` bumps the minor, and its entry is an **Added**
line that states the break. A derived section carries only **Added** and
**Fixed**: the composer that writes it sees the records that shipped and
not the previous release's surface, so it does not claim what changed,
was deprecated or was removed (iss-2609011207114761). The version number
is the signal; sections at v0.6.0 and earlier were rolled by hand and
some carry a **Breaking** heading.

## [Unreleased]

## [0.7.1] - 2026-09-02

These notes list what was added and what was fixed; changes to earlier behaviour are not claimed until the composer can see the previous release.

### Added

- **Updating abcd is now one verb, and it is always the user's verb.** `abcd update` completes what `abcd version --check` reports: it names the release it resolved, or takes an explicit tag, fetches the binary for this platform over a pinned transport, verifies it against that same release's `checksums.txt` before anything is replaced, swaps the PATH-installed copy atomically, and prints a receipt naming the origin, the old and new versions, and the digest it verified. Where another owner is in charge it refuses and names that owner's path instead — a plugin-root binary takes a plugin update, a package-manager install takes the manager's own upgrade command, and the dev shim is never silently replaced. abcd still never looks for an update on its own. (itd-130)
- `abcd version --check` now follows an available update with a `next:` line naming the command your install shape actually takes, carried as `check.next_step` in `--json`, and the eight schema-too-new refusals name that command too instead of saying only "upgrade abcd". (iss-2609012111168872)
- `scripts/pr-keep-current.sh` merges the base branch into every armed auto-merge pull request that is behind and not yet queued, once or on a watch loop, so a repository whose main protection combines a merge queue with the strict up-to-date policy stops stranding armed pull requests outside the queue that already gates them. The strict policy stays, and CONTRIBUTING points at the script. (iss-2609012202237613)

### Fixed

- **A command line the guard cannot parse is no longer run unguarded.** GHSA-5wx3-2c86-fjpx (CWE-636): the wired pre-tool-use hook failed open on two tokenizer states every shell an agent runs under executes — a backslash as the last byte of the line, and a here-document body that never reaches its delimiter — and ordinary input reached them, because a here-document whose redirection line ended in a list operator, and a delimiter that is not letter-led such as `cat <<20` or `cat <<$D`, both put document text into command position where one apostrophe raised the unparsable error. A stray later delimiter could instead swallow the real commands between as document body — a silent allow. A pending body is now consumed at the newline that ends its redirection line whatever that line ends in, a trailing backslash is parsed as bash parses it, an unterminated document takes a fail-closed block, and `guard check` and `guard hook` return the same verdict for the same command whether or not it ends in a newline. (iss-2609012040003576, iss-2609020347585681, iss-2609020543544634, iss-2609020348030423, iss-2609020450405136)
- GHSA-3w99-pgv4-8g55 (CWE-184): a hazard whose argv bash rewrites by glob expansion was an allow, because nothing recorded that a metacharacter was unquoted. The tokenizer now records per token that it carried an unquoted, unescaped `*`, `?` or `[`, and the matcher compares such a token as the pattern it is at every position an entry constrains, reading a globbed short cluster fail-closed and opening the payload of a globbed interpreter name such as `s? -c` or `ev?l`. Unconstrained positions are never compared, so `ls *` and `git add *.md` are unchanged. (iss-2609012040008545)
- GHSA-m2r8-fx7r-rq34 (CWE-184): the guard allowed a git hazard whose subcommand git rewrites from configuration the matcher stepped over. It now reads the alias declarations a segment carries itself — `-c key=value`, `--config-env` in both spellings, the `GIT_CONFIG_COUNT`/`KEY_n`/`VALUE_n` triple and `GIT_CONFIG_PARAMETERS` — and appends the command git would actually run, following a nested alias up to four hops and handing a `!`-alias body to the same inspection an execute-a-string payload gets. Each bang body also gets its own chain range now, so a `cd` in one body no longer reads as preceding an `rm` in another, while a `cd` and an `rm` in the same body still chain and block. (iss-2609012040016236, iss-2609020450498573)
- **A pasted private key no longer reaches disk with its body intact.** GHSA-29jw-3jg9-qmhx, GHSA-5qr6-f78x-g2cx and GHSA-gmp7-9rvm-qcr3 (CWE-312): the issue ledger, the memory store and the transcript store each matched the PEM BEGIN header alone, masked that one line and committed the base64 body and the END line verbatim — while reporting the text redacted. The shared scanner now covers a same-line body through its END marker and consumes the body-shaped lines that follow, through the END line, into one placeholder bounded at 4096 lines, so the record filename, the page body, the kept original and the stored transcript carry no body line. It covers the common renderings and not every one; the shapes the header pattern still cannot detect are recorded as an open follow-up rather than claimed closed. (iss-2609012029335752, iss-2609012029335717, iss-2609012029344995)
- **Every git identity you have is redacted, not only the one this repository resolves.** GHSA-rvhr-3455-c5jw, GHSA-gxhr-pmwv-r99p and GHSA-v826-5jf4-p8xg (CWE-359): the issue ledger, the memory store and the transcript store keyed redaction on the single effective `user.name` and `user.email`, so a repo-local or `includeIf` work persona displaced the global identity in the matcher set and the other one was stored in clear text. The probe now lists every value git resolves for the repository — system, global with its includes, and repo-local — and the matchers alternate over all of them. (iss-2609012029342529, iss-2609012029341137, iss-2609012029342817)
- A CI, `direnv` or rebase persona is redacted too. `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME` and `GIT_COMMITTER_EMAIL` outrank every config file when a commit is written but `git config` never lists them, so they were an identity source nothing redacted. They are folded into the probe's other-identity sets under the same guards as the config values — and only as others, so an injected variable can add a redaction target but can never displace the identity the config resolves. (iss-2609020128539293)
- **A memory page's frontmatter is redacted, like its body.** GHSA-x46m-mw9h-5jwj (CWE-312): `WritePages` redacted only the page body and dumped the frontmatter verbatim, so host-supplied `recall`, `contradicts` and citation scalars, a fetched origin and title, a detected licence and the weighting note reached the committed store unscanned — and on `abcd memory ask --file-back` those fields are host-supplied. Every string leaf a write introduces now goes through the store redactor under the same fail-closed gate as the body, and the sources registry is judged against a baseline copied before the merge, so a fresh entry is redacted while a leaf the registry already held is not re-judged. (iss-2609012037432966, iss-2608291941064448)
- A secret can no longer enter the memory store as a map key. The write-time leaf walk sanitised every string value a write introduced and never looked at the keys it walked, asserting in its own comment that keys are schema-fixed — false for the page `source:` block, which admits any identifier-shaped key. An introduced key is now judged by the same redactor before the value under it, and a key carrying a blocking secret or identity span refuses the whole write, naming the label and the kinds but never the key, because here the key is the secret. Refusal rather than rewriting is the rule: renaming a key renames the field a reader looks up, and dropping it discards the value it names. (iss-2609020239008775)
- `abcd memory ingest` no longer writes a URL credential into the committed store. Basic-auth userinfo and credential-shaped query keys are stripped from the fetched final URL before it becomes the page origin and title, so they reach neither the page frontmatter, nor `.sources_index.json`, nor the `--json` result, and every fetch failure, byte-cap refusal and redirect-count refusal renders the source with the password masked and those keys dropped. (iss-2609020238535054)
- GHSA-35fj-9w6f-7h62 (CWE-319, CWE-311): `abcd memory ingest` fetched plaintext `http` as readily as `https` and followed a redirect that hopped off `https`, so a man-in-the-middle on a source could rewrite its text and plant a licence. It now admits a local path or an `https` URL only, refusing any other scheme by name — naming the scheme it wanted and the one it got, with userinfo masked — before the fetcher is reached, and refusing a redirect hop that leaves `https` ahead of the SSRF host guard. (iss-2609012037433158)
- GHSA-xj89-cc2c-wgwr (CWE-313): `abcd memory lint` imported no scanner, so a page whose frontmatter or body already held a token or a home path yielded zero findings and a clean exit — and `ask --file-back` deep-copied the contaminated citation and licence onto the new page, so a tainted store propagated itself. Lint now runs the store redactor's read side over every stored page, the sources registry and each text kept-original, and reports a blocker naming the finding kind and the line but never the span, so its own report stays clean; a degraded scanner is itself a blocker rather than an abort. Lint reports and never rewrites the store. (iss-2609012037430191)
- GHSA-4fmm-95pf-32c6 (CWE-150): on an empty store `abcd memory ask` returned the question raw, so ESC, C1, bidi-override and zero-width runes given on the command line reached stdout and the `--json` question field. The question is sanitised once and that one value feeds both return paths and the JSON field; retrieval still tokenises the raw question, so ranking is unchanged. (iss-2609012037425074)
- GHSA-xq36-hcgf-9wrj (CWE-693): transcript staging deduplicated on the session id alone and took no lock, so a second `SessionEnd` for one session carrying different bytes was a no-op and the drain then stored the stale copy and deleted the only copy of the newer transcript. Staging now lists, compares content and writes under the staging lock and keys idempotency on the bytes — identical bytes are a no-op, different bytes replace the staged copy and the hook says so — and a drain removes a staged file only while it still holds the captured bytes, so a mid-drain re-stage is left for the next pass instead of being lost. (iss-2609012035100572)
- GHSA-j7v5-q7x6-v3rp (CWE-354, CWE-391): the armed gitleaks augmentation silently dropped any finding whose reported value spanned lines or was not verbatim, so a secret gitleaks had found was stored unredacted. A multi-line value is now split into one finding per line, so the line-scoped seal masks every line; a non-verbatim secret falls back to the reported match; a report that places nothing fails closed; and after redaction the capture refuses when any augmented finding's bytes still occur in the text, so verification is symmetric with detection. (iss-2609012035110990)
- A benign recurrence of one secret fragment no longer refuses every transcript capture for good. Detection located the whole reported value while verification checked each fragment's presence anywhere in the redacted text, so a transcript that also quoted a standalone PEM end-of-key line could never be captured — and the unredacted staged copy sat waiting for a drain that could never succeed. Both now work at one scope: each non-blank line of a reported value is located at every occurrence and sealed there, and a value counts as located only when every one of its lines is found. (iss-2609020231145566)
- **A hostile checkout can no longer redirect or hang the issue ledger.** GHSA-fh9j-8xmg-m33f (CWE-59, CWE-400) and GHSA-865x-5m7q-qm79 (CWE-59): the ledger readers loaded every well-formed record through a bare read with no regular-file check and no byte cap, so a committed FIFO at a record path blocked `abcd capture list` indefinitely; and the store was provisioned with a recursive mkdir that followed every ancestor, so a committed symlink at `.abcd` or `.abcd/work` redirected the whole store out of the tree. Both readers now go through one guarded, capped primitive that skips a FIFO, device, symlinked or oversize leaf, and every segment between the repository root and the ledger is judged before any verb touches the store — the read-only ones included — with a missing segment created one level at a time rather than through a link. (iss-2609012036271396, iss-2609012037119454)
- GHSA-cxmf-gw6r-2pf5 (CWE-20, CWE-696): `abcd capture promote` minted the intent draft before validating the source record, so a record the package's own invariants reject — a frontmatter slug disagreeing with the filename, a severity outside the enum — left an orphan draft behind on every attempt. The pre-flight now runs the strict and invariant validators before anything is minted and refuses with the validators' own error. The orphan-draft error a failed stamp does return now prints a repair that runs as printed: it carries the required `--grounds`, quoted for a POSIX shell, so the retry stamps the same conjecture the failed promotion was pursuing. (iss-2609012037122078, iss-2609012037130181)
- The status board's next moves for an open issue run as printed. `abcd <iss-N>` and `/abcd iss-N` printed `abcd capture promote <iss-N>` and `abcd capture resolve <iss-N> "<note>" --impact <...>` without the `--grounds` both routes have required since v0.7.0, so a reader who copied either remedy — README's first-run section points at the board — got exit 2 and "requires --grounds" instead. Both now carry the argument, and a test refuses any promote or resolve next move that omits it. The wontfix remedy was already correct. (iss-2609020746547554)
- **abcd trusts a binary only from a place an attacker cannot write.** `ahoy install` took any `CLAUDE_PLUGIN_DATA` value as its verified cache, including one that is relative, lies inside the repository being installed, or is world-writable; and the hook shims' PATH rung accepted an `abcd` that `command -v` resolved from inside the working tree or from a world-writable directory. Both now ignore such a resolution, with one line naming what was ignored and why, and take their existing loud degraded path — the pre-tool-use hook still says UNGUARDED and exits 1, and no symlink heal is offered from an untrusted cache. (iss-2609012039116200, iss-2609012039117381)
- GHSA-qc3w-8pv5-crc3 (CWE-312): the `origin` remote URL was stored verbatim, credentials included, reaching `~/.abcd/history/<sha>/meta.json`, the history index, `abcd ahoy --json`, its dry run and `abcd ahoy doctor --json`. The userinfo is scrubbed before the value enters the repository identity, so every one of those surfaces inherits a credential-free URL, and a legacy credentialed entry is rewritten rather than left for an install that would never revisit it. (iss-2609012039102035)
- GHSA-m8pg-chhv-hxvq (CWE-200): `abcd ahoy doctor --json` printed two raw absolute home paths in the `history.path_stale` gap, while every other path-bearing gap in the package already rendered them in tilde form. Both render through the same display path now, and the whole `doctor --json` output carries no home prefix. (iss-2609012039103376)
- GHSA-mchq-gm34-3j34 (CWE-754): no install step rewrites an `.abcd/config.json` it cannot parse. `ahoy install` used to discard the parse error, treat the file as empty and republish a meta-only config, and to plant marker blocks against the default target while ignoring the `docs.target` it could not read. Both refuse now, through one note per run naming the file, the parse error and the repair; detection raises a single non-resolvable diagnostic instead of the gaps that armed those rewrites under `--yes`; and a run that refused its config reports partial rather than clean. (iss-2609012039102703, iss-2609012039108437)
- GHSA-vvqc-3mv2-5p49 (CWE-15, CWE-426): the rules loader walked every ancestor of the working directory and took the first one holding a `.abcd` directory, so a `.abcd/rules.json` or `.abcd/guard.json` planted above the working tree — a world-readable temp directory, or the user-scope `~/.abcd` — decided what a session's agent was told. The root is bounded at the git working tree now: the walk stops at the toplevel, a tree with no `.abcd` resolves to its toplevel, a nested repository or submodule resolves its own toplevel instead of inheriting the superproject's, and outside git there is no walk at all. The banlist root and both guard loaders share the resolver. (iss-2609012039211929)
- GHSA-22f8-qf5r-gjgq (CWE-451, CWE-1427): a `.abcd/rules.json` override replaced a bundled domain's rules wholesale and nothing downstream recorded where the words came from. Every domain now carries its provenance: the injected context and `abcd rules` render an overridden domain as `## NAME (repo override)`, the prompt-router diagnostic prints the labelled names, and `abcd rules --json` carries a source field. One effect on upgrade: an overridden domain re-injects once, because the marker moves its dedup signature. (iss-2609012039215212)
- A rules override that declares no rules is dropped rather than injected as a heading with nothing beneath it — suppression wearing the domain's name, which an agent reads as a domain that says nothing. The rest of the file still loads and one diagnostic per dropped domain names it, the file, and the deliberate route, `"state": "dormant"`; both front doors print those notes out of band, never into the injected context. The bundled defaults are still refused outright, because there a ruleless domain is a build error. (iss-2609012039210402)
- A multi-line rule body stays inside its own bullet. Only a continuation line beginning with `#` was indented, so every other one rendered flush-left and un-bulleted: a continuation beginning with `- ` forged a sibling rule and one beginning with `#` forged a heading. Every continuation line is indented under its bullet now, a line already indented keeps its deeper indent, and a blank line stays blank. No bundled default carries a newline, so bundled signatures do not move; a multi-line repo override re-injects once. (iss-2609012039227456)
- **`abcd reading ingest` no longer deletes a committed record on its way to refusing.** With an orphaned stage present and its record already in the ledger, a payload that failed the type check deleted a committed reading record from the durable tier and printed only the type error. The ingest now probes the stage root read-only, validates the whole payload, and only then sweeps — the sweep is the verb's first delete in the committed tier, and a refused run never reaches it — and a refused or failed run reports the orphans it left in place, in text and in JSON alike. (iss-2608311517509690)
- The provenance field is no longer a signature-free channel through the supply-regime gate. The semantic detectors read the body text only and excluded the envelope pattern field, so an item whose pattern carried the registry's own phrasing landed with exit 0 at every regime — a registrative pattern saying the fix is to rewrite the charter, an explicative one carrying a disposition. The signatures now read every text value an item carries, the pattern included: it is untrusted text the reading chose, every item at every regime must carry it, and it lands in a committed record. An ordinary pattern still raises nothing. (iss-2608311517547712)
- The bare reading status tells a leftover stage from an orphan, instead of calling every stage directory an orphan and promising a rollback a committed run will never get. It probes each stage's commit marker — the same probe the sweep makes — and reports the two apart: a run with no marker is an orphaned ingest whose records the next validating ingest rolls back, and a run that committed and merely failed to clear its stage is reported separately, with the render saying that only the stage goes. (iss-2609012043437282)
- The terminal help for `abcd reading ingest`, and the reference page generated from it, no longer state three things the code does not do. It says now that a refusal becomes durable once the run's identity is proven and that nothing is written before that point; that the orphan sweep rolls the run's reading records out of the committed ledger and reports the ids on every exit; and which reserved names each regime actually declares, naming the generative position as declaring none. (iss-2608311518199679)
- Three smaller defects in `abcd reading ingest`. A definition that does not resolve now refuses through the recording path, so the refusal record exists with an empty regime and a reason saying why, rather than the run being refused with nothing durable to find it by. The bounded refusal list's elision entry names no item in either durable record. And U+2800 BRAILLE PATTERN BLANK is folded to a space, so a pattern that renders as nothing is refused at every regime. (iss-2608311518250688)
- The shared untrusted-prose cleaner stops mangling what it should leave alone and stops passing what it should break. A documented placeholder inside a code span shipped as a shell redirect, because the cleaner inserted a space after every `<` — correct and load-bearing outside a span, wrong inside one — while a faithful quotation of the form `items[0] (itm-0001)` landed as a live markdown link that made record-lint refuse the whole tree. Code spans are left byte-for-byte now, with every HTML opener and comment close still neutralised outside one; the length cap cannot cut inside a span and expose its content; and the bracket-then-parenthesis and bracket-then-bracket adjacencies are broken by a single space, with the intent audit's ingest routed through the one cleaner instead of keeping a second. (iss-2609011217083577, iss-2608311504353427)
- **Updating the abcd plugin is a non-event.** The checksum-verified hook binary is no longer kept in the commit-stamped plugin cache directory that every update replaces and garbage collection later deletes: the verified release artefact is kept once in the harness's persistent per-plugin data directory and copied into each fresh plugin root, so an update no longer re-downloads about 11MB and the window in which hooks run without their binary shrinks from once per plugin update to once per released binary. The `abcd` on PATH is a regular file abcd owns and refreshes, not a symlink into a directory the harness will delete. Each plugin root still runs and verifies its own binary, and a steady-state session pays one file test and no network. (itd-132)
- An unknown command or flag says so when the binary is stale. Where abcd can prove it from disk — the command surface at the resolved plugin root documents the refused verb or flag, or the disk-only vintage says the binary is behind its checkout tip or differs from the pinned release — the error carries a second line, and the remedy follows where that binary sits: `make build`, a plugin update, or `abcd update`. A typo with no evidence behind it stays verbatim, and the exit code and JSON envelope are unchanged. (iss-2608230943088357)
- Intents and specs mint their ids through the same timestamp seam the captures family already used, so two checkouts minting in the same window no longer allocate one `itd-N` or `spc-N` by construction. The max+1 scans over the refs union, the mint warnings and the integer-ceiling guards are gone, and the per-checkout lock now only serialises the presence check and the write inside one checkout. (iss-2608210737260468)
- A blocking issue the ledger reader had to skip still blocks its dependents. The `blocked_by` projection was built from the records that parsed and dropped the skipped roster, so an open record refused as a FIFO, an oversize body or a symlinked leaf was invisible to it — the dependent rendered with an empty `blocked_by_open` and sorted ahead of genuinely unblocked work. The projection now counts the whole of the open folder, adding back the id each skipped file's name claims, in the list and the status board alike, and the skip is reported in the same result so the cause is visible rather than silent. (iss-2609020154404742)
- A derived changelog no longer claims what it cannot see. The composer's inputs are the records that shipped and never the previous release's surface, yet `Changed`, `Deprecated` and `Removed` are claims about exactly that — so the ingest refuses those three and `Security` by name, only `Added` and `Fixed` are writable, and every derived section carries one sentence under its heading saying what the notes list and what they do not claim. A record declaring a breaking impact is an `Added` line that states the break. (iss-2609011207114761)
- The documented implement loop names its last step. `abcd spec close <spc-N>` ships the linked intent, and nothing in the build-review-merge routine invoked it, so an intent whose code was on main sat in `planned` while the release cut composed from terminal folders alone — two intents delivering a breaking CLI change were about to be released with no changelog line, invisibly, because the cut exits 0. The step is now stated in the intent command page, in the definition of done beside the issue-resolution rule it mirrors, in the lifecycle diagram as a manual step nothing runs for you, and in the bundled INTENTS rule domain. (iss-2609011026401952)
- The brief describes the binary that shipped. The `abcd reading` verb and the four cold-reading agent definitions, documented nowhere at v0.7.0, are now covered with their positions, their regime fields and corrected counts; and twenty-one drifted surface claims the machine-checked layer cannot see are corrected against the code — the required `--grounds` on promote and resolve, the seven-check ready gate and the Grounds section in the intent template, the ahoy, disembark, launch, site and identity details, and the guard citation — with duplicated enumerations collapsed to one home so they cannot drift twice. (iss-2609011424033149, iss-2609011424033603)
- The release-gate receipt is held to the tree it claims to have searched. The pinned-input manifest every receipt echoes as its hash had gone stale and under-scoped the record the crosscheck must read; it now pins all 22 command pages, all 15 agent prompts and every surface chapter, and a test fails when one of them ships unpinned or the prompt is edited without its hash. The gate also refuses a `judgeModel` that is a floating alias, not only a blank one, so a receipt naming a bare family name or a `latest` alias no longer passes as the pinned snapshot the runbook requires. (iss-2609011423385217, iss-2609012020521049)
- The issue-resolution check diagnoses a stale branch as a stale branch. On a branch 235 commits behind the base, the pre-push preflight emitted 84 violations each telling the reader to resolve an issue that already sat in a terminal folder at the base; RS001 now tells the shapes apart from the base-side history and says which — rebase and name the commit, drop the trailer, or check an id that names no record anywhere. The script also runs every git call with path quoting off, so a record whose slug carries a non-ASCII byte is found by its real path rather than silently read as unresolved. (iss-2609012023256534, iss-2609012047552618)
- Tests that build a throwaway git repository no longer race git's own background maintenance. The isolated git environment carries `gc.auto=0`, `gc.autodetach=false`, `maintenance.auto=false` and `core.fsmonitor=false`, after the parent's own injection is scrubbed, so a detached `gc --auto` cannot remove a `.git` file or churn a directory under a test's own tree walk — the class that dropped one pull request from the merge queue twice within an hour. (iss-2609020319494139)

## [0.7.0] - 2026-09-01

### Added

- **A cold reading sees exactly what the assembler passed it.** `abcd reading assemble` builds one reading's input by positive inclusion and by projecting fields out of files rather than copying paths, so a record type added later is excluded by default rather than included by oversight, and every run emits a manifest naming what was passed, by path and field, hashed and carrying the run identifier, so a reader can judge contamination instead of accepting a disclosure on trust. (itd-183)
- **Four reading definitions, one blindness core.** Widening, entailment, comparative and detection each hold their own object, question and regime value, with the seven blindness conditions carried byte-identically across all four, and no position may hold another's licence. (itd-184)
- **A reading is commissioned about something.** `abcd reading assemble --scope` takes a record id (`itd-N` or `spc-N`), a material kind such as `spec`, `discipline` or `test`, or the name of a preset committed at `.abcd/config/reading-presets.json`, and passes that rather than the position's entire corpus. Presets map position to scope, `warm` is `cold` plus a delta by construction, no repository path is accepted at the invocation, and the manifest records the effective scope, its hash, and whether a run overrode the committed preset. (itd-199)
- **An assembly says what a reading would cost before anyone commissions it.** Every assembly reports bytes and a byte-derived token estimate per material kind and in total, whether or not it writes an artefact, with test files counted apart from other source. No budget is enforced and none is invented. The assembler version pin is mechanical rather than advisory, so changing the include table without moving the version is refused rather than left to a convention. (itd-198, iss-2608311949385350)
- **A reading that exceeds its licence is refused at ingest.** `abcd reading ingest --reading-json` takes the path to one reading's returned output and validates it before any durable record is written: the instrument name, manifest reference, target state and regime value are checked per run, and each item's identifier is minted at validation rather than supplied by the reading. The flag is named for what the JSON contains, following the house pattern its siblings use. (itd-185, iss-2608311725218136, iss-2608310912206941, iss-2608301726130926)
- **The read-block eval can falsify the blindfold rather than assert it.** Sentinel warm content is planted across every warm location class in a fixture repository and asserted absent from the assembler's output, on fields rather than on paths, so a leak fails the build loudly. (itd-186)
- **Amnesia is a repository property, proven by an eval.** One definition assembled twice over an unchanged repository state produces byte-identical input, with the manifest outside the comparison and paths walked in lexicographic order, so no reading run is spent evidencing it. (itd-187)
- **The scribe sees the ledger and nothing else.** Machine assistance in maintaining the ledger transcribes reading outputs and researcher dispositions, authors nothing, and never receives the shipped repository as an object of judgement, so no session holds both a reading and the ledger. (itd-188)
- **A reading record and a disposition are two acts and two writes.** A reading record carries a run-scoped identifier, a run identifier, a manifest reference, its position and regime value, and a position-typed body; the researcher's response is a second record keyed to that identifier, from a vocabulary of four states (`accepted`, `rejected`, `declined`, `held`) whose availability varies by position. The record can therefore always show that a finding existed before it was answered. (itd-180)
- **What a widening reading proposes is admitted or declined on the record.** Every admission into the candidate set carries recorded grounds, every declined proposal carries a disposition, and surprises are their own entries, so uniform adoption can be told apart from judgement. (itd-189)
- **An intent says what kind of claim each of its sections carries, and its scope conditions survive their own rewording.** Criteria stay mandatory, the mechanism claim is prompted and may be declined with a recorded nullity, and scope conditions are required or declared absent as `None stated.` Each condition carries a persistent identity stamped by `abcd intent plan` and rendered by `abcd intent --json`, so a later disposition attaches to the claim rather than to wording an edit replaces. (itd-177)
- **A shipped intent's scope conditions are dispositioned by the fidelity verdict.** Each condition, by its identity rather than its wording, receives `survived`, `narrowed`, `falsified` or `untested`, so a narrowing is stated rather than implied by silently changed text and later work inherits only what held. (itd-181)
- **abcd records why a thing was pursued, at the moment it is pursued.** The readiness gate and capture's triage routes take a `--grounds` argument over a small vocabulary (`pursued`, `deferred`, `declined`), so the reasoning behind what goes forward is recorded rather than evaporating at the gate. (itd-179)
- **Every record written through a command says where its items came from and how their text was produced.** `origin` names the arrival path (`researcher-authored`, `contributed-by-reading` carrying the run and item identifiers, or `extracted-from-record`) and `--production-mode` distinguishes `hand-written`, `dictated-and-formatted` and `scribe-transcribed`. Both are stamped by the command, never typed by hand. (itd-178)
- **A lapse in the recording discipline is itself a capture category.** A `lapse` entry names the point in the process at which the discipline was suspended, deferred or evaded, and is timestamped at the lapse rather than at write-up. (itd-182)
- The assembler's exclusion floor holds against the ways a heading or a frontmatter key can be spelled: setext and indented headings, ATX closing sequences, case variants, character and entity references, raw HTML with inline markup, HTML comments, quoted and escaped keys, block scalars and nested flow mappings are all redacted before a bundle is written. The redaction is verified against the manifest's own claim on every heading path, the verdict is deterministic rather than dependent on map iteration order, and the scans are linear rather than quadratic in file size. (iss-2608300229101513, iss-2608300258519825, iss-2608300320162381, iss-2608300349300138, iss-2608300402591658, iss-2608300846123075, iss-2608300914552505, iss-2608300915534350, iss-2608301237450133, iss-2608301237457008, iss-2608301237458660, iss-2608301237459461, iss-2608301251385581, iss-2608301350527962, iss-2608301350537306)
- An assembly refuses rather than passes a bundle it cannot vouch for: the dirty-tree gate reads renames and copies so a file renamed out of the include set cannot be silently absent from a bundle the manifest calls HEAD, a store whose directory is missing or retargeted refuses the scan, a prior run's own bundle or manifest is refused by content signature rather than by file extension and top-level parse, a definition whose regime disagrees with its position is refused, the path deny binds case-insensitively on every component as its documentation states, and the scope carried in the bundle names no repository path. (iss-2608300229102495, iss-2608300229107715, iss-2608300258510460, iss-2608311145258479, iss-2608312019544150, iss-2608312058244357)
- Everything a reading returns is treated as untrusted text on its way to a terminal and to a committed record: item bodies and refused field names are sanitised and length-capped, bidi overrides, C1 controls and zero-width runes are refused rather than written into a markdown record, `claim_type` is held to its closed vocabulary of `criterion`, `causal` and `context`, the durable refusal reason keeps the payload-derived detail that explains it instead of truncating trusted prose, the refusal list is bounded in count, and the supply-regime gate is not evaded by a non-breaking space or an invisible combining mark inside a registered phrase. (iss-2608311211235195, iss-2608311230204063, iss-2608311230204232, iss-2608311234532073, iss-2608311234540729, iss-2608311306533109, iss-2608311306535485, iss-2608311351290623)
- An ingest owns its run. A run id that already committed cannot be re-ingested, two ingests landing in the same second cannot mint one item identifier, the ledger walk and the orphan sweep are contained against a symlinked ancestor and take capture's ledger lock so a concurrent invocation's records are not deleted under it, the definition is read and hashed through the repository root, an item whose record would exceed the ledger read limit is refused before it is written rather than committed permanently unreadable, and the size estimate accounts for both the record escaper and the ledger redactor. `commands/reading.md` states which list-level refusals leave a refusal record and which cannot, because the run's identity is not yet proven. (iss-2608300227228575, iss-2608311230116767, iss-2608311230301576, iss-2608311230305308, iss-2608311234537042, iss-2608311306530338, iss-2608311306536114, iss-2608311306536908, iss-2608311351239562, iss-2608311234541392)
- The conjecture recorded behind a gate decision is written once and kept. A malformed `--grounds` value is refused before any id is minted, so a bad argument cannot orphan a draft; the substance floor counts letters rather than runes and accepts a language written without inter-word spaces, so padding, zero-width filler and invisible text do not satisfy it; control, DEL, C1 and bidi runes are refused; the write takes the ledger lock so a concurrent second grounds write cannot discard the first; a later resolve or wontfix leaves the conjecture a promote recorded intact; an unclosed comment or fence in an issue record cannot blind the triage routes; and the lint gate, the writer and the reader agree on every scalar spelling of the field, with the refusal messages naming what the author must actually do. (iss-2608300927577163, iss-2608300930057882, iss-2608301206032013, iss-2608301206034359, iss-2608301206036067, iss-2608301212428844, iss-2608301301044588, iss-2608301455387735, iss-2608301620346560, iss-2608301646042379, iss-2608301657354776, iss-2608301803423101, iss-2608301805069999, iss-2608301808193750, iss-2608301908284034)
- A scope condition's identity survives ordinary editing. The marker is recognised anywhere in its bullet rather than only at the close of the first physical line, so an eighty-column reflow does not orphan the disposition keyed on it; the claim parser and the stamper skip fenced code blocks and HTML comments, so an example under a Scope Conditions heading is neither counted nor stamped and no marker is written inside a comment; a planned intent can still receive markers; and a draft carrying two or more unmarked conditions gets its pre-write size check rather than silently skipping it. (iss-2608300210588874, iss-2608300235377731, iss-2608300235388164, iss-2608300259316871, iss-2608300352403199)
- `abcd capture resolve --production-mode` and `abcd capture wontfix --production-mode` refuse a restamp on a record that predates disclosure, rather than appending a lone production mode that the provenance blocker then reports as a state no command produced. (iss-2608300925423309)
- The outstanding-answers report speaks whenever an item is unanswered or contested. Two standing dispositions on one item are both reported rather than the first by id hiding the second; an unreadable admissions tree is reported as unreadable rather than as a missing disposition; and the admission join keys on run and proposal together, resolves handles the way the report keys them, and constrains the proposal to a widening item, so an admission naming nothing real is refused instead of passing green and admitting nothing. Both standing-disposition readers converge on one parser in the shared schema leaf, so a preamble-led or duplicate-keyed record retires nothing on either side, and each guard is pinned by a test that fails when the guard is removed. (iss-2608300257429193, iss-2608300326346554, iss-2608300349499636, iss-2608300935215868, iss-2608300939008274, iss-2608301327013320, iss-2608301411017768, iss-2608301519254240, iss-2608301519254418, iss-2608301519255871, iss-2608301649339636, iss-2608301656193936)
- `abcd lint`'s `record_schema` rule covers the record stores this release adds, and configuration cannot blind it: a `record_stores` entry is validated against the stores the scanner knows, so no committed line can point a prefix at another store's bucket and exempt that directory from the walk. A required property that is quoted-empty, blank, an empty flow sequence or mapping, or an explicit null is treated as absent by every leg that reads it rather than as present by one and absent by another; the reader clause on a missing-property finding is stated per store rather than claimed for stores that have no validating reader; and a join to a retired handle gives one answer rather than two. (iss-2608300227224016, iss-2608300320001985, iss-2608300935218982, iss-2608301308369559, iss-2608301327012166, iss-2608301411010342, iss-2608301649337965, iss-2608301656192369)

### Fixed

- `abcd lint`'s `record_schema` findings no longer assert reader behaviour a store's reader does not have. The duplicate-key finding claimed the file was skipped by every surface of its store: no ADR reader refuses a duplicated key, and the release cut reads a resolved issue record leniently and folds it into the generated changelog, so an author told the record was invisible everywhere could find it in both. The finding now names the reader that does refuse, says what does not skip the record, and scopes the first-value claim to this rule's own scanner. (iss-2608301656200729, iss-2608301813253101, iss-2608301901260678)
- The test suite builds and passes on a clean machine: the reading CLI fixture pins its own git identity instead of relying on an ambient one a CI runner does not have, and the assembler's tree walk tolerates a `.git` file that git's background maintenance removes between enumeration and stat. (iss-2609010759382400, iss-2609010818066790)

## [0.6.9] - 2026-08-29

### Fixed

- **`abcd ahoy install --attribution` no longer fails silently when the opt-in was not persisted.** A read or write failure on `config.json` appends a change-note to the receipt, so a hook written to disk without its recorded opt-in is visible instead of silently making every later plain install a no-op for that hook. (iss-2608291814553362)
- **The check job fetches full history, so the record lint's agent-diff arm actually runs.** The unbumped-edit check on agent prompts has a present base commit to diff against, and the unarmed fallback prints a warning naming the missing base rather than reporting green. (iss-2608291814568191)
- **Linting an agent prompt whose frontmatter carries only `name:` reports every applicable finding.** The missing trust declaration is recorded and the check falls through to the `prompt_version` and changelog checks, rather than stopping at the first. (iss-2608291814563353)
- **The installer names the environment it ignores when a fetch fails.** The failure message lists the proxy, `CURL_HOME` and CA-bundle variables the installer deliberately ignores and says how to install by hand, so a host that reaches the download through a proxy or a custom CA bundle is diagnosed rather than guessed at. (iss-2608291941067275)
- **A blank licence is refused on both source shapes.** One shared check trims the value before testing it, so a licence of only whitespace is refused on a `sources[]` entry exactly as it is on a single-source page. (iss-2608291814563403)

### Security

- **GHSA-fg9r-3f8g-89m6: a hostile checkout can no longer make the gitleaks adapter execute a binary it committed.** A configured or PATH-located gitleaks binary runs only when it is absolute, resolved outside the repository both lexically and after symlink resolution, regular, executable and named `gitleaks` — a wrapper or renamed binary is refused; a refusal is loud and never falls back to PATH. (iss-2608291807456485)
- **GHSA-9wv7-88w3-f77m: a release payload can no longer ship a secret inside a skip-listed file.** Skip-listed bundle files are read through the guarded, capped primitive and their bytes scanned with the secret, harness-leak and long-literal identity rules, so the verdict for a home path, an email address, a session URL, and a real name that is multi-word or at least 8 bytes does not depend on the file's extension; a file that cannot be read or is oversized is an `Unscanned` refusal that says why, rather than an unverified allow. (iss-2608291807454357)
- **GHSA-fpf2-pg82-72rj: `abcd site build` and `abcd site check` refuse a symlink in the output path** — at the leaf, at any ancestor inside the repository, at the repository root and at any `.git` holder — and a purge additionally requires a marker naming this repository's root commit and a directory git tracks nothing in. A repository that commits its built site directory is refused by `abcd site build` and must be pointed at an untracked output directory. (iss-2608291807455620)
- **`abcd launch ship` no longer publishes the operator's absolute home path inside a binary file.** The byte scan waives both halves of the home anchor — a raw blob carries no path syntax on either side of the literal — so the operator's home is reported wherever it sits in a blob, and bytes are judged at least as strictly as text. (iss-2608292034215745)
- **A page written back by `abcd memory ask --file-back` is redacted before it reaches the store.** Redaction now happens inside the single page-write primitive, so no page body lands in the committed memory store unscanned and a file-back write is scanned exactly as an ingest is. (iss-2608291814566067)
- **A home path is replaced only where it stands as a path.** The sweep, the `home_path_self` detector and the placeholder rewrite share one anchor: a name may continue only with an alphanumeric byte, so a longer name such as `/rootfs` is another name and left intact, while a punctuation suffix such as `/root-cause` or `/root.old` is swept to the safe side. The survivor gate that refuses a home path surviving redaction is boundary-aware and can actually fire. (iss-2608291814568971)
- **A caller's own home nested under a longer root is swept.** A home of two or more segments — a temporary or CI home such as one under `/var/folders` or `/workspaces` — is taken by the sweep and by the detector wherever it sits, closing the gap that left it reported by nothing when `HOME` is not under `/Users` or `/home`. (iss-2608292036125100)
- **An install receipt no longer names your home directory or username.** Every change-note that embeds an error — the history-registration lock note and the attribution opt-in notes — renders through the receipt's path scrub, in the printed receipt and in `--json` alike. (iss-2608291941060605)
- **Every occurrence of a secret on one line is redacted, not just the first.** The gitleaks adapter's findings advance a cursor past each match and key one finding per line, column and value, so a retry log or a request-and-response echo carrying the same secret twice no longer leaves the second copy verbatim. (iss-2608291814551110)

## [0.6.8] - 2026-08-29

### Added

- **`abcd ahoy remote apply` turns on GitHub native secret scanning, then push protection, on a managed repo.** It sits behind a config opt-out and the caller's confirmation, and re-running it is idempotent. (iss-2608270636272755)
- **Repository-object settings now have a source of truth in the tree.** `.abcd/work/rulesets/repo-settings.json` mirrors the desired state and the merge-hygiene snapshot (delete-branch-on-merge and the allowed merge methods), the sibling the branch-ruleset mirror lacked. (iss-2608270512210664)
- **`DECISIONS.md` is gated append-only.** Checks DA001-DA004 (tail position, preservation, per-parent merge multiplicity, no NUL bytes) run in `make preflight`, the pre-push hook and CI, so a line inserted above an existing entry is refused rather than reordered. (iss-2608271804494867)
- **A lifeboat's provenance can declare a Pass B exemption.** `_provenance.json` carries a `pass_b_exemption` marker with its reason, the embark coverage handoff reads it as a declared exemption, and an unmarked record is treated exactly as before. (iss-136)
- **Docs lint refuses an em dash inside a list item.** `punctuation/em-dash-in-list-item` is a blocker and the `spelling/*` rules warn; the seventeen existing violations under `docs/` are fixed and the writing-style guide's labels match the shipped machinery. (iss-2608280706531199)
- **The writing-style guide gains an Audience section.** Audience-by-placement is ratified as adr-53: one register, density set by the Diataxis type, self-contained sections, terminology links, and a warn-then-promote rule for preamble. (iss-2608280750324657)
- This repository's own `DOCUMENTATION` rules override in `.abcd/rules.json` tells an agent to run `abcd docs lint` over a drafted docs artefact before presenting it; conversational replies stay unlinted. The bundled default domain shipped in the binary is unchanged. (iss-2608280825450623)

### Fixed

- **A fresh plugin install no longer dies silently without a binary.** `bootstrap.sh` closes with exactly one terminal line on every path, an EXIT trap turning a silent death into the same loud refusal; the launch bundler denies released platform artefacts structurally rather than by omission; and the Cut A section 4 assertions run as an automated fresh-install self-check on a Go-free PATH. (iss-253)
- **Auto-release no longer tags the wrong commit after a batched merge-queue push.** The release content commit is derived from the receipts directory instead of `HEAD^2^` ancestry, so a batch tip cannot resolve to an unrelated PR's commit. (iss-355)
- **The secret scanner no longer truncates a match at a fixed 512-byte window.** A galloping (exponential-doubling) probe grows the adjacency window only while a match keeps running into its edge, so a match end is never a window-edge artefact; the spurious over-redaction and broken-recovery-chain shapes land as regression tests with a deterministic cost-class guard. (iss-229)
- **Record lint now sees every ledger record capture would refuse.** `checkIssueRecordShape` mirrors capture's enum-membership, kebab-slug and unknown-key checks from the shared `issueschema` home, and the per-store filename match uses `recordid.FilenameNumRe`, so a record with an unknown frontmatter key or a divergent filename is flagged instead of being lint-green and silently dropped by the reader. (iss-2608261447039180, iss-2608270908342889, iss-2608270908346617)
- **Filename and frontmatter slugs must agree across all four record stores.** The shared `recordid` splitter and the `record_schema` blocker enforce it, and the two drifted records are corrected. (iss-2608271804178239)
- **`prepare-this-repo` is self-contained.** Every adopt-phase asset resolves from the record or the embedded binary, `ahoy install --attribution` scaffolds the `prepare-commit-msg` template, and an onboarding self-containment test refuses a machine-local path. (iss-87)
- **The site's by-links arrangement comes to rest.** It is sized from what each region holds and settles under the coil's own packing rule, so it publishes no overlapping positions, and the overlapping-bubbles gate now measures both arrangements in the renderer's own space instead of the coil alone. (iss-2608231350127745, iss-2608231322321751)
- `FoldPath` applies NFC after case-folding, so NFC and NFD spellings of one directory match in every folded gate (payload destination, pack overlap, embark's claimed key) on a normalisation-insensitive filesystem such as APFS. (iss-2608270926030978)
- `ahoy` repo registration surfaces a history-lock contention failure as a change note instead of discarding it, and a concurrent double-install re-applies a pending lineage link without overwriting the winner's. (iss-128)
- `docs/requirements.txt` and dotfiles are excluded from the built site via `exclude_docs`, so the Python pin file no longer ships as a public page. (iss-2608271711538151)
- The writing-style guide and the rules loader's DOCUMENTATION text no longer claim staged masking lints: adr-54 rules punctuation enforcement mechanical-only, the em-dash rule is enforced as a banned token, and the casing rows carry permanent review labels with their corpus-evidence reasons. (iss-2608280801428646, iss-2608280813441672)
- **`ACKNOWLEDGEMENTS.md` is backfilled from the attribution review.** Inspirations entries for CARL, PAUL, SpecStory, the Karpathy LLM-wiki gist, gitleaks, TruffleHog and Rams; the Diataxis entry completed with author and CC-BY-SA 4.0; the mattpocock entry widened to four adaptations; Dell'Acqua et al. 2023 added to the references and CSL; the Horn entry carries a caveat that its URL was not recoverable; in the principles, the Weng essay is source-linked and the Iacob and GLM-5 citations carry arXiv ids; the mglgit authorship is recorded in the skills-evaluation note; and the itd-27 to-prd link no longer dangles. (iss-2608280824478819)
- `ACKNOWLEDGEMENTS.md` credits the external reporter's two reports and the README requirements section shape they proposed, and the adopt-their-branch default for contributors with working branches is recorded as a process lesson. (iss-2608271322159887)

### Security

- **A rule body can no longer forge a domain heading in the injected rules block.** The render site strips control characters, bidi overrides and zero-width runes and space-indents hash-leading lines, with the forgery corpus as regression tests. (iss-2608261550394120)
- **A broken repo `guard.json` no longer switches off the whole guard registry.** `guard.Load` falls back to the bundled defaults on a repo-layer error and the hook checks them, so bundled hazards stay armed instead of failing open. (iss-2608261551087492)
- Rule injection is bounded by a 64 KiB per-repo budget with a loud truncation notice; a truncated domain retries on a later prompt and is never silently dropped. (iss-2608261551077971)
- **The guard closes two shell-syntax bypasses.** An unquoted brace group such as `git push {--force,} origin main` folds into a fail-closed block instead of being read as a literal token (quoted braces, `${VAR}` expansion and a reserved-word group command keep their prior verdicts), and the zsh `noglob` and `nocorrect` precommand modifiers are treated as command wrappers so a Tier-1 blocker behind them is refused rather than degraded to a warning. (iss-2608221457227161, iss-2608270655497992)
- **The harness-leak class is defined once and enforced on three surfaces.** A live agent-session URL or a tool attribution footer is caught by the store-before-commit redactors, by `abcd lint`'s privacy rule and by the `harness_leak` record and docs-lint rule, from a single canonical pattern set carried as `scanner.OutboundPolicy`. The posting-time primitive `ScrubOutbound` exists and is tested but is wired to no command, so protection at posting time remains the re-read-and-strip policy. (iss-178)
- **Commits from a linked worktree run the private name guard.** The committed pre-commit hook resolves the primary checkout's private banlist store from inside a worktree and enforces it there, and the CLI renders the same inherited layer. (iss-370)
- The scanner's identity matchers and `$HOME` backstop now run over the percent-decoded copy of a line too, so a percent-encoded home path or email is redacted before it can enter a committed record. (iss-2608270720336165)
- `ahoy` path redaction and `fsutil.RedactRoot` fold case on a case-folding filesystem, so a case-variant spelling of HOME or the repo root is redacted instead of leaking into rendered output. (iss-2608270908341622)
- The memory store's write path sanitises frontmatter scalars and derived fields, so a committed memory page no longer replays raw control characters through `cat`, `git diff` or a pager, while the newline, tab and CR escaping provenance depends on is preserved. (iss-2608270655495573)
- **Relative-path containment is canonical and closes a Windows traversal.** `fsutil.ValidRelPath` rejects a backslash segment, so `..\..\x` can no longer walk out of the intended root on a Windows target, and the launch install-surface tree resolver delegates to it instead of a bespoke guard that missed an embedded `a/../../b`. (iss-2608280807166510, iss-2608270655490198)
- `abcd identity` refuses a positioning surface under `.git/`, so a credential-bearing remote URL in `.git/config` cannot be quoted into identity output. (iss-150)
- The `intent plan`, `intent link` and `intent create` renders route the record path through `termsafe.Sanitize`, so a hostile filename tail cannot reach the terminal as escape sequences. (iss-259)

## [0.6.7] - 2026-08-27

### Fixed

- The memory licence gate (ML001) now holds where it previously waved a page through: a licence declared as a plural classes list, a page carrying both a scalar class and a classes list, a missing source hash, an empty or junk sources list, a trailing-space or HTML-comment-led frontmatter delimiter, and a legacy body demoted by a leading comment are all caught. (iss-2608270500191643, iss-2608270500197290, iss-2608270500202696, iss-2608270926037660, iss-2608270500199536, iss-2608270500194738)
- Null and quoted-null impact values are now judged the same way across capture, record-lint and the release derivation, so a quoted, uppercase, or NULL/Null-spelt impact no longer passes one gate while another rejects it. (iss-285, iss-286, iss-287, iss-2608261132593151)
- record-lint now refuses records the capture parser and the record loaders reject but it used to pass: a duplicated or space-before-colon key, a .MD extension, a trailing-space or preamble-led frontmatter delimiter, an unvalidated intent or spec id, and a record missing its schema version. This closes the gap where a lint-green record broke every capture verb or fail-closed the whole store. (iss-2608270500206633, iss-2608270500207078, iss-2608270500208672, iss-2608270500198764, iss-2608270500207987, iss-2608270926031827, iss-2608261437041050)
- docs-lint and record-lint no longer pass with zero findings when a configured roots entry points at a directory that does not exist, which had silently disarmed every per-file rule for that tree. (iss-2608270500208736)
- A record-lint or docs-lint rule whose severity is missing or off-enum now fails the gate, instead of printing findings while the gate exits zero. (iss-2608261533033894)
- A zero-width no-break space mid-record no longer closes frontmatter early, which had let abcd intent plan write kind and spec_id into the record body and report a success that a reload showed empty. (iss-2608270926036966)
- The graveyard interpretation now canonicalises ADR ids by text, so adr-012 and adr-12 dedupe and an overlong id no longer drops its finding, and it notes every signal it truncates at the per-signal cap instead of dropping findings silently. (iss-2608270926036528, iss-2608270500195431, iss-2608270500190007)
- The issue-resolution and reviews gates no longer mis-report: they run from the repository root rather than the current directory, guard against a shallow checkout that read every commit as unreachable (which also made the reviews gate pass vacuously), refuse a bare deletion of an open record as a resolution, treat a git failure or a pathspec-scoped run as a real result, and run their self-test under a hermetic git environment that cannot corrupt the caller's repository. (iss-2608261040378346, iss-2608261132596224, iss-2608261133091276, iss-2608261437044382, iss-2608261041020040)
- A release cut now distinguishes a back-filled resolution from a fresh one, so a ledger hygiene sweep no longer re-announces old work as the current release's content. (iss-2608241612087533)
- **SessionStart notices reach the user again.** The hook now writes its text to stderr and exits zero rather than relying on a non-zero exit the harness rendered as an opaque banner with the text dropped; fixed in both the binary and the bundled bootstrap.sh and hooks.json. (iss-2608241115201044, iss-2608251011427187)
- abcd intent's spec-link scan now propagates an intent-tree read error instead of swallowing it, matching the spec half. (iss-2608261437049307)
- abcd intent reconcile and ready now canonicalise the spec id the way lint does, so a lint-green slug or zero-padded spelling no longer bricks them. (iss-2608261437047643)
- abcd capture --blocked-by now verifies the referenced record the way its own blocker check does, so it cannot write a cross-reference it would then refuse. (iss-2608261437046287)
- Lifeboat lesson prose is now cleaned through the consolidated prose path, so the pre-migration body is no longer retained. (iss-2608261437040578)
- The served install-script template now carries the eol=lf pin its committed siblings have, so it is not delivered with CRLF line endings. (iss-2608261437040448)
- Unpacking a lifeboat on a case-insensitive filesystem no longer lets two files differing only in case both plan a create, silently overwriting the first. (iss-2608270500193735)
- The documentation now states that git is a required runtime dependency, under a Requirements heading it previously lacked. (iss-2608270500210055)
- The version-recovery guidance no longer tells a user to install Go, which neither supported install route needs; it points at the prebuilt-binary routes instead. (iss-2608271014245874)
- The CHANGELOG no longer advertises a Breaking-changes section the derived release ingest cannot emit. (iss-2608261437046261)
- Corrected stale user-facing documentation: the install guide no longer attributes an ahoy-install warning to the plain one-liners, the README plugin-command count and the superseded issue-resolution rule are current, CONTRIBUTING and the CI header no longer understate the Linux leg and the record-lint job, the site README drops a theme control nothing ships, the site command description names the write posture of every form, the terminology reference no longer claims the sources corpus ships, and the preflight gate list is now derived from the recipe so it cannot drift. (iss-2608261133090830, iss-2608261437042550, iss-2608261437041111, iss-2608261437047992, iss-2608261437047965, iss-2608261533419396, iss-2608261533174500, iss-2608242043243131)
- Corrected stale references in the design record: a phantom task-classes enum and a retired review token in itd-5, work specified against a retired terminology tree in itd-43, a retired glossary term in itd-24, and an unswept intent-auditor rename across the disciplines bucket. (iss-2608261437043962, iss-2608261437044340, iss-2608261437046944, iss-2608261437043634)

### Security

- **abcd guard sees through more ways of hiding a blocked command.** A blocked command concealed in backtick substitution, launched through coproc, run under a restricted or alternative shell (rbash, yash), wrapped in a single-string launcher (watch, GNU parallel), or spelt with case-variant capitals on a case-insensitive filesystem is now flagged instead of passing unchecked. (iss-2608270500192596, iss-2608270500198285, iss-2608270500198416, iss-2608270500200205, iss-2608270500205262)
- **Transcript redaction catches secrets it previously wrote raw.** A live token embedded after a percent-encoded URL delimiter, and an ASIA-prefixed temporary AWS access key, are now redacted before a session transcript is written rather than committed in the clear. (iss-2608270500202144, iss-2608270500209138)
- abcd capture now redacts the home path before it forms the slug, so a username no longer lands in the issue filename even though the body was already redacted. (iss-2608270500209352)
- **Terminal output sanitises attacker-controlled text before printing it.** Memory-ask citations, the memory-ingest licence value, and docs-lint config severity and rule ids are now escaped, so a control or escape sequence held in a stored page or a committed config file can no longer reach the terminal raw. (iss-2608270500184205, iss-2608270500186632, iss-2608261533033587)
- Every memory-store read, and the memory lint and coverage passes, now resolve within the store root, so a committed page symlink can no longer redirect a read out of the store or hang the CLI on a device file. (iss-2608261532379188, iss-2608270735427309)
- abcd memory ingest now scans and redacts acquired source text before it is written to the store. (iss-2608270735436138)
- **Working with an untrusted lifeboat stays bounded.** A lifeboat built purely of directories, a single very wide directory, and an unbounded git read on the probe path can no longer exhaust memory before the file cap fires. (iss-2608270500202786, iss-2608270500203019, iss-2608261437048689)
- The disembark scan no longer reads files git ignores or leaves untracked, no longer falls open to a wide read on an all-ignored repository, and no longer mis-sorts the first entry of a NUL-delimited ignore list, so a packed lifeboat cannot cite or carry work kept out of git. (iss-2608241828356533, iss-2608261206490430, iss-2608261533297309)
- Persisted disembark review notes now neutralise CommonMark HTML-block openers in untrusted provenance fields, so a hostile lifeboat cannot inject markdown structure into the durable review file. (iss-2608270500196428)
- abcd launch now scans the payload on the render path that actually materialises the files, and fails closed on any include-selected file it did not scan, so a secret in an included file cannot ship unscanned. (iss-2608270735422516, iss-2608270735433799)
- launch's structural deny of the reserved namespace now applies per path segment and case-insensitively, so a nested or case-varied denied path cannot enter the release payload. (iss-2608270735428304)
- Reads a committed symlink could redirect out of the tree are now contained: site build resolves repository sources through an os.Root, and the secret scanner's config read is guarded against a symlinked .abcd ancestor. (iss-2608270735422697, iss-2608261133204171)
- urlguard now blocks the host-platform magic IP, so a fetch cannot be steered into an SSRF against the provider platform's metadata endpoint. (iss-2608270735423546)
- abcd intent now redacts caller-supplied text before persisting a draft, so a secret or home path is not committed. (iss-2608270735422415)
- The git check-ignore probes now run under the same isolated-exec pins (core.hooksPath, core.fsmonitor) as abcd's other git calls. (iss-2608270735420161)
- The published install one-liners now match the bootstrap curl lockdown: quiet-first, proxy and CA scrub, and protocol pins. (iss-2608270735421311)
- abcd ahoy install now contains its writes within the repository root, refuses a non-real .abcd directory, and no longer writes a self-referential or dangling PATH entry from a relative plugin-root setting. (iss-2608270735428527, iss-2608270500201901)
- Payload write paths are now contained: a manifest path from version-location.json is rejected when it escapes the destination, and the destination-inside-repository gate compares paths case-insensitively so a case-variant payload directory cannot slip it and write inside the working tree. (iss-2608270500214438, iss-2608270500192615)
- abcd banlist add --private now refuses to write the secret-pattern store when git cannot confirm the file is ignored, instead of writing it unignored and reporting success. (iss-2608261132593030)
- **The privacy-hygiene scan no longer reports a repository clean when it scanned nothing.** A git-unanswerable tree, an unreadable tracked file, and the audit rule's dead degradation fallback now surface as a finding rather than passing silently. (iss-203, iss-2608261132597732, iss-2608261533290815)

## [0.6.6] - 2026-08-25

### Fixed

- **The production site deploy runs as part of the release, instead of needing a manual `gh workflow run site.yml` after every tag.** A called workflow's `secrets` context is only what its caller passes: a job inside one that declares `environment:` gets the environment applied — GitHub creates a deployment record for it, which is why this read as an environment problem — and still resolves none of that environment's secrets unless `inherit` unlocked the context first. `release.yml` has carried `secrets: inherit` on its call to `site.yml` since v0.6.3, while `auto-release.yml`'s call to `release.yml` carried no `secrets:` line at all, so the chain ran empty from the top and the deploy failed on a credential that had never been conveyed — on v0.6.2, v0.6.3 and v0.6.5, each time after the binaries, checksums and attestation had already published, leaving the site on an older version. v0.6.4 never reached the credential at all: its render failed first and the deploy was skipped, which is the defect the `## [0.6.5]` entry records. The line sits on both live calls and on the same two calls in the scaffold templates, so a repository scaffolded from them starts with every level of the chain passing secrets, and `TestReleaseChainPassesSecretsAtEveryLevel` in `internal/core/site` pins all four call sites. This retracts the `## [0.6.4]` entry below, where a reader meets the disproven mechanism: a job that calls a reusable workflow genuinely cannot declare `environment:`, but the conclusion drawn from that — that the inherited set is therefore empty inside the callee — is false. Measured on a canary environment secret through the same two-level shape: a top-level job declaring the environment saw it set, nested twice with no `secrets:` line empty, and nested twice with `inherit` at the outer call set. (iss-2608250536214009)
- **Both slash-command argument hints — the one string a user reads in the picker before invoking — name only what the surface dispatches.** `/abcd:ahoy` lists `install`, `uninstall`, `doctor` and `dry-run`, each with its own flow section on the page; the bare invocation is the status render, and `identity-check`, scoped as a bare-CLI entrypoint so a hook or CI job can read its exit code, stays out of the picker. `/abcd:launch` shows `--dry-run` as the flag it is, beside the registered `ship` and `scaffold`. (iss-2608250743421381)

## [0.6.5] - 2026-08-24

### Changed

- **The install one-liner, the marketplace-add command, the badges and the citation metadata name the current organisation.** `go.mod`'s module path and the import lines in 226 Go files move with them, along with `README.md`, `docs/how-to/install.md` and two docs READMEs, `CITATION.cff`, `SECURITY.md`, the Makefile ldflags that stamp the version symbol, `mkdocs.yml`, and `site-src/install.sh.tmpl` — the script the site serves, whose URLs are what a curl install fetches. `TestInstallGuideDocumentsTheInstallAndUpdatePath` derives the documented marketplace slug from the module path, so `go.mod`, the README and the install guide move in one change or the gate fails. Changing the module path breaks any Go importer, judged near-zero cost because abcd is a CLI rather than a library and is pre-1.0. The CHANGELOG entry for the earlier transfer and the historical records under `.abcd/` keep the old name, because they state facts about the past. (iss-2608241959573830)

### Fixed

- **A record whose markdown the site renderer refuses fails before it reaches the default branch.** `abcd site build` renders the issue ledger as well as `docs/` against a fixed markdown subset and never passes text through unrendered, so one four-space indented code block in a committed issue record fails the render — and nothing caught it, because the four lint gates read records without rendering them, the screenshot workflow is path-filtered, and the CI changes classifier stands its optional jobs down for a pull request confined to `.abcd/`. The block is fenced; `make site-render` builds the whole site into a throwaway directory and keeps nothing, `preflight` depends on it so the pre-push hook catches it, and a site-render gate runs on ci.yml's Linux lane, which the classifier never stands down, so a records-only pull request is covered. It matters at this level because the production site renders the tagged commit's tree with that release's own binary: a content defect in a released tree waits for the next release rather than for a later commit. (iss-2608241845109280)

## [0.6.4] - 2026-08-24

### Added

- **The private banlist layer catches machine identifiers, not only names.** Hostnames, IPv4 and IPv6 addresses, CIDR prefixes and MAC addresses sit in the keyed private store and are refused by entry key, with the value itself never echoed, and a keyed corpus test pins the coverage. The work ships in v0.4.2; the ledger record is reconciled here. (iss-158)
- **`abcd disembark` grounds open questions, naming, glossary and internals from a repository's own conventions.** A conventions-tier adapter walks the repository's files for in-code work markers — TODO and FIXME with a trailing delimiter, plus XXX, HACK and BUG — to fill `evidence/open-questions`, and three more read `GLOSSARY.md` or `docs/glossary` for naming and glossary and `docs/architecture` and the package layout for internals — so a repository with conventional docs and no abcd record still grounds these sections instead of reporting blank. The work ships in v0.4.0; the ledger records are reconciled here. (iss-99, iss-100)
- **A lifeboat's graveyard reads what a repository abandoned, not only what it reverted.** The graveyard adapter grounds on files deleted after substantial history and on branches abandoned unmerged, so a project that walks away from work without a revert still has that work read back; `evidence/what-didnt` grounds on reverted commits alone, which is the section's declared source. The work ships in v0.2.0; the ledger record is reconciled here. (iss-98)

### Fixed

- **A fresh plugin install resolves the latest release again after the organisation rename.** `hooks/bootstrap.sh` reads — rather than follows — the redirect from the releases/latest location and parses a tag out of it with `sed`; pinned at the old organisation, that location answers with an organisation hop instead of a tag URL, so `resolved_tag` came out empty. With no cache the download path refused with a message blaming the network, and with a cache the script provisioned from an unauthenticated offline copy while asserting the wrong cause. The bootstrap, the hook manifest, the plugin manifest and the issue-template config name the current organisation, and the bootstrap test's pinned origin constants move with them. `bootstrap.sh` ships from the repository tree rather than a release asset, so the correction reaches installs on merge rather than on publish. (iss-2608241659573856)
- **The release chain's site deploy does not receive its Cloudflare credentials, and the v0.6.3 entry saying it does is false.** `secrets: inherit` is necessary and not sufficient: it conveys the caller's secrets, a job that calls a reusable workflow cannot declare an environment, and this repository holds no repository- or organisation-scoped secrets, so the inherited set is empty inside the callee and `secrets.CLOUDFLARE_API_TOKEN` resolves to nothing. The credentials were correct and correctly scoped throughout. Dispatching `site.yml` directly deploys the newest published release and is the release-day recovery documented in `/abcd:launch`. The record returns to the open ledger carrying the evidence and the three remaining options, because the choice wants a decision rather than a third patch. (iss-2608231912566984)
- **`abcd ahoy install --dev` refuses loudly when an unowned wrapper holds the target path.** `stepSymlink` names the foreign entry as a gap instead of keeping it silently, so the install says what stopped it rather than writing nothing, reporting no gap and leaving `install_mode` empty. The silent no-op ships in v0.4.1 and the loud refusal ships in v0.5.0; the ledger record is reconciled here. (iss-222)
- **The abcd binary survives a plugin update, and the PATH entry pointing at it keeps working.** The binary and its metadata live in the harness's persistent data directory rather than in the commit-stamped plugin cache directory the host re-clones on every update, so an ~11 MB download stops repeating and the cache garbage collection stops dangling the pinned PATH entry roughly a fortnight after each update; the default PATH entry is an owned copy, with a cache-directory symlink kept only as a loud degraded fallback. An owned copy has no plugin root among its ancestors, so `writePathEntry` records a home-scoped `plugin_root=` in `~/.abcd/path-entry` and `resolvePluginRoot` reads it — which is what lets `ahoy install`, `abcd update`, uninstall and the version and staleness verbs work when abcd is invoked by name from a plain terminal, where the harness variable naming that directory is absent. The work ships in v0.6.2; the ledger records are reconciled here. (iss-2608210934566221, iss-2608210934566222, iss-2608210934566230)
- **The references page fits a 360 px screen.** Long unbroken link tokens wrap inside their column instead of escaping it — `overflow-wrap: anywhere` on the references and inspirations list content, with a panel-level anchor overflow net — after a screenshot audit measured a 484 px scroll width on a tree the static mobile gate called clean. The work ships in v0.6.2; the ledger record is reconciled here. (iss-2608221342503802)
- **`abcd site build` cannot leave a file from an earlier build in its output.** The build purges an output directory that carries its own marker before writing, and refuses a non-empty directory it did not write rather than deleting one it does not own. The work ships in v0.6.2; the ledger record is reconciled here. (iss-2608221342506046)

### Security

- **The scanner's per-repo `pii.json` override is read through the shared guarded open.** `scanner.New` routes the read through `fsutil.ReadGuarded` — `O_NOFOLLOW`, a size cap and a regular-file check, with a symlinked `.abcd` ancestor refused before the leaf is opened — so a FIFO planted at that path cannot hang the read indefinitely and a committable symlink to a device file cannot grow it toward memory exhaustion. The path is reachable automatically through the session-end history capture, which holds an unbounded repository lock across such a hang and wedges transcript capture for the repository. The scanner still fails closed, and it names which of the four refusals it hit. (iss-202)
- **The bootstrap authenticates a cached binary against the release's published checksums, and refuses a non-regular file at the cache path.** Re-verifying a cached artefact against the hash recorded beside it is a corruption check and not a tamper check, because artefact and metadata are equally writable by the same user; online cache promotion fetches that release's `checksums.txt` and verifies the cached digest against the published one, while the offline path stays a corruption check with a notice naming the trust level honestly, and a poisoned-pair test pins the self-consistent case the earlier corrupt-cache test never exercised. The migration seed takes the same `[ -e ] && [ ! -f ] -> refuse` form as the main install site, checked before the lock and again under it, closing a path where one `mkdir` turned the seed's `mv -f` into a move into a directory and left every session running shell commands unguarded until a human cleared it by hand. The work ships in v0.6.2; the ledger records are reconciled here. (iss-2608210934566228, iss-2608210934566229)
- **`abcd capture` runs the redaction scanner over free text before it reaches the committed ledger.** Both ledger write paths — the capture render and the resolve or wontfix transition — redact through the scanner first and report the count of altered spans, so the person who filed the record can judge whether the redacted text still says what they meant. It redacts and reports and never refuses: the transcript store is fail-closed on the same detector, and the asymmetry is deliberate, because refusing a capture loses the finding and a ledger that rejects writes stops being written to. A per-repo `pii.json` that cannot be parsed redacts with the bundled defaults and returns a reason rather than blocking every capture in the repository. The work ships in v0.6.2; the ledger record is reconciled here. (iss-2608231025198888)
- **The hook-supplied transcript path is opened through the shared guarded read.** `readTranscript` routes through `fsutil.ReadGuarded` — `O_NOFOLLOW` plus a regular-file check on the open descriptor plus a size cap — so a symlinked transcript is refused rather than followed. This is defence in depth rather than a reachable exploit, and it closes the last bespoke external-input read in the CLI. The work ships in v0.6.2; the ledger record is reconciled here. (iss-369)

## [0.6.3] - 2026-08-23

### Fixed

- **The plugin's two halves are described separately, because they move at
  different speeds.** The surface tracks this repository and the binary tracks
  the latest published release, and prose in the README, in
  `/abcd:update`'s page, and in a refusal string `abcd update` prints had all
  said they arrive together from one release. That is the behaviour a planned
  intent wants and not the behaviour that ships, so three surfaces stated a
  future in the present tense (iss-2608231346137587 records the wider class).
  The install commands the README repeats are now pinned by the same detector
  that already pinned the install guide, so a renamed marketplace fails a test
  rather than misdirecting a reader.

- **The release chain's site deploy receives its credentials.** `release.yml`
  invokes `site.yml` as a reusable workflow, and a called workflow receives no
  secrets unless the caller passes them: declaring `environment:` on the
  callee's job gates the job but leaves `secrets.*` empty inside it. So the
  first production deploy reached wrangler with no `CLOUDFLARE_API_TOKEN` and
  failed at the last step of a release whose binaries were already published.
  The secrets were correct and correctly scoped throughout. `secrets: inherit`
  is added to the call and to the scaffold template it is regenerated from, so
  a managed repo does not inherit the defect (iss-2608231912566984).

### Changed

- **The README speaks to the person holding the intent.** It opens with what
  abcd is for and who for, then why the project's own public record is the
  demonstration rather than a claim, and it names the plugin's one-harness
  limit plainly instead of implying wider support. Installing as a plugin is
  now two copy-pasteable commands with a check afterwards, and the difference
  between updating a plugin and reloading one is stated: reloading re-reads
  what is on disk, so it refreshes the commands while leaving the binary as it
  was.
- **`/abcd:launch` explains release day to whoever is doing it.** The page
  described the verbs and not the day, so nothing told a reader that the
  release stops and waits for a human approval, where to click for it, or that
  a merge landing between the release merge and the tag invalidates the
  receipts. It now carries the seven steps, the approval gate in detail, what
  to check afterwards, and the three failure modes that look like something
  else. It also documents proving the release gate locally before merging,
  which is what turns a receipt refusal into a branch-local failure rather
  than a consumed version.

## [0.6.2] - 2026-08-23

### Added

- **The record explorer gains two pages: `/record/development/` and
  `/record/health/`.** Development reads the stores that MOVE — decisions,
  intents, specs, issues — as Foundation reads the ones that hold: a lifecycle
  bar over a folded list per bucket, each bucket stating its own true total.
  Health collects every check the record can be run against itself: unresolved
  typed references, records nothing reaches, candidate duplicate contributor
  identities with the `.mailmap` line that would fold them, authored commits
  declaring nothing, commits declaring more than one model, and the
  supersessions. A family with nothing to report says so, so a reader can tell
  a check that passed from one that never ran.
- **A print stylesheet.** The record's pages print without their interactive
  chrome, card decks fragment across sheets instead of jumping whole to the
  next one, and the relationship chart keeps its proportions on paper.

- **abcdev.app is rendered from this repository alone.** The `abcd site`
  verb family builds the whole site — the landing page composed from the
  docs under the single-source rule, the record explorer with every
  decision, intent, spec, issue and principle as a page, the relationship
  chart and genealogy, contributors and references, and `/install.sh` from
  the shared template — and `abcd site check` gates the rendered tree on
  seven fail-loud checks; production deploys per release from the tag via
  the release chain, previews ride every push to main stamped as
  unreleased, and README becomes a contributor page with its product
  narrative living in `docs/` (itd-135/spc-37, itd-136/spc-38,
  itd-137/spc-39, itd-138/spc-40; adr-47, adr-48).
- **A bare `abcd` on an interactive terminal opens with its banner.** The
  a-b-c-d signal-flag hoist renders in half-block pixels on its painted
  panel — true ICS geometry, from the livery grids — with the version beside
  it, the canonical tagline beneath, and two runnable next steps, all above
  the unchanged status board. The words are baked in at build time from the
  canonical identity block behind a drift gate (an installed abcd in a
  foreign repo never wears that repo's tagline), and the colour ladder
  resolves truecolor → 256 → 16 → mono automatically, honouring `NO_COLOR`
  and a root-local `--no-color`; mono renders the art as shade-block glyphs,
  and machine-consumed streams — pipes, hooks, `--json`, every subcommand —
  receive no decoration byte at all, the boundary now recorded as its own
  decision (adr-49) and brief invariant. Unstamped builds say
  `abcd (dev build)` honestly. (itd-112/spc-41)

- **abcd has a face: the livery assets ship as one drift-gated source.** A new
  internal `livery` package holds the canonical pixel grids for the visual
  identity — the duckling mascot, the a-b-c-d signal-flag logo (true ICS
  geometry at full size and in a two-by-two icon arrangement; the compact
  variant declares itself approximate), and
  the lifeboat mark — and generates the committed SVG assets under
  `docs/assets/img/livery/`: panel variants legible on any background,
  transparent variants labelled dark-surface-only, each on a natural and a
  square avatar/icon canvas. A package test regenerates
  every asset and fails on a single byte of drift, so the grids and the
  artwork cannot diverge. No surface is rewired yet — the forge/web logo stays
  as it is, and terminal rendering arrives with the planned banner work.
  (itd-133/spc-36)
- **The OPINIONS rules domain carries the memory-graduation principle.** A
  lesson whose "why" is a correction any agent should receive belongs in the
  repo's committed record — the `AGENTS.md` conventions section, a custom
  `.abcd/rules.json` domain, or a ledger capture — never only in one user's
  local agent memory; a memory item recalled twice is a graduation candidate.
  The bundled domain line is self-contained so managed repos receive the whole
  rule; the full principle lives in the development record.
  (iss-2608210923436594)
- **The writing style guide is a reference page.** `docs/reference/writing-style.md`
  is the canonical home for prose rules — the British/US language split,
  present-tense doctrine, Diátaxis page types, and the punctuation rules (no
  em dash in list items; a capital after a colon; lower case after a
  semicolon) — with every rule labelled machine-enforced (naming its shipped
  docs-lint rule) or review. CONTRIBUTING.md and the DOCUMENTATION rules
  domain point at it; the punctuation rules stay review-labelled until their
  lint ships.

- **`abcd banlist --json` carries the private layer's `reach` caveat.** The
  private report now includes a `reach` field holding the one-sentence statement
  of what the layer does and does not protect, unconditionally — mirroring the
  ahoy status board's `banlist.reach`. A surface that summarises the JSON can now
  relay the caveat verbatim instead of paraphrasing a reach that CI cannot
  enforce. (iss-362)

### Security

- **The issue ledger redacts on write.** `abcd capture`, `abcd capture
  resolve` and `abcd capture wontfix` now pass the rendered record through
  the same detector the launch bundler and the transcript store use, so an
  absolute home path or identity span in free text is rewritten before it
  reaches a committed file. Redaction runs before validation, so the
  validator sees the bytes that get written. It redacts and reports rather
  than refusing — a ledger that rejects writes stops being written to — and
  `abcd capture` names the number of spans it rewrote in its render, and all three report it as `redacted` in `--json`, because redaction alters
  what the caller filed. A degraded scanner redacts with the bundled
  defaults and warns rather than blocking every capture in the repo.
  Redaction runs on the free-text inputs before the slug is normalised, so
  a path in the issue text cannot reach a validator-constrained field and
  turn a leak into a refused capture. (iss-2608231025198888)

- **The shell guard recognises the `&>` / `&>>` redirection operators.** The
  guard tokenizer read a leading `&` as a background/`&&` operator, so gluing or
  spacing bash's both-streams redirection into a command (`git push &>/dev/null
  --force origin main`) split the simple command in two and dropped its
  dangerous flag out of command position — a silent allow on every blocker-tier
  entry. The tokenizer now recognises `&>`/`&>>` as a redirection (whose target
  is dropped, whose fd digit is preserved as a real argument) before the
  list-operator split. (iss-2608220131352917)
- **Record-lint reads frontmatter behind a BOM or a multi-line comment.** The
  frontmatter scanner anchored on line 0 and `TrimSpace` does not strip a UTF-8
  byte-order mark, so a record led by a BOM or a multi-line `<!-- … -->` comment
  yielded an empty field map and slipped the `no_git_metadata` blocker and the
  `record_schema` id/supersession gates entirely. The shared scanner now strips
  a leading BOM and the lint/glossary scanners skip a multi-line comment before
  the `---`. (iss-2608220134344680)
- **The shell guard recognises ANSI-C and locale quoting.** The tokenizer
  implemented single, double and backslash quoting but not bash's `$'...'` /
  `$"..."` forms, so the leading `$` prefixed the token (`$'--force'` →
  `$--force`) and a blocker-tier entry missed while bash handed the child
  byte-identical argv — a silent allow. The tokenizer now decodes `$'...'`
  (ANSI-C escapes included, so an encoded spelling resolves to the same bytes)
  and reads `$"..."` as a plain double-quoted word. (iss-2608221456463223)
- **The `local_username` redaction matcher folds case.** The hard-fail matcher
  for the caller's own login was built without the `(?i)` its home-path sibling
  carries, though the login is the last segment of that same home path, so on a
  case-folding filesystem a case variant of the login was neither redacted nor
  caught by the blocking residual scan and survived into the stored transcript.
  It now folds case; the system-directory exemption is lower-cased so the iss-31
  suppression still holds. (iss-2608221456469938)

### Fixed

- **The published AI-assistance disclosure rate was wrong by twenty-four
  points.** The site reported that 71% of commits disclose assistance; the
  figure is 95%. The numerator counted `Assisted-by:` trailer OCCURRENCES, so a
  commit naming two models counted twice; the denominator counted merge
  commits, which the forge writes and no convention asks to declare anything;
  and the residue was then presented as a disclosure gap. `Authorship` gains
  `authored`, `merges`, `assisted_commits` and `multi_trailer_commits`
  additively, so `record.json`'s `schema_version` is unchanged, and the
  excluded merge count is shown rather than silently subtracted.
- **`abcd site check` refused every page the documentation build writes.** The
  scanner faults on HTML comments, which the documentation theme emits, so
  `docs/**` never parsed — and before the gate was made fail-closed those pages
  dropped out of every check silently. The tree is excluded at the walk, which
  is the scope the gate already declared, and its `--json` report lists exactly
  the pages it examined.
- **A whole family of declared interface strings was invisible to the
  provenance gate**, which walked structs and strings but not maps, so the gate
  refused words `site-src/ui.json` plainly permits.
- **Inlined drawings lost their embedded images' dimensions.** The optimiser
  stripped `width` and `height` from every element rather than the root, so a
  drawing's embedded pictures rendered at their intrinsic size and were clipped
  to slivers.
- Citations on the references page rendered one character per line; the
  landing page's chapter rail, the roles table on a phone, the install tabs'
  jumping height and broken narrow-width strip, and uneven panel heights across
  every dashboard grid are all repaired. Contributor rows link forge profiles
  where a noreply address names one.

- **The release preview no longer hides the gate that refuses releases.**
  `abcd launch --dry-run` listed five gates and omitted `receipt_gate`, so
  it could report a clean bundle, a clean scan and a green smoke while the
  semantic-receipt gate that actually blocks the publish went unmentioned —
  an absent row reading as "no such gate". The preview now carries a
  `semantic-receipts` row in every state, reporting which receipts are
  recorded for the candidate commit and pointing at the runbook. It reports
  presence only and never a pass: `release.yml` owns the required-gates list
  and judges receipt validity, and a second copy of that decision in the
  preview would be exactly the false confidence this fixes
  (iss-2608231226342272).
- **The launch command page documents the whole release cut.** It described
  the three-step changelog flow and never mentioned the two host-run
  semantic passes, the receipts they produce, or the two-commit release
  branch the gate requires — so following the page end to end produced a
  release that tagged cleanly and then fail-closed. The ship flow now
  carries the semantic passes as first-class steps
  (iss-2608231226274000).

- **`abcd history capture` accepts what the hooks accept.** The verb read
  its operand through the 8 MiB JSON-operand cap while the SessionEnd path
  read through the 64 MiB transcript cap, so every transcript between the
  two was capturable automatically and unrecoverable by hand — the recovery
  verb bounded eight times tighter than the thing it recovers from, with no
  stdin workaround. Found refusing an ordinary 11.8 MB session during a
  backlog recovery. The caps themselves are unchanged and still refuse an
  over-cap file whole rather than truncating (iss-2608231029040602).
- **Session transcripts past a couple of megabytes are no longer dropped
  at exit.** `hook session-end` redacted the whole transcript in-line
  before writing, at roughly 0.7s per megabyte, and the host cancels a
  shutdown hook rather than wait for it — so the long, dense sessions most
  worth keeping were exactly the ones lost, silently, with the store unable
  to tell an uncaptured session from one that never ended. Capture is now
  split across the two hooks that can each afford their half: SessionEnd
  stages the raw transcript in one write, so its cost no longer scales with
  the transcript, and the next SessionStart redacts and stores it through
  the same fail-closed path. Nine of this repo's own ended sessions were
  absent from its store before this (iss-2608230817034768).
- **A session that ended but was not stored is now visible.** `abcd history
  staged` lists transcripts awaiting redaction — the outcome the store alone
  could never report, since an absent record spans "never ended", "ended
  before the store existed" and "ended and lost" alike. `abcd history drain`
  finishes a backlog without waiting for another session, and exits non-zero
  if anything could not be stored. A SessionStart drains a bounded number so
  it cannot stall the first prompt, and says out loud what it left
  (iss-2608210934566224).

- **The ideate record grill now sweeps id-less records.** Leg 2 of
  `/abcd:ideate` reads the research notes and the decision log alongside the
  id-bearing families, reporting a hit that rests on one in the `note` field
  of the nearest citable record — closing the blind spot where a standing
  verdict recorded in a research note was invisible to the leg that exists
  to prevent re-litigation (iss-2608230748418054; the root-cause decision on
  record-bearing ids for notes stays parked in that issue).
- **A mistyped `abcd intent` or `abcd capture` no longer files a record.**
  Both verbs take free text as their canonical create path, so `abcd intent
  nosuchthing` was swallowed as a draft title and `abcd capture nosuchthing` as
  issue text — each printing a created id and exiting 0, each leaving a durable
  file behind and, on the intent side, burning an id under the `max+1` allocator
  and leaving a record-lint `index_drift` blocker until the stray was noticed.
  The did-you-mean guard only ever caught a NEAR-miss of a real sub-verb, so a
  token resembling nothing fell straight through it. A lone bare word — one
  whitespace-free positional — is now refused on both verbs at exit 2 with
  nothing written, because `capture nosuchthing` and `capture resolve` are the
  same invocation shape and only the second happens to reach the dispatcher
  first. Prose is untouched: quoted text arrives as one argument carrying
  whitespace, an unquoted title as several, and both still file. The tree-wide
  sweep that asserts every parent refuses an unknown sub-verb at exit 2 had
  `capture` and `intent` on an exemption list — that exemption was the defect
  recorded as a design choice, and it is gone, so the two verbs are now held by
  the same detector as the rest of the tree. This closes the gap the
  unrecognised-input-never-writes principle was written about: its founding
  evidence is a misspelled `capture` sub-verb filing an issue when the user asked
  to resolve one, in the 2026-07-08 review. (iss-2608221328552172)
- **`abcd site build` and `abcd site check` agree on what a page may carry, and
  gate every page.** The build inlined an SVG's XML prolog, comment or CDATA
  verbatim while the emitted-page reader refused all three, so a normal exporter
  drawing made the build pass and the check refuse a page whose message named no
  asset; and a page that failed to parse dropped out of the page-walking gates,
  which then printed `ok` for a page they never examined, masking a real finding
  behind the parse fault. The build now refuses the prolog/comment/CDATA at the
  asset, and an unparsed page fails every gate. `LoadRepoMeta` also screens the
  repository address for an executable scheme (a `javascript:` address reached
  an `href` on every page), and the record export refuses a store id that
  collides with a typed record id. (iss-2608221456462677, iss-2608221456469558,
  iss-2608221456466438, iss-2608221457123924)
- **The adopter release scaffold no longer wedges a repo with no `internal/`
  tree.** The race leg hardcoded `go test -race ./internal/...` outside the
  abcd-only guard, so a bare adopter module failed every release verify run with
  `lstat ./internal/: no such file or directory`. The bare profile now runs the
  race leg over the whole module. The scaffold also folds a symlinked target
  leaf (an `ELOOP`, not `ErrNotRegular`) into the path-free non-regular reason,
  so its `--dry-run --json` no longer embeds an absolute path.
  (iss-2608221456597173, iss-2608221456599559)
- **`abcd capture list`/`status --json` no longer leak an absolute path in a
  skipped record's error.** The `skipped[].error` string carried the raw path
  beside the deliberately-relativised `skipped[].path`, in an exit-0 envelope
  the CLI error scrub never sees; it is now scrubbed at the same choke point.
  (iss-2608221456599229)
- **The history date walk reads non-ASCII record paths.** `git log` emitted a
  C-quoted, double-quoted path for a non-ASCII byte, so its key never matched
  the filesystem path and the record lost its dates; the isolated git
  invocation now sets `core.quotePath=false`. (iss-2608221457123924)
- **`abcd update` and `abcd history` no longer leak an absolute home path.** The
  update refusal detail and receipt (`target_path`) and the history record
  `path` field carried the developer-identity home root raw into their success
  and `--json` envelopes — which the CLI error scrub never sees — including the
  documented plugin-session refusal the plugin relays into agent chat. All are
  now redacted to `~` at the render boundary. (iss-2608220142158516)
- **`abcd ahoy` recognises a linked worktree or submodule.** Detection tested
  `.git` for dir-ness, so a worktree or submodule (where `.git` is a gitfile) was
  misclassified as an unmanaged folder: every gap detector was skipped and
  `ahoy install` exited 0 with a wrong "not a git repository" reason. It now
  tests existence, the last `isDir(.git)` holdout after iss-72. (iss-2608220136593438)
- **`abcd ahoy --json` omits the guard object for an unmanaged folder.** The
  guard-health field was a value serialised even when never computed, so an
  unmanaged folder reported four `false` guard facts (a broken guard) beside a
  resolved plugin root. It is now a pointer omitted for an unmanaged folder,
  matching the banlist sibling. (iss-2608220136597127)
- **The privacy scanner refuses a file that grows past the read cap.** The
  tracked-file reader read exactly the cap, so a file that grew past it between
  the stat and the read was scanned as a truncated prefix and reported clean; it
  now reads one byte past the cap and routes a grown file to the same
  not-scanned warning an over-cap file already takes. (iss-2608220144233519)
- **`abcd docs lint` exits 2 when the engine cannot run.** An engine or config
  fault returned a bare error mapped to exit 1 — the code a blocker finding uses
  — while `abcd lint` and record-lint exit 2 for the same fault; a CI gate keying
  on `>=2` read a docs lint that never ran as a findings-pass. The engine-fault
  path now exits 2 with a path-scrubbed message. (iss-2608220145356167)
- **`abcd update` fails closed on an unrecognised install shape.** The dispatch
  switch had no default, so a target kind it could not classify fell through to
  fetch-and-swap; the regular-file case is now explicit and every other kind a
  named refusal. (iss-2608220142154022)
- **`abcd adr-N` resolves an ADR file whatever its zero-padding.** Dispatch
  routed by a fixed four-digit filename prefix, so a differently-padded ADR file
  — lint-green and citation-resolvable, since those readers compare numerically —
  was reported not found. It now routes by the filename's numeric ordinal.
  (iss-2608220148289898)
- **Empty `--json` collections render as `[]`, not `null`.** `abcd spec`,
  `abcd intent`, `abcd memory`, and `abcd capture` emitted bare `null` for an
  empty collection, so a consumer iterating the value errored; they now emit an
  empty array, the invariant history list already held. (iss-2608220147106835)
- **Documentation corrections.** `commands/version.md` now notes that
  `install_mode` is omitted from the JSON when no abcd-owned PATH entry is
  resolvable; `AGENTS.md` and `CONTRIBUTING.md` record that the CI classifier
  stands the macOS leg, race lane, `zizmor`, `govulncheck` and smoke jobs down on
  a docs-only pull request (the merge-queue run still gates the merge with the
  full set); and `docs/requirements.txt` no longer claims a reproducibility its
  unpinned transitive dependencies do not deliver. (iss-2608220150154972,
  iss-2608220150152332, iss-2608220150152535)
- **The hook binary survives plugin updates.** The checksum-verified release
  artefact is now kept once in the plugin's persistent data directory and each
  fresh plugin root is provisioned from it by a re-verified copy, so a plugin
  update no longer re-downloads ~11MB and the first-hook window without a
  binary shrinks from once per plugin update to once per released binary. The
  `abcd` command on PATH is now a regular file abcd owns and refreshes — never
  a symlink into a directory the plugin update lifecycle deletes — recorded
  with its hash in the data directory's `path-entry` file; a legacy symlink is
  healed to an owned copy by `abcd ahoy install`, and `abcd ahoy uninstall`
  removes the copy and its record. Every promotion out of the cache re-verifies
  the artefact against its recorded SHA-256 and refuses loudly on a mismatch;
  a harness that provides no persistent data directory degrades loudly to the
  previous per-root fetch. The version-skew notice is now computed against the
  live plugin root at render time, so it stays truthful when one cached binary
  serves many roots. (itd-132 / spc-35; iss-2608210934566221,
  iss-2608210934566222)
- **The lifeboat coverage render neutralises terminal-control sequences from an
  untrusted report.** A coverage report is a cross-repo artefact, and its
  `status`, `tier`, `tiers_present`, and section-name fields are bare strings
  with no enum validation on the render path, so a crafted report replayed a raw
  ANSI escape or bidi override on the operator's terminal — only the repository
  name was sanitised. Both the per-repo and cross-repo renders now pass every
  repo-derived string through `termsafe`, with column widths computed from the
  sanitised strings. (iss-361)

- **`termsafe` masks the modern zero-width characters, not only the deprecated
  ones.** The sanitiser masked the BOM/ZWNBSP (`U+FEFF`) but not its
  Unicode-designated successor `U+2060` WORD JOINER, nor the `U+2061`–`U+2064`
  invisible operators or `U+00AD` SOFT HYPHEN, so those default-ignorable
  characters passed through every render surface and two distinct byte strings
  could display identically. They are now masked. (iss-366)

- **`abcd capture` accepts the live spec namespace in `related_specs`.** The
  field was validated against the retired `fn-N` namespace, so linking a real
  spec (`spc-N`) was rejected and the field was unusable — the `fn-` prefix was
  itself renamed to `spec` and is a banned token elsewhere. The validator now
  accepts `spc-N`. (iss-364)

- **Two user-facing command docs corrected.** `/abcd:lint` and
  `/abcd:prepare-this-repo` enumerated the conventions the binary checks as five,
  omitting `identity-positioning` — a shipped default rule the same
  prepare-this-repo flow arms (iss-363); and `/abcd:launch` documented a
  top-level `files` count in `--dry-run --json` that does not exist, when the
  file list is the `bundle.files` array (iss-367).
- **`abcd update` completes a chosen update in one verb.** It fetches the
  named release (or resolves the latest, naming the tag before acting),
  verifies the platform binary against the same release's `checksums.txt`
  over a pinned transport (no proxy or CA overrides from the environment,
  redirects only onto the release origin's own hosts), and swaps the
  PATH-installed copy atomically, printing a receipt with origin, tag,
  digest, and old→new versions. A plugin-root binary, the dev shim, a
  stranded entry, a Homebrew-owned install, and any file abcd cannot prove
  is its own are refused loudly, each naming its remedy. Nothing ambient
  changes: this verb and `version --check` remain the only two paths to the
  release origin, each only when invoked. (itd-130)
- **`abcd capture` mints collision-proof record ids.** A captured issue's id is
  now timestamp-numeric — `iss-<yymmddHHMMSS><4 random digits>`, a UTC second
  stamp plus a uniform random suffix — so two agents minting at the same
  instant on different branches produce different ids with no coordination, no
  network, and no registry; nothing is ever renumbered, and existing
  sequential ids stay exactly as minted. The id grammar is unchanged
  (`iss-[0-9]+`), so listing, resolution, promotion, dispatch, and release
  cuts hold as before; the capture result no longer carries the max+1 era's
  `mint_warning` degrade note, since the mint consults no refs. (itd-114,
  spc-33, adr-45; resolves iss-330)
- **The docs-lint/record-lint config read is guarded like its `.abcd/*.json`
  siblings.** `abcd docs lint`, `abcd lint`, `abcd docs cite refresh`, `ahoy`,
  the session hooks, and `record-lint` read `.abcd/docs-lint.json` /
  `.abcd/record-lint.json` with a raw, unguarded read, so a committed config
  symlink to a FIFO wedged those verbs, a `/dev/zero` target exhausted memory,
  and an out-of-repo symlink target ran the whole ruleset from a file the
  repository does not own. The read now uses the shared guarded primitive
  (`O_NOFOLLOW`, regular-file check, non-blocking open, size cap), matching
  `guard.Load`/`rules.Load`. (iss-2608211132061930)
- **Over-cap stdin operands are refused whole, not truncated.** The `-`
  (stdin) transport for `ideate record --verdict-json`, the lifeboat lesson and
  synthesis payloads, and `history capture` read exactly the byte cap, so an
  over-cap payload was silently cut into a prefix while the file transport
  refused it — and on `history capture` the truncated transcript was stored
  under a sha256 idempotency key computed over the prefix. All four now refuse
  an over-cap payload whole. (iss-2608211134281912)
- **`abcd adr-N` resolves an ADR whose id is quoted, zero-padded, or
  case-shifted.** The dispatch confirmed the frontmatter id byte-for-byte while
  record-lint and the citation resolver treat `adr-0012`, `"adr-12"`, and
  `ADR-12` as one handle, so a present, lint-green ADR (or a padded invocation
  like `abcd adr-0003`) was reported absent. The confirm now compares parsed
  handles and renders the canonical id. (iss-2608211140410761)
- **The record-id lint rules key every keyspace canonically.** The delivery-state
  drafts-citation gate, the spec-intent existence and supersession lookups, and
  the spec-id uniqueness backstop keyed on the raw id spelling, so a zero-padded
  intent or spec id could slip a gate open or false-block a valid link. Every
  key and lookup now canonicalises the id. (iss-2608211139076040)
- **The README front page reflects the shipped toolchain and mint scheme.** The
  Go badge advertised 1.25 (the retired version) and `abcd capture` was still
  described as minting "the next free id", the abolished max+1 protocol; both
  corrected. (iss-2608211143184945, iss-2608211143185943)
- **`abcd launch --dry-run` output cannot be corrupted by a committed filename,
  and no longer leaks an absolute path.** The refusal-reason lines were printed
  unsanitised, so a repo filename carrying raw terminal escapes (a
  control-char-rejected path carries its own bytes) reached the terminal and any
  CI log; each reason now passes through the terminal sanitiser like the citation
  line beside it. Separately, when a pinned manifest was unreadable the lockstep
  detail embedded a raw absolute path — and the dry-run report is a success
  envelope that never passes the error-surface path scrub — so the
  developer-identity root leaked into the report and its `--json`; the path is now
  stripped at the read, leaving the named cause.
  (iss-2608211432258689, iss-2608211432257954)
- **`abcd lint` fails closed when a rule cannot run.** A stat that hit a symlink
  loop, an unreadable tracked file, or a git-index fault returned an engine fault
  that mapped to exit 1 — the code the documented Conftest tri-state reserves for
  "warnings only" — so a CI gate keying on exit `>=2` read a lint that never ran
  as an advisory pass. An engine fault now exits 2 (the tri-state's "any error").
  (iss-2608211432258975)
- **`abcd launch scaffold` derives the right default branch from a linked
  worktree.** `deriveBranch` assumed `.git` was a directory, so in a worktree or
  submodule (where `.git` is a gitfile) both ref reads failed and the fallback
  branch `main` was stamped into the generated release workflows — leaving the
  release gate silently inert on a repo whose default branch is not `main`. It
  now resolves the gitfile, reading `HEAD` from the worktree gitdir and
  `origin/HEAD` from the shared `commondir`. (iss-2608211432254405)
- **The intent verdict operand is read through the guarded primitive.**
  `intent audit ingest --verdict-json` used an `Lstat`-then-`os.ReadFile` pair
  whose comment wrongly called the window benign; `os.ReadFile` follows a symlink
  swapped in after the `Lstat` and ignores the pre-checked size. It now reads
  through `fsutil.ReadGuarded` (`O_NOFOLLOW` + regular-file + cap in one open),
  the last CLI operand reader to join the shared primitive.
  (iss-2608211432389181)
- **The `history` rootSHA diagnostic names both accepted widths.** A rejected
  rootSHA was told it must be a 40-char SHA though the validator also accepts a
  64-char SHA-256 root; the message is now a shared const beside the regex naming
  both widths. (iss-2608211432384430)
- **Three user-facing docs corrected.** The install guide overstated that every
  non-start hook self-bootstraps, when SessionEnd deliberately never downloads
  (a fetch there would race shutdown and lose the transcript); `commands/memory.md`
  named the citation field `source.class` where the JSON key is `source_class`;
  and `commands/ahoy.md` omitted the ` (shadowed on PATH)` `install_mode` suffix.
  (iss-2608211432384091, iss-2608211432389477, iss-2608211432389791)
- **Record cross-references reconciled.** Seven record links carried a display
  path that did not resolve from the containing file (the href did), two
  backticked bare paths named directories that do not exist, `itd-26` gained the
  reciprocal `related_rfcs` for `rfc-1`, and the `.abcd/work` tier roster in
  `AGENTS.md` and `.abcd/README.md` now names the issue ledger, reviews charter,
  and ruleset mirror that also live there.
  (iss-2608211432489481, iss-2608211432489288, iss-2608211432482363, iss-2608211432483805)
- **The command guard recognises shell redirection operators.** The tokenizer
  knew the compound and grouping operators but not `>`, `>>`, `<`, `>&` and
  their kin, so a redirection glued to a token — `git push --force>/dev/null` —
  mutated the flag and the blocker missed, a silent allow, and a leading
  redirection displaced the command and degraded a Tier-1 block to a warn. A
  redirection now terminates the word and its target is dropped, so every glued
  and leading form blocks again. (iss-2608211849467013)
- **The record/docs lint reporting and glossary walks stay inside the
  repository.** Both read a cloned-repo-controlled config path (the `roots` and
  `glossary_dir` fields) with an unguarded `os.ReadFile` while the citation
  collector guards the same class, so a committed lint config could point either
  outside the tree — or a committed symlink leaf resolve outside it — and the
  lint read, followed, and reported a file the repository does not own. Both now
  run through the same containment and guarded-read stack; the remaining
  config-derived record reads in the package are tracked for a follow-up sweep.
  (iss-2608211849463840, iss-2608211914592726)
- **`no_git_metadata` sees comment-led records.** A record whose frontmatter is
  preceded by a leading attribution comment yielded no fields to the shared
  scanner, so a git-inferable metadata key there slipped the blocker entirely.
  The lint now reads past the comment. (iss-2608211849461061)
- **Citation staleness is timezone-stable.** The age arithmetic took each date's
  local calendar day, so the 180-day boundary — and, under the release gate, the
  citation-overdue blocker — depended on the maintainer's timezone. Both ends are
  now converted to UTC before the date is taken. (iss-2608211849466878)
- **The attribution gate accepts CRLF pull-request bodies.** The trailer and
  human-only presence checks are line-end-anchored over a class excluding the
  carriage return, so a web-UI pull-request body — which arrives CRLF — false-red
  a correct `Assisted-by:` trailer on the required check. The gate now normalises
  line endings before every rule. (iss-2608211849468791)
- **`abcd identity init` exits 2 on a fault.** A structural fault mapped to exit
  1 — the code a rendered refusal reserves — while init renders nothing on that
  path and its sibling verbs use 2. (iss-2608211850070541)
- **`abcd history list --json` emits `[]` for an empty store.** An empty store
  marshalled to bare `null` while the command doc promises an empty list.
  (iss-2608211850070318)
- **The lifeboat probe refuses a file that grows past its read cap.** The read
  sized with an fstat then read exactly the cap, so a file that grew in between
  was silently truncated to a prefix rather than refused. (iss-2608211850074600)
- **The install guide describes the persistent provisioning cache.** The provisioning
  section still described a download on every plugin update and a `.binary-meta`
  remedy that is a no-op on a cache-provisioned root; both now describe the
  shipped persistent per-plugin cache. (iss-2608211849580624)
- **The contributing guide scopes the fenced-quotation carve-out to the body.**
  The gate strips fences on the pull-request-body arm only, but the guide granted
  the carve-out for commit messages too; the guide now says a commit message is
  read verbatim. (iss-2608211849582190)

### Changed

- **The record's navigation reads in one order** — Dashboard, Foundation, Work,
  Relationships, Health — with Contributors and References at the end.
  `Foundations` is now `Foundation`, and the genealogy moves into the
  dashboard, folded, rather than holding a page of its own.
- **The dashboard's counts lead somewhere.** Each tile links the page that
  reads that store, anchored at the store itself; a store whose page the build
  did not write keeps an inert tile rather than a dead link.
- **The release-cadence panel is removed**, and the per-day commit history that
  fed it with it.
- **The relationship chart tells its two encodings apart**: colour is what kind
  of record, border is what state it is in, and each is listed in its own
  block. Disciplines carry their own colour, their own filter chip and their
  own word on a record card, rather than being coloured as one thing and
  labelled as another.

- **Released binaries build on Go 1.26.** Go 1.25 has left the support window,
  so the toolchain moves to the current 1.26 line (1.26.7) in lockstep across
  the `go.mod` directive, the CI pins, and the release workflow's pin via the
  scaffold substitutions — `govulncheck` scans clean on the new line. (iss-329)

### Fixed

- **A hostile source repo cannot inject terminal escapes through `disembark
  plan`.** The dry-run render printed planned file paths raw while the same
  view already sanitised the source name and omission lines, so a crafted
  filename carried C1/bidi runes to the operator's terminal. Every path in the
  render now passes the terminal-safe sanitiser. (iss-382)
- **The transcript history store's read path is guarded.** `history list` and
  `history show` read records from the cross-repo store under HOME with a raw
  read while the write path was fully hardened, so a planted symlink was
  followed and a FIFO wedged `history capture` under the store lock. Both reads
  now use the guarded primitive (no symlink follow, no blocking open, a size
  cap). (iss-383)
- **A committed citation baseline cannot smuggle escapes through the lint
  gate.** The baseline validator sanitised the final URL but echoed the entry
  key and the other field values raw into a refusal that reaches the terminal,
  so a hostile entry could replay ESC/bidi/zero-width through the blocking
  docs-lint gate. Every echoed value is now sanitised. (iss-384)
- **`abcd launch scaffold` pins the adopter's Go toolchain.** The generated
  release workflow carried a floating `go-version` (the go.mod patch was
  stripped), so an adopter shipped binaries built on whatever patch a runner
  resolved on release day. A patch-pinned `go.mod` now scaffolds that exact
  toolchain. (iss-386)
- **`/abcd:version` names every network-touching verb.** Its guidance claimed
  only `version --check` and `update` reach the network; `docs cite refresh`
  and `memory ingest <url>` do too. The enumeration is corrected. (iss-385)
- **`record-lint` catches a zero-padded record-id collision.** The
  id-uniqueness backstop keyed on the raw filename, so a hand-added
  `iss-0100-*.md` beside `iss-100-*.md` read as two distinct ids and slipped
  through. It now keys on the canonical id, matching the resolver. (iss-392)
- **`ahoy install` heals the PATH entry a plugin update strands.** The entry
  pointing into a deleted previous plugin cache dir classified as a foreign
  occupant — never healed, refused by `ahoy uninstall` against the dangling
  gap's own fix hint. Ownership now extends to exactly that shape, one
  predicate serves detection, install, and uninstall alike, and `--dev`
  cannot rewrite a stranded entry past a declined approval. (iss-345)
- **A redirect cannot smuggle hidden runes into the citation record.** The
  final URL a fetch ends at is redirect-controlled, and `encoding/json`
  escapes neither C1 nor bidi/zero-width runes, so a hostile Location query
  could land a terminal escape or Trojan-Source override raw in
  `docs cite refresh --json` and the committed baseline. The fetch boundary now
  percent-encodes them losslessly and the baseline validator refuses them from
  any producer. (iss-359)

- **`guard check` reports every matched entry.** The human report assumed the
  winner led the match list, so a synthetic block over registry warns dropped
  the matched warn entry and echoed the winner twice; non-winners are now
  selected by id. (iss-346)

- **Session end refuses a symlinked or growing transcript.** The hook-supplied
  transcript path was the one guarded read left off `fsutil.ReadGuarded`: a
  planted symlink was followed into the history store, and a file crossing the
  cap mid-read stored a silently truncated prefix. Both are refused whole now.
  (iss-347)

- **An over-cap hook payload names the cap.** The hook readers truncated at the
  limit and blamed the host for the resulting severed JSON; they now read one
  byte past the cap and say what actually happened, on the prompt router and
  the guard hook alike. (iss-201)

- **The URL guard blocks CGNAT and the other reserved IPv4 ranges.** Go's
  `IsPrivate` covers RFC 1918 + ULA only, so 100.64/10 — live internal
  addressing on Tailscale tailnets and carrier networks — passed every
  predicate; the RFC 6890 table closes the class and the NAT64/6to4 unwrap
  re-checks it. (iss-356)

- **`abcd lint` says when the privacy scan skips a file.** A tracked textual
  file over the 4 MiB cap was skipped silently and the repo reported
  conforming; the skip is now a warn naming the file, while oversize binary
  assets stay quiet. (iss-356)

- **`capture resolve --commit` accepts a SHA-256 repo's 64-hex sha.** The shape
  check stopped at 40 chars, refusing a legitimate resolution the receipt gate
  already accepts. (iss-356)

- **The CLI help states what ships.** Bare `abcd`'s help now declares the
  record-id positional (`abcd iss-N` describes a record) that shipped in
  v0.6.0 but appeared on no CLI discovery surface (iss-348); `launch`'s
  summary says `--dry-run` is required rather than reading as optional, and
  `guard check`'s gap list discloses the backtick-substitution limit the
  package map says it states. (iss-356)

- **The terminology crosswalk states the shell guard's real safety property.**
  The Guardrails row listed the pre-tool-use guard among fail-closed gates —
  the opposite of its deliberate fail-open-loud design — on the page written
  for an evaluator. (iss-349)

- **The README promises the PATH hint from the verb that prints it.** After
  the copy one-liner install, bare `abcd ahoy` never reports the off-PATH gap
  the README claimed; `ahoy install`'s reachability note is what prints the
  one-line fix. (iss-350)

- **GitHub-handle redaction fires regardless of the remote's letter case.** The
  remote-URL parser matched the `github.com` host case-sensitively, so a
  hand-typed mixed-case remote (`git@GitHub.com:…`) left the derived username
  empty and the `github_username` redaction kind was never armed — the caller's
  handle survived `abcd history capture` redaction and the launch/pack PII scan.
  The host match is now case-insensitive, matching its sibling patterns. (iss-339)

- **`abcd history show` neutralises terminal-control sequences in a stored
  transcript.** The transcript body is untrusted input, and capture redacts only
  secrets and home paths, so an ESC/CSI/C1/bidi sequence in ingested content
  replayed raw on the reader's terminal. The human render now passes the body,
  and the metadata fields the read path does not re-validate, through `termsafe`
  while preserving the transcript's line structure. (iss-340)

- **The lifeboat read no longer hangs on, or trusts, a swapped file.** `embark`
  vetted a file with `Lstat` then opened it separately, so an entry in an
  untrusted lifeboat swapped for a FIFO blocked the open indefinitely, and an
  in-root symlink was read in place of the vetted file. The read now routes
  through the guarded primitive (`O_NONBLOCK`, fstat-on-descriptor, `os.SameFile`).
  (iss-341)

- **`abcd disembark probe`/`plan`/`pack` no longer hangs on a planted FIFO.** The
  probe listed a directory's entries with a blocking open, so a FIFO named where
  an adapter expects a directory (for example `docs`) hung the command over an
  untrusted target repository, with no race required. The listing open is now
  non-blocking, matching the probe's file reads. (iss-342)

- **Two user-facing command docs corrected.** `/abcd:prepare-this-repo` located
  the record at a path and level count that predate the command-file flattening,
  sending an agent one directory above the repository root (iss-343); and
  `/abcd:lint` presented the `abcd-lint:allow` line waiver as applying to any
  finding when only `privacy-hygiene` honours it (iss-331).

- **`abcd lint` privacy-hygiene no longer hard-fails on ordinary committed
  content.** A relative path segment or a URL path that merely contains `home/`
  or `Users/` — a route directory, an import path, a docs URL — tripped the
  rule at hard-fail severity because it lacked the leading-boundary predicate
  its scanner twin already carries; the rule now requires the match to begin a
  path (iss-305). In the same rule, the Windows arm of the absolute-path
  pattern is now case-insensitive, so a lowercase `c:\users\<name>` leak is
  caught — the POSIX arm deliberately stays case-sensitive, since folding it
  would flag ordinary API-route text (iss-308).

- **Redaction no longer misses an address glued to a word character.** The
  scanner's IPv4 and MAC patterns ended in an ASCII word boundary that a
  fixed-length, pure-word-char token can never satisfy before another word
  char, so `192.168.1.44_gw` or `a4:83:e7:11:22:33_eth0` was silently dropped
  from a hard-fail redaction path. The trailing boundary is gone, with a
  compensating truncated-number guard keeping dotted version strings and
  four-digit tails silent — measured zero added false positives. (iss-307)

- **`abcd guard` no longer claims a warn it does not raise.** The 0.6.0 note, a
  code comment, the CLI help's scope text, and the pinning test all said
  `python -c` and `perl -e` get a loud warn; the shipped behaviour is a silent
  allow — a non-shell interpreter's payload is one opaque token the Tier-2
  fail-safe never lands on, and a loud warn for it is a recorded design
  target, not yet implemented. Every surface now states the silent allow, and
  the pinning test asserts the actual allow posture (with a shell-tokenizable
  fixture that would block if the language were wrongly folded into the shell
  family), where it previously asserted only "not block" and so pinned
  nothing. No behaviour change: only the false claim is removed. (iss-315)

- **The attribution-gate corpus is hermetic against the ambient git
  environment.** Every scratch-repo command in its harness is `git -C`, but an
  inherited absolute `GIT_DIR` overrides `-C` — the harness would have
  committed to and hard-reset the ambient repository while reporting
  all-green — and an inherited `commit.gpgsign` could break the scratch
  commits and skew every case. The ambient git environment is neutralised
  before the first git call, and a scratch commit that cannot be created now
  fails loudly. (iss-313)

- **The SessionEnd hook no longer downloads the binary at exit.** SessionEnd
  carried the same bootstrap salvage as the other hooks, but it fires exactly
  when the session is going away and the host cancels a slow hook rather than
  wait — so after a plugin update landed a fresh binary-less cache dir,
  update-then-quit exited through a blocking ~11 MB download, the hook was
  cancelled mid-flight, and the session's transcript capture was silently
  lost. The hook now runs the plugin-root binary if present, falls back to an
  `abcd` on PATH, and otherwise says in one line that the transcript was not
  captured — no network work at session end. (iss-2608210934566223)

## [0.6.1] - 2026-08-20

### Fixed

- **Releases build on the patched Go toolchain.** The release workflow's Go version floated at the minor — resolving to a toolchain carrying four standard-library vulnerabilities fixed upstream — while ci scanned green on the patched one; the scaffold substitutions and the committed workflow now pin the patched toolchain in lockstep, held identical by the self-scaffold parity test. (iss-289)
- **`abcd guard` no longer allows a blocker hidden behind a value-taking short-flag cluster.** An exec-string verb whose payload flag sat in a cluster after a value-taking flag (`script -Tc out.txt -c '< blocker>'`, where `-T` takes `c` as its value) was mis-read as `-T -c`, resolved a bogus payload, and switched the Tier-2 fail-safe off for the segment — so the real `-c` command reached neither tier and was silently allowed. The cluster reader now consults the verb's value flags and declines such a cluster, letting the scan find the genuine payload flag. (iss-291)
- **Release-tag and changelog version parsing rejects an out-of-range component.** `ParseSemver` discarded the integer parser's range error, so a version component wider than the platform int clamped to its maximum — colliding distinct versions and wrapping negative on the next bump. It now returns the error, matching the guard already shipped for spec ids. (iss-293)
- **Wording corrected in two user-facing surfaces.** The `abcd version --check` flag help no longer claims to be "the only command that touches the network" (three verbs can fetch; the network model is recorded), and the README onboarding section describes the shipped facilitator flow with its discovery-ingest and Socratic-interview automation marked as a design target rather than a present capability. (iss-294, iss-295)
- **The development record catches up with its own releases.** The hand-maintained ADR index gains the records it omitted and the rulesets README counts all eight required checks; the v0.6.0 verb renames are swept out of their last present-tense stragglers in the roadmap and brief; the brief's broken and stale intra-record deep links, its phantom internals chapter, and the persona glossary's contradiction of the by-role selection rule are all corrected. (iss-296, iss-297, iss-298, iss-299, iss-300)

## [0.6.0] - 2026-08-19

### Breaking

- **The lifeboat verdict says what it is.** `abcd disembark oracle` is now `abcd disembark review`, its flag `--oracle-json` is `--review-json`, and the verdict artefact moves from `audit/oracle-<manifest12>.{json,md}` to `review/review-<manifest12>.{json,md}` (the `audit_path` key in the JSON envelope is now `review_path`). The `lifeboat-oracle` agent becomes `lifeboat-reviewer`. The verb emits family-1 change-judgement verdicts (`SHIP`/`NEEDS_WORK`/`MAJOR_RETHINK`) — a *review* in adr-40's vocabulary — and the planning investigation found it never invokes the oracle seam at all, so naming it for that seam claimed something untrue and collided with the reserved `/abcd:oracle ask`. **The seam itself is untouched**: adr-25 stands, `oracle` remains the model-access seam's name, `ahoy install --oracle-backend` is unchanged, and the `oracle_review` task-class token stays. A re-run over a lifeboat reviewed before this release replaces cleanly — it writes the new artefact, removes that manifest's stale pair, and prunes `audit/` if empty, never touching another manifest's files. Verdict logic, the enum gate, cite-or-be-dropped, and attestation stamping are behaviour-frozen. (itd-125, spc-30; adr-40 §5 amended in place)

- **The conformance check calls itself lint.** `abcd audit` is now `abcd lint` (and `/abcd:audit` is `/abcd:lint`): the verb applies deterministic rules about a repo's form, which the record's vocabulary names a *lint*, and the `audit` name returns to its reserved seat — itd-16's hash-chain fidelity surface (adr-40). The tri-state exit contract (0 clean / 1 warnings only / 2 any error), the rule set, and the JSON envelope are unchanged; the conformance core moves to `internal/core/repolint`. The privacy waiver's current spelling is `abcd-lint:allow`, and every committed `abcd-audit:allow` line stays honoured forever. **Managed repos:** re-download the binary, and re-run `prepare-this-repo`/`launch scaffold` where a repo's own instructions or CI referenced `abcd audit`. (itd-124, spc-29; the lint-engine merge question is iss-251)

- **The intent audit says audit.** `abcd intent review` is now `abcd intent audit`, and `intent review ingest` is `intent audit ingest`: the verb emits family-2 promise-vs-reality verdicts (`MET`/`NOT_MET`/…), which the record's own vocabulary rules an *audit*, not a review (adr-40). The `intent-fidelity-reviewer` agent renames to `intent-auditor` with it. Clean break, no alias: the old spelling is refused (with the successor named) and never swallowed as a free-text intent create. Stored artefacts are format-frozen — the `abcd-review:` audit-note markers, the `abcd/intent-fidelity-verdict/v1` payload type, and every previously ingested verdict remain valid; only the verb, the agent, and the code identifiers move. (itd-123, spc-28)

### Added

- **An external pull request needs two invited reviewers.** A required check
  (`external-review`) holds any pull request whose author is not an invited
  collaborator until at least two collaborators of role triage or above have
  approved it; a collaborator's own pull request passes trivially, and the
  check runs the base repository's own logic so a fork cannot edit it green.
  Ledger issue captures land through pull requests and inherit the rule.

- **The contribution surface tells the truth about its gates.** `CONTRIBUTING.md`
  reflects the public repository: inbound = outbound MIT with no CLA and no DCO,
  issue-first intake with a per-author volume cap, the merge queue, the
  code-owner-reviewed publish surface, and the fact that the shipped hooks are
  per-machine opt-in (`git config core.hooksPath .githooks`) rather than
  self-installing. The attribution rules now state that the `Assisted-by:`
  trailer stands on its own line, and cite the kernel's
  `coding-assistants.rst` as the convention's shared design. `SECURITY.md`
  routes vulnerability reports to private reporting (with an issue-template
  contact link doing the same), and a pull-request template carries the
  disclosure line every PR body needs.

- **The documentation renders as a site.** `mkdocs.yml` builds `docs/` — the
  single source of truth — into a disposable HTML site (Material theme, pinned
  in `docs/requirements.txt`): any static-site host runs
  `pip install -r docs/requirements.txt && mkdocs build` and serves `site/`.
  Directory-style links in the docs pages became explicit file links so every
  page resolves both on the repository host and on the rendered site; the
  generated `site/` output is gitignored, never committed.

- **The surface registry can no longer wave its hands.** Every surface file under the brief's `04-surfaces/` carries a machine-checked `## Sub-verbs` table recording two facts per verb — its adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a non-assessment verb) and whether it exists (`shipped` / `staged`) — and the `surface_coverage` record-lint rule gains a sub-verb pass that checks every row against the committed command-tree snapshot in both directions: a `shipped` row must be registered, a `staged` row must not be, a registered sub-command must have a row, and a sub-command-bearing verb cannot hide without a table. Exemptions (host-delegated surfaces, operator-internal verbs, the bare command) are explicit config, never silent skips; a duplicate table heading and a malformed row are findings too. The bucket enum is registered as reserved vocabulary, closed and PR-to-extend, and every sub-verb now has one machine-checked home that a stale prose claim can be corrected against. (itd-122, spc-27; closes iss-246)

- **Type the id, get your next move.** `abcd <id>` dispatches on any record id — `iss-N`, `itd-N`, `spc-N`, `adr-N` — and reports, strictly read-only, what the record is, its links, and the concrete next move for its lifecycle state: a draft intent points at the planning interview, a planned one at its spec body or at implementation (the `intent ready` checks decide), an open issue at `capture promote`/`resolve`/`wontfix`, a promoted one at the intent it graduated into, and decisions are read. Bare `abcd` answers *what can I do*; `abcd <id>` answers *what is this*. Any other positional stays on the unknown-command path unchanged, and every recommended verb is pinned to the live command tree by a test, so a future rename breaks the build instead of shipping stale advice. (itd-121, spc-26)

- **A resolved issue points at what fixed it.** `abcd capture resolve` gains optional provenance flags — `--intent itd-N`, `--spec spc-N`, `--commit <sha>` — that write the structured `resolved_by` pointer the schema has modelled all along, in the same atomic transition as the resolution note and impact. Ids must exist in their record store; the sha is shape-checked only; an unknown or malformed value refuses the whole resolve and writes nothing. Without the flags the record stays byte-identical to a plain resolve. `wontfix` is untouched — a non-action points at nothing. (itd-120, spc-25; the `resolved_by` half of iss-245)

- **`abcd guard` no longer waves through a hazard reached by a launcher it does
  not recognise.** `nice git push --force origin main` exited 0 with no output,
  and so did nine other spellings — `setsid`, `flock`, `stdbuf`, `chroot`,
  `busybox sh -c`, `su -c`, `runuser -c`, `script -c` — because the matcher
  stepped a hand-maintained list of programs that launch other programs, and
  anything not on that list simply *became* the command. Nothing matches a
  command called `nice`, so the verdict was a confident allow. **That list can
  never be complete**: any binary that execs its arguments belongs on it, the
  ones that matter grant no privilege so no security catalogue lists them, and a
  repository adds one with a line in a Makefile. So it stops being what stands
  between a hazard and a silent allow. When nothing matches a segment, the
  matcher now re-runs from each later command position, and a hazard found there
  is a **loud warning** — never a refusal, because an unknown program is not
  proof that it runs the rest of the line, and `rg git push --force docs/` is a
  search. Naming the launchers is still worth doing and fourteen are now named,
  which turns each warning into a precise refusal that teaches the safe form; the
  point is that being incomplete no longer costs silence. Four verbs that carry a
  command *string* — `su`, `runuser`, `script`, `flock` — are read too, in every
  documented spelling including the long ones. **What this does not change:** a
  hazard quoted as text is still not a hazard, ordinary commands are unaffected,
  and the guard is still a mistake filter rather than a boundary — see below.
  (adr-42, iss-272)

- **The guard's own limits are on the record, and so are the sources that
  settled them.** ADR-42 rules the hazard guard's parse layer a *mistake filter*
  rather than a security boundary — it catches accidents and casual evasion by a
  cooperating agent, and anything needing a real boundary needs an
  execution-layer control behind it. **Nothing in the guard's behaviour or its
  own documentation changes yet** — what lands here is the decision, the evidence
  behind it, and credit; saying it on the guard's own surfaces is the first thing
  the implementation builds. `ACKNOWLEDGEMENTS.md` gains five § Inspirations
  entries for the precedents that shaped it — one agent harness's permission and
  sandboxing model, another's command-denylist failure, a third's
  sandbox/approval split, GTFOBins' shell/command taxonomy, and sudo's `NOEXEC` —
  with the full source list registered in the design record. (adr-42, iss-272)

- **An issue graduates into an intent without retyping.** `abcd capture promote <iss-N>` is the native verb for step 2 of the record walk: one invocation mints an intent draft — slug reused from the issue, body carrying a by-id pointer to the issue rather than a copy, `promoted_from: iss-N` in its frontmatter — and stamps the issue's `promoted_to` with the minted `itd-N`. Promotion works from any status folder and never moves the issue, a second promote is refused with the existing `itd-N`, and a stamp failure after the mint names the orphan draft and its repair: `capture promote <iss-N> --intent <itd-N>`, the stamp-only mode that links an existing draft instead of minting. (itd-119, spc-24; the `promoted_to` half of iss-245)

### Changed

- **The repository lives at `Partnermedia/abcd`.** The repo transferred from
  `REPPL/abcd-cli` to the Partnermedia organisation and dropped the `-cli`
  suffix; the Go module path, install one-liner, marketplace add command, and
  plugin metadata all follow (`github.com/Partnermedia/abcd`). Every old URL —
  clones, plugin installs, release downloads — keeps working through GitHub's
  redirect, but new work should reference the new path.

- **A human stands between merge and publish.** The release job runs in the
  `release` environment, which carries a required reviewer: a push to `main`
  that cuts a release now pauses as a pending deployment until a maintainer
  approves it, instead of publishing on the merge alone. Rehearsals
  (`workflow_dispatch`) are unaffected — they publish nothing and gain no gate.

### Fixed

- **The launch scan no longer mistakes the caller's public handle for a real
  name on an org-owned repository.** The `real_name` suppression compared
  `user.name` only against the remote URL's owner, so transferring a repo to an
  organisation turned the caller's own GitHub login into hard-fail findings
  that blocked the launch. The suppression now also recognises the login
  embedded in the caller's `users.noreply.github.com` address (both its forms),
  which travels with the caller whatever the remote's owner is. A name that
  does not match that login still scans as a real name. (iss-283)

- **`abcd guard` no longer misses a force-push behind an unlisted git global option.** The six git entries listed the global options that consume the following token so the matcher could step over them and find the subcommand — but the list named seven and was wrong about which — two take no value at all, while three genuinely value-taking globals were missing. `--attr-source <tree>` (git 2.40+) and `--config-env <name>=<var>` were absent, so their value was read as the subcommand: `git --attr-source HEAD push --force origin main` performs the identical forced update and was silently **allowed**, as were `--no-verify` commits and `reset --hard`. Every git blocker was bypassable by the same prefix, and only the separate-token spelling evaded — the glued `--opt=value` form was already skipped as an ordinary flag. A third, `--shallow-file`, was missing from the fix's own first cut and is a live force-push bypass in its own right — it has been in git since 1.9 and is absent from `git --help`, as are the other two. All three are now listed on all six entries. The test no longer *asserts* the list is complete: it **re-derives the classification from the git on the machine**, probing whether git reads the next token as the subcommand, and treats an unrecognised flag as value-taking so an unclassifiable option demands listing rather than being ignored. The earlier shape — a hard-coded size plus a superset-tolerant membership check — read as authoritative while being unable to notice a missing member, and duly certified a list containing a live bypass. (gh-299)

- **The attribution gate can be written about again.** Its banned-footer and co-authorship rules are anchored to line start so prose quoting them mid-sentence stays documentation — but a fenced code block, the natural way to show a literal shape, was refused exactly as a real footer would be, and the change that tightened the rule tripped over this in its own pull-request body. A fenced block is quoted material rather than the document's own voice, so it is now removed before any check runs. That reads both ways: a banned form inside a fence is an example, and a trailer inside a fence is an example too — it no longer counts as the disclosure. **The pull-request body only:** the justification is that a forge *renders* a fence as a code block, and a commit message is never rendered — `git log` shows it verbatim — so the commit check keeps the unstripped text and the mid-sentence convention. **What this defends is footer-last.** Bodies picked up a tool's default footer across 78 pull requests — an accident at scale, not an adversary — and a tool appends its footer last, which is caught whatever precedes it. A body *constructed around* the footer is not defended, and never was: a plainly fenced footer passes by design. Within a document-level context the matcher pairs fences the way CommonMark does — same marker character, at least as long, nothing after it, and a backtick opener carrying a backtick in its info string is an inline code span rather than a fence — because a looser test let a footer be stripped while the forge rendered it as an ordinary paragraph. An unterminated block strips nothing, which both fails closed and lets a longer fence quote a shorter one, the shape needed to document this rule at all. **A document containing an HTML block strips nothing**, since the matcher cannot model one and striking less can only over-reject. A fence indented four spaces or led by a tab is not a fence; one inside a first-level list item is *under* the indent limit and is stripped although the forge renders it visibly — an **accepted** limit rather than an outstanding defect, since it needs a body constructed around the footer, which this gate has never defended against (iss-270 records the decision). Bad invocations exit 2 again, as documented. (iss-262 follow-up, iss-268)
- **The attribution gate's fence matcher is now pinned clause by clause.** Four clauses were correct but untested, so a plausible refactor could have removed any of them with the corpus still green — and one had a bypass direction: substituting the marker-character check for a bare emptiness test keeps the matcher terminating while making `---`, `***` and `___` runs into fence delimiters, so a thematic break or a setext heading would start stripping. Seven cases close those, each verified by mutation to fail when its clause is removed, plus the boundary case nothing was covering (a checkable line immediately after a closer) and the known list-item residual for ordered items as well as bulleted ones. (iss-271)
- **`abcd guard` no longer waves through a hazard wrapped in `zsh -c` or `ksh -c`.** The guard descends into an interpreter's `-c` string so a hazard carried as an argument is still matched — but the interpreter set was written out at three separate sites and named only `sh`, `bash`, `dash` and `eval`. A command wrapped in `zsh -c '…'` was treated as an ordinary unknown command, its payload stayed one opaque token, and **every bundled blocker was silently allowed** — including the critical `gh repo delete` and `git push --force`. zsh is the default login shell on macOS, one of the two systems this project's CI runs on, so the wrapper is ordinary rather than exotic. The set is now one shared predicate covering `sh`, `bash`, `dash`, `zsh`, `ksh`, `mksh` and `ash`, with **nowhere left to widen but that one place** — re-listing is how a fix reaches some sites and leaves the rest latent. That is the *interpreter* set only: a command that merely execs another (`nice`, `setsid`, `busybox sh`, `su -c`) is handled by the separate wrapper set, which has gaps of the same shape and is not touched here (iss-272). `eval` keeps its own branch (a builtin, with its own end-of-options rule), and a different *language* stays out: `python -c` and `perl -e` carry source this tokenizer cannot read, so guessing a verdict on them would be unsafe. Their recorded posture is a loud warn — a design target, not yet implemented, so today they remain a silent allow (corrected in Unreleased, iss-315). (gh-297)

- **The rest of the hook plane fails open too.** A skewed plugin install no longer blocks a session through any usage error, not just an unknown sub-verb. On a host hook exit 2 is the host's instruction to *block*, and the previous fix covered only the two parents — a stray positional on a leaf (`abcd guard hook zzz`) and **any unknown flag** still exited 2, so `abcd guard hook --strict` against a binary predating that flag would have blocked every shell command in the session. That is the reachable one: `hooks/hooks.json` ships with the plugin clone while the binary comes from the latest release, so the manifest runs ahead, and adding a flag to an invocation is an ordinary change. Every command a hook can reach now refuses at exit 1 — loud, non-blocking, naming the skew and its remedy. **`guard check` is deliberately excluded** and keeps its exit 2: it is the human and scriptable verb, where a fault must never read as clearance. The guarded set is derived from the manifest in the test, so it cannot drift from what a hook actually invokes. (iss-269)

- **A skewed plugin install no longer blocks the session it is meant to guard.** On a host hook an exit status is an instruction, not a diagnostic: the host reads 2 as *block this action*. Since every command parent began refusing an unknown sub-verb at cobra's usage status, a renamed hook sub-verb made `abcd guard <old-name>` exit 2 — so the guard claimed a hazard verdict it never reached and blocked **every shell command** in the session, and `abcd hook <old-name>` blocked **every prompt**. The manifest and the binary can genuinely skew, because `hooks/hooks.json` ships with the plugin clone while the binary is fetched from the latest release, and the PreToolUse wrapper could not rescue it: it treats 2 as a recognised code, so its *FAILED TO RUN … UNGUARDED* net never fired. The unknown-sub-verb path under the two parents a hook reaches now refuses at exit 1 — loud, non-blocking, naming the skew and its remedy. A mistyped sub-verb still never reads as success; only the code moves. **Scoped, not universal:** a stray positional on a leaf, an unknown flag, and an unknown top-level token still exit 2, none of them reachable from today's manifest (iss-269 carries that gap). One consequence worth knowing: `abcd guard <typo>` now exits 1, which is also `guard check`'s code for a block verdict, so a script keying on the exit code alone can no longer tell a typo from a hazard — the stderr text distinguishes them. The tree-wide usage-error mapping no longer overwrites an exit code a validator chose deliberately, which is what had been silently undoing this. A new test resolves every invocation in `hooks/hooks.json` against the live command tree, enforcing the doctrine that actually survives the skew: the manifest's spellings are **frozen**, and a rename is absorbed by an alias in the binary — because the clone runs ahead of the binary, an alias only ever helps the new side, so editing the manifest to a new spelling is the move that strands older binaries. (iss-267)
- **The attribution gate now catches the footer it was written to catch.** Its banned-footer rule anchored on optional leading whitespace, but the footer a hosted agent platform actually appends is wrapped in markdown emphasis — a leading underscore — so the precise shape that motivated the gate went straight through the pull-request-body check. It was found live on two pull requests whose failing leg happened to be something else, which is how it stayed hidden. The rule now admits an optional emphasis run — italic, bold or bold-italic — with the robot-emoji form allowed on either side of it, since a footer may put the emoji inside the italics or before them. The marker must be attached to the word, exactly as markdown reads it (`*text*` is emphasis, `* text` is a list item), so a bullet describing the banned footer stays documentation rather than a violation — the same writable-about property the line anchor already protects. Eight new corpus cases pin the refused forms and two pin the list-item forms that must keep passing; narrowing the run, dropping either emoji position, or allowing a space after the marker each fails the corpus, so every clause earns its place. (iss-262)
- **A mistyped sub-verb no longer reads as success.** A cobra parent with no `RunE` is not runnable, so cobra printed help and exited **0** without ever running the command's argument validator: `abcd docs nonsense`, `abcd guard nonsense`, `abcd embark nonsense`, `abcd ideate nonsense`, `abcd docs cite nonsense` and `abcd hook nonsense` all reported success to a calling script. Every parent in the tree is now runnable, so its declared argument validator actually runs and refuses an unknown sub-verb at exit 2, while bare invocation still prints help at exit 0. Refusing needs both halves — a runnable parent that declares no validator falls through to cobra's arbitrary-args default and exits 0 anyway — so the guarding test asserts runnability *and* a declared validator across the live command tree, and derives the parents it exercises from that tree rather than a hand-kept list, so a parent added later cannot reintroduce the hole by missing either half. `capture` and `intent` are unchanged — their positional is free text by design, and they keep their own suspected-typo guard. (iss-266; the sweep spc-30 owed after fixing the `disembark` parent alone)

## [0.5.1] - 2026-08-16

### Added

- **abcd is citable, and its sources are on the record.** A root `CITATION.cff` (CFF 1.2.0) powers the forge's cite-this-repository box; the References & sources section of `ACKNOWLEDGEMENTS.md` records the academic literature the design record draws on as a curated, alphabetically ordered list of primary sources; and the canonical CSL-JSON metadata lives at `.abcd/development/research/references.csl.json`, with a documented on-demand `.bib` export for LaTeX toolchains — generated, never committed. (iss-257)

### Fixed

- **A missing plugin binary no longer degrades the whole session in silence.** Session start was the only hook that ever provisioned the binary, so when that one hook did not fire the session stayed broken for good, with every later hook failing as a raw `No such file or directory`. Every binary-invoking hook — the prompt router, the shell guard, the compaction reset, the session-end capture — now guards on the plugin root, attempts a rate-limited silent bootstrap salvage when the binary is absent (one attempt per ten-minute window), resolves the binary plugin-root first and then from `PATH`, and on continued absence prints one plain line naming exactly what is degraded and the install remedy. Session start keeps its loud primary-provisioner role unchanged. (iss-254)
- **`ahoy install` no longer hides a repo's committed record tiers from git.** The public visibility block used to ignore `/.abcd/` wholesale, so on any repo that commits the durable tiers — this one included — new intents and issues vanished from `git status` and `git add` refused them. When `.abcd/` holds tracked files, the block's `.abcd/` entry — and only that entry — now narrows to the local-ephemeral tier, and the install receipt says out loud that the committed record tiers remain published. Narrowing needs positive evidence: a directory whose git cannot be asked, or no repo at all, keeps the declared set, and the `memory/` snapshot fence is unchanged. (iss-255)

## [0.5.0] - 2026-08-16

### Breaking

- **The documented install location is `~/.local/bin`, and nothing abcd runs asks
  for administrator rights** (iss-171). The README one-liner drops `sudo` and
  copies the verified binary into `~/.local/bin`; `abcd ahoy install` writes its
  `PATH` entry there too, creating the directory when absent. **What existing
  users see depends on how their binary got there.** An abcd-owned symlink — the
  thing `ahoy install` writes — is found anywhere on `PATH` and adopted in place,
  so nothing changes. A binary the old one-liner *copied* into `/usr/local/bin`
  is a plain file abcd does not own: it is never adopted, and a new entry in
  `~/.local/bin` lands behind it on `PATH`, so the copy is still what runs. That
  case now reports a `symlink.shadowed` gap naming the copy, both from `ahoy`
  and on the install run itself; the remedy is to delete the stale copy
  (`rm /usr/local/bin/abcd`, which needs the rights that put it there) or to
  install ahead of it with `--bin-dir`. abcd will not remove it — it never
  touches a binary it does not own. A system-wide directory is reachable only
  through an explicit `--bin-dir`, which fails loudly when it is not writable
  rather than re-running itself with privilege: abcd escalates nothing, so there
  is no fallback to hide the refusal behind. Two more gaps arrive with it:
  `~/.local/bin` not on `PATH` is its own named gap carrying the one-line
  `export PATH="$HOME/.local/bin:$PATH"` fix (abcd prints it and never edits a
  shell profile), and an abcd-owned `PATH` entry whose binary has gone is
  reported as dangling rather than silently trusted. Install refuses to create a
  link whose target does not exist, because a dangling `abcd` early on `PATH`
  shadows every working one behind it, and it reports every such refusal on the
  result rather than leaving a gap to speak for it. The old detector recognised
  exactly one blessed target, so a working `~/.local/bin/abcd` reported
  `symlink.missing` while the detector was itself running as that very binary,
  and "fixing" it would have written the shadowing link.

### Added

- **A stale abcd never answers silently** (itd-111). Every surface now knows its
  own vintage and says so. `abcd version` and `abcd ahoy` print the install mode,
  the running binary's vintage (its build revision in a source checkout, its
  pinned version otherwise), and whether it is up to date, stale, or of an
  undeterminable vintage relative to the on-disk reference. At session start, a
  binary behind (or a dirty rebuild atop) its own source tip is named with the
  one-command rebuild fix. `abcd ahoy install` refuses before any write when the
  binary is stale against its tip or its vintage cannot be determined — the trap
  where month-stale install logic silently ran against a machine — with
  `--allow-stale-binary` as the documented override. And `abcd version --check`
  is the one command that reaches the network: it fetches the latest release
  once, compares, and reports with its source named. Every other path reads only
  what is on disk.
- **`/abcd:intent` decomposes a proposal before filing it** (itd-84, MVP
  rung). The surface page now carries a hand-run protocol that routes each
  part of a proposal to its record home (capability → intent, trust rule →
  ADR plus brief invariant, stance → principle, plumbing → brief), surfaces
  typed links to existing records, and renders an advisory FILE-AS-IS /
  SPLIT / HOLD verdict a human confirms; the planning interview runs the same
  step. Not yet automated: the deterministic pre-pass and the capture-time
  validator are future rungs, and every hand-run is graded into a calibration
  corpus that gates them.
- **The attribution gate reads the git identity, not only the message.** A commit
  authored and committed as `Claude <noreply@anthropic.com>` carried a fully
  compliant message — `Assisted-by:` trailer, no banned footer — and sailed
  through the gate, because the gate read only messages and bodies. The
  contributor graph is built from commit authorship plus `Co-authored-by:`
  trailers, so that one identity put an AI at #2 in the graph twice over: once
  for the commit itself, and again on every squash merge, where the forge
  auto-appends a `Co-authored-by:` for any branch author who is not the PR
  author. The commits half now refuses an AI author or committer identity —
  whole-name match on the assistant names AI tools stamp by default, plus the
  vendors' address space, vendor-agnostic in intent like the co-authorship ban —
  while the bot exemption is untouched and a human whose name merely contains an
  assistant's name still passes. The corpus at
  `scripts/check-attribution-cases.sh` grows a commits-mode section that proves
  all of it against a scratch repository.

- **The attribution gate accepts the trailer forms actually in use, and refuses AI
  co-authorship whoever wrote it** (iss-214, iss-215). Two defects in the gate's first cut. Its trailer
  pattern had no bracket in its character class, so `Assisted-by:
  Claude:claude-opus-5[1m]` — an exact model identifier, carried by 38 commits of
  this repository's own history — was rejected while the unbracketed form passed,
  so an agent disclosing its precise model followed the prose and failed the check.
  And the co-authorship ban named one vendor, so `Co-authored-by: ChatGPT` passed
  while `Co-authored-by: Claude` did not, leaving the gate effective only for as
  long as the assisting model was Claude. The trailer now admits an optional
  bracketed context suffix; the co-authorship ban matches the trailer KEY rather
  than a vendor name; and the "generated with" ban matches the footer's shape, so
  `Generated by [tool]` is caught alongside `Generated with [tool]`. A test corpus
  at `scripts/check-attribution-cases.sh` pins all of it — including the historical
  identifiers and the shell-metacharacter body — and runs in the gate's own
  workflow, so the rules are proven rather than asserted. A bare vendor with no
  version stays refused, and refusing every co-authorship trailer refuses a human
  one too: abcd defers DCO until the repo is public or takes an outside
  contribution, so there is no such case today.

- **The AI-attribution convention is enforced, not merely written down.** A new
  `attribution` workflow fails a pull request whose commit messages or body break
  the rule `AGENTS.md` and `CONTRIBUTING.md` state: the kernel trailer
  `Assisted-by: Claude:<model-version>`, never `Co-Authored-By:` for an AI, and
  never a tool's own "Generated with <tool>" footer. `scripts/check-attribution.sh`
  holds the logic and runs locally as `make check-attribution` for the commit half.
  The convention had been prose since the beginning and drifted anyway — a
  reconciliation sweep across 78 pull requests was needed once before, after PR
  bodies picked up a tool's default footer. The body half is the half that slips,
  because it comes
  from a tool default rather than from a contributor's habit, so the trigger
  includes `edited`: a body corrected or broken after opening is re-checked. Bot
  authors are exempt — announced in the log rather than silent, and applied inside
  the steps so the job always runs and the check always reports, which is what makes
  the workflow safe to mark required: a required check that never reports leaves a
  pull request waiting on it forever. The gate binds only once it is added to the
  branch's required status checks.

- **A dependency bump that lands on a release workflow can be carried into the
  template it was rendered from** (iss-209). `.github/workflows/release.yml` and
  `auto-release.yml` are rendered from
  `internal/core/launch/scaffold/templates/*.tmpl`, and `TestSelfScaffoldParity`
  holds the two byte-identical so that abcd's own release exercises the machinery
  a managed repo receives. Dependabot sees only the rendered side — its
  github-actions ecosystem discovers `.github/workflows/*` and composite action
  manifests, and no ecosystem scans a `.tmpl` under `internal/` — so every action
  bump it opens broke that parity with no mechanical way back. `make
  scaffold-sync` now propagates the pinned refs from the committed workflows into
  the templates, and `make scaffold-sync-check` reports the same drift without
  writing. Only the ref moves: indentation, list markers and version comments
  survive byte-for-byte, an action the workflow does not pin is left alone, and an
  action pinned to two different refs in one workflow is refused rather than
  guessed. The propagation is one-way by design — re-rendering the template over
  the workflow would revert the bump. `TestSyncRepoPinsIsCleanToday` fails
  preflight when a workflow pin has moved without the template following, and
  names the command that fixes it.

### Fixed

- **The guard now reads the execute-a-string wrapper family, and a warn is no
  longer silent on the hook** (iss-200, iss-231). A command carried as a data
  argument — `sh -c '<payload>'`, `bash -lc "<payload>"`, `eval '<payload>'`, and
  GNU `env -S<value>`/`--split-string` — used to sail past every blocker: the
  payload was one opaque token the matchers never looked inside, so `env -S 'gh
  repo delete owner/repo'` and `sh -c 'git push --force'` were accepted. The guard
  now expands each payload once and matches it as if it had been typed inline, so a
  hazard inside blocks (or, for a warn-tier hazard, warns) exactly as the bare
  command would. The `sh`/`bash`/`dash` `-c` command string is read as the shell's
  first non-option operand, so an option wedged after `-c` (`sh -c -x '<payload>'`,
  `bash -c -- '<payload>'`) can no longer hide it, and `eval` drops a leading `--`
  before joining; when options make the operand impossible to locate the guard warns
  rather than allowing. `env -S` is read on the raw tokens in every spelling — separate,
  glued, `--split-string=`, its abbreviations, and bundled short clusters — at
  every `env` in a wrapper chain, and its value is split only when it decodes to a
  provably plain command; anything else (an expansion, a quote, a leading option, an
  escape env itself would reject) is refused rather than guessed. A payload the
  guard cannot read splits by posture: an uninspectable `sh -c`/`bash -c`
  (a `$(...)` substitution, a pipe into an interpreter) is a loud warning, while an
  uninspectable `env -S` or a nest deeper than two layers is blocked outright. And a
  warn on the pre-tool-use hook now exits non-zero-but-non-blocking so its message
  is actually seen — a hook that exits zero has its stderr discarded, so every warn
  (for example `git reset --hard`) had been running as if allowed with nobody told.
- **The one instruction that resolves the no-binary-on-`PATH` state can now be run
  in that state** (iss-207). The bootstrap's success notice and the README both
  said to run `abcd ahoy install` once — a command whose whole premise is that
  `abcd` is not a name the shell can resolve, so it failed with "command not
  found" for precisely the reader it was written for. On the first manual install
  the consequence was not cosmetic: the agent reading the notice could not run the
  printed command, invented a `go run` incantation reaching into the harness's
  plugin cache, and told the user to run that instead — a source-build path
  needing a Go toolchain, and not the documented install at all. The notice now
  prints the absolute plugin-root path the script already holds, shell-quoted so
  a plugin root containing a space, an apostrophe, a `$` or a backtick still
  pastes as one word, and with the invocation last on the line so it stays
  copy-pasteable to the end. The README carries the same form with the one part a
  committed file cannot know left as a placeholder, says what that placeholder is
  in host-agnostic terms, and points a reader who cannot instantiate it at the
  install one-liner, which needs no plugin root. CI holds both surfaces — every
  `ahoy install` either one prints must be reached through a path, not a bare
  name, and the printed command is handed to a real shell against a hostile path
  to prove it runs as pasted — while the end-to-end reading of it on a real
  plugin cache remains the manual install gate.

- **The session-start hooks run after the bootstrap that provisions their binary,
  and a successful install reads as success** (iss-204, iss-208). The hook
  manifest listed the bootstrap and the two binary-backed commands as three
  sibling `SessionStart` entries and relied on list order; the harness runs every
  hook matching an event in parallel, so both gated entries raced a ~10.7 MB
  download, lost, printed "the plugin binary is not installed", and genuinely did
  not run — on every fresh install and every plugin update, since an update lands
  in a fresh cache directory with no binary. The three entries are now ONE
  command that runs the bootstrap and then both binary calls in a single shell,
  so the sequencing is owned by the manifest rather than assumed of the harness.
  Chaining them makes two further properties load-bearing, and both are held
  explicitly: the hook payload is read once and piped to each call separately,
  because every hook verb consumes the whole of stdin and a shared stdin would
  leave `session-start` reading EOF and silently disabling its notices; and
  `session-start` runs ahead of `prompt-router-reset`, whose unconditional
  success diagnostic would otherwise be the one line the transcript renders.
  The bootstrap's own message is emitted first, which is what the transcript
  renders: on a fresh install the visible line is the checksum-verified success
  rather than one of two missing-binary complaints, and the two complaints
  collapse into one. The honest-failure posture is unchanged — a refusal keeps
  its message and its exit code, a binary that is genuinely absent is still said
  out loud, and the binary calls' stdout still reaches the model untouched. The
  spec that shipped the bootstrap carried the false warrant ("ordering within one
  event's hook list is preserved by the harness") as a load-bearing claim; it is
  corrected in place, with the brief's two descriptions of the manifest. Parallel
  hook execution and the plugin cache are not present in CI, so the end-to-end
  proof is the manual install gate; what CI holds is the manifest's shape and the
  chained command's behaviour against fixtures.
- **`ahoy install` prompts read a piped answer, in a fixed order, and `--yes` says
  what it does not cover** (iss-167, iss-166). The prompter attached to stdin only
  when stdin was a terminal, so `yes | abcd ahoy install` — the first thing an
  agent reaches for — arrived as a decline on every question, and the interactive
  path could not be driven at all: the agent reported failure and handed the step
  back to the human. Prompts now read stdin whether or not it is a terminal, and
  off a terminal each question's answer is echoed to stderr, so a piped run leaves
  a transcript of what was asked and answered. Piped answers are positional, so
  the approval questions are now asked in a **fixed order** — dependency,
  safe-autocreate, config-change, user-state, plugin-owned, the order the apply
  pass acts in — where the walk previously ranged over a map and handed out a
  fresh permutation on every run: the same command approved a different category
  each time, exiting 0 and reading as a clean install. One line answers one
  question, which is why `yes` is the documented form. The safe default is
  unchanged:
  answers that run out read as EOF, and EOF declines every confirm and takes the
  default for every prompt, so an unattended run still adopts nothing it was not
  told to adopt. The interactive path at a terminal is untouched. Folded in:
  `--yes` deliberately does not adopt the optional git-identity pin — the pin
  records whatever identity is currently configured, and a blanket approval would
  canonicalise a sandbox or agent identity, the very value the identity gate
  exists to reject — but it reported "already up to date" without mentioning the
  skip. The exclusion is now stated in the flag's own help, carried in the install
  envelope as `optional_skipped`, and printed with the way to apply it —
  `yes | abcd ahoy install` — which the piped answer makes available to a
  non-interactive caller for the first time. A run that must neither block nor
  prompt closes stdin and pre-answers
  (`abcd ahoy install --yes --refuse-adopt < /dev/null`); the plugin surface says
  so, because reading a non-terminal stdin means a stdin held open and silent
  makes a prompt wait rather than decline.
- **An `ahoy install` receipt is safe to paste** (iss-177). Every apply step
  reported its write as an absolute path and the CLI printed them verbatim, so a
  receipt pasted into an issue or a transcript carried the developer's home
  directory and username — while the sibling verbs already routed their error
  text through a shared path scrub the receipt did not use. The receipt now
  reports a repo write repo-relative (`.abcd/config.json`) and a user-scope write
  home-relative (`~/.abcd/history/index.json`), leaving a location that names no
  developer (`/usr/local/bin/abcd`) exactly as written — the same limit the error
  scrub already states, through the same primitive, which moved to
  `internal/fsutil` so the two cannot drift. The scrub sits at the one seam every
  step reports through rather than in each step's string, and a test holds that
  seam to being the only writer of the receipt, so a step added later cannot
  reintroduce an absolute path by forgetting.
- **The command surface reaches the binary a plugin install actually provisions**
  (iss-205). Every command file resolved the binary as a bare `abcd` on `PATH`
  with a `go run ./cmd/abcd` fallback, and none named `${CLAUDE_PLUGIN_ROOT}` —
  while the bootstrap hook installs its checksum-verified binary *into the plugin
  root* and leaves nothing on `PATH`. The two halves never met: a fresh install
  worked only because the marketplace clone happened to carry `cmd/`, costing 54
  seconds and a Go toolchain, and on a machine without Go the whole `/abcd:*`
  surface was non-functional despite a healthy binary sitting in the plugin root.
  The resolution ladder in all 17 binary-invoking command files now runs
  `"${CLAUDE_PLUGIN_ROOT}/abcd"` first, `abcd` on `PATH` second, and `go run
  ./cmd/abcd` third and explicitly only in a source checkout — the published
  payload carries no `cmd/`, so an unqualified third rung prints an instruction a
  plugin user cannot follow. Every fenced command line, which is what an agent
  runs verbatim, carries the plugin-root form.
  `TestCommandSurfaceResolvesBinaryFromPluginRoot` keeps it that way: it fails if
  any file under `commands/` names the binary without the plugin-root rung first,
  hands over a fenced invocation that resolves any other way, leaves a `go run`
  rung unqualified, or drops the ladder paragraph. It runs under `go test ./...`,
  so `make preflight` and CI already execute it rather than needing a target
  anyone can forget to wire.

- **The build plumbing's own comments describe the gate suite that runs**
  (iss-182). The `Makefile` preflight comment claimed the target ran "the same
  steps CI's check job runs" and named only the reviews-charter gate, though the
  recipe takes three lint prerequisites (`lint-reviews`, `record-lint`,
  `docs-lint`) and CI additionally runs a `gofmt -l .` format gate that preflight
  does not; the `.githooks/pre-push` comment called what the hook enforces a
  "check-job trio", undercounting those lint gates, and listed only the
  secret-scan and workflow-audit lanes as Actions-only; and the `ci.yml` header
  omitted the check job's gofmt, record-lint and docs-lint steps along with the
  reviews-charter and smoke jobs. All three now state what actually runs.
  Comment-only — no recipe, hook logic, workflow step, or gate behaviour moves.

- **The record describes the `.abcd/**` exclusion by the channel it is true of**
  (iss-183). The blanket present-tense claim — `.abcd/**` "never ships", or is
  "excluded from the release artifact by packaging" — survived across
  `CONTEXT.md`, `AGENTS.md`, both `.abcd/` READMEs, four brief sections, the
  release glossary term, the Phase 1 expectation and a scanner code comment,
  after the README alone was corrected. The exclusion is implemented but has
  never run on a release: the launch bundler denies the `.abcd` namespace
  structurally, while a marketplace install takes the repository root and GitHub
  attaches an auto-generated source archive to every release, so only the
  released binaries omit the directory. Each descriptive instance now says so —
  present in every repository checkout, marketplace installs and release source
  archives included, never in the released binaries — and names the bundler as
  the implemented mechanism it is. Decision records, dated plans and intent
  bodies keep their original wording.

- **Merging a rule-set overlay onto a base that declares no domains no longer
  panics** (iss-187). `rules.Merge` promises that new domain keys are added, but
  it wrote them into a map it never allocated when the base carried no domains of
  its own — a valid rule set that the validator accepts — so the merge crashed
  instead of returning the overlay's domains. It now allocates that map before
  adding the overlay's keys, matching the guard registry's loader. No behaviour
  changes for the rules a repo loads today, where the base is always the bundled
  default set.

- **`abcd capture resolve`/`wontfix` no longer strand an issue across two status
  directories on a failed move** (iss-186). The transition wrote the destination
  file, then removed the source; if the removal failed for a reason other than
  "already gone" (a read-only remount or a restrictive attribute on the source
  directory), the error surfaced after the destination already existed, leaving
  the same issue id present in both `open/` and its target directory. From then
  on every later transition on that id was refused as a duplicate, with no
  repair verb — the file had to be deleted by hand. The move now rolls the
  destination back when the source can't be removed, so a failed transition
  leaves the ledger exactly as it was before the attempt and a retry is all
  that's needed.

## [0.4.2] - 2026-08-06

### Added

- **`context_citation_currency` — the orientation doc cannot ground a live
  caveat in a record that is finished** (iss-42). `CONTEXT.md` is the first file
  a session reads and its sharp-edges list is the paragraph a reader trusts
  before they have read anything else, yet nothing required that list to be
  revisited when the record it cites moved — so the staleness was structural, not
  incidental, and recurred every time a cited record shipped, resolved, closed,
  or was superseded. The rule resolves every `iss-N`, `itd-N`, `spc-N`, and
  `adr-N` handle the sharp-edges section names against the store that holds it,
  and blocks any citation whose target sits in a terminal lifecycle state —
  `resolved`/`wontfix` for an issue, `shipped`/`superseded` for an intent,
  `closed` for a spec, and a declared supersession or a retired status for an
  ADR, whose flat store carries its lifecycle in frontmatter rather than in a
  directory. Scope is one section of one document on purpose: a shipped intent's
  evidence trail, a supersession chain, and a dated plan all name closed records
  legitimately, and only the living orientation doc claims to describe what is
  true right now. A handle that resolves to no record is left alone — dangling
  cross-references are `record_schema`'s question, already answered there. It
  fails closed on the three ways an armed gate could check nothing: no target to
  read, no sharp-edges section in it, and configured stores that resolve no
  records at all, which reads exactly like a clean section. Any abcd-managed repo
  enables it by declaring its own `target` and `record_stores` in
  `.abcd/record-lint.json`, and may name a different section with `section`.
- **The development record map's `research/` row is derived rather than
  hand-kept.** An `index_drift` region holds the row to the directory's actual
  subdirectories, so a routing claim naming a child that does not exist — or
  omitting one that does — fails the record gate instead of quietly misdirecting
  a reader.
- **`delivery_state` — the changelog cannot credit an intent the record calls
  unbuilt** (iss-41). An intent in `drafts/` is a captured idea nobody has
  committed to build, so the intent tree and a delivery entry citing it say
  opposite things about the same work — and the tree is the side nobody re-reads.
  The rule reads each version entry's delivery sections — `Added` and `Changed`,
  plus any heading a repo names in `delivery_sections`, which is unioned with
  those rather than substituted for them so a config can widen the gate but never
  narrow it — and blocks any `itd-N` citation whose intent still sits in
  `drafts/`. Non-delivery sections are out of scope on purpose: an id
  under `Fixed` is normally provenance for a defect — which draft two branches
  minted at once — not a claim that the intent is built. Its two remedies are the
  two truths a finding can be reporting: the intent shipped whole and was never
  promoted out of `drafts/`, or the entry credits an intent for less than it
  promises, in which case it describes the capability without the citation, since
  an intent is delivered whole or not at all. It fails closed on the three ways an
  armed gate could check nothing — no changelog to read, no intents store to
  resolve against, and a store holding none of the lifecycle buckets, which is
  what a root pointed one level too deep looks like and reads exactly like a clean
  corpus. Any abcd-managed repo enables it by declaring its own `changelog` and
  `intents_root` in `.abcd/record-lint.json`.
- **`index_drift` gains `dir_entry`, so a listing can enumerate records by id.**
  The rule compared a document's entries against whole filenames, which only ever
  agreed when the document transcribed slugs — a listing nobody writes by hand.
  `dir_entry` is the mirror of `entry` on the directory side: a regexp that
  reduces each file's stem to the part the document enumerates — a record's id
  rather than its id-plus-slug filename — and drops any stem it does not
  describe. That is what lets the brief's later-phase list be gated by the same
  one rule instead of a second one shaped like it. It is rejected in `absent`
  mode, which resolves listed paths rather than reading a directory.
- **`record_schema` — the design record is checked across its stores, not just
  inside each one** (iss-39). Every existing record rule asks a question about one
  store: is this intent's bucket schema right, does this spec agree with its
  intent. None of them asked the questions that only make sense between stores,
  which is where a record actually drifts. The new rule reads the ADR, intent,
  spec, and issue stores in one pass and holds four invariants: a cross-reference
  field (`supersedes`, `related_adrs`, `related_intents`, `builds_on`,
  `blocked_by`) names a record the corpus has — or one a successor declares it
  superseded, since a pruned record is accounted for rather than lost; a
  supersession is declared from BOTH sides, so a record can no longer contradict
  itself about which decision is in force; a filename and the id inside it agree,
  so one record cannot answer to two handles; and every directory that should
  hold records is enumerated — an undeclared bucket in a store root, and a
  subdirectory inside a bucket or a flat store — so a directory nobody declared is
  a finding rather than a place records hide from every check. A supersession may
  cross stores — an ADR that redecides the question an
  intent rested on retires that intent — so `superseded_by` accepts `adr-N` as
  well as `itd-N`. That widening makes `intent_lifecycle`'s own target check
  looser on its own: it reads only the intent tree, so a repo arming it WITHOUT
  `record_schema` accepts an `adr-N` successor without resolving it. The two are
  meant to be armed together, and `record_schema` is what resolves a cross-store
  target. Cross-references are checked in frontmatter rather than in prose on
  purpose: a frontmatter handle is a machine-readable claim that the record
  exists, while prose legitimately names ids that do not resolve. A retirement
  declaration is bounded by what the store has issued, so `supersedes:` cannot
  mint a phantom id that then resolves everywhere else. Any abcd-managed repo
  enables it by declaring its own `record_stores` in `.abcd/record-lint.json`.
- **`index_drift` — a hand-written directory index is gated, not trusted**
  (iss-38). A README that enumerates its sibling files by hand is a second copy
  of something the filesystem already knows, and it drifts the moment a file is
  added, renamed, or shipped — silently, because nothing was checking. The rule
  reads a region a document fences with `<!-- index: <id> -->` and holds it to
  the directory it enumerates: in `exact` mode the listing and the directory must
  agree in both directions, so a file added without a line and a line kept for a
  file that has gone are each a finding; in `absent` mode every listed path must
  still be missing from the tree, which is the shape a "planned seams" list has,
  where the drift is a seam that shipped while the list still calls it planned.
  Scoping is by explicit marker plus a configured entry pattern rather than by
  parsing arbitrary markdown, so unrelated prose around a list cannot produce a
  finding and no bespoke parser is needed per README. It fails closed on the
  three ways a gate could be quietly disarmed — a region deleted while its config
  entry remains, a region that parses to no entries at all, and a malformed index
  spec (no document, no directory, no entry pattern, an uncompilable pattern, an
  unknown mode) — the last as a loud configuration error rather than a pass. Any
  abcd-managed repo enables it the same way: an `index_drift` block naming its
  own document/directory pairs in `.abcd/docs-lint.json`.
- **`abcd banlist` — the names a repo must not publish, in two layers** (itd-74,
  spc-20). Enforcement splits by sensitivity, because a deterministic CI gate is
  the right tool for a public banned name and the wrong place for a private one:
  the rule would have to contain the very string it forbids. The **public** layer
  is the `banned_tokens` family of `.abcd/docs-lint.json` — the same primitive
  that already gates this repo's harness names, not a second mechanism — with
  verb-written entries under a `names/` id prefix that marks what the verb owns:
  `list` renders the whole family, and a removal is refused for a hand-curated
  entry. Config edits are byte surgery on the located array rather than a
  re-marshal, so an add is one inserted line, a remove is one deleted line, and
  add-then-remove returns the file to its exact bytes. The **private** layer is a
  gitignored per-machine store read by the committed pre-commit guard, and its
  visibility follows: entries render by key only, never their pattern, and the
  redaction is structural — the entry type carries no pattern field, so no
  rendering can leak one. A private pattern is entered by piping it on stdin
  (`printf %s 'PATTERN' | abcd banlist add --private KEY -`), which is the
  recommended form because an argument is world-readable in `/proc/<pid>/cmdline`,
  is captured by process auditing, and lands in shell history. Each layer is
  validated against the engine that enforces IT — a private pattern by the guard's
  own grep, a public one through the linter's compile path — so an entry cannot be
  stored as healthy while it matches nothing; and `add --private` refuses outright
  if git does not ignore the store's path, since the layer rests on that file being
  untracked. `add` and `remove` name their layer explicitly (neither flag and both
  flags exit 2); bare invocation and `list` are read-only, both state their reach
  plainly, including that CI cannot enforce the private layer, and `list --private`
  separates a line the guard cannot use from one it accepts but reads differently,
  because the first stops every commit and the second stops nothing.
- **`abcd ahoy` scaffolds the whole two-layer name banlist** (itd-74, spc-20). A
  repo becomes name-safe by being abcd-managed rather than by a maintainer
  hand-wiring the files. Install writes FIVE artefacts, each reported by name on
  the status board: `.githooks/pre-commit` (the guard), `.githooks/pre-merge-commit`
  (git runs no pre-commit for a merge commit, so a banned name would otherwise walk
  into history the moment a merge commit carrying it is made), an appended
  `.gitattributes` line keeping both hooks at LF (a `core.autocrlf` checkout
  rewrites a script git executes, and its shebang stops resolving),
  `.abcd/docs-lint.json` carrying an empty public banned-names family, and the
  documented private stub in the gitignored local tier. A clone arms the hooks once
  with `git config core.hooksPath .githooks`; abcd never sets it, and no surface
  reports a committed hook as a running one. The merge half is written ONLY beside
  abcd's own guard, identified by a whole `# abcd-name-guard: v1` line: beside a
  maintainer's own hook it would both claim coverage it has not got and silently
  start running that hook on merge commits, which git never did. The public family
  is seeded EMPTY on purpose — abcd cannot know which names a repo may not publish,
  and a ban nobody declared would fail a build over a word the maintainer never
  chose. Every write is create-if-absent and contained: a hook, a CI-gating config,
  and above all a populated private store are the maintainer's, and paths resolve
  through a containment root so a symlink committed at `.githooks` or at the local
  tier cannot land an artefact outside the repo while the surfaces report the
  in-repo path. The stub is written only where `git check-ignore` itself reports the
  store's path as ignored, not where a comparison of `.gitignore` text suggests it:
  a repo can carry a byte-perfect block and still track the store, and a stub git
  would track is the hazard rather than the remedy. Where git cannot be asked at all
  — missing from PATH, a corrupt `.git` — a repo-shaped directory fails closed
  rather than borrowing a plain folder's answer. The public config is held to the
  same rule: abcd does not write one into a path git ignores, because it would have
  to report it unenforceable in the same breath. The stub's worked examples are all
  commented out — a fresh scaffold parses to zero entries, and the guard says so
  loudly at commit time instead of looking like protection — and every illustrative
  value in it is a reserved documentation value (RFC 5737, RFC 3849, RFC 2606, RFC
  7042) or a persona-derived fixture host, judged by the repo's own
  network-identifier detector.
- **The name guard refuses copies of the private store, with a published escape**
  (itd-74, spc-20). The store-path refusal matched only the local tier, so a COPY of
  the private banlist anywhere else — `notes.txt`, a `.bak` beside it, or a `git mv`
  out of the tier — committed every pattern in clear while the guard announced a
  clean check: the entries cannot catch their own text, because they are escaped
  regular expressions and a pattern does not match itself. Three tests now: a staged
  path inside the local tier (including a rename's source path, and not escapable),
  a staged blob whose first line is the format declaration, and a staged path whose
  basename is the store's filename. They are shape tests on a mistake rather than a
  net against someone determined — a copy with the declaration stripped or displaced
  still commits — and they carry a per-file escape, because a repo that legitimately
  commits a store-shaped file (a fixture corpus, a doc quoting the declaration) needs
  one that is not `--no-verify`: a second line reading `# abcd-banlist-example`
  exempts a blob from the copy refusals and from nothing else, its content still
  scanned against every entry. The guard also pins `PATH`, `IFS` and xtrace and
  unsets any inherited shell function shadowing a command it runs, as its first
  statements, and BLOCKS loudly when a tool it needs is missing rather than failing
  as a mute exit 127; a machine with a nonstandard prefix extends the pin with
  `git config --local abcd.guardPath <dir>`, never an environment variable, since a
  repo-scoped environment is the hole the pin exists to close. The format declaration
  must be line 1 or nowhere: a blank line, a comment, a duplicate below it, or any
  prefix bytes before it is a damaged declaration rather than a silent downgrade to
  the legacy format, and both readers say so identically.
- **Every surface that describes the private layer states its reach** (itd-74,
  spc-20). `abcd ahoy` now reports the name guard's state — what occupies each hook
  path, whether the public family can actually be enforced, and what shape the
  private layer is in on this machine — and the status line, the JSON envelope, and
  the `abcd banlist` verb all carry the same sentence: CI cannot enforce the private
  layer; it protects only machines that have opted in, and only the commits git runs
  a hook for, so a fast-forward `git pull`, a rebase, a `git am`, a `git revert` or a
  cherry-pick bypasses it, as does `--no-verify`. The list is explicitly
  non-exhaustive and names nothing that is in fact covered. The reach travels inside
  the reported state rather than being added by a renderer, because a machine
  consumer reading "hook committed" beside a present store would otherwise draw
  exactly the wrong conclusion. Where a claim cannot be supported it is withdrawn
  rather than softened: a docs-lint config git ignores — the state
  `visibility: public` puts every repo in, since the installed fence ignores the
  whole `.abcd/` namespace — is reported as NOT ENFORCEABLE instead of as the
  committed, CI-enforced layer, with the placement question left for a maintainer to
  settle (iss-176). The status pass reads the private store's shape (its format and
  two counts) and never its content: the patterns are the secret, and a status board
  is the surface that must not hold them.
- **The private name guard refuses by key and says when it is inactive** (itd-74,
  spc-20). The committed `.githooks/pre-commit` guard checks the CONTENT of every
  staged file, read out of the index, and on a match refuses the commit naming the
  entry key alone: the matched text and the pattern value never reach stdout,
  stderr, or a log, because a refusal that echoed the string would defeat the layer
  at the moment it worked. The pattern reaches grep on stdin rather than in argv,
  for the same reason. Hostnames, IP and CIDR values, MAC addresses, and device
  names are ordinary entries, matched exactly as a name is, and so are binary
  blobs — a name in one is in history just the same. The store declares its own
  format on its first line: `# abcd-banlist: keyed` means every line is
  `KEY<space-or-tab>PATTERN`, and no declaration means every line is one whole-line
  pattern under a synthetic key, so an older store keeps matching exactly what it
  always matched and no part of any line is ever read — or printed — as a key. A
  line that does not parse, a pattern the engine refuses, and any git step that
  fails are each a refusal naming a step or a line number: an unusable entry is
  never skipped, and a check that could not run must never look like a check that
  passed. A store that is absent, or present with no entries, prints a loud warning
  that the layer is inactive on this machine and lets the commit through: it
  protects machines that opted in, and silence must never impersonate protection.
  It reads each staged blob stage-explicitly (so a file literally named
  `0:README.md` cannot hide behind git rev-magic), scans the staged PATH strings as
  well as content (a banned name in a filename enters history just the same), skips
  a staged gitlink rather than fail-closing on a submodule it cannot read, refuses
  to commit the private store itself, and announces the format and entry count it
  read before the scan so a stripped format declaration cannot silently downgrade it.
- **A citations family in `abcd docs lint`, with zero network in the gate**
  (itd-101, spc-17). Cited references rot silently — pages retitle, URLs
  redirect, whole platforms announce their own shutdown — but a gate that dials
  out to notice is a gate that flakes. So the checking splits in two. The lint
  side, landing here, reads only committed markdown, committed config, and a
  committed baseline: `citation_footnotes` holds a page's footnote markers and
  definitions in bijection (an unreferenced definition counts, being the way a
  reference page quietly stops meaning what it says); `citation_crosswalk_rows`
  requires every crosswalk table row to carry a footnote; `citation_url_syntax`
  checks cited URLs and DOIs are well-formed; and `citation_source_policy`
  refuses aggregator domains named in config — a list that ships empty, because
  naming one is a project's editorial policy and never something the gate
  invents on its behalf. A page's citations are its footnote definitions, not
  its prose, so ordinary body links stay `links_resolve`'s business.
  `citation_baseline` enforces the committed record offline — no cited URL
  without an entry, none recorded broken, none whose recorded final address has
  drifted from what the page cites, and a 180-day staleness warning. Each rule
  is opt-in per repo, and the whole family is inert until configured.
- **`abcd docs cite refresh` — the one verb that fetches, so the gate never has
  to** (itd-101, spc-17). It collects every cited URL through the same collector
  the lint uses, gives each exactly one bounded attempt, and rewrites the
  committed baseline. It never retries — a run's cost cannot become a function of
  how many links are failing, nor a burst against a struggling host — and it never
  reads a response body, because liveness is a property of the status line and a
  citation to a huge file should cost a response header, not a download. Sources
  that refuse automated fetchers (401, 403, 406, 429) are **not** recorded as
  broken: a refusal says the fetcher may not look, which is a different fact from
  the source being gone, so those URLs get no invented entry and are printed as a
  manual checklist naming exactly where each is cited. A current human-verified
  receipt is preserved verbatim and not even re-requested; once it ages past the
  staleness threshold it is re-checked like any other, because human and machine
  verifications share one clock. Receipts for addresses the documentation no
  longer cites are dropped.
- **`abcd docs cite confirm` — recording that a human cleared a queued link.**
  Name the URLs, or pass `--receipt` with a receipt file; both assemble the same
  schema, so the generated checklist page that lands later is a different producer
  of one input rather than a second pathway. Only URLs the documentation actually
  cites can be confirmed, and one bad line refuses the whole receipt rather than
  half-applying it. The receipt records **that** a human verified a citation and
  **when**, never how — the schema declares no such field and decoding rejects
  unknown keys outright.
- **The citation baseline surfaces where maintainers already look.** `abcd ahoy`
  reports its coverage and age on one line in any repo that has armed the rule;
  `abcd launch --dry-run` gains a `citation-baseline` gate that names entries
  approaching the staleness blocker while still letting a release cut, and refuses
  on ones that are overdue, broken, or unreceipted. `abcd docs lint
  --release-gate` promotes an overdue citation from a warning to a blocker — the
  flag is the trust root, so a repository cannot defang its own release by editing
  a committed config, and an ordinary commit is never blocked by the calendar.
  The flag is built and tested but **not yet passed by `release.yml`**, which
  still runs the plain `abcd docs lint`; wiring it into the release workflow is a
  CI change needing its own sign-off, and until it lands the 365-day threshold
  warns at release time rather than blocking.
- **The citation gate is armed for this repository.** 50 of the 51 URLs cited
  under `docs/` carry a receipt, and four citations that had silently drifted
  behind redirects now name the address they actually resolve to — rot nobody
  would have caught by reading. The fifty-first answers HTTP 403 to every
  automated fetcher and is waiting in the manual queue, so `abcd docs lint`
  reports it as unreceipted until a maintainer opens it and runs `abcd docs cite
  confirm`. That report is the mechanism working: no receipt is ever written on
  a human's behalf.
- **A schema-versioned citation baseline at `.abcd/citations-baseline.json`.**
  Per cited URL it records the final resolved address, when it was last checked,
  the outcome, and whether verification was automatic or manual with its date.
  It records nothing about *how* a human verified, and cannot be made to: the
  schema declares no such field, and loading rejects unknown keys outright, so a
  hand-added `method` or transcript is a refusal rather than a quietly-kept
  note. Manual entries age on the same clock as automatic ones, so a human
  confirmation buys no exemption from going stale.
- **The hazard registry refuses destructive GitHub remote operations and a force
  push spelled as a refspec** (iss-159, iss-148). Three shapes that verdicted
  `allow` now block. `gh repo delete` destroys the remote copy of a repository
  and everything kept with it — issues, pull requests, releases, review history —
  and nothing an agent can run brings any of it back, so the refusal names the
  human-only successor: tell the person who owns the repository, and archive it
  (`gh repo archive owner/repo`) where retiring it rather than destroying it was
  what was meant. `gh api -X DELETE repos/{owner}/{repo}` is the same deletion
  written as a raw API call, with no confirmation prompt anywhere in the way; it
  is depth-limited on purpose, so a `DELETE` deeper inside a repository — a
  branch ref, a release — is ordinary work and stays allowed. The path operand is
  normalised before that depth check — a `scheme://host` prefix, a query, and a
  fragment are dropped — so the same call written as
  `https://api.github.com/repos/owner/repo` is the same refusal; what remains
  stated rather than closed is a host that serves the API under a mount prefix
  (a GitHub Enterprise Server `/api/v3/…`), because matching a root segment
  wherever it appeared would falsely refuse
  `DELETE /teams/{id}/repos/{owner}/{repo}`, which removes a repository from a
  team and destroys nothing. And
  `git push origin +main:main` overwrites the remote branch exactly as `--force`
  does, without the flag anyone reviews for. Entry patterns gained the fields
  those shapes need, each optional and overridable per repo like the rest: a
  second-level subcommand (`gh repo list` and `gh repo delete` share their first
  level and only one of them is a hazard), a flag's SETTING rather than its
  presence (`-X DELETE` refused, `-X GET` allowed, all three spellings read and
  the case of the value ignored), an operand's path root and exact depth, and an
  operand prefix. Every new entry ships through the same admission gate as the
  rest: known-bad and known-good fixtures in the registry file itself, a 100%
  true-negative floor, and known-good at least 40% of its own corpus.
- **The plugin provisions its own binary, so a fresh install and every update
  yield a working hook surface** (itd-105, spc-21). The hooks call
  `$CLAUDE_PLUGIN_ROOT/abcd`, the plugin root is a clone of a repository that
  commits no binary, and the harness re-clones each update into a fresh
  commit-stamped cache directory — so every install and every update produced a
  plugin root with nothing to call, and each hook failed with a raw "No such
  file or directory" until someone hand-copied a binary in. `hooks/bootstrap.sh`
  closes that: committed POSIX sh, needing no abcd binary to run (the binary is
  exactly what is missing), wired as the FIRST `SessionStart` hook so it lands
  before the binary-backed ones in the same event. It downloads the latest
  release binary for the host platform plus `checksums.txt`, verifies the
  SHA-256 against the manifest, and installs it with an atomic rename from a
  temp directory on the same filesystem — a mismatch or an absent manifest line
  deletes the download and refuses, so a corrupted or unpublished artefact never
  reaches the binary path. The trust bar is the README one-liner's: same-origin
  checksums, with `go build ./cmd/abcd` documented as the full-trust route. The
  script fetches from two hardcoded origins and offers no way — no environment
  variable, no flag, no branch — to point it anywhere else, and it pins HTTPS
  including redirects on every request unconditionally: the binary and the
  `checksums.txt` that verifies it come from one origin, so anything able to
  name that origin would supply both the payload and its own manifest, and what
  is installed then runs unattended as the Bash shell guard on every tool call.
  Pinning the transport is not enough on its own, because `curl` reads a
  configuration surface the command line knows nothing about: every fetch
  therefore passes `-q` as its first argument, so `$CURL_HOME/.curlrc` and
  `$HOME/.curlrc` are never loaded — one `connect-to` or `resolve` line there
  re-points the connection while the URL still reads `https://github.com/…`, and
  the checksum then verifies the substitute against its own manifest. The names
  `curl` reads without being told to are removed from the environment before the
  first request for the same reason: the proxy variables (`HTTPS_PROXY`,
  `ALL_PROXY` and their lowercase forms, plus `HTTP_PROXY`/`http_proxy`) and the
  certificate-authority overrides (`CURL_CA_BUNDLE`, `SSL_CERT_FILE`,
  `SSL_CERT_DIR`) that would make such a route succeed on TLS. **The accepted
  cost: a machine that can only reach the network through a proxy does not
  bootstrap automatically** — install the release binary by hand or build from
  source with `go build ./cmd/abcd`, both of which the refusal message names. A
  plugin root that already holds an executable binary exits on one file test
  with no network, so steady-state sessions pay nothing; concurrent sessions
  serialise on an atomic `mkdir` lock whose loser exits quietly and whose stale
  remains (older than ten minutes, from a killed run) are broken and retaken. A
  lock that cannot be taken and is not there afterwards is not a lost race —
  the plugin root is unwritable, or something that is not a directory occupies
  the lock path — and that says so out loud instead of sharing the loser's
  silence, because it repeats every session and nothing else would report it. A
  platform outside the released matrix — darwin and linux on amd64 and arm64 —
  is reported, not retried: it states the matrix and changes nothing.
  Every failing path prints one plain-language message naming what is missing,
  what it costs (hooks cannot run, the shell-hazard guard is inactive and
  commands run UNGUARDED), and the three ways out; `abcd ahoy`'s guard health
  names the same script and its recovery step, so one fault has one story
  wherever it is read. Every message the bootstrap has for a person — the
  unsupported-platform statement and the one-time `abcd ahoy install`
  suggestion included — goes to stderr with a non-zero (non-blocking) exit,
  because a SessionStart hook's stdout becomes model context while only a
  non-zero exit puts its stderr in front of the human who can act on it. The
  two binary-backed `SessionStart` commands that follow the bootstrap check for
  the binary first, so a session that begins before one exists reports the gap
  in the same plain language instead of a raw "No such file or directory".
  Every message strips control characters from the values it echoes — a
  plugin-root path, `uname`'s answer, the release tag read off a redirect — so
  an escape sequence in one of them cannot recolour or visually rewrite a
  message whose whole job is to be believed.
- **A binary at a different commit from the surface it serves says so at session
  start** (itd-105, spc-21). The plugin surface tracks the repository tip while
  the newest binary is the last tagged release, so a fix can merge without a
  release cut and leave a session running old-binary logic against new-surface
  expectations. The bootstrap records what it installed in a `.binary-meta` file
  beside the binary — release tag, release commit, the SHA-256 it verified the
  binary against, fetch time, and the plugin commit it provisioned for, written
  by the same atomic rename the binary gets — and the session-start hook renders
  one line when the plugin commit and the release commit differ. The line names
  both directions the difference could run in and asserts neither, because
  comparing two commits establishes that they differ and never which is ahead.
  What the bootstrap could not resolve it records as `unknown`, and a commit
  that is not forty lowercase hex characters — unresolved, or truncated by a
  crash mid-write — produces no line at all: a skew notice that guesses is worse
  than no notice. The plugin commit comes from the cache directory's name, which
  is an assumption about the harness this repository cannot verify, so the raw
  basename is recorded beside the gated value: if that naming ever changes, the
  notice goes quiet for everyone and `.binary-meta` is the one place that says
  why. The release tag is sanitised before it is rendered, being the only value
  in the line that arrives unchecked from off the machine.

### Changed

- **`abcd ideate record` files its verdict record where dated research notes
  live.** The record it writes is a dated research note by spc-18's own
  reasoning, and `research/README.md` files those under `notes/`; it was landing
  in `research/` root, so the verb was a standing producer of a convention
  violation the record then had to clean up by hand. It now writes
  `.abcd/development/research/notes/<date>-ideate-<slug>.md`. The path is still a
  constant, not a parameter — a configurable target would let a verdict land
  somewhere no future session looks.
- **The intent tree's schema rules stop honouring the `superseded/` lint
  exemption** (iss-39). Being historical excuses a record from how it is
  WRITTEN — the banned tokens, the persona roster — never from being
  well-formed. The exemption used to reach the intent-lifecycle schema too, so
  the `superseded/` bucket excused its own records from the one rule that
  checks a supersession is recorded: two intents sat there carrying a prose
  note and no `superseded_by` field at all, invisible for as long as the
  exemption held. Both rules that read the intent tree — `intent_lifecycle` and
  `intent_impact_valid`, which share one scan of it — now run everywhere,
  `superseded/` included; if you arm `intent_impact_valid` with `exempt_paths`,
  an exempt record carrying an illegal `impact:` value is a finding on upgrade
  where it was silent before. The spec-store rules (`spec_lifecycle`,
  `spec_id_unique`) still honour the exemption as before — widening those is
  separate work. The new `record_schema` is cross-store and never consulted the
  exemption at all.

### Fixed

- **The README layout line says where the development record travels**
  (iss-43). `.abcd/` was described as "never shipped", which holds for the
  released binaries alone: a marketplace install takes the repository root, and
  every release carries an auto-generated source archive, so both hold the
  directory. The line divides on the real boundary — any repository checkout
  against the released binaries — and the section on marketplace installs names
  those source archives alongside the binaries and checksums a release
  publishes.
- **The installed plugin surface is the surface the documentation describes**
  (iss-44, iss-160, iss-161, iss-162). The verb files lived in
  `commands/abcd/`, a subdirectory named after the plugin itself, and a harness
  maps each `commands/` subdirectory to a namespace segment — so every verb
  registered as `/abcd:abcd:<verb>`, the plugin name twice, and each documented
  `/abcd:<verb>` was an unknown command. The verb files move flat into
  `commands/`, which also makes the surface's own next-step guidance runnable:
  it already spelled `/abcd:<verb>` throughout, so it was correct prose about a
  name nothing registered. `commands/README.md` shipped as a spurious
  `/abcd:README` because the loader reads every markdown file under `commands/`
  as a command, requiring no frontmatter and exempting no name; the directory's
  documentation moves out of the auto-discovery root into the brief's surface
  registry, which already enumerates the same surface. `/abcd:ahoy` reaches the
  sub-verbs the binary registers — `install`, `uninstall`, `doctor` and
  `dry-run` each have a section, and `identity-check` an explicit note saying
  why it stays CLI-only — where the command file previously drove only the bare
  read-only detect. Every "the `abcd` binary is not on `PATH`" remedy named
  `make build`, which cross-compiles `bin/abcd-<goos>-<arch>` and never a
  PATH-resolvable `abcd`; each now names the `go run ./cmd/abcd …` fallback the
  same paragraphs already carried. And `abcd memory ask` run from a plain
  terminal no longer heads its output with a plugin slash command: the renders
  live in the transport-agnostic core, so they name the binary invocation and
  stay true whichever front door produced them. A surface-parity test holds all
  of it — the command surface is one flat level of markdown commands and
  nothing else, every binary sub-verb is reachable from its command file or
  carries a scoping note, and no remedy offers a build that cannot put `abcd`
  on `PATH`.
- **The developer docs describe the gate suite that actually runs** (iss-37).
  `AGENTS.md`, `README.md` and `CONTRIBUTING.md` name the three lint gates
  `make preflight` runs (`lint-reviews`, `record-lint`, `docs-lint`) alongside
  build, vet, test and the race-enabled internal tests; `AGENTS.md` and
  `CONTRIBUTING.md` attribute the format gate to CI's own `gofmt` step rather
  than to preflight; and the `ci.yml` step comment states that the record-lint
  step blocks.
- **The record can say what has been delivered, and the brief's later-phase list
  is derived again** (iss-41). Two entries credited an intent that never left the
  uncommitted bench: the v0.1.0 `abcd docs lint` entry cited `itd-60`, whose
  deterministic layer shipped while its semantic layer is still five open
  questions, and the v0.2.0 `abcd audit` entry cited `itd-85`, which shipped
  whole and was never promoted. Both entries now describe the delivered
  capability without the citation, and `intents/README.md` states the rule the
  new `delivery_state` gate holds: leaving `drafts/` is what represents delivery,
  and nothing else does. `itd-85`'s missing promotion is `iss-180` — it is
  blocked on the `shipped/` schema, which wants a spec id the audit verb has
  never had. The brief's canonical later-phase list, which claimed to be derived
  from `drafts/` and not hand-counted, had drifted nineteen entries: three intents
  it still listed had moved to `planned/`, `shipped/`, and `superseded/`, and
  sixteen captures since the last hand edit were missing. It is regenerated and
  now sits inside an `index_drift` marked region, so the claim is enforced rather
  than repeated. The four later-phase items that never had an intent id move out
  of the derived list, which could not have contained them.
- **One canonical glossary, and its index is derived** (iss-40). The record held
  two artefacts each calling itself the glossary: `.abcd/development/brief/glossary/`,
  a directory of per-term files with the frontmatter the `GL002` forbidden-synonym
  rule reads, and `02-constraints/04-naming.md`, whose vocabulary-registration
  section declared that terms are registered in *it*. Nothing said which one won
  for a given term, so "the glossary" resolved to two places. The naming file now
  states the split it actually implements — the glossary directory holds
  cross-cutting prose vocabulary, that file's own tables hold the naming
  convention, controlled enums, and spec-pinned reserved names — and stops
  claiming to be a glossary; its ~60 reserved-vocabulary rows are untouched,
  because a closed enum is not a glossary term. The `VR001` descriptions that
  carried the old framing across the brief and itd-37 are corrected with it.
  The glossary's own index was stale in the way a hand-kept index always is: a
  whole bounded context (`distribution/`, three terms) existed on disk and
  appeared in neither the context list, the directory tree, nor the term index.
  Both blocks are now RENDERED from the term files by `internal/core/glossary`
  and held to them by a drift test, so a term file that lands without an index
  row fails the build — the same generate-and-gate shape the brief↔lifeboat
  mapping table already uses. The README also promised a JSON schema
  (`internal/core/schema/terminology.schema.json`), a CLI verb
  (`abcd lint terminology`), and a config allowlist key
  (`terminology_exclude_files`), none of which exist; it now describes what does
  — the frontmatter tables as the shape's own specification, `GL002` through
  the record-lint gate, and the index drift gate — and says plainly that neither is a
  full schema validator. `glossary/core/brief.md` no longer calls the brief
  "immutable once approved", which contradicted adr-5's living-record decision,
  and the brief's own navigation table and directory tree now list `glossary/`,
  which the generated brief↔lifeboat mapping had been alone in knowing about. A
  public banlist entry (`names/record/glossary-self-identification`) blocks the
  retired self-identifying phrase from returning to the published surface.
- **The plugin root is resolvable when abcd runs through the PATH symlink
  `ahoy install` itself pins** (iss-170). Resolution falls back to walking the
  executable's ancestors for a `hooks/` directory, but the executable path was
  taken as invoked rather than canonicalised — and the installed layout puts a
  symlink on PATH (`/usr/local/bin/abcd` -> `<plugin-root>/abcd`), so the walk
  climbed `/usr/local/bin`, `/usr/local`, `/usr` and never reached the plugin
  root (the walk's own guard excludes `/`, so it was never a candidate either).
  The path is now resolved through its symlinks before the walk — falling back
  to an absolutised form of the original path, then the original path itself,
  only if resolution errors, which `os.Executable` never does on a shipped
  platform, so no case that worked before can regress.
  The gap was invisible inside the harness, where `CLAUDE_PLUGIN_ROOT` is set and
  answers first; it appeared in a plain shell, on exactly the layout the
  installer writes. Linux is unaffected — `os.Executable` already resolves
  symlinks there; the unresolved path this fix accounts for is macOS's.
- **Two frontmatter-parser defects that silently dropped a memory page's
  provenance** (iss-30). Both were found by the detectors written for that
  issue's last acceptance instances, and both broke the same invariant the parser
  documents at the top of `internal/core/memory/yaml.go`: what the writer emits,
  the reader gives back. (1) A block scalar opened with `key: |` and left without
  an INDENTED body took the next unindented line as its content and went on
  consuming the rest of the document — every later key, `source` and its hashes
  among them, vanished into that value with no parse error. The block's body must
  be indented deeper than the line that opened it; an unindented line now ends an
  EMPTY block and is read as the top-level key it is. (2) The dumper decided a
  string needed quoting on `\n` and `\t` but not `\r`, so a bare carriage return
  went into the YAML region raw — and since the reader normalises `\r` to `\n`
  before splitting lines, a value carrying `\r---` re-read as an early
  frontmatter terminator, pushing the keys below it into the page body. A page
  written that way reported a successful ingest while reading back with no source
  hashes at all, which is the state the reconcile/repair path treats as an
  orphan. `\r` now triggers the same quoting as `\n` and `\t` (the escape side
  already emitted `\r` correctly). Defect (2) was reachable from distiller
  output alone, so a page's provenance could be stripped without any
  hand-editing; defect (1) needs a hand-authored or externally-written page
  (a bare `key: |` cannot survive the dumper's own quoting of `|`).
- **`ahoy install` writes a `.gitignore` block that matches the tier layout it
  documents** (iss-169). The managed block ignored a root-level `.work/` under
  both visibilities — a path the three-tier layout does not have — while never
  naming `.abcd/.work.local/`, the one tier that must be gitignored. So a fresh
  install ignored nothing that existed and left the local-ephemeral tier tracked:
  precisely the state `abcd audit`'s `three-tier-layout` rule then reports as an
  error, the installer and the auditor disagreeing about the same convention. The
  block now follows the brief's visibility table exactly — a private repo ignores
  `.abcd/.work.local/` alone, because the rest of the `.abcd/` namespace is meant
  to be committed; a public repo ignores `.abcd/` outright, one switch with no
  per-subdirectory exceptions, plus the legacy root-level `memory/` snapshot. An
  existing repo carrying the old block needs no migration step: the block reads
  as drift, `abcd ahoy` reports it, and one apply replaces it, leaving the
  repository's own ignore rules untouched. A repository still on the historical
  root-level `.work/` layout stops ignoring `.work/` when the block refreshes;
  the layout migration in `/abcd:prepare-this-repo` is the path off that state.
- **The `PII` rules domain fires on network work, and says never to commit a
  hostname or an address** (iss-156). Its recall keywords were the vocabulary of
  credentials — secret, token, credential, pii, redact, hostname, email — so a
  session investigating a mesh VPN's reachability, a firewall rule, or an
  address in a network config matched nothing and the hook injected nothing.
  The keywords now cover that context — `ip`, `ips`, `ipv4`, `ipv6`, `vpn`,
  `tailscale`, `tailnet`, `wireguard`, `firewall`, `network`, `reachability`,
  `reachable`, `unreachable`, `dns`, `ssh`, `subnet`, plus the `mac address`
  phrase alias — and
  one new rule line forbids committing hostnames, IP or MAC addresses, and other
  live network identifiers: redact or omit them, and reach for a reserved
  documentation value (RFC 5737, RFC 3849, RFC 2606, RFC 7042 — the same set the
  audit privacy-hygiene rule cites) only where an illustrative
  example is actually needed. That rule previously existed only in a repo's own
  instructions file, which meant the one discipline this domain most needed to
  state was the one it never said — an agent could recall `PII` and still read
  nothing about the machine identifiers in front of it. Recall matching is
  word-bounded already, so the bare `ip` keyword matches a standalone token and
  never the middle of a word such as "script" or "zip"; the plural, the
  version-qualified forms and the `-able` adjective need their own entries
  because the stemmer has a three-character floor and does not bridge
  `-ability` to `-able`. `mac` is an alias phrase rather than a bare keyword so
  that an Apple Mac does not recall the domain.
- **`abcd audit` flags local-tier artefacts sitting in a committed tier**
  (iss-155). The `three-tier-layout` rule verified that the committed tiers
  exist and that `.abcd/.work.local/` is gitignored, but never the reverse
  containment: a `NEXT.md`, `scratch/` or `logs/` placed directly in
  `.abcd/work/` or `.abcd/development/` passed clean — which is exactly how a
  handover file carrying machine-local detail rides a committed tier into
  public history. Those three names are the local-ephemeral tier's
  conventional contents, so their presence in a committed tier is now an
  error, one finding per misplacement, each naming the offending path and
  fixing it with a move to `.abcd/.work.local/`. Presence is checked on the
  filesystem, like the tiers themselves: an untracked `NEXT.md` in
  `.abcd/work/` is one `git add -A` from being committed, and the audit
  should say so before that happens, not after.
- **Network identifiers are detected, redacted and audited — one pattern set,
  three surfaces** (iss-154, iss-157, iss-125, iss-153). The scanner carried
  token-shaped secrets and identity matchers but nothing for addresses or
  hostnames, so a lifeboat pack or launch bundle carrying a tailnet address and
  device names shipped clean, a stored transcript kept a LAN hostname verbatim,
  and `abcd audit` passed the whole class in silence. The detector is an
  allowlist inversion rather than a leak-recognition heuristic: because every
  illustrative identifier in a committed file comes from a reserved range, the
  small closed set of values that are ALLOWED is what a pattern can recognise,
  and everything outside it is a finding. Allowed are the documentation ranges
  (RFC 5737, RFC 3849, RFC 2606/6761, RFC 7042), device names derived from the
  persona registry, and the values that name no individual host at all —
  loopback, unspecified, netmasks, masked CIDR prefixes, and the IANA
  special-use ranges for link-local, multicast, benchmarking, protocol
  assignments and the NAT64 well-known prefix. Flagged is what identifies
  private topology: private ranges, CGNAT/tailnet, IPv6 unique-local, 6to4,
  public unicast, LAN suffixes (`.local`, `.lan`, `.fritz.box`) and
  device-hostname shapes. The set is built once and folded into the scanner's
  defaults, so Stage-1 redaction and the launch/lifeboat scan inherit it and the
  audit rule consults exactly the same patterns — the surfaces cannot disagree
  about what a leak is. Addresses are hard_fail; the two hostname shapes are
  warn, and the audit surface carries that split through rather than flattening
  it. A line carrying `abcd-audit:allow` stays exempt, so a deliberately
  illustrative value needs no weakening of the patterns. An identifier is masked
  WHOLE — to a readable placeholder in redacted text, and fully starred in a
  serialised finding — never to the head-and-tail fingerprint a credential gets:
  on a MAC or a hostname that fingerprint preserves the vendor bytes and the
  head, which is enough to re-identify the machine the masking was meant to hide.
  The two hostname shapes tell a host from source code, because Stage-1
  redaction rewrites every finding and a false positive there corrupts a stored
  transcript: a mixed-case suffix where code puts a value (`zone := time.Local`),
  a selector position (an assignment target, a block head, a call) and a
  determiner before a device noun ("a synology-nas") are all read as code or
  prose rather than machines. Mixed case on its own exempts nothing — a shifted
  key in a command or a URL would otherwise be a bypass anyone could type — and a
  fixture host must be spelled the way the persona registry spells it, in lower
  case, because a capitalised given name in front of a device noun is how macOS
  names a real person's machine. A digest is told from an address by the length
  of the colon-separated run around it, counting only the groups that carry hex:
  an address is eight of them and may sit beside one more (the port a tool
  prints), while the shortest fingerprint is sixteen. A
  repo that wants the hostname shapes to block raises their severity in
  `.abcd/config/pii.json`, and `abcd audit` reads that merged set, so the
  override reaches the surface that reports it. The transcript store's
  verification rescan refuses the write on ANY surviving identifier, not only a
  hard_fail one, so a warn-severity hostname cannot reach disk in silence, and
  the refusal it prints names the surviving kinds rather than a severity it does
  not gate on.
- **`/Users/Shared` and `/Users/Guest` stop reading as usernames** (iss-153).
  privacy-hygiene flagged every segment under a `/Users` root, so the macOS
  system directories that live there were reported as though they named a
  person, taxing product code that legitimately writes to one. The exemption is
  narrow: it is scoped to the `/Users` root, it covers the system directory
  itself and not what sits beneath it (a name nested under one — `Shared/<user>`
  — is still a user, and still flags), a segment that merely begins with a
  system-directory name is still a leak, and an empty or dots-only segment in
  between (`Shared/../<user>`) restores no shield. The audit rule holds those
  terms for both spellings it matches, the POSIX one and the Windows one — and
  because Windows accepts either separator within one path, a name written after
  a forward slash inside a Windows path is read as the nested name it is; the
  scanner's own identity matcher shares the allowlist for the POSIX paths it
  recognises, which are the only ones it has ever matched.
- **A wrapper carrying its own flags no longer defangs the hazard entry behind
  it** (iss-148). Of the
  matcher's gaps this was the one a facilitator would reasonably assume was
  covered: only the wrapper NAME was stepped over, so `sudo <hazard>` was seen
  and `sudo -u bob <hazard>` was not — `-u` was read as the command name, and one
  extra token turned an entry the registry does describe into an allow. The same
  held for `env -i` and `time -p`. A wrapper's own arguments are now stepped over
  with it, off two explicit tables: the flags each wrapper documents as taking a
  value, and the mandatory operand that no flag stepping can reach —
  `timeout [OPTIONS] DURATION COMMAND...`, where `timeout 30 rm -rf /` read as a
  command called `30`. `xargs`, `timeout` and `exec` join the wrapper set they
  were missing from. What stays unseen is stated rather than implied: a wrapper
  outside the known set, a value-taking flag the per-wrapper table does not
  name (a bundled `sudo -Hu bob`), where the miss is a non-match and never a
  false block, and a backtick command substitution, which stays a disclosed v1
  gap: the fourth part of iss-148 was attempted on this branch and reverted, and
  the issue stays open on it alone.
- **The guard no longer treats an arithmetic shift as a here-document, so a
  command after it is no longer silently unchecked** (iss-184). An unquoted
  arithmetic left shift with an identifier operand (`$((1<<shift))`) parsed
  as a heredoc delimiter; the tokenizer then scanned for a closing line and,
  finding one — even one an attacker supplies on purpose — consumed every
  line up to it as unchecked body text, so a later `git push --force` or
  `rm -rf` never reached command position and the guard returned an allow
  verdict with no error and no signal at all. The delimiter word is now
  recognised as never real when a bare `(` or `)` immediately follows it
  with no separator (the shape `$((expr<<ident))` always produces), so the
  arithmetic expression is read correctly and the rest of the command is
  checked normally regardless of what later lines contain. A genuinely
  unterminated heredoc (a real delimiter whose closing line never appears)
  still surfaces as an unparsable command rather than silently swallowing
  the rest of the input, the same error class an unterminated quote already
  uses.
- **The secret scanner now detects a fixed-length secret token that
  immediately abuts another, same-family or not, with no separator, instead
  of silently missing it** (iss-185). Every bundled secret pattern anchors
  its start on a leading `\b`; when two such tokens are concatenated with no
  separating byte, the byte just before the second token is itself a word
  character — the first token's own last byte — so that boundary can never
  hold and the second token was never matched at all. Because detection
  missed it, `Redact` left its whole body raw in the output, and the
  fail-closed residual re-scan that stage-two history capture relies on
  reported the redacted text clean anyway, letting a live secret reach disk.
  Fixed at the scan pass: immediately after each match, every pattern's
  `\A`-anchored, `\b`-stripped variant is tried (within a small bounded
  window, so one attempt can never cost more than a constant amount of work
  regardless of what follows it) at the exact byte offset where the match
  ended — anchored by the adjacency itself rather than by `\b`, so it cannot
  introduce a false match anywhere else in the line. A pattern whose
  quantifier is open-ended rather than fixed-length can still greedily
  consume into a following token before this recovery ever runs; that is a
  separate, broader gap, tracked as iss-188.
- **The secret scanner now also detects an abutting token whose leading bytes
  the preceding pattern's greedy quantifier had already swallowed** (iss-188).
  A pattern with an open-ended length bound consumes as many class-matching
  bytes as it can find, including the next token's own prefix when those bytes
  fall in the same character class — so the boundary between the two tokens
  sits before the reported end of the match, where the adjacency probe above
  never looks. Two concatenated GitHub PATs were reported as one over-long
  finding and the second token's tail survived redaction raw, with the
  fail-closed residual re-scan reporting the output clean, so a live
  credential still reached disk. Before a match's end is accepted as final,
  the scan now walks back over a bounded window looking for a cut where the
  shortened match is still valid for its own pattern and another token can
  begin, and probes there as well. Whether a pattern needs that search is
  decided from the pattern itself — one whose match cannot survive losing a
  byte has a rigid length and is skipped — so a pattern added later is covered
  without being annotated. The backward window, like the forward one, is a
  small fixed size and candidates come from one combined probe over the whole
  pattern set, so the work done per match never scales with what follows the
  match — a legitimately huge match (a base64 blob, a minified line) keeps the
  whole scan linear in the length of the line rather than quadratic. The
  recovery is bounded, not exhaustive: a recovered token longer than the probe
  window is still truncated to it (or, for a pattern whose required separators
  fall outside the window, missed), so an abutting token behind one of those is
  not yet covered — tracked as iss-190.

## [0.4.1] - 2026-07-28

### Added

- **`abcd identity` — one canonical identity block, and every surface held to
  it** (itd-102, spc-19). A project's positioning fragments silently: the README
  strapline, the package or plugin manifest description, and the conventions
  file's opening are edited at different moments until three surfaces tell three
  stories (the recorded iss-143 drift, which is this check's acceptance corpus).
  A repo now records one canonical markdown block — `- **Title:**`,
  `- **Tagline:**`, and an optional, wrappable `- **Pitch:**` under a recorded
  heading — and `.abcd/positioning.json` records only where that block lives, how
  loudly the family reports, and which surfaces render from it. Markdown stays
  the single source of truth. The registry is data, not branches: a surface names
  candidate files (the first present wins, so one entry covers several manifest
  formats), a locator (a capture-group regexp or a top-level JSON field), the
  block fields it must carry, and the template a proposal renders from; the three
  defaults are the README strapline, the manifest description, and the
  conventions opening, and a declared list replaces them so nothing is registered
  silently. A new `identity-positioning` rule runs the check on **every**
  `abcd audit`, naming the file, the exact drifted line, and the canonical line it
  should carry — warn-tier by default (it highlights, it never gates) and
  upgradeable per-repo to blocker. `abcd identity` renders the block and each
  surface's verdict; `abcd identity render` prints the proposed correction as a
  unified diff and **writes nothing** (autonomous rewriting is permanently out of
  scope — adopting a proposal is always the maintainer's move); `abcd identity
  init` records the block and the pointer at onboarding, adopting an existing
  block rather than re-interviewing over it. `/abcd:prepare-this-repo` gains the
  interview, detect-first. Every path the registry names is read and written
  inside an OS-enforced containment root, so an audited repo that commits a
  symlinked directory cannot make the check read a file the repository does not
  own and quote it into the report.

- **`/abcd:ideate` — the idea-admission gauntlet** (itd-104, spc-18). A big,
  unproven idea can be put through three legs before it becomes a record entry:
  primary-source research (each load-bearing claim checked against its **primary**
  source, never a secondary citation), a grill against the existing record, and an
  adversarial review that is fresh-context, off-policy, and receives the idea as an
  artefact of unknown authorship. The legs are host work; `abcd ideate record
  <idea-slug> --verdict-json <file|->` is the deterministic frame that validates
  them and writes the durable verdict — the dated record under
  `.abcd/development/research/` plus one pointer line in `.abcd/work/DECISIONS.md`.
  The verdict is recorded **whether the idea survives or dies**, and its rejected
  alternatives may be empty only behind an explicit marker, because silence and
  "nothing was weighed" read the same to a session tempted to re-propose the idea.
  Every record-grill hit is **cited by id and proved to resolve** in the
  repository: an id naming no record refuses the whole verdict and names the id.
  The legs travel as an ordered array so "three legs, in order" is checked rather
  than assumed, and refusals are whole-document — nothing is written unless
  everything validates. Ideate is **optional and never a gate**: the `intent` and
  `capture` routing help names it for big unproven ideas, and nothing requires it
  or warns when it is skipped.

- **`abcd guard` — the shell-hazard guard, wired into a live session** (itd-103,
  spc-16). `abcd guard check --command "<line>"` evaluates one candidate command
  against the hazard registry and reports allow, warn, or block: a blocker exits
  1 and answers with the plain-language why and the safe successor, a warn exits
  0 with the warning rendered, and a guard that cannot be evaluated at all exits
  2 rather than letting silence read as clearance. `abcd guard hook` is the host
  adapter: it reads a pre-tool-use hook payload, refuses a matching command with
  the successor as the block message, and lets everything else through. The
  installed hook entry wraps the binary call so a missing or broken abcd **fails
  open, loudly** — the command runs and the session carries an unmissable
  UNGUARDED warning, never a silent no-op and never a stuck session. `abcd ahoy`
  gains a `guard:` line reporting the three things that can independently be
  false — hook installed, binary reachable, registry loadable — plus a
  deliberately disabled registry, so a broken guard is visible from outside the
  session too. Every unguarded state is loud, including the deliberate one: a
  registry switched off in `.abcd/guard.json` makes each command it lets through
  carry the warning, so "off" can never pass for "clear". Per-repo overrides live
  in that one file and nowhere else — no flag, environment variable, or prompt
  disarms the guard for a session, so the change lands in a diff. An allow means
  no entry matched, never that a command is safe: a hazard reached another way —
  a string handed to an interpreter (`eval`, `sh -c`), a launcher the guard does
  not step over, a backtick substitution, or a form no entry describes — is not
  seen. Coverage is what the registry names, and the command reference says so.
  A registry that is switched off answers `abcd guard check` with a fault rather
  than a clearance, so a script using the verb as a gate cannot be waved through
  by an edit to `.abcd/guard.json`.
- **`abcd launch scaffold` — the changelog-driven release-gate scaffolder**
  (itd-93, spc-14). Writes the fixed release machinery into a managed repo that
  lacks it: `.github/workflows/release.yml` (verify → build → publish, the verify
  gate armed against the reviewed **content** commit so the first public release
  cannot hit the receipt-vs-tag self-reference), `.github/workflows/auto-release.yml`
  (newest dated CHANGELOG heading → tag that commit → call `release.yml`), and the
  adr-37 runbook — wired to the repo's own default branch and Go version,
  `GITHUB_TOKEN`-only and injection-safe. The workflows ship from a **single
  embedded template** that abcd-cli's own release workflows are regenerated from
  (self-scaffold parity, proven by a byte-exact test), so every abcd release
  exercises the exact machinery a managed repo receives. The scaffolded
  `release.yml` carries a `workflow_dispatch` **rehearsal** that arms the full gate
  against a simulated changelog roll and reviewed-content commit and publishes
  nothing — a green rehearsal is the runbook precondition for the first real
  release. A bare repo with no semantic detector degrades cleanly to the
  deterministic gates and a generic build. The verb is idempotent and fail-safe: a
  re-run on current machinery is a no-op, a hand-edited file is refused rather than
  clobbered (unless `--confirm`), and it refuses rather than half-writing.
- **A public terminology crosswalk at `docs/reference/terminology.md`** (itd-100).
  One reference page maps 26 established agentic-AI terms — protocols, the core
  loop, context, safety, governance, operations — to abcd's position on each:
  USES (naming the native verb or principle), ADAPTS (the sharper native name and
  why), REJECTS (with the recorded reason), or WATCHING (with the record id).
  Every established definition carries a footnote citation to a primary source
  (specification, standards body, DOI-bearing paper, or origin engineering doc);
  every abcd claim is grounded in the committed record. Vendor names appear only
  inside citation footnotes, keeping the page body host-agnostic.

- **`abcd ahoy install --dev` — a track-latest dogfood install mode** (iss-75).
  Normal `ahoy install` symlinks the pinned built binary, so tracking live
  development meant hand-rolling a `~/.local/bin/abcd` wrapper that ran
  `go build -C <repo> && exec` on every call. That manual workaround now dies:
  `--dev` installs a shim at the same `PATH` target that rebuilds abcd from the
  source tip on every invocation and execs the fresh binary. A broken build fails
  loudly and never execs a stale binary (loud-staging). `abcd ahoy` status reports
  the mode as `install: dev (tip build)`, detected from the installed shim itself
  (never recorded in the tracked repo config, so it can never go stale), so a dev
  install is never invisible. Installing over an existing
  install applies-as-update in either direction — `--dev` replaces the pinned
  symlink with the shim, a plain re-install restores the symlink — and a foreign
  occupant is still never clobbered.
- **Record-id minting now sees every branch, and a spec-id uniqueness lint closes
  the class** (iss-115, iss-120). Sequential ids (`iss-N`, `itd-N`, `spc-N`) were
  minted from the local working tree only, so two branches cut from the same base
  silently minted the same next id — invisible on each branch and surfacing only
  at merge. Minting now folds in the highest id committed on every local and
  remote-tracking branch (a single canonical refs-union scan), so once one branch
  commits an id, the other mints past it. When git cannot be read over a present
  repository the mint degrades to working-tree-only and says so loudly on stderr
  (never a silent fallback); a directory that is not a repository has no branches
  to collide with and mints quietly. The residual window — two branches that both
  mint before either commits — is caught by the record-lint uniqueness rules on
  the merged pull request, which now cover spec ids too: the new `spec_id_unique`
  rule flags every file claiming a duplicate `spc-N`, mirroring the existing
  `issue_id_unique` and intent-id guards.
- **`abcd capture resolve` and `abcd intent "<text>"` can now stamp a product
  `impact`** (iss-117). A resolved issue and a shipped intent are in the release
  set, so the `issue_impact_valid` and `intent_impact_valid` record-lint blockers
  require a valid `impact` on those records — but the verbs that mint them had no
  way to set one, so the tool's own path produced records its own gates rejected.
  `capture resolve` now takes a mandatory `--impact <additive|breaking|fix|internal>`
  (there is no default: an absent or misspelled value is refused, not guessed),
  and `abcd intent "<text>"` takes an optional `--impact <additive|breaking|fix>`
  that is stamped onto the seeded draft and travels unchanged through planning to
  `shipped/`. `internal` is rejected on an intent (a press-release-first intent is
  user-facing by definition). `capture wontfix` is unchanged — a non-action ships
  nothing, so `wontfix/` carries no impact.

### Fixed

- **Untrusted prose can no longer open raw HTML in a record abcd writes.** The
  shared cleaner every host-delegated ingest routes through — the release
  changelog ingest, the lifeboat synthesis writers, and the ideate verdict —
  neutralised newlines and HTML comment markers but left a bare tag intact. In
  CommonMark a `<` followed by a letter, `/`, `!`, or `?` opens an HTML block, and
  several of those block types run to the end of the document: a single
  `<script>` in one model-supplied field made every later section of the record
  render as inert text inside an unclosed element, while a forged section above it
  rendered normally — so an artefact whose whole value is that a later session
  trusts it could hide its own evidence. Every tag opener is now broken apart in
  the one canonical primitive, so the fix lands at all three boundaries at once.
  The neutralisation runs **after** terminal sanitisation, which closes a second
  hole: sanitising substitutes `?` for a masked rune, so `<` before an escape byte
  previously became `<?` — a processing instruction — after the old ordering.
- **The disembark probe's recursive file walk is bounded per directory, opens
  each child in O(1), and skips the common ecosystems' dependency trees**
  (iss-112, iss-114, iss-116). The walk now reads every directory with a bounded
  `ReadDir` (the same 50 000-entry guard `ListDir` uses), so a single directory
  of millions of entries can no longer balloon memory before the file cap
  applies. It holds a sub-root per directory (`os.Root.OpenRoot`) instead of
  re-resolving every path from the containment root one component at a time, so a
  deep tree costs O(entries) rather than O(entries × depth) — a 48 000-directory
  depth-30 tree walks in ~1.4 s where the old walk took ~7 s, and the cost is now
  independent of depth. The `os.Root` containment guarantee is unchanged: a
  symlink is still refused rather than followed out of the tree. The skip set
  widens beyond Node and Go to the common dependency, cache, and build-output
  trees — Python (`.venv`, `venv`, `.tox`, `__pycache__`), Rust and generic
  build output (`target`, `build`, `dist`), and CocoaPods (`Pods`) — so a
  vendored `TODO` is no longer cited as the project's own open question, and a
  large dot-prefixed dependency tree can no longer exhaust the walk cap before
  the project's own `src/` is reached.
- **The open-questions marker scan no longer reads documentation about markers as
  open questions** (iss-111). The pattern that grounds `evidence/open-questions`
  admitted a bare uppercase `TODO`/`FIXME` followed by whitespace, so on a
  repository that documents its own conventions every prose mention of a marker
  was cited as a work marker — 14 such false positives across the durable record
  (`.abcd/development/`) on this repository, all documentation, none a real
  marker, down to 3 after the fix (each an irreducible prose quotation of the
  literal `TODO:` form). `TODO` and `FIXME`
  now require a trailing `:` or `(` (the conventional `TODO:` / `TODO(alice):`
  spellings), which is how genuine markers are almost always written; `XXX`,
  `HACK`, and `BUG` still admit their conventional bare form, because they are
  rarely written as bare words in prose and carry no measured false-positive
  cost.
- **Concurrent runs can no longer drop a repo registration or delete a
  just-committed issue file** (iss-101, iss-102). Two `abcd ahoy install` runs
  from different worktrees shared one `~/.abcd/history/index.json`, and its
  registration was an unlocked load-modify-write: atomic rename kept the file
  intact but the last writer clobbered the other's update, silently erasing a
  repo entry or a re-founding lineage link. The history registry now serializes
  its load-modify-write behind an inter-process lock and re-loads inside it, so
  concurrent registrations compose instead of overwriting; the store bootstrap
  creates `index.json` with an exclusive create, so exactly one racing run seeds
  it. The re-founding lineage confirmation is still asked before the lock is
  taken — never across an interactive prompt — and the state it validated is
  re-checked under the lock, surfacing a conflict rather than writing a link the
  user approved against a stale index. Separately, the capture ledger's orphan
  sweep and its commit write now take the same ledger lock, closing a window in
  which a capture stalled more than sixty seconds could have its committed issue
  file swept away after the capture reported success. The inter-process lock is a
  single shared primitive; the capture allocator and the history registry both
  route through it.

## [0.4.0] - 2026-07-22

### Breaking

- **The record-lint banlist now requires a machine-readable successor and
  context.** Every `banned_tokens` entry must declare a non-empty `successor`
  (what to use instead of the retired token) and a non-empty `allow_context`
  (where the token is legitimately allowed); a config whose entry omits either
  is rejected at load rather than lints with a replacement that lived only in
  prose. Each finding auto-cites its successor, so the reader is told the
  replacement inline. The bundled record-lint and docs-lint banlists carry the
  new fields.

### Added

- **Generated CLI reference with a drift gate** (iss-47). `docs/reference/cli/`
  now holds `commands.md`, a per-command Markdown reference walked deterministically
  from the Cobra command tree (`cli.GenerateReference`) — usage line, summary, and
  flags for every user-facing verb, with the operator-internal hook subtree omitted.
  Refresh it with `go generate ./internal/surface/cli`; a `go test` drift gate
  regenerates the tree and fails the build whenever the committed page and the tree
  disagree, so the reference can never silently go stale. The walker is hand-rolled
  over stdlib and the existing Cobra dependency — no new module dependency.
- **`agent-observation` is now a valid `--source` value for `abcd capture`.**
  An autonomous run's self-observation had no honest surfacing channel and was
  reusing `agent-finding`. `agent-observation` parallels it ("an agent
  observed") without being tied to one run mode (iss-57).

- **`issue_id_unique` record-lint rule — a duplicate issue id is a blocker.**
  The lint pass now scans the issue ledger's three status directories
  (`open/`, `resolved/`, `wontfix/`) and flags any `iss-N` id claimed by two or
  more files, mirroring the existing intent-id uniqueness check (both share one
  `validateIDUnique` primitive). The capture allocator already rejects a
  duplicate on the reservation path, but a hand-added issue file that bypassed
  it slipped straight through until now; this is the backstop that catches it.
  Every file in a colliding set is flagged, since the linter cannot know which
  claimant is authoritative.
- **`abcd intent ready <itd-N>` — the implement-readiness gate.** A read-only
  verb reporting whether an intent may be implemented now: planned
  (directory-as-truth), enumerable Acceptance Criteria, a bidirectional spec
  link, and a spec body written past its minted stub. Every check carries a
  reason and, when failing, the exact remedy command. The exit code is the
  machine seam: 0 ready, 1 not ready (the rendered report is the output), 2
  structural fault — so an autonomous run can gate on it (step 0 of the run
  protocol) instead of re-deriving readiness from prose, and an unplanned
  intent is refused rather than improvised against.
- **`/abcd:intent` plugin command surface** (`commands/abcd/intent.md`),
  covering the full verb family — status, quoted-text create, `ready`, `plan`,
  `link`, `review`/`ingest` — plus the host-run planning interview an unready
  intent is routed to: the human confirms the press release, resolves open
  questions, and accepts, edits, or strikes every acceptance criterion before
  `abcd intent plan` is run as their sign-off act.

### Fixed

- **The published plugin now ships its agents and hooks.** The launch payload's
  include list named neither the `agents/` nor the `hooks/` surface directory,
  so the released bundle omitted both — the plugin installed without its agents
  and without its hook wiring. Both directories are added to the payload
  includes, and a bundle-completeness check now fails if any auto-discovered
  plugin-surface directory present on disk is neither included nor explicitly
  excluded with a reason, so a newly-added surface can never be silently dropped.
- **The history-store bootstrap error names the verb that exists.** When a
  transcript capture found the store's owned directories absent, the preflight
  error told the user to "run `abcd install`" — a verb that does not exist. The
  remediation now names `abcd ahoy install`, the verb that actually bootstraps
  the store (iss-58).
- **`abcd ahoy` no longer overclaims `managed-repo` for a stray `.abcd/`
  directory (iss-88).** Folder classification treated the mere presence of an
  `.abcd/` directory as a strong managed signal, so a repo with an unregistered,
  markerless `.abcd/` reported `managed-repo`. Only index registration or a
  marker block now promotes a folder to managed; a stray `.abcd/` reports
  `unmanaged-repo` (or `unmanaged-folder` outside a git repo).
- **The identity pin round-trips through the self-contained commit guard.** The
  pin was stored with Go's default JSON encoder, which escapes `&`, `<`, `>` (and
  always `"`, `\`, control characters); the pre-commit identity guard reads the
  raw bytes between the quotes with a naive parse, so an escaped value never
  matched `git config` and fail-closed a correctly configured identity (e.g. a
  `user.name` of `Marks & Spencer`). The pin is now marshalled without HTML
  escaping so `&`/`<`/`>` are stored literally, and the characters a parse can
  never read back (`"`, `\`, control) are refused at pin time with a clear remedy
  — keeping the commit guard zero-dependency rather than delegating it to a
  possibly-stale binary.
- **`abcd ahoy install` now honours an explicit config-value flag on an
  already-configured repo (apply-as-update).** Passing `--visibility`,
  `--docs-target`, `--oracle-backend`, or `--scan-deep` on a repo whose value
  was already set and valid was silently dropped: the persisted value
  short-circuited the install before the override was consulted. An
  explicitly-passed flag whose value differs from the persisted one now
  overwrites it, echoes the change (`changed: visibility: private -> public`),
  and — for visibility and docs-target — refreshes the `.gitignore` block and
  marker files so nothing is left inconsistent. A re-install with no such flag
  is still an exact no-op and never clobbers a valid value.
- **GL002 no longer fires a spurious blocker when a line has a closed inline-code
  span before a stray backtick.** `stripInlineCode` restored the entire original
  line whenever it reached end-of-line with an unpaired backtick, un-masking an
  enforced synonym that sat inside an earlier, correctly-closed span. It now
  blanks matched backtick pairs only and leaves a trailing unpaired backtick (and
  its tail) literal, so the earlier span stays masked (iss-106). Full CommonMark
  double-backtick span parsing remains out of scope.
- **`abcd intent "<text>"` no longer files a draft from a mistyped subcommand.**
  A near-miss for an intent subverb (`intent paln`, `intent lnk itd-5`) is
  refused with a did-you-mean and writes nothing, mirroring `abcd capture`'s
  guard; a genuine prose title still files. The shared typo heuristic is now
  record-id aware (`iss`/`itd`/`spc`), which also sharpens `abcd capture`.

## [0.3.0] - 2026-07-18

### Security

- **The redaction scanner no longer lets a secret survive a trailing underscore.**
  Nine hard-fail token patterns (GitHub `ghp_`/`ghs_`/`gho_`/`ghu_`/`ghr_` and
  fine-grained PATs, AWS `AKIA`, Stripe `sk_live_`/`sk_test_`) used a pure
  alphanumeric charset closed by a `\b` word boundary. Because `_` is itself a
  word character, a credential immediately followed by `_` (a JSON key, a
  concatenation, `token=ghp_..._old`) had no boundary and slipped through
  unredacted into the stored transcript. The trailing `\b` is dropped (matching
  the existing Google-key fix); the leading boundary, prefix, and minimum length
  keep the match precise. The same fix extends to Slack `xox` tokens (whose
  charset also excludes `_`) and the Anthropic/OpenAI `sk-ant-`/`sk-proj-`/
  `sk-svcacct-` keys (a minimum-length key ending in `-`), so no token pattern
  relies on a trailing word boundary.
- **Untrusted file reads are guarded against symlink-follow and unbounded
  reads.** Reads of content that can originate outside the local worktree — the
  sources-index registry, packed-lifeboat layer files, and the CLI JSON operands
  (`disembark coverage`, the memory `--pages-json`/`--page-json` transport, and
  the lesson/synthesis payloads) — now route through a single guarded primitive
  (`O_NOFOLLOW` + regular-file on the open fd + size cap, in one call). Previously
  some followed a symlink to its target's content or read an endless/oversized
  file unbounded, and a symlinked registry surfaced a raw, path-leaking error;
  all are refused consistently, closing a class of `lstat`→`read` swap windows.
- **The last three ahoy reads route through the guarded primitive, closing a
  residual read-time TOCTOU.** The hook-manifest check and the two `.gitignore`
  reads (one at the attacker-influenced working-directory boundary) each ran a
  separate `lstat`, regular-file, and size-cap check and then a distinct
  `os.ReadFile`, so a type or symlink swap in the window between the check and the
  read was not refused on the descriptor that was read. All three now read through
  the single guarded open (`O_NOFOLLOW` + regular-file on the open fd + size cap),
  and every structured signal — the manifest's reason strings and the
  `.gitignore` overwrite refusals — is preserved unchanged.
- **Machine (`--json`) output no longer leaks an absolute local path.** The
  error-only path scrub covered failures but not the success envelope, so
  `capture`, `capture resolve`/`wontfix`, and `capture list` echoed the absolute
  ledger path in their `path` field; a successful `memory ingest` recorded the
  absolute (and, for a `~/…` source, home-rooted) source path in its
  `citation.origin`; and a `memory ingest` failure on a source outside the
  working directory embedded an absolute source path the scrub could not reach.
  Every such locator is now rendered repo-relative — in machine output and in the
  persisted citation alike — so no verb emits a developer-identity path into
  machine output.

### Added

- **A one-line, checksum-verified installer in the README.** The command
  detects OS/architecture, downloads the binary and `checksums.txt` from the
  latest GitHub Release, verifies the binary's SHA-256 against the manifest
  fail-closed (a mismatch — or a binary the manifest does not list — refuses
  to install), and installs to `/usr/local/bin`. The README also documents
  the inspect-first manual equivalent.

## [0.2.0] - 2026-07-17

### Security

- **Two more git call sites are isolated — finishing the env-inheritance sweep.**
  An ultracode sweep-completeness pass found the two the earlier round missed.
  `identity.gitConfig` (the commit-identity gate) read `user.name`/`user.email`
  with the ambient environment, so an injected `GIT_CONFIG_*` could forge — or an
  inherited `GIT_DIR` redirect — the very identity the gate verifies; it now uses
  `gitutil.ScrubbedEnv` (keeping global config, like the identity probe).
  `capture.discoverRepoRoot` ran `git rev-parse --show-toplevel` with the ambient
  environment, so an inherited `GIT_WORK_TREE` redirected repo-root discovery — and
  thus where the issue ledger is read and written — at an attacker-chosen tree; it
  now uses `gitutil.IsolatedEnv`.
- **The embark/pack path guards every read of an arbitrary target repo.** Five
  reads on the lifeboat embark/pack path — `embark`'s target `CLAUDE.md`
  (`embarkMarker`), the target record compare (`classifyEmbark`), the coverage
  handoff (`readCoverageHandoff`), the pinned provenance (`readProvenance`), and the
  destination-gate provenance probe (`isAbcdLifeboat`) — used a plain `os.ReadFile`
  (three of them after a separate `Lstat`, a TOCTOU window; one checking the size
  only *after* reading it all). The source is an arbitrary, possibly hostile repo,
  so a symlinked or device/endless file could be followed or read unbounded. All
  now route through `fsutil.ReadGuarded` (`O_NOFOLLOW` + regular-file on the open fd
  + size cap), closing the swap window and the resource-exhaustion path in one call.
- **The identity probe and the ahoy git helper ignore inherited `GIT_DIR`/
  `GIT_WORK_TREE` and injected `GIT_CONFIG_*` — completing the `IsolatedEnv`
  sweep.** Two git call sites still ran with the ambient environment. Worse of the
  two: `scanner.ProbeIdentity` reads the caller's `user.name`/`user.email` to
  build the hard-fail identity-redaction matchers, so an injected
  `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_*` (as a CI/agent sandbox can export) forged a
  fake identity and the caller's *real* name/email then sailed through the ship
  gate and the transcript sanitiser unredacted. It now runs with a new
  `gitutil.ScrubbedEnv` — the repo-selection and config-injection vars stripped but
  global config kept, since the caller's identity legitimately lives there and full
  isolation would blind the probe. `ahoy.runGit` (which derives the root-commit SHA
  and origin URL that key the cross-repo history registry) now uses the fully
  isolated `gitutil.IsolatedEnv`, so an inherited `GIT_DIR` can no longer register
  one repo's transcripts under another's immutable key.
- **The secret scanner detects a Google API key whose 35th character is `-`.** The
  `\bAIza…{35}\b` pattern is fixed-length and its class includes `-`, so a key
  ending in `-` had no shorter match to satisfy the trailing ASCII `\b` and the
  `hard_fail` secret slipped both the launch gate and the transcript sanitiser. The
  trailing `\b` is dropped (the `AIza` prefix and fixed length still bound it).
- **`home_path_self` redaction is case-insensitive.** The overlapping-secret and
  Unicode-boundary hunt added `(?i)` to the email/name/github identity matchers but
  left the caller's own home path case-sensitive, so on a case-folding filesystem a
  differently-cased spelling of `$HOME` — the same directory on disk — escaped the
  hard-fail `home_path_self` gate. It now folds case like its siblings.
- **Untrusted repo content is sanitised on every terminal render path, not just
  the lifeboat report.** The escape/C1 sanitiser that guarded `disembark`'s human
  report is now a shared `internal/termsafe` primitive, and the other render paths
  that printed repository-derived text raw — `abcd audit`, `docs lint`, `capture
  list` skip rows, the `intent`/`spec` boards, `memory` (bare + `ask`) — route
  through it. A crafted commit subject, file path, error string, or memory-page
  summary can no longer inject an ANSI escape to recolour or corrupt the report.
  The primitive also now masks the bidirectional-override and zero-width
  ("Trojan Source") classes, so untrusted text cannot visually reorder or hide
  characters so the rendered line differs from the bytes. JSON output is unaffected.
- **The identity pin is written atomically.** `identity.WritePin` persisted
  `.abcd/config/identity.json` with a plain in-place `os.WriteFile` — the fifth
  writer the iss-32 atomic-write consolidation missed (the guard flags only
  divergent atomic primitives, not a non-atomic one). It truncated the pin before
  rewriting, so a crash mid-write left a corrupt or empty gate config, and it
  followed a symlink at the path. It now routes through the canonical
  `fsutil.WriteFileAtomic` (temp + fchmod + fsync + rename + parent fsync).

- **The secret scanner now detects GitHub fine-grained PATs (`github_pat_…`) and
  PEM private-key headers.** Neither was in the bundled pattern set, so a
  current-generation GitHub token (GitHub's default since 2022) or a committed
  private key passed both the `abcd launch` ship gate and the history-transcript
  sanitiser unflagged.
- **Write-time transcript redaction no longer leaks raw secret bytes from
  overlapping matches.** `scanner.Redact` masked by longest-first substring
  replacement, so two partially-overlapping secret spans (e.g. an `sk-ant-` key
  running into a JWT) left the shorter token's tail verbatim, and the fail-closed
  re-scan could not catch the now-truncated remainder. Redaction is now by
  authoritative byte span (overlap bytes forced to `*`), matching the serializer.
- **History capture fails closed when the per-repo `pii.json` is broken.**
  `Scanner.ScanText`/`Redact` could not signal the degraded (`unavailable`) state
  the way `ScanBundle` does, so a malformed config silently redacted transcripts
  with a weakened pattern set and still reported success. A new `Unavailable()`
  accessor is now consulted before capture.
- **A per-repo config can no longer neuter a bundled detector by replacing its
  regex.** The severity floor clamped an override's severity but not its regex, so
  swapping a bundled pattern's regex for a never-match one disabled detection at
  full `hard_fail` severity. Bundled regexes are now immutable (a config may only
  raise severity / adjust label; new detection must use a new pattern name), and
  the config merge is all-or-nothing.
- **The launch bundle no longer ships denied namespaces or gitignored files
  reached through a symlink.** A symlink to (or into) the repo root let a
  dereferenced walk descend into `.git/**` and `.abcd/**`, and a symlink whose
  target was gitignored shipped the ignored content under the symlink's benign
  name. The structural deny is now re-applied to real paths at every level of a
  dereferenced walk, and the ignore probe covers the symlink target.
- **Command-error output no longer leaks a path equal to `$HOME` itself.** The
  redactor only rewrote paths *under* a root, so a message or `PathError` naming
  exactly the home directory slipped through — and its base segment is the
  username. `record-lint` likewise printed raw `*os.PathError` config-load paths;
  both now scrub the home/root prefix.
- **The launch bundle's gitignore exclusion fails closed on a git failure.** It
  delegated to a fail-open probe, so if git errored the exclusion pass admitted
  every gitignored file (typically `.env`-style secrets). A launch-local strict
  probe now distinguishes "nothing ignored" from a real git failure and rejects
  the affected candidates on failure; a plain non-git directory still resolves.
- **Git queries ignore inherited `GIT_DIR`/`GIT_WORK_TREE`/`GIT_INDEX_FILE` and
  config-injection env vars.** An inherited repo-selection variable could redirect
  abcd's isolated git queries at a different repository; those variables are now
  stripped before every invocation.
- **Transcript snippets no longer leak the head/tail of a short multi-byte
  identity value.** `sealLine` measured its fingerprint threshold in bytes while
  `maskSecret` used runes, so a short non-ASCII identity value kept visible
  characters; the two are now both rune-based.
- **`WriteFileAtomic` sets permissions on the open descriptor** (`fchmod`) rather
  than by name after close, closing a TOCTOU symlink-swap window; and
  `WriteFileAtomicPreserveMode` no longer silently widens an existing file to
  `0644` on a transient stat error (it fails closed).
- **The PII scanner detects real names, emails, and third-party home paths that
  RE2's ASCII `\b` was silently skipping.** A `hard_fail` `real_name` whose first
  or last character is non-ASCII (accented, CJK, Cyrillic) never matched — the
  name shipped unredacted; the `real_email`/`github_username` matchers were
  case-sensitive, so a case variant of the caller's own address slipped the gate;
  and `home_path_other` never fired at a realistic boundary (line start, after a
  space or `=`), so third-party home paths published verbatim. All three now use
  Unicode-aware, case-folding boundary predicates. Relatedly, a warn-level
  `home_path_other` span no longer suppresses a `hard_fail` `local_username`
  finding underneath it (which downgraded a username leak out of the ship gate).
- **The privacy-hygiene audit flags a bare home path with no trailing separator.**
  `absPathRe` required a path component *after* the username, so the leak itself —
  `HOME=/home/alice` at end of line — was never caught. The trailing separator is <!-- abcd-audit:allow -->
  now optional (matching the Windows branch).
- **The release-bundle gitignore probe ignores inherited `GIT_DIR`/`GIT_WORK_TREE`
  and config-injection env vars.** The strict probe appended to `os.Environ()`, so
  an inherited repo-selection variable could redirect it at a different repository
  and make a gitignored secret read as "not ignored" — promoting it into the
  release. It now runs with the same scrubbed environment as every other isolated
  git call (also applied to release-tag retention).
- **Terminal-report sanitisation strips the C1 control range (`0x80`–`0x9F`).**
  It masked C0 controls and DEL but let U+009B (CSI) through — an 8-bit terminal
  acts on it exactly like `ESC[`, reopening the escape-injection path from
  untrusted commit subjects/refs that masking `ESC` alone was meant to close.
- **The lifeboat pack overlap gate is case-insensitive on case-folding
  filesystems.** On macOS's default filesystem a differently-cased destination
  inside the source (`.../REPO/lifeboat` vs source `.../repo`) computed as an
  out-of-tree sibling and slipped the gate, so the pack wrote into the source
  tree.
- **The graveyard probe rejects option-like git refs from a hostile repo.** The
  lifeboat probe (whose stated threat model is hostile/archived repositories) fed
  branch names and an `origin/HEAD`-derived default branch straight to `git
  merge-base`/`branch`/`rev-list` as positional args; a crafted ref such as `-x`
  (written into `.git/refs`) parsed as a flag. Repo-derived refs beginning with
  `-` are now refused before reaching git — closing the argument-injection vector
  before a future richer subcommand makes it exploitable.
- **Memory-store reads are guarded against symlink and size attacks.** The sources
  registry (`.sources_index.json`) and each memory page were read with a bare
  `os.ReadFile` — no size cap, following symlinks — inside the repo working tree
  (a trust boundary) on every `abcd memory` verb, some under the store lock. A
  committed symlink to `/dev/zero` could OOM or hang the CLI. Both now route
  through a shared `fsutil.ReadGuarded` (`O_NOFOLLOW` + regular-file check on the
  open fd + byte cap).

### Fixed

- **`abcd install` no longer deletes a user's `.gitignore` content after an
  orphan `# BEGIN ABCD` fence.** An unbalanced BEGIN with no matching END made
  the rewriter drop every line to end-of-file, taking the user's own ignore rules
  with it. An unmatched BEGIN is now dropped alone; the content after it is
  preserved (mirroring the stray-END policy).
- **`git check-ignore` no longer inverts the answer for force-added files.**
  Dropping `--no-index` means a tracked file that matches a `.gitignore` pattern
  is correctly reported not-ignored, so the privacy audit stops flagging a
  force-added `DECISIONS.md` and the bundler stops dropping force-added files.
- **Recall keyword matching now handles inflected forms.** The stemmer could not
  round-trip e-drop (`merging`→`merge`) or doubled-consonant (`committing`→
  `commit`) inflections, and multi-word aliases bypassed stemming entirely, so
  guardrail domains like `COMMITTING` silently failed to match common phrasings.
- **Frontmatter and intent records agree on delimiter handling.** An unclosed
  frontmatter block is now treated as no-frontmatter (it previously harvested body
  prose as top-level fields), and the intent writer tolerates a trailing-space
  `---` delimiter exactly as the reader does.
- **YAML frontmatter flow-map keys are quoted.** A citation key containing a YAML
  metacharacter previously corrupted the record or broke the write→read round-trip.
- **Concurrent `abcd intent plan` runs mint distinct spec ids** (an exclusive
  advisory lock now guards the id-scan-and-write), and history capture attributes
  a byte-identical transcript from a distinct session to its own record instead of
  the first session's.
- **Several confident-but-wrong diagnostics are now guarded:** `abcd audit
  --root <missing>` and `disembark coverage <non-report>` return a usage error
  instead of fabricated findings; the privacy-hygiene rule surfaces an error
  rather than reporting clean when it cannot read the repo; brittle-reference
  linting skips fenced code blocks; and Cobra usage errors exit `2`, not `1`
  (which `abcd audit`'s tri-state reserves for "warnings only").
- **Robustness:** a malformed glob in the include config is a preflight error
  instead of a panic; an overflowing manifest pointer index is rejected instead
  of panicking; the intent identity gate compares the author git will actually
  stamp (honouring `GIT_AUTHOR_*`); inline-list items with quoted commas round-trip
  faithfully; and the lifeboat probe's tier gate matches what its adapters read.
- **The modular-rules loader resolves `.abcd/rules.json` from the repo root, not
  the current directory.** Run from any subdirectory, `abcd` looked up the
  per-repo overrides — and the kill switch — under the subdirectory, found
  nothing, and silently injected the default ruleset a repo had disabled. The
  loader now walks up to the nearest `.abcd` directory.
- **`abcd install` no longer rebuilds a malformed `.abcd/config.json` from
  scratch,** which destroyed whatever the user had. A JSON parse error is now
  respected: the file is left untouched and the install reports partial.
- **The issue-ledger reader rejects malformed records it used to accept
  silently:** a duplicate top-level (or nested) frontmatter key — where the reader
  kept the last value but a status transition rewrote the first — and a
  non-string `resolved_by` sub-value that validated clean then dropped to `""` on
  read.
- **The disembark voyage ledger logs SHA-256-format repositories.** Its root-SHA
  key accepted only a 40-char SHA-1; a 64-char SHA-256 root was rejected and the
  pack silently went unlogged.
- **The history transcript store accepts a SHA-256 root key too.** The same
  40-char-only assumption in `history.store`'s `rootSHARe` (a sibling of the voyage
  key above) made `history capture`/`list`/`read` all fail for a repo in git's
  SHA-256 object format — the ahoy layer derives the 64-char root SHA, but history
  refused it, so no session was ever stored. The key now accepts 40 or 64 hex.
- **Spec id minting cannot wrap to a negative id.** `specNum` discarded the
  `strconv.Atoi` overflow error, keeping the clamped `MaxInt64`, so an over-int64
  spec number made `NextID` compute `max+1` and mint `spc--9223372036854775808`. An
  unparseable/over-range number is now treated as no reservation.
- **Release-tag retention ignores prerelease/build tags.** `Tag()` renders the
  core `MAJOR.MINOR.PATCH` only, so a real tag `v1.2.3-rc1` surfaced in the plan
  as a phantom `v1.2.3` and collapsed against the real release; prerelease/build
  tags are now excluded (retention operates on release cores).
- **Distinct deleted paths key distinct graveyard findings.** The id cleaner
  *deleted* spaces and control characters, so two paths differing only in
  whitespace collided onto one finding id and one shadowed the other; the
  transform is now injective (percent-encoding), leaving ordinary paths unchanged.
- **The lifeboat pack destination gate treats an `ENOTDIR` stat as "absent"**
  (a prefix component being a file) rather than an uninterpretable error that
  refused a writable destination.
- **A relative PATH symlink is resolved against the symlink's own directory,**
  not the process working directory — so `abcd ahoy` no longer reports a bogus
  "foreign symlink" gap (and uninstall no longer refuses to remove a link it
  owns) for a correct relative install such as `/usr/local/bin/abcd ->
  ../lib/abcd/abcd` when run from another directory.
- **`source.classes` on a memory page is validated as a set, not an ordered
  list.** The same classes declared in a different order from their first
  appearance in `sources[]` were rejected, contradicting the schema's (and the
  error message's) set semantics.

### Changed

- **The coverage report is now schema v2.** Each brief section carries a `kind`
  (`extractable` — a source or a better adapter could ground it, so a blank is
  coverage debt — versus `human-owned` — a question only a person can answer, so a
  blank is not a failure), the durable form of the M2 cross-repo gate decision
  (adr-36). A blank additionally carries a `resolution` (`open`/`answered`/
  `deferred`) and, once answered, an authored `answer` whose provenance is a
  person and a date rather than a file it did not come from. `abcd disembark
  coverage` still refuses a report from a newer schema with an upgrade message.

### Added

- **`abcd intent "<text>"` files a new draft from quoted text — a symmetric
  create path.** Typing `abcd intent "I want users to feel X"` mints the next
  `itd-N` under the intent-store lock and writes
  `.abcd/development/intents/drafts/itd-N-<slug>.md`, seeded from the text with the
  canonical draft frontmatter and a minimal, lint-valid body skeleton — no `new`
  sub-verb required, mirroring `abcd capture "<text>"`. The old `abcd intent new
  "<text>"` still works as a backwards-compatible alias but prints a deprecation
  warning on stderr naming the quoted-text shape. Bare `abcd intent` stays
  read-only status + help and mutates nothing. Both ledgers' bare-form help now
  carry a one-line decision rule (nitpick/observation -> capture; user-facing
  change to ship -> intent). The `/abcd:capture promote` flow hands the issue text
  to this create path (itd-46).
- **`GL002` — a glossary-driven forbidden-synonym gate for the record lint.**
  The lint now reads each glossary term's `forbidden_synonyms` and flags an
  *enforced* synonym used as a standalone word in live prose, so terminology
  drift is caught by a detector instead of by eye (itd-43). Enforcement is a
  deliberate subset (`epic` first): most forbidden synonyms are common English
  words whose false-positive rate would sink the gate, and each enforced word
  must be one the glossary actually forbids. Matching uses explicit Unicode word
  boundaries — not the ASCII-only regexp `\b` — and skips code spans, YAML
  frontmatter, dated/historical records, and the glossary term files themselves.
- **Bare `abcd ahoy` now names the next step for the folder it classified.** An
  unmanaged git repo report points at `/abcd:ahoy install` as the way to adopt
  it, and a plain (non-git) folder report states there is nothing to act on —
  the read-only classification never mutates either (itd-40).
- **Synthesis over the record — `abcd disembark principles`, `press-release`,
  and `oracle`.** Three post-pack verbs interpret a packed lifeboat, each in
  one of two self-recorded modes. Without a payload they run **deterministic
  mode**: principles distilled evidence-only from the packed ADRs'
  Decision/Consequences bullets, the press release composed from the brief's
  own page (or the spine, or an honest placeholder), and the oracle scoring
  mechanically — a failed manifest verification is a `MAJOR_RETHINK` verdict,
  not an error; more blanks than grounded sections is `NEEDS_WORK`; a healthy,
  verified lifeboat ships `SHIP` — the first code home of abcd's registered
  review-verdict vocabulary. With `--*-json <file|->` they ingest a
  host-delegated agent's output behind the same trust guards as an intent
  verdict, under cite-or-be-dropped (a principle or oracle finding citing no
  live record id, graveyard finding, or packed path is dropped and reported; a
  press release citing nothing resolvable is refused whole). The binary stamps
  the oracle's attestation fields itself, so a model cannot fabricate a
  manifest hash. All synthesis artifacts live outside `manifest_sha256`, are
  fully replaced per run, and carry no wall-clock — the audit is keyed by the
  lifeboat's own manifest hash. The four agents (`principle-distiller`,
  `graveyard-interpreter`, `press-release-composer`, `lifeboat-oracle`) ship
  under `agents/` with itd-5's prompt discipline: versioned prompts in the 0.x
  calibration band, `reads_untrusted_input` declared, and an injection-canary
  fixture each (itd-88, adr-35).

- **`abcd embark` — a lifeboat comes ashore.** `embark probe <lifeboat> [target]`
  is the read-only reconciliation: it refuses a lifeboat whose provenance schema
  is newer than the binary (with an upgrade message), re-hashes every archived
  file against the pinned `manifest_sha256` so a tampered or truncated lifeboat
  is caught before anything is read in anger, and reports — in one bulk report,
  not a per-file barrage — every conflict with the target repository. `embark
  from <lifeboat> [target]` is the write path: it refuses entirely on any
  conflict (identical bytes are an idempotent skip, and a re-embark is a no-op),
  writes only the record families (ADRs, issues, intents, specs) through
  `os.Root` containment plus independent path validation — two layers, so a bug
  in one is not an escape — and never copies lifeboat prose into `CLAUDE.md`:
  it re-injects the *current* abcd marker block instead. The rendered result
  leads with the coverage report's blanks and their questions — the handoff to
  the human who must answer them. The packer now also carries the spec store
  (`rescue/specs/`), and every lifeboat's provenance records
  `record_manifest_sha256`, the seal over exactly the record-derived families
  that must survive a round-trip byte-for-byte: pack → embark → re-pack
  reproduces it, and embarking into a byte-copy of the source reproduces the
  full original manifest hash (itd-88, adr-35; closure re-scope in the
  2026-07-16 decision log).

- **The graveyard — what the project abandoned, in three strictly-ordered
  layers.** Every packed lifeboat now carries `graveyard/archaeology.json`
  (deterministic git archaeology: reverted commits, branches never merged into
  the default branch ranked by divergence age, paths deleted after substantial
  history, dependencies adopted then dropped, wholesale-rewrite commits — pure
  evidence, no interpretation, from any git repo) and `graveyard/abandoned.json`
  (what the record itself declared dead: superseded intents and ADRs, wontfix
  issues with their reasons, each ADR's Alternatives-Considered options, and
  rejected options named in the decision log). A new
  `abcd disembark graveyard <lifeboat-dir> --lessons-json <file|->` verb ingests
  a host-delegated interpretation over those two layers into
  `graveyard/lessons.json` under a **cite-or-be-dropped** validator: every
  lesson must cite live layer-1/2 evidence ids or it is dropped (reported, never
  fatal), low-confidence lessons are quarantined under
  `graveyard/low-confidence/` instead of the main file, and the untrusted
  payload is read behind the same trust guards as an intent verdict (size cap,
  no symlinks, unknown fields refused, schema version gated). Each ingest fully
  replaces the prior interpretation, so a promoted or later-dropped lesson
  leaves nothing stale behind. The validator — not the model's good intentions
  — is the difference between a graveyard and a séance (itd-88, adr-35).

- **`abcd disembark probe <repo>` — a read-only coverage probe over any
  repository.** It walks a repo without touching it and reports, per brief
  section, whether a lifeboat could ground it: `grounded` / `partial` / `blank`,
  with the tier it was grounded from, a confidence, and the evidence cited. A
  blank is a first-class result — it carries what abcd searched and the question
  a human must answer, so the report is a to-do list, not a shrug. Adapters
  degrade across three tiers — git (any repo), conventions (README, docs,
  CHANGELOG, manifests, ADRs wherever they live), and abcd-native (`.abcd/`) — so
  a richer repo grounds more, and the `graveyard` section grounds from git
  history alone (reverts, deleted files, dependency churn). Every read is
  contained to the repo (`os.Root`), bounded, and non-blocking, and the source
  tree is byte-identical afterwards — the probe never writes to a source. A
  companion `abcd disembark coverage <report.json>...` reduces several probe
  reports to one section×repo table with an always-blank verdict per section:
  the delta between a record-rich repo and a git-only one is what keeping a
  record is worth, legible as a number. Both are read-only operator verbs (no
  `/abcd:disembark` command surface yet); the packer that writes a lifeboat is a
  later milestone (itd-88, adr-35). The dependency-manifest detector spans Go,
  Node, Rust, Python (pip/poetry/pdm/uv/pipenv), Ruby, and PHP, so a real project
  is not reported as having no dependencies merely because the probe did not know
  its packaging tool (found probing a Python/uv repo in the M2 cross-repo run).

- **`abcd disembark plan <repo>` — a dry run of the packer.** It shows the
  complete file set a lifeboat pack would write — brief citation maps for the
  grounded sections, `coverage.json`/`coverage.md`, verbatim copies of the ADRs
  and the issue ledger, the rescue spine (the intent corpus where one exists, a
  git-derived summary where it does not), and a `_provenance.json` carrying a
  pinned `manifest_sha256` over every other file — and writes **nothing**. Plan
  and the eventual packer are one code path, so the dry run cannot describe a pack
  a real pack would not perform; a re-plan of an unchanged source is byte-for-byte
  identical (the manifest carries no timestamp). `--json` emits the manifest
  (paths, sizes, and the hash — never file content). Still a read-only operator
  verb (no `/abcd:disembark` command surface yet); the destination write path is
  a later milestone (itd-88, adr-35).

- **`abcd disembark pack <repo> <dest>` — writes a lifeboat out-of-tree, and the
  `/abcd:disembark` command surface.** It writes the planned file set to `<dest>`
  and never to the source (a test hashes the source tree before and after).
  Everything that stops a pack destroying real work is enforced: a **destination
  safety gate** refuses unless `<dest>` is absent, an empty directory, or an
  existing lifeboat abcd produced (it carries a parseable `_provenance.json`) —
  and refuses a symlinked destination, one inside a `.git/` directory, or one that
  overlaps the source tree. The planned bytes are **secret-scanned before any
  write** and a hard-fail refuses the whole pack — a secret is fixed at source,
  never redacted into the artefact. Files are written into a staging directory
  through `os.Root` (no crafted path or symlink escapes it) and renamed into
  place, so a crash leaves staging, never a half-lifeboat; `_provenance.json` is
  written last. Any abcd marker block in a copied record is stripped so embarking
  the lifeboat cannot plant a stale rules-loader. Each pack appends one line to an
  append-only voyage ledger at `~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl`,
  keyed on the source's root-commit SHA and carrying the manifest hash. `--json`
  emits the result (destination, file/byte counts, hash, voyage status).

- **Session transcripts are captured automatically when a session ends.** A
  `SessionEnd` hook now runs `abcd hook session-end`, which redacts the session
  transcript through the existing two-stage, fail-closed scanner and files it in
  the local per-repo store — no flag to pass, no command to type. `abcd history
  list` shows the records. It is wired to `SessionEnd` (which fires once when a
  session terminates) rather than `Stop` (which fires once per assistant turn) —
  the transcript grows through a session, so a `Stop`-wired capture would store a
  fresh, larger superset every turn. The store has existed since the native transcript
  corpus landed (adr-29) and was **called by nothing**: `history.Capture` was
  built, correct, and unused, so no session was ever stored. That gap was the one
  cost on the board that could not be recovered later — a session that ends
  without being captured cannot be reconstructed by any amount of future work.
  The hook is operator-internal, never blocks the host, and always exits `0`: a
  malformed payload, a missing or non-regular `transcript_path`, a hostile
  session id, or a directory that is not a git repo each capture nothing, say why
  on stderr, and exit cleanly — a `Stop` hook that errors or hangs would wedge
  the session, which is strictly worse than a missed transcript. Re-capture is
  idempotent (a `Stop` hook may fire more than once per session), the transcript
  open is non-blocking so a FIFO cannot hang the hook, and nothing is ever
  written to stdout. It needed a new verb because `history capture` cannot be
  wired to a `Stop` hook: from stdin it *requires* `--session <id>`, and a `Stop`
  hook delivers its session id inside a JSON payload (itd-89, adr-29).

- **A session that starts in a repo where abcd is not installed now says so.** A
  `SessionStart` hook runs `abcd hook session-start`, which — when the current
  repo is a git repository whose transcript store has not been bootstrapped —
  prints a one-line notice telling the user their sessions will not be captured
  and how to fix it (`abcd ahoy install`). Without it the automatic-capture hook
  above fails silently: the plugin is enabled, the user assumes their transcript
  corpus is accruing, and it is not. The notice rides `SessionStart`'s visible
  channel (stderr on a non-zero exit) and never blocks the session; every case
  that is *not* a bootstrappable-store problem — a non-git cwd, a malformed or
  empty payload, an already-installed store — stays completely silent and exits
  `0` (iss-95, itd-89).

- **`abcd audit` — a read-only repo-conformance check.** One command reports
  whether a repository follows the working conventions: the three-tier `.abcd/`
  layout, an `AGENTS.md` router, decisions durable in a committed
  `.abcd/work/DECISIONS.md`, docs currency (reusing the docs-lint engine where
  `docs/` exists), and privacy hygiene (no absolute local paths in committed
  files, waivable per line with `abcd-audit:allow`). It runs against any repo
  given only a working directory, prints a grouped human report with a fix per
  gap or machine JSON (`--json`, stable rule ids, `{ "findings": [] }` when
  clean), and exits with a tri-state code — `0` clean, `1` warnings only, `2`
  any error — so it gates CI as well as onboarding. It answers a different
  question from `abcd ahoy doctor`: `doctor` is tool-setup health, `audit` is
  repo conformance. `/abcd:prepare-this-repo` now runs `abcd audit` for its
  Phase 2 gap report instead of hand-auditing (iss-86).

### Fixed

- **`--json` and stderr command errors no longer leak the developer's home or
  working-directory paths.** `cli.Run` routes every command error through the
  machine envelope, so identity-bearing paths reached it three ways: an
  `os.PathError`/`os.LinkError` (e.g. `memory ask --page-json` on a missing file),
  a path `fmt`-formatted into a core error (e.g. `capture` on a symlinked ledger
  dir), and a custom error type (e.g. history's home-rooted store path). The
  `Run()` boundary now redacts the working-directory and home roots (to `.` and
  `~`) and reduces any remaining `PathError`/`LinkError` path to its base name.
  Generalises the per-branch fix made in `iss-29` (iss-76). A verb echoing a
  user-supplied absolute path outside both roots is out of scope, tracked in
  `iss-81`.
- The `intent_lifecycle` record-lint rule now **blocks duplicate intent ids**.
  Id allocators are branch-local — parallel agents on separate branches each
  scan for `max + 1` and mint the same id — so two intents both claimed
  `itd-82` and both merged with every gate green. The rule flags *every* file in
  a colliding set, not just one: the linter cannot know which claimant is
  authoritative, and flagging a single file would imply the others are fine. The
  collision itself is resolved (the later claimant renumbered to `itd-83`); the
  underlying minting scheme is tracked as `iss-80`.
- `memory ingest --keep-original` writes the stored source copy through the
  canonical `fsutil.WriteFileAtomic` (temp + fsync + **chmod + parent-directory
  fsync**) instead of an inline temp+rename that omitted both — the fifth
  divergent atomic write the `iss-32` consolidation left untouched. The
  one-canonical-primitive detector now also flags inline `os.O_EXCL`+`os.Rename`
  sequences, not just named primitives (iss-79).

### Added

- **Four reviewer agents ship with the plugin**: `abcd:ruthless-reviewer`
  (correctness, resource handling, error paths, dead code),
  `abcd:security-reviewer` (adversarial review of a trust boundary),
  `abcd:docs-currency-reviewer` (every user-facing claim verified against the
  code), and `abcd:sota-researcher` (evidence-tiered state-of-the-art research).
  Every repo with the abcd plugin enabled gets the same review bar, versioned in
  the repo rather than in a per-machine harness config. Each renders a binary
  verdict, and every finding it emits must carry a concrete failure scenario —
  the LLM-judge calibration discipline (itd-81).
- **Intent-fidelity review** (itd-80): the ship move now emits a report-only
  fidelity-review receipt, and `abcd intent review ingest --verdict-json <path>`
  applies the host-produced verdict back onto the record. When `abcd spec close`
  ships a linked intent (`planned/ → shipped/`), it parks a deterministic OWED
  receipt marker in the intent's `## Audit Notes` and writes an ephemeral review
  request under `.abcd/.work.local/reviews/` (gitignored); the emit is
  non-fatal, so a failure never un-ships the intent. `abcd intent review ingest`
  validates an untrusted intent-fidelity verdict JSON fail-closed (schema,
  in-enum verdicts, cited evidence, and each `criterion_id` bound to an actual
  Acceptance-Criteria bullet), then either replaces the OWED stub with the
  rendered per-criterion verdicts and honoured/diverged/missing audit
  (`INGESTED`, idempotent — a re-ingest is a no-op) or quarantines a bad payload
  (`DEAD_LETTER`: all criteria `INCONCLUSIVE`, raw payload retained) — never a
  partial application. Bare `abcd intent review <itd-N>` re-emits a shipped
  intent's request. The single source of truth is the intent file's Audit Notes;
  there is no side receipt store.

- The **intent lifecycle** verbs `abcd intent` and `abcd spec` (itd-80), the
  front doors onto the native intent store (`internal/core/intent`). Bare
  `abcd intent` renders a read-only lifecycle summary (intent counts by bucket,
  spec counts by status, and the linked intent↔spec pairs); `abcd intent plan
  <itd-N>` mints a native spec for a draft intent that carries a non-empty
  `## Acceptance Criteria` section (the itd-1 gate), writes both sides of the
  bidirectional link (the spec's `intent: itd-N` and the intent's
  `spec_id: spc-N` plus a default `kind: standalone`), and moves the intent
  `drafts/ → planned/` — fail-closed, so every intermediate on-disk state stays
  valid under the `intent_lifecycle` record-lint rule. `abcd intent link <itd-N>
  <spc-N>` retroactively links a planned intent to an existing spec, refusing a
  spec that realises a different intent. Bare `abcd spec` renders the spec-store
  status; `abcd spec close <spc-N>` moves a spec `open/ → closed/` (the
  lifecycle reconcile that trails a close lands in a later phase). The
  frontmatter line-scanner shared by these stores now lives in
  `internal/core/frontmatter`.
- The **modular rules loader** core and its `abcd rules [domain]` verb (itd-3,
  phases 1 + 3). `internal/core/rules` holds binary-bundled default rule domains
  (COMMITTING, DOCUMENTATION, ROADMAP, ISSUES, INTENTS, LIFEBOAT, PII, and
  OPINIONS — whose rules point at the canonical conventions under
  `.abcd/development/principles/` rather than copying them) merged
  with an optional per-repo `.abcd/rules.json` override (per-field domain
  override, sticky kill switch), with word-bounded recall matching (including a
  conservative suffix stemmer so `commits`/`issues` recall their keyword),
  `*<DOMAIN>` star-commands, and per-domain dedup signatures. Bare `abcd rules` renders the
  active rule set; a positional `DOMAIN` (case-insensitive) scopes to one; a
  malformed `rules.json` fails closed. A Claude Code prompt-router hook
  (`abcd hook prompt-router` / `prompt-router-reset`, operator-internal) injects
  the matched rules just-in-time on `UserPromptSubmit` with per-session
  signature dedup, clears the ledger on a `SessionStart`/`PreCompact` reset
  (event-driven refresh; a large fixed-N counter is only a backstop), and is
  fail-closed and non-blocking — a malformed payload, unreadable `rules.json`,
  or state error injects nothing and logs out-of-band, never wedging a session.
  The `hooks/hooks.json` manifest wiring lands with ahoy in the next phase.
- A `surface_coverage` record-lint rule (iss-35): the deterministic half of the
  brief↔surface cross-check. It reads the plugin surface
  (`rules.surface_coverage.commands_dir`, `skills_dir` — outside the lint roots)
  and the brief's surface registry table (`rules.surface_coverage.registry`, by
  convention `.abcd/development/brief/04-surfaces/README.md`), and asserts three
  invariants: every real surface has a registry row; every row marked `shipped`
  in the registry's **Status** column has a backing surface while every `staged`
  row (a design target) has none; and every row's status is `shipped` or
  `staged`. The bare `/abcd` top-level is binary-backed and exempt from the file
  check. Chapter-link resolution stays with `links_resolve`; the semantic half —
  each row's prose vs. binary behaviour — stays a release-gate agent check.
- A managed-repo **git-identity gate** (iss-62): a repo can pin its expected
  commit identity in `.abcd/config/identity.json`, and every commit is checked
  against it. `ahoy doctor` reports a divergence (a repo-local override that
  differs from the pin, or an unset identity) or an un-pinned repo; `ahoy
  install` adopts the gate by pinning the current git identity; `ahoy
  identity-check` exits non-zero on a mismatch; and the `pre-commit` hook
  fail-closes so a stray identity (e.g. a sandbox default) is caught at commit
  time rather than discovered later. A repo with no pin is unaffected.
- A `context_status_free` record-lint rule: the shared orientation file
  (`rules.context_status_free.target`, by convention `.abcd/work/CONTEXT.md`)
  must carry no phase/status claims — status is read live from the CLI and
  the ledger, never hand-written into orientation docs. Patterns are
  configurable (`rules.context_status_free.patterns`) with sensible defaults;
  lines matching inside fenced code blocks are skipped.

- A `/abcd:prepare-this-repo` command — audits the current repository against
  the abcd record and adopts the three-tier `.abcd/` layout, a marked
  working-conventions section in `AGENTS.md`, and the commit gates; an interim
  bridge until repos are managed directly. Owned repos only (it refuses
  elsewhere), and it migrates the older root-level `.work/` scaffold layout
  with explicit sign-off.
- `/abcd:consult` and `/abcd:ingest` commands — consult the user-level sources
  corpus (confidential entries are never cited or named in public artifacts)
  and ingest a URL or document into it with extracted reference metadata,
  keywords, and a text-quality check. Both are thin fronts on the corpus's own
  tooling and stop gracefully when no corpus exists.
- A `persona_registry` record-lint rule: press-release quote attributions
  (`said <Name>,`) must name a persona from the registry file the rule's
  `registry` key points at; unknown names are blocker findings. Configured
  per repo in `record-lint.json`; the historical record is skipped via the
  standard content-drift exemptions.
- `abcd capture --blocked-by <iss-N,…>` records typed dependency edges on a new
  issue, and `capture list` / the status board now render a derived-priority
  view: unblocked issues first, then by severity, with blocked rows annotated
  `[blocked-by iss-N,…]`. There is no stored priority — the ordering is a
  read-time projection, so resolving a blocker re-prioritises its dependents
  automatically.
- A store-contract README for the issue ledger (`.abcd/work/issues/README.md`).

### Changed

- `abcd intent plan` seeds a new native spec with a clear author-guidance
  placeholder in its `## Summary`, rather than a bare `TODO` (iss-68).
- `abcd spec close <spc-N>` now reconciles the linked intent (itd-80): it moves
  the intent `planned/ → shipped/` and then closes the spec, so one command
  completes the lifecycle transition. It is fail-closed (a missing/empty intent
  link, a non-existent or ambiguously-linked intent, bidirectional drift, or an
  intent in an unexpected bucket refuses with no partial move) and idempotent (a
  re-run on an already-shipped intent / already-closed spec is a clean no-op).
  The intent's `## Audit Notes` are left untouched. A new `spec_lifecycle`
  record-lint rule mirrors `intent_lifecycle` on the spec side: every spec under
  `specs/{open,closed}/` must carry a well-formed `id`/`slug`/`intent` link whose
  named intent EXISTS and points back at this spec (bidirectional agreement).
- The issue ledger moved from `.abcd/development/activity/issues` to
  `.abcd/work/issues` (the committed shared-working tier).
- The atomic-write and real-directory primitives are consolidated onto
  `internal/fsutil` (iss-32): the ahoy, capture, and memory store writers no
  longer keep their own divergent temp-file+rename copies. Two observable
  effects of routing through the canonical primitive: the ahoy and capture
  writers now fsync the parent directory after the rename (a crash-durability
  strengthening they previously lacked), and memory pages are written at a
  fixed `0644` (an explicit chmod, where the old writer left the mode subject to
  the process umask). A `TestNoNonCanonicalAtomicWritePrimitives` guard keeps a
  fifth copy from reappearing.

### Removed

- The `created` and `updated` frontmatter fields on issues. Git is the canonical
  source of an issue's timeline; the ledger no longer duplicates it.

### Fixed

- **Launch dogfood gate — identity false positive and resolver race** (iss-31).
  The secret/PII scanner no longer hard-fails on a system path such as
  `/dev/null` when the machine username collides with a system directory name
  (e.g. a user called `dev`): a local-username match is suppressed only when it
  is the top segment of an absolute system path, so genuine username leaks
  (nested under a home root, or bare) are still caught. The launch bundle's
  compiled-glob cache is now guarded by a mutex, removing a data race when the
  transport-agnostic core resolves bundles concurrently.
- **Memory-ingest boundary — partial-failure reporting and CRLF parity**
  (iss-30, continued). When `abcd memory ingest --keep-original` fails to store
  the original *after* the pages and registry are durably written, it no longer
  reports total failure: the successful ingest is reported (pages listed) with a
  warning and a non-zero exit, and the failure message names only the
  repo-relative store location — no absolute path, in text or `--json`. CRLF
  documents now split identically to their LF form (`splitFileFrontmatter`
  normalises line endings like its sibling parsers), so a `\r\n` closing `---`
  delimiter is no longer rejected and hashes/summaries no longer silently
  degrade.
- **Fail-closed capture surface** (iss-29). A mistyped `capture` subcommand
  (e.g. `abcd capture resovle iss-1 …`) is no longer swallowed as free text and
  filed as a new issue; it is refused with a did-you-mean and writes nothing.
  Errors requested with `--json` are now emitted as a `{"error": …}` envelope
  rather than raw Go text, and `abcd docs lint` with a missing or unreadable
  config reports a clean, repo-relative diagnostic instead of a raw file error
  that leaked the absolute config path.
- `abcd` status now reports `IsGitRepo` correctly in a linked git worktree or a
  submodule, where `.git` is a regular gitfile rather than a directory (iss-72).
- `abcd intent plan` now refuses an `## Acceptance Criteria` section with no
  top-level `-`/`*` bullet, matching the ingest gate — an intent can no longer be
  planned into a state where every fidelity verdict dead-letters for having zero
  positional criteria. The intent template's Audit Notes placeholder is cleared
  when the first review block lands, so a populated audit carries no stale "Empty"
  claim (iss-67).
- The frontmatter scanner (`internal/core/frontmatter`, used by `abcd intent`/
  `spec` and record-lint) now tolerates a trailing space or tab on the `---`
  delimiters; previously a `--- ` closing delimiter went unrecognised and every
  body line after it was misread as a frontmatter field. `record-lint` no longer
  keeps a divergent copy of the scanner — it routes through the canonical one and
  inherits this fix (iss-69).

### Security

- **Memory-ingest fetch/read hardening** (iss-30). `abcd memory ingest` now treats
  a non-2xx HTTP response as an error instead of storing the 404/500 error page as
  source content; the SSRF guard additionally rejects NAT64 (`64:ff9b::/96`) and
  6to4 (`2002::/16`) IPv6 addresses that embed a metadata/loopback/private IPv4; a
  local source file is size-capped like the URL path; and a `~user` path is left
  literal rather than being mangled into `home`+`user`.
- **Spec-store hardening** (iss-68). The spec-store reader now opens a file once
  with `O_NOFOLLOW`+`O_NONBLOCK` and validates the file descriptor before reading,
  closing a symlink-swap window (and never blocking on a FIFO leaf). `NextID` fails
  closed on an intent `spec_id` that carries no parseable reservation number (e.g.
  `spc-` with no digits) instead of silently dropping it from the id-reservation
  scan (which could hand out a colliding id); a `spc-N` or `spc-N-<slug>` form still
  reserves N, consistent with record-lint. (The leaf-only ancestor-symlink guard
  and the atomic-rename clobber check are documented as accepted under the
  trusted-worktree model.)
- **Release receipt-gate hardening** (iss-70). The `receipt_gate` record-lint
  rule now binds each semantic-pass receipt to the gate it attests: a receipt
  satisfies a required gate only when its `policy.detector` equals that gate name,
  not merely when a `<gate>.json` file exists. This closes a hole where one
  genuine PROMOTE receipt copied across every gate's path satisfied them all.
  Arming (`record-lint --release-gate`) now treats the caller's required-gate list
  as authoritative even when empty — an argless arming clears the gates and fails
  closed rather than inheriting the committer-editable in-tree list. The
  `gate_lockstep` workflow parser no longer mistakes a nested `with: name:` for a
  step name. (The receipt-gate remains disabled outside release time.)
- **Secret-scanner serialisation hardening** (iss-65). A serialized scan finding's
  snippet now masks *every* secret on its source line, not only the finding's own
  token — two secrets sharing a line (a minified `.env`, collapsed JSON) no longer
  leak each other into the `abcd launch --json` report. The content sniff no longer
  misclassifies a valid UTF-8 file as binary when a multibyte rune straddles the
  8 KB boundary (which would have skipped scanning it), and a bundle file that
  cannot be read is now surfaced in `unscanned` rather than silently dropped.
- **Issue-ledger transition hardening** (iss-71). `abcd capture resolve`/`wontfix`
  now run their find→move under the same ledger lock id allocation uses, so two
  concurrent conflicting transitions on one issue can no longer land it in two
  status directories at once. A migrator-supplied `ForceID` is validated against
  the `iss-N` shape before any path is built, so a traversal id cannot touch the
  filesystem outside the ledger.
- **Rules-loader trust hardening** (iss-66). The per-repo `.abcd/rules.json` is now
  opened once with `O_NOFOLLOW` and validated on that file descriptor, closing a
  Lstat-then-read window where the file could be swapped for a symlink. The
  prompt-router's per-session dedup state moved off the world-writable shared temp
  dir to the per-user cache dir (`ABCD_RULES_STATE_DIR` still overrides), so a local
  co-tenant can no longer pre-create the predictable state path to suppress rule
  injection.

## [v0.1.0] - 2026-07-07

First tagged milestone: the Go rebuild through Phase 2. abcd is a single,
host-agnostic Go binary that is also a plugin for compatible agent harnesses, holding all
behaviour in a transport-agnostic `internal/core` behind a Cobra CLI front door and
a markdown plugin surface that shells out to it.

### Added

- Phase 0 scaffold: Go module (`github.com/REPPL/abcd-cli`), a
  transport-agnostic `internal/core`, a Cobra CLI front door (`abcd` status
  board and `abcd version`), the plugin surface, and the design record carried
  forward as the build specification.
- Phase 1 — install and launch. `abcd ahoy` installs abcd into a repo
  (folder-kind detection, visibility-driven gitignore, idempotent marker blocks in
  CLAUDE.md/AGENTS.md). `abcd launch --dry-run` renders a curated release bundle
  that excludes `.abcd/**` by default-deny, running a native secret + PII scanner,
  strict SemVer, marketplace-lockstep anti-drift, and newest-per-line retention over
  the bundle.
- Phase 2 — native capture substrates. `abcd history` is a SHA-keyed, redacted,
  gitignored transcript store (`list`, `show`, and a fail-closed `capture` write
  path); `abcd capture` is a directory-as-status issue ledger; `abcd memory`
  provides deterministic ingest / ask / lint.
- `abcd docs lint` — a deterministic docs-currency gate over `docs/` and the repo
  root: change-narration in a doc body, a broken relative link, or a stray
  top-level markdown file each fails the gate.
- `record-lint` — a deterministic drift gate for the `.abcd/development` design
  record (banned tokens, git-metadata, link resolution, intent lifecycle), wired
  blocking into CI and the pre-push preflight.
- Derived-versioning design record (intent itd-73 + ADR-31): the release version
  is derived from intents' declared impact, never hand-authored. The derivation
  itself lands in a later phase.
