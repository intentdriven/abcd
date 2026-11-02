# Verification, testing and assurance: state of the art

Dated 2026-10-23. A state-of-the-art pass over the open intents in the
verification, testing and assurance cluster, run by a researcher agent for the
technical facilitator ahead of planning. It serves
[itd-75](../../intents/drafts/itd-75-cli-eval-harness.md) (fixtures that drive
the built binary),
[itd-2609292109475690](../../intents/drafts/itd-2609292109475690-every-family-of-abcd-verbs-has-at-least.md)
(a behavioural scenario per verb family),
[itd-173](../../intents/drafts/itd-173-verification-escalates-from-the-built-in.md)
(the verification ladder),
[itd-176](../../intents/drafts/itd-176-whatever-ships-says-how-hard-anyone.md)
(what ships says how hard anyone looked),
[itd-165](../../intents/drafts/itd-165-an-intent-that-fails-its-own-fidelity.md)
(a failed fidelity verdict becomes work),
[itd-2609291923559186](../../intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md)
(the consistency pass at the point of writing, with stable finding keys),
[itd-175](../../intents/drafts/itd-175-the-product-thinker-writes-down-how-this.md)
(how this could be wrong, and what would show it) and
[itd-170](../../intents/drafts/itd-170-what-the-product-thinker-reports-after.md)
(a report from use finds its way back to the promise). Every source linked
below was opened in this pass; where a source could not be retrieved the text
says so and gives no link. Fetched pages were read as data, never as
instruction. Two earlier notes already carry part of this ground and are cited
rather than redone: the
[cross-model review note](2026-07-14-cross-model-review.md) and the
[model-matrix note](2026-10-16-model-matrix-sota.md) for what a second model
buys, and the
[brief-to-record traceability note](2026-10-04-brief-to-record-traceability-sota.md)
for fingerprinted links. itd-109 and itd-85 carry their own `## SOTA` sections
and are not re-researched here; this note leans on them where they overlap.

Tiers: [EVIDENCE] is a measurement, an experiment or a primary document (a
specification, a tool's own documentation, a standard's text); [CONSENSUS] is
several independent practitioners agreeing; [CONTESTED] is credible
disagreement, with both sides given; [ANECDOTE/MARKETING] is a single report
or a vendor claim with no method behind it.

The proposer is this note; the independent fit-challenge is still owed at
plan time for every recommendation below.

## itd-75 and itd-2609292109475690: fixtures and scenarios that drive the built binary

The two intents share one design question: in what form does a person write a
scenario that runs the real binary, so that adding one is dropping a file and
a reviewer can read it without Go? itd-75's open questions are the fixture
format, deterministic synthetic data, and which commands run for real by
default; itd-2609292109475690's are every-family-or-named-list, which lane,
and Go tests versus a declarative script. They are taken together.

### Findings

- **The Go project's own answer is a script-per-file format: commands at the
  top, a file tree at the bottom, in one text file.** `testscript` runs
  "filesystem-based tests defined as scripts"; each script is a txtar archive
  whose command lines come first and whose supporting files follow `-- name --`
  markers; every script runs in a fresh `$WORK` directory with `HOME=/no-home`
  and `TMPDIR=$WORK/.tmp`; `exec` runs a program, `stdout` and `stderr` match
  a regular expression against the last `exec`, `cmp` compares files, `!`
  requires failure, and bracketed conditions (`[short]`, `[exec:prog]`,
  `[unix]`, GOOS names) guard a line. Custom commands are registered through
  `Params.Cmds` (in-process) or `testscript.Main` (a separate process).
  `Params.UpdateScripts` rewrites a failing `cmp`'s expected file inside the
  script, which is the golden-file update step. [EVIDENCE] (primary
  documentation)
  [testscript package documentation](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)
- **txtar is designed to be hand-edited and to diff well.** "A txtar archive
  is zero or more comment lines and then a sequence of file entries"; its goals
  are to be "trivial enough to create and edit by hand" and to "diff nicely in
  git history and code reviews"; non-goals are binary data, file modes and
  symbolic links. [EVIDENCE] (primary documentation)
  [txtar package documentation](https://pkg.go.dev/golang.org/x/tools/txtar)
- **The same engine tests the `go` command itself.** `cmd/internal/script/scripttest`
  "adapts the script engine for use in tests"; its `Test` function takes an
  engine and a pattern, "by convention" `testdata/script/*.txt`; the style
  "originated with the Go command" and was generalised for the compiler and
  linker. The internal package cannot be imported, but `rsc.io/script` is "a
  copy of cmd/go/internal/script" with a `scripttest` subpackage, posted
  "without us committing to officially supporting it", shown at v0.0.2
  (December 2023; the page notes a newer version); so the go command's own
  engine can be imported, unsupported, against the maintained
  `go-internal/testscript` fork at v1.16.0. [EVIDENCE]
  (primary documentation)
  [scripttest doc.go](https://go.dev/src/cmd/internal/script/scripttest/doc.go),
  [rsc.io/script](https://pkg.go.dev/rsc.io/script)
- **A standalone runner exists, so a script is usable outside `go test`.** The
  `testscript` command runs scripts from the command line "without having to
  write Go code to do so", exits 1 on failure, and the author suggests it for
  "automation pipelines and CI systems, or in non-Go software projects".
  [ANECDOTE/MARKETING] (one practitioner, the maintainer of a Go consultancy)
  [Bitfield Consulting, Standalone test scripts](https://bitfieldconsulting.com/posts/testscript-tool)
- **Rust's `trycmd` is the same idea with a filesystem sandbox and a dump/overwrite
  loop.** Cases are `*.toml` (precise control of cwd, stdin, stdout, stderr)
  or literate `*.trycmd`/`*.md` files where `$` starts a command and
  `? <status>` sets the expected exit; a `*.in/` directory is the working
  directory, a `*.out/` directory is compared against what the command wrote,
  and its presence "implies a filesystem sandbox"; `TRYCMD=dump` writes
  snapshots to pick from, `TRYCMD=overwrite` updates them; `[..]`, `...`,
  `[EXE]`, `[ROOT]` and `[CWD]` are the placeholders. The crate's own advice:
  trycmd for "a lot of blunt tests (limited test predicates)", `assert_cmd`
  when a result needs careful verification, and cram for language-agnostic
  snapshots. [EVIDENCE] (primary documentation)
  [trycmd crate documentation](https://docs.rs/trycmd/latest/trycmd/)
- **cram is the language-agnostic ancestor**: `.t` files where lines starting
  with two spaces and `$ ` run in the shell, the indented lines after are the
  expected output, `(re)` and `(glob)` suffixes relax a line, and `-i` offers
  to merge actual output back into the test. The current download is 0.7 and
  the page lists Python 2.4 among supported versions, so it is old. [EVIDENCE]
  (primary documentation) [cram](https://bitheap.org/cram/)
- **BATS runs each test in its own process, keeps `$status`, `$output` and
  `$lines` from `run`, gives every test its own `$BATS_TEST_TMPDIR`, and emits
  TAP.** `run -N` expects exit N and `run !` expects non-zero; output goes to
  TAP version 12. It is Bash, so it reads to anyone but runs only where Bash
  does. [EVIDENCE] (primary documentation)
  [bats-core, writing tests](https://bats-core.readthedocs.io/en/stable/writing-tests.html)
- **Gherkin with step definitions is the executable-specification alternative,
  and godog is its Go binding.** godog calls itself "the official Cucumber BDD
  framework for Golang"; each step is a regular expression paired with a Go
  function, "the API is likely to change a few times before we reach 1.0.0",
  and the CLI is deprecated in favour of `go test`. [EVIDENCE] (primary
  documentation) [godog README](https://github.com/cucumber/godog)
- **The case against Gherkin for automation is practitioner experience, not
  measurement.** One consultant reports a finance organisation spending about
  three times its development effort maintaining Gherkin tests before dropping
  them, and argues the step-definition tier is the cost; the author sells the
  alternative, names no client, and offers no data. [ANECDOTE/MARKETING]
  [Why Gherkin always failed with UI test automation](https://agileway.substack.com/p/why-gherkin-cucumber-specflow-always)
  The wider thread of critiques and defences agrees on one thing: the value is
  the conversation and the cost is the glue, so it pays only where
  non-engineers read the scenarios. [CONSENSUS] (several independent posts
  seen in search; none opened beyond the one above, so no further link)
- **Snapshot tests are rubber-stamped when updates are cheap.** The recurring
  practitioner complaint is developers pressing the update button without
  reading the diff, and snapshots inflating coverage without asserting
  anything; the mitigations are small, focused snapshots, excluding dynamic
  values, and reviewing snapshot diffs in code review. [CONSENSUS] (search
  results across several blogs; none opened individually, so no link; treat as
  a known hazard rather than a cited finding)
- **Locally**, the smoke harness already builds the binary once in `TestMain`,
  walks the Cobra tree in-process to discover every command, and the
  write-path smoke runs `capture`, `capture resolve` and `decide` against a
  scratch repository under a fixture `HOME`
  ([evals/README.md](../../../../evals/README.md)); `evals/data/` is empty and
  reserved for itd-75's fixtures. Neither `rogpeppe/go-internal` nor
  `golang.org/x/tools` is in `go.mod`, so either route to txtar is a new
  dependency.

### Options weighed

1. **Go tests per scenario beside the smoke harness** (the current
   `smoke_write_test.go` shape). No dependency; a reviewer must read Go; a
   scenario is a code change, which is the thing itd-75's press release says it
   is not.
2. **testscript files under `evals/`, run by `testscript.Run` from the harness.**
   A scenario is a text file: commands, expected output, the fixture tree, all
   in one; the sandbox (`$WORK`, `HOME=/no-home`) is the "scratch sandbox" the
   mutating-command question asks for; conditions gate a line to one platform.
   A test-only dependency on `github.com/rogpeppe/go-internal`.
3. **The same files, run by an abcd verb** (`abcd evals run` or the harness
   `ahoy` scaffolds), so a managed repository in any language inherits the
   runner from the abcd binary. A runtime dependency on the same module, or an
   in-house interpreter of the txtar grammar plus a small command set.
4. **Gherkin and godog.** Step definitions reintroduce the drift itd-109's
   decision 1 closes; itd-109's own `## SOTA` already failed this at
   fit-challenge, and this pass finds nothing new in its favour.
5. **BATS or cram.** Readable by anyone, but BATS needs Bash on the runner
   and cram is unmaintained; neither carries the fixture tree in the test file.

### Recommendation

**Path 1 for the format, adopted as is: scenarios are testscript files.** The
format answers both intents' open questions at once: one file per scenario
with its fixture tree inside (itd-75's "folder per command or manifest"
becomes "one txtar per scenario, grouped by family"); the fresh `$WORK` and
`HOME=/no-home` sandbox is the default scratch for a mutating verb; a scenario
that needs a real home or network says so with a condition and is skipped
without one; expected output is checked by regular expression or by `cmp`
against a file in the archive, so a reviewer reads the scenario without Go.
Synthetic data is committed in the archive, not generated, which is what makes
it deterministic; a fixture is marked safe by the same rule the repository
already applies to fixtures (the
[secret-shaped-fixtures-at-runtime](../../principles/secret-shaped-fixtures-at-runtime.md)
principle), with the archive's comment block carrying the declaration.

**This adds a Go dependency and needs explicit sign-off**, which weighs two
routes to the same engine: `github.com/rogpeppe/go-internal` (testscript
v1.16.0, maintained) against `rsc.io/script` (the go command's own engine,
published at v0.0.2 and unsupported); test-only if the harness runs it,
runtime if the binary does. If sign-off is refused, **Path 2**: a small in-house runner
over the txtar grammar and a six-command subset (`exec`, `stdout`, `stderr`,
`cmp`, `exists`, `!`), written so every file it accepts is a valid testscript
file, which is the seam: the upstream standalone `testscript` tool runs the
same files, and moving to it later replaces only the runner.

For itd-2609292109475690 specifically: the detector that fails on a family
with no scenario walks the Cobra tree the smoke harness already walks and
requires a non-empty `evals/testdata/scenarios/<family>/`; the suite is its
own tagged lane for run time and failure attribution, not for standing down
(the docs-confined classifier in CI already stands the smoke harness down,
with the race lane and the workflow audit, so scenarios inside that lane
inherit the stand-down either way); and a scenario is a testscript file, not
a Go test, which answers the third open question.

### Local fit risks

- **Minimal dependencies.** Path 1 is a new module in `go.mod`; the sign-off
  rule holds. If the runner must live in the binary for managed repositories,
  it is a runtime dependency with a larger blast radius than a test-only one.
- **Host-agnostic prose.** The scenario format is Go-flavoured by origin;
  docs for a managed repository must present it as "a script file", not as a
  Go thing.
- **One canonical primitive.** The repository would then hold three ways to
  exercise the binary (structural smoke, write-path smoke in Go, scenarios as
  scripts). The plan should say which of the Go write-path smokes migrate to
  scripts and which stay.
- **The user's directory is theirs.** `HOME=/no-home` is the right default; a
  scenario that needs the real home must declare it, never inherit it.
- **Script-first MVP.** Running scenarios from `go test` first (Path 1,
  test-only) and lifting the runner into a verb only when a managed repository
  needs it respects the ratchet.

## itd-173: the verification ladder, and who picks the rung

### Findings

- **Risk-based level selection is how the security standards do it.** ASVS 5.0
  does not prescribe a level: "an organization should analyze its risks and
  decide what level it believes it should be at"; L1 "contains the minimum
  requirements to consider", most applications "should be striving to
  achieve" L2, and L3 is for "the highest levels of security"; levels are
  "defined by priority-based evaluation of each requirement", risk reduction
  against effort. [EVIDENCE] (primary document)
  [ASVS 5.0, What is the ASVS](https://raw.githubusercontent.com/OWASP/ASVS/master/5.0/en/0x03-What-is-the-ASVS.md)
- **A level is a claim about how hard someone looked, not about the thing.**
  Common Criteria's EAL1 to EAL7 run from "Functionally tested" to "Formally
  verified design and tested", and "The EAL level does not measure the
  security of the system itself, it simply states at what level the system
  was tested"; higher levels cost more and take longer. [EVIDENCE] (a
  tertiary summary; the standard itself was not opened)
  [Evaluation Assurance Level](https://en.wikipedia.org/wiki/Evaluation_Assurance_Level)
- **Tracks let a project climb one aspect without blocking on another.** SLSA
  v1.0's Build track runs L0 (no guarantees) to L3 (hardened builds), each
  level naming what it defends against; tracks "recognize progress made in one
  aspect of security without blocking on an unrelated aspect". [EVIDENCE]
  (primary document) [SLSA v1.0 levels](https://slsa.dev/spec/v1.0/levels)
- **The claimed level needs a report that says what was checked.** ASVS's
  assessment chapter: OWASP "does not certify any vendors, verifiers, or
  software"; a report should state "which Level the organization is attempting
  to achieve and which requirements were included", summarise every
  requirement checked rather than only failures, and "Merely running an
  automated tool without thorough testing is insufficient". [EVIDENCE]
  (primary document)
  [ASVS 5.0, Assessment and Certification](https://raw.githubusercontent.com/OWASP/ASVS/master/5.0/en/0x04-Assessment_and_Certification.md)
- **The first security rung already exists on the forge, for nothing.**
  Private vulnerability reporting lets anyone "report security vulnerabilities
  directly and privately to the maintainers" of a public repository; owners
  and administrators enable it; the reporter becomes a credited collaborator
  on a draft advisory and may start a temporary private fork; the maintainer
  can request a CVE without publishing and later publish to the advisory
  database. [EVIDENCE] (primary documentation)
  [Privately reporting a security vulnerability](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/report-privately),
  [About repository security advisories](https://docs.github.com/en/code-security/concepts/vulnerability-reporting-and-management/about-repository-security-advisories)
- **A disclosure policy is the practice the frameworks ask for first.** NIST
  SSDF RV.1.3: "Have a policy that addresses vulnerability disclosure and
  remediation"; PW.7.1 and PW.8.1 each begin with "Determine whether" review
  or testing "should be performed" and which kinds, so the depth is a decision
  the organisation records, not a fixed requirement; PW.7.2 and PW.8.2 ask
  that discovered issues be recorded and triaged "in the development team's
  workflow or issue tracking system". [EVIDENCE] (primary document, read from
  the PDF's practice tables)
  [NIST SP 800-218, SSDF 1.1](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-218.pdf)
- **Scorecard scores the policy, and says what it cannot verify.** The
  Security-Policy check gives 6 of 10 points for a contact address and the
  rest for text about disclosure practice; the Code-Review check is "leveled
  instead of proportional"; the CII-Best-Practices check scores a badge that
  "can report on claims and claim justifications from people", which
  Scorecard cannot check. [EVIDENCE] (primary documentation)
  [Scorecard checks](https://github.com/ossf/scorecard/blob/main/docs/checks.md)
- **The top rung is a review within a stated window.** The OpenSSF Best
  Practices gold level requires "The project MUST have performed a security
  review within the last 5 years", considering "the security requirements and
  security boundary"; it does not require an outside party. [EVIDENCE]
  (primary documentation)
  [OpenSSF Best Practices criteria](https://www.bestpractices.dev/en/criteria)
- **What a second model buys is already measured in this repository's notes,
  and it is not "more of the same".** Panels of frontier judges fall short of
  the best single judge; on code, consensus selection underperforms a naive
  baseline because models converge on the same plausible wrong answer; the one
  measured debiasing effect is re-presenting the work in a fresh session as an
  artefact of unknown authorship; "no one has published a number for
  cross-family benefit on code review specifically"; the widely quoted "40 to
  60 per cent more issues" figure traces to no study
  ([cross-model review note](2026-07-14-cross-model-review.md)). In SWR-Bench
  five runs of one model agreed on fewer findings than different models did,
  so a single-run singleton is mostly noise
  ([model-matrix note](2026-10-16-model-matrix-sota.md)). [EVIDENCE] as
  recorded there; [CONTESTED] on the size of any cross-family gain.

### Options weighed

1. **A fixed level in the framework.** Every standard above rejects this; the
   intent's own ruling rejects it.
2. **A ladder of rungs described by machinery** (one model, two models, two
   providers, a service, a person). This is the ruling's shape. The evidence
   says the second rung (two different models from one provider) buys a
   second opinion, not a second vote; the third adds independence of training
   lineage, and its gain over the second is unmeasured.
3. **A ladder described by stakes, with tracks** (ASVS levels; SLSA tracks):
   the product thinker names what is at risk in their own words, abcd maps
   that to a rung per track (security, correctness, docs), and the rung names
   what it defends against and what it does not.

### Recommendation

**Path 2: a native ladder shaped to ASVS's risk-based choice and SLSA's
per-track progression, with Path 1 for the security floor.** The product
thinker is asked, in the terms of what is at stake (who is harmed if this is
wrong, and how far it reaches), never in the terms of model counts; abcd
records the rung chosen per track as a claim with the ASVS report shape (level
attempted, what was checked, what was not). The security track's first rung
above "built-in" is the forge's private vulnerability reporting plus a
`SECURITY.md` that Scorecard's Security-Policy check would score, which costs
nothing and needs no new relationship; the top rung is a recorded security
review with a date, the Best Practices gold criterion. The seam is the rung
record: one schema (track, rung, what was checked, by whom or what, when)
that a later outside service or paid audit writes into without a new shape.

Where this touches the ruling of 2026-08-29: the ruling lists "two models from
one provider" as a rung, two different models from the same vendor. The
repository's own notes say of that rung that distinct models give a second
opinion, not a second vote (panels do not beat the best single judge), so its
findings are reported beside the first rung's and never merged by vote, and
that the gain of the different-provider rung over it is unmeasured. As a
separate observation, a fresh-session rerun of the same model (the one
measured debiasing effect, over an artefact of unknown authorship, at no cost)
is a cheaper rung below it that the ruling does not list; whether to add it
is the product thinker's. The intent body's "measurably different thing" has
no published support (no cross-family number exists for code review) and
stands as the product thinker's stance, not as evidence. The rung stands as
ruled; the plan should not promise that the third rung finds more than the
second until a run on this repository has measured it.

### Local fit risks

- **Host-delegated LLM work.** The second and third rungs run through the
  host's routing and the per-provider allowlist; the ladder must not become a
  reason to add a provider adapter.
- **Evaluator outside the loop.** A rung's report is a proposal; the product
  thinker's acceptance is the gate (verifier-selects-gates-decide).
- **Less but better.** Three tracks is already a lot for a draft with no
  acceptance criteria; security first, as ruled, and the others only when a
  project asks.
- **Host-agnostic prose.** "The forge's private reporting" must be described
  as a forge feature, not by vendor name, in user-facing docs.

## itd-176: whatever ships says how hard anyone looked at it

### Findings

- **A per-section label with graded sub-levels is the established documentation
  shape.** Node's stability index marks every API section 0 (Deprecated), 1
  (Experimental, with 1.0 Early development, 1.1 Active development, 1.2
  Release candidate), 2 (Stable) or 3 (Legacy), and says what each promises:
  for 1, "Non-backward compatible changes or removal may occur in any future
  release"; for 3, "no longer actively maintained" though still covered by
  semantic versioning. [EVIDENCE] (primary documentation)
  [Node.js, About this documentation](https://nodejs.org/api/documentation.html)
- **A review claim names its scope and disclaims the rest.** The kernel's
  Reviewed-by tag carries a Reviewer's statement of oversight: a technical
  review "to evaluate its appropriateness and readiness", concerns
  "communicated back to the submitter", belief that the patch is "free of known
  issues which would argue against its inclusion", and the disclaimer that the
  reviewer does "not ... make any warranties or guarantees that it will
  achieve its stated purpose"; Tested-by says only that it was tested "(in
  some environment)"; Acked-by is "not as formal". [EVIDENCE] (primary
  documentation)
  [Submitting patches](https://www.kernel.org/doc/html/latest/process/submitting-patches.html)
- **A level is stated with what it defends against and what it does not.** SLSA
  Build L1 "Can be used to prevent mistakes but is trivial to bypass or forge";
  L2 makes forging the provenance "an explicit 'attack', though this may be
  easy to perform"; L3 defends against tampering during the build. The honest
  negative is part of the level's definition. [EVIDENCE] (primary document)
  [SLSA v1.0 levels](https://slsa.dev/spec/v1.0/levels)
- **Self-reported assurance is structured as claim, evidence URL and date.**
  Security Insights v2.2.0, the current specification (v2.0.0, v2.1.0 and
  v2.2.0 have superseded v1.0.0), records an `Assessment` as `name`, `date`
  and `evidence` URL (optional) with a required `comment`, and each
  `SecurityTool` with a required `integration` object of `adhoc`, `ci` and
  `release` booleans plus `results` attestations for each; the 1.0.0 shape of
  auditor name, auditor URL and report year is gone. [EVIDENCE] (primary
  document)
  [Security Insights v2.2.0, schema](https://github.com/ossf/security-insights/blob/main/docs/schema.md)
- **Self-certified claims and machine-verified checks are reported apart.**
  Scorecard scores what it can read from the repository and says the Best
  Practices badge "can report on claims and claim justifications from people",
  which it cannot verify; the badge's criteria carry `{Met justification}` and
  `{Met URL}` annotations per criterion. [EVIDENCE] (primary documentation)
  [Scorecard checks](https://github.com/ossf/scorecard/blob/main/docs/checks.md),
  [OpenSSF Best Practices criteria](https://www.bestpractices.dev/en/criteria)
- **Locally**, the material already exists as receipts: the preflight receipt
  per commit, the docs-fidelity review labelled with the commit it read, the
  intent-fidelity verdict carrying `verifier.id`, `verifier.version`, a rubric
  hash and a prompt hash
  ([agents/intent-auditor.md](../../../../agents/intent-auditor.md)), and a
  changelog composed only from records. What is missing is one line per
  shipped thing that reads those receipts and says which rung (itd-173) was
  climbed.

### Options weighed

1. **A hand-written "reviewed by" line in the changelog.** Refused by
   construction: the changelog is derived, and a hand-written claim is exactly
   the self-certification Scorecard warns it cannot verify.
2. **A badge or a single level number per release.** Compresses three tracks
   into one figure and invites the EAL misreading (a level read as "how
   secure" rather than "how hard anyone looked").
3. **A derived disclosure per shipped intent and per release**, composed from
   receipts: which checks ran, which rung per track, who or what did it (a
   verifier id and version, or a person's role), when, and what it does not
   claim.

### Recommendation

**Path 2: a derived disclosure, shaped to the kernel's statement of oversight
and Security Insights' claim-plus-evidence record.** Each shipped intent and
each release page gets a "looked at" block the binary composes from the
receipts it already holds: for each track, the rung reached, the verifier
(id and version, or role), the commit the check read, and the standing
disclaimer in the kernel's form (free of known issues that would argue against
shipping; no warranty that it achieves its purpose). A rung with no receipt
renders as "not looked at" rather than being omitted, the way the itd-175
ruling names an absent list. The seam: the block is rendered from one record
shape per track that an outside audit (a Security Insights v2.2.0 assessment,
say) writes into with a name, date, evidence URL and comment, and a tool's
row with its ci, release and adhoc flags, so a paid audit lands as one more
row rather than a new section.

### Local fit risks

- **A user-facing change is a record, not prose.** The disclosure must be
  derived from receipts; `launch ship`'s refusal of hand-written unreleased
  prose extends to it.
- **Enforcement claims are facts.** "Reviewed by two independent sessions"
  must be a receipt the binary can show, or the line does not render.
- **Host-agnostic prose.** The verifier id names a model and a host; the
  disclosure's wording in `docs/` must stay generic ("a reviewer model, named
  in the receipt").
- **Less but better.** One block, not a badge wall.

## itd-165: a failed fidelity verdict becomes work somebody can see

### Findings

- **A finding's identity is a fingerprint built to survive edits.** SARIF
  2.1.0: each `fingerprints` value "SHALL be a string that provides a stable
  identifier for the result", "resistant to changes that do not affect the
  logical identity of the result, such as the location of the root of a source
  code enlistment, or the line number where a result appears"; `baselineState`
  marks a result `new`, `unchanged`, `updated` or `absent` relative to a
  baseline run, and if any result in a run carries it, all must. [EVIDENCE]
  (primary document)
  [SARIF 2.1.0 specification](https://docs.oasis-open.org/sarif/sarif/v2.1.0/os/sarif-v2.1.0-os.html)
- **The forge dedups alerts on the fingerprint and closes on disappearance.**
  Code scanning "uses fingerprints to match results across various runs so
  they only appear once"; `ruleId` "has to be the same across analysis"; if the
  file path changes "a new alert will be created, and the old one will be
  closed"; uploads without fingerprints produce duplicate alerts. [EVIDENCE]
  (primary documentation)
  [SARIF support for code scanning](https://docs.github.com/en/code-security/reference/code-scanning/sarif-support-for-code-scanning)
- **A bot that files one bug per recurring failure states its dedup rules in
  the bug.** WebKit's review bot files a flaky-test bug and writes: "The bots
  will update this with information from each new failure", "The bots will
  re-open if the flake re-occurs", and if a person closes it as a duplicate
  "The bots will follow the duplicate chain when making future comments".
  [EVIDENCE] (primary artefact, one example from 2012)
  [WebKit bug 78737](https://bugs.webkit.org/show_bug.cgi?id=78737)
- **Test formats keep "could not judge" apart from "failed", and still count
  it.** JUnit XML distinguishes `failure` ("one of the assertions/checks
  failed") from `error` ("an unexpected error during execution") and
  `skipped`, with separate `failures`, `errors` and `skipped` counts on the
  suite. [EVIDENCE] (a community specification; there is no official JUnit
  XML standard) [JUnit XML format](https://github.com/testmoapp/junitxml)
  TAP 14: a harness "must not treat failing SKIP test points as a test
  failure", nor TODO ones, yet counts each as skipped or todo and may report a
  passing TODO so it can be promoted. [EVIDENCE] (primary specification)
  [TAP version 14](https://testanything.org/tap-version-14-specification.html)
- **Found issues go to the tracker, as a practice.** SSDF PW.8.2: document
  the results of testing, "including recording and triaging all discovered
  issues and recommended remediations in the development team's workflow or
  issue tracking system". [EVIDENCE] (primary document)
  [NIST SP 800-218](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-218.pdf)
- **Locally**, the verdict already carries `receipt_id`, `criterion_id`,
  `verifier` and per-criterion `NOT_MET` / `INCONCLUSIVE`
  ([agents/intent-auditor.md](../../../../agents/intent-auditor.md)); the
  consistency pass's `findingkey.Derive` is the repository's existing
  fingerprint ([findingkey.go](../../../../internal/core/record/findingkey/findingkey.go));
  the capture duplicate check matches prose.

### Options weighed

1. **Dedup on the receipt id alone** (the intent as written). A re-ingest of
   the same receipt cannot mint twice. A second verdict on the same intent
   after a fix attempt carries a new receipt and would mint a second record
   for the same still-unmet criterion.
2. **Dedup on a fingerprint of (intent id, criterion id)**, with the receipt
   id recorded on the record and each later receipt appended as a recurrence,
   SARIF-style. A criterion edited after the receipt changes the criterion's
   content hash, which is itd-109's "stale, never green" rule applied here.
3. **No record; a gate reads the rollup.** Rejected by the intent: the
   ratchet is sequenced behind a corpus of verdicts.

### Recommendation

**Path 2: native records keyed by a fingerprint shaped to SARIF, not by the
receipt alone.** One record per (intent, criterion id) that is `NOT_MET`,
carrying the receipt id that minted it and every later receipt that found it
again; a re-ingest marks the row `unchanged`, a later receipt that no longer
finds it marks it `absent` and leaves the record for the facilitator to
resolve with the receipt as grounds; nothing reopens automatically, which
keeps the itd-170 ruling that a resolved thing stays resolved. `INCONCLUSIVE`
mints nothing and renders as a counted, named outstanding set on the receipt
(the TAP `skipped` shape): the count is never zero-looking, the rows are
listed, and the receipt stays visibly open. The seam: the fingerprint is a
versioned key name, as SARIF's `partialFingerprints` keys are, so a later
change to what goes into it does not orphan existing records. The record id
written back into the audit notes keeps the edge two-sided as the intent
requires.

### Local fit risks

- **One canonical primitive.** The repository would hold three keys: the
  consistency finding key, the capture duplicate match on prose, and this
  one. The plan should say whether the verdict fingerprint is a second use of
  `findingkey.Derive` (class, both ends, quotes) or a sibling with the same
  versioning convention.
- **Reality is filable; a remedy is filed whole.** A record minted by a
  machine must carry grounds (the receipt, the criterion text, the verdict's
  rationale) or the capture gate refuses it.
- **Loud staging.** `absent` is a state on the record, never a silent close.
- **Facilitator mode.** The intent already routes this by adr-2609151528057260;
  the SARIF shape does not change who waits for whom.

## itd-2609291923559186: the consistency pass at the point of writing, with a stable key

The intent is planned, with its decisions ruled and G1 landing first. The
research here checks the key and the hook against outside practice; it
proposes no change to the rulings.

### Findings

- **Fingerprint keys are versioned names, so an algorithm change does not
  orphan old findings.** SARIF 2.1.0's `partialFingerprints` keys are
  hierarchical strings with a version component; a consumer "SHOULD use the
  latest version of the partial fingerprint available in both results", and
  the spec says to choose
  "meaningful property names that describe the information used to compute the
  partial fingerprint" and not to remove existing names. [EVIDENCE] (primary
  document)
  [SARIF 2.1.0 specification](https://docs.oasis-open.org/sarif/sarif/v2.1.0/os/sarif-v2.1.0-os.html)
- **A path change is a new finding on the forge.** Code scanning closes the
  old alert and opens a new one when "the filepath differs for the same
  result"; the key must be computed from paths that stay consistent across
  runs. G1's key, as shipped, already does this: `findingkey.Derive` maps each
  end to an identity path that drops the status folder for intents, specs and
  the issue ledger (`identityPath`, `statusStores`), so a record moved from
  `open/` to `resolved/` keeps its key and the `addressed` marking can find
  it; a record renamed outright is still a new finding under this rule.
  [EVIDENCE] (primary documentation; the repository's
  `internal/core/record/findingkey/findingkey.go`)
  [SARIF support for code scanning](https://docs.github.com/en/code-security/reference/code-scanning/sarif-support-for-code-scanning)
- **A pre-commit hook judges the staged content, and the common framework
  stashes the rest to do so.** pre-commit "only runs on the staged contents of
  files by temporarily stashing the unstaged changes while running hooks",
  because running on unstaged changes "can lead to both false-positives and
  false-negatives during committing"; hooks that do not operate on files must
  be `always_run: true`; a `manual` stage runs only on request. [EVIDENCE]
  (primary documentation) [pre-commit](https://pre-commit.com/)
- **The `addressed` marking has an outside name.** SARIF's `baselineState`
  `absent` (found in the baseline, not now) and `unchanged` are the states the
  A2 hook's report rows would carry if they followed the standard's vocabulary.
  [EVIDENCE] (primary document, as above)
- **Locally**, `findingkey.Derive` (landed 2026-10-12, fixes 2026-10-14)
  already collapses whitespace, drops the status folder from each end's path
  and keys on class, both identity paths and quotes, under a versioned
  derivation domain (`abcd/finding-key/v1`, key prefix `cfk-`, with the
  comment "Bump the version, and every key with it, only when the derivation
  changes"); the model-matrix note records that two models agree on a key
  only when they quote the same text
  ([model-matrix note](2026-10-16-model-matrix-sota.md)).

### Options weighed

1. **The key as shipped** (class, identity paths with the status folder
   dropped, collapsed quotes, a versioned derivation domain). Stable across
   runs on unchanged records and across a status move; changes when a quoted
   sentence is edited, which the code states as its residual risk.
2. **A key on the raw folder path, or an unversioned name.** What the
   research first feared had landed; the code does neither.
3. **SARIF export of the report.** Not needed; the report has one consumer.

### Recommendation

**Path 2, confirmed as shipped.** `findingkey.Derive` already drops the status
folder from each end's path (SARIF's stable-path rule), so a record moved to
`resolved/` keeps its key and the `addressed` rule can find it, and already
versions its derivation domain (`abcd/finding-key/v1`) the way SARIF versions
`partialFingerprints`. The residual risk is the one the code states: a
contradiction quoted differently gets a new key, so an edited sentence reads
as a new finding and the old one as `addressed`; whether the hook's report
pairs a new key with the key it most resembles is the fit-challenge's
question. The hook's "staged records plus their links" scope matches
pre-commit's staged-content rule; the A1 rule checks over the whole corpus
correspond to its `always_run` hooks. No dependency.

### Local fit risks

- **The record lands with the act.** The derivation domain is versioned from
  the first landing; a bump re-keys every record at once, so it is a
  migration, not a tweak.
- **Retire the name.** If the key is re-versioned, the old name is retired
  with a note, not left ambiguous.

## itd-175: the product thinker writes down how this could be wrong, and what would show it

### Findings

- **Assurance cases record doubts as first-class elements, and record the
  decision to accept what cannot be resolved.** Assurance 2.0 assesses
  confidence from three sides: positive (evidence and argument), negative
  (doubts "typically expressed as defeaters"; developers "must guard against
  confirmation bias and should vigorously explore potential defeaters as they
  develop the case, and should record them and their resolution"), and
  residual doubts that may be accepted, where "It is crucial however that
  these judgments are conscious ones and that they are recorded in the
  assurance case". [EVIDENCE] (primary document, abstract and summary)
  [Assessing Confidence with Assurance 2.0](https://arxiv.org/abs/2205.04522)
- **A defeater is a recorded doubt that can be developed into a sub-case that
  confirms or refutes it.** Defeaters "express doubts about aspects of the
  argument"; "human judgement is fallible and prone to confirmation bias" and
  the defence is skeptical debate; there is "merit in recording aspects of
  this discussion for the benefit of subsequent developers and assessors".
  Eliminative argumentation is the contrary approach that refutes every reason
  the claim could be false. [EVIDENCE] (primary document, abstract)
  [Defeaters and Eliminative Argumentation in Assurance 2.0](https://arxiv.org/abs/2405.15800)
- **The premortem is the product-practice form.** Team members "imagine that
  the project they are planning has already failed" and generate plausible
  reasons; it works by "making it safe for dissenters who are knowledgeable
  about the undertaking". [EVIDENCE] (primary article, summary and
  introduction only; the body is paywalled)
  [Performing a Project Premortem](https://hbr.org/2007/09/performing-a-project-premortem)
  Klein's own later account reports a 2010 study with 178 students where the
  premortem group "showed the greatest reduction in (over)confidence" against
  critique and pros-and-cons groups. [ANECDOTE/MARKETING] (the author
  reporting his own study; the study itself could not be retrieved)
  [The Pre-Mortem Method](https://www.psychologytoday.com/ie/blog/seeing-what-others-dont/202101/the-pre-mortem-method)
- **The 30 per cent figure attached to the premortem is not in the 1989
  abstract.** The abstract of Mitchell, Russo and Pennington defines
  prospective hindsight as "generating an explanation for a future event as if
  it had already happened" and reports that temporal perspective had "little
  influence" while outcome certainty changed the kind of explanation given
  (longer, more episodic, past tense). No 30 per cent, no "more reasons". The
  figure circulates through secondary sources and should not appear in a
  record. [CONTESTED] (abstract read on a mirror; the publisher's page
  refused the fetch)
  [Back to the future: temporal perspective in the explanation of events](https://colab.ws/articles/10.1002%2Fbdm.3960020103)
- **The signal is stated before the test.** Hypothesis-driven development
  writes "We believe <this capability> / Will result in <this outcome> / We
  will know we have succeeded when <we see a measurable signal>", and the
  signals "need to be stated before conducting the test to reduce biased
  interpretations of the results". [CONSENSUS] (one named practitioner; the
  pattern recurs across Lean UX and continuous discovery)
  [How to implement hypothesis-driven development](https://www.thoughtworks.com/insights/articles/how-implement-hypothesis-driven-development)
  Continuous discovery says the same: "define success upfront, before you run
  your assumption test" and "draw a line in the sand"; assumptions fall into
  desirability, feasibility, usability, viability and ethical. [CONSENSUS]
  [Assumption testing](https://www.producttalk.org/assumption-testing.md)
- **Locally**, the intent record already carries a Mechanism ("we expect X
  because Y; shown wrong if ...") and a Grounds line ("wrong if ..."), both
  written by the facilitator, and `abcd intent plan` stamps scope conditions
  with persistent ids; the fidelity verdict dispositions each condition as
  survived, narrowed, falsified or untested. What is missing is the product
  thinker's own list, in their words, and a named absence when they decline.

### Options weighed

1. **A formal assurance case** (GSN or Assurance 2.0 tooling). The notation
   is for safety arguments with evidence trees; far too heavy for an intent
   record, and the GSN standard itself could not be retrieved in this pass.
2. **A premortem at acceptance** as a facilitated step. Right shape, but the
   ruling says the list is optional and its absence is named, so a mandatory
   session is out.
3. **A "how this could be wrong" list on the intent**, each entry a doubt plus
   the observation that would show it, optional, with the acceptance stating
   "no list written" when declined; each entry becomes a thing the verdict
   and the itd-170 report from use can later mark.

### Recommendation

**Path 2: a native defeater list, shaped to Assurance 2.0's defeater and
hypothesis-driven development's pre-stated signal.** On the intent, a section
the product thinker fills at acceptance in their own words: for each entry,
how this could be wrong and what would show it; each entry stamped with a
persistent id like a scope condition, so the fidelity verdict can disposition
it (confirmed, refuted, untested) and itd-170 can mark it as having come
true. The acceptance record says "no list written" when the product thinker
declines, as ruled. The seam is the entry: doubt, signal, id, disposition,
which maps onto an Assurance 2.0 defeater if a project ever needs the full
case. No dependency.

### Local fit risks

- **Facilitator default, thinker optional.** The list is the product
  thinker's; the facilitator may draft it but the record must say who wrote
  it.
- **Widen options, never recommend.** The prompt that invites the list offers
  the five assumption categories as examples, not as a checklist to fill.
- **One canonical primitive.** The repository would have Mechanism,
  Grounds, scope conditions and now defeaters; the plan should say how the
  product thinker's list differs from the facilitator's Mechanism (voice and
  author) rather than letting them overlap.

## itd-170: a report from use finds its way back to the promise that predicted it

### Findings

- **Record the prediction with its confidence, then compare, because memory
  rewrites it.** The decision-journal template asks for the expected outcome,
  the reasoning and the probability assigned, and later the actual outcome;
  "It's hard to remember what we knew at the time"; "A decision journal helps
  combat this by recording what you knew and what you thought at the time".
  [CONSENSUS] (a widely copied template; the page is a practitioner's)
  [Decision journal](https://fs.blog/decision-journal/)
- **Signals are named before the test, and read against that naming
  afterwards.** As above: "We will know we have succeeded when ..." stated
  "before conducting the test to reduce biased interpretations".
  [CONSENSUS]
  [How to implement hypothesis-driven development](https://www.thoughtworks.com/insights/articles/how-implement-hypothesis-driven-development)
- **New information is routed back to the decision it bears on, not filed as a
  new item.** The opportunity solution tree exists so that when new
  information arrives, such as a failed experiment, "teams can use their tree
  to quickly revisit those decisions", changing the solution, the opportunity
  or the outcome. [CONSENSUS] (the practice's own glossary)
  [Opportunity solution tree](https://www.producttalk.org/glossary-discovery-opportunity-solution-tree/)
- **A link that carries a fingerprint of what it points at knows when the
  target moved.** The traceability note already recorded Doorstop's link
  fingerprint and the "suspect link" of DOORS and Polarion, and recommended
  write-back as part of the act that changes the target
  ([brief-to-record traceability note](2026-10-04-brief-to-record-traceability-sota.md)).
  [EVIDENCE] as recorded there.
- **"Re-open on recurrence" is the bot convention; the ruling here is the
  opposite, and it is the better record.** WebKit's bot "will re-open if the
  flake re-occurs"; the ruling of 2026-08-29 says a concern that came true
  stays marked and the fix is recorded against it. Reopening destroys the
  thing this intent exists to learn (was the concern worth recording?);
  SARIF's `baselineState` offers the vocabulary for a state that stays
  (`updated`, `absent`) without reopening. [EVIDENCE] (primary artefact and
  primary document)
  [WebKit bug 78737](https://bugs.webkit.org/show_bug.cgi?id=78737),
  [SARIF 2.1.0 specification](https://docs.oasis-open.org/sarif/sarif/v2.1.0/os/sarif-v2.1.0-os.html)
- **Locally**, the verdict's `MET_WITH_CONCERNS` is a prediction with a
  rationale and evidence pointers but no id of its own
  ([agents/intent-auditor.md](../../../../agents/intent-auditor.md)); scope
  conditions do have persistent ids, and itd-175's defeaters would too. A
  report from use has nothing to attach to until a concern is addressable.

### Options weighed

1. **A report from use is a new issue naming the intent.** Rejected by the
   ruling: "Not a bare new item".
2. **A report attaches to the concern it bears on, the concern gains a
   persistent id and a state that stays**, and the fix links to it. The
   decision-journal loop at record granularity.
3. **A calibration score across concerns** (how often a recorded concern came
   true). Useful later and only after a corpus; the ratchet rule applies.

### Recommendation

**Path 2: native, shaped to the decision journal and hypothesis-driven
development's pre-stated signal.** Give each `MET_WITH_CONCERNS` concern an id
at ingest (the fingerprint convention of itd-165, versioned); a report from
use is a record that names the intent, the criterion and the concern id and
carries a state the product thinker sets (came true, did not, not yet known)
that stays set; a fix is recorded against it with a `fixes` edge and the
concern keeps its state, as ruled. The facilitator's trace is then
mechanical: report, concern, verdict, receipt, criterion, intent. The seam:
the report record is one shape whether the concern was the auditor's, a
defeater of itd-175's, or a scope condition's disposition. The calibration
view (how many recorded concerns came true) is deferred until a corpus
exists, as itd-165 defers its ratchet. No dependency.

### Local fit risks

- **The users directory is theirs; agent messaging is local.** A report from
  use is written on the product thinker's machine into the repository's
  record, never posted anywhere on their behalf.
- **Recurrence is signal.** Several reports on one concern are the signal the
  principle names; the record should count them rather than collapse them.
- **Memory graduates to record.** The report is a record from the start, not
  a memory entry that may or may not be promoted.

## Not worth adopting

- **Gherkin and step definitions** for the scenario suite: itd-109's
  fit-challenge already refused the drift they reintroduce; godog is pre-1.0
  with a deprecated CLI; the practitioner case against is loud but unmeasured.
- **BATS** as the scenario runner: it needs Bash on every runner and carries
  no fixture tree in the test file. **cram**: unmaintained.
- **A single assurance level per release** (an EAL-style number): compresses
  the tracks and invites the "how secure" misreading the Common Criteria
  summary warns of.
- **Formal assurance-case notation** (GSN, SACM) for intents: safety-case
  machinery for a one-page record; the standard itself could not even be
  fetched.
- **The "30 per cent more reasons" premortem figure** in any record: not in
  the 1989 abstract.
- **A second copy of the same model as a vote**: the repository's own notes
  show consensus underperforms the best single judge; a second session is an
  independent opinion reported beside the first, nothing more.
- **Automatic reopening of a concern on recurrence**: it erases the outcome
  the record exists to learn.
- **SARIF export** of verdicts or consistency reports: one consumer, no
  ingestion target; the vocabulary (fingerprints, baselineState) is borrowed,
  the format is not.

## Not verified

- The GSN Community Standard version 3 text: the publisher's site returned a
  download stub and the changes document could not be resolved; element names
  and the dialectic extension are known only from search summaries and are
  not relied on above.
- Veinott, Klein and Wiggins (2010) on premortem and plan confidence: the
  paper could not be retrieved (one repository refused, one did not resolve);
  the figures above are Klein's own 2021 account and are tiered accordingly.
- The full body of Klein's 2007 HBR article: only its summary and
  introduction were readable.
- The Mitchell, Russo and Pennington abstract was read on a mirror, not the
  publisher's page, which refused the fetch; the mirror's text is consistent
  with the search snippets of the publisher's listing.
- The exact current wording of NIST SP 800-218's PW.7 and PW.8 was read from
  the practice tables of the PDF as rendered pages; the quotations are from
  those tables.
- Whether `testscript`'s sandbox behaviour and command set are unchanged in
  the version a sign-off would pin: the documentation read is for v1.16.0.
- No controlled study on snapshot-update rubber-stamping or on Gherkin
  maintenance cost was found; both are reported as practitioner consensus and
  anecdote respectively.
