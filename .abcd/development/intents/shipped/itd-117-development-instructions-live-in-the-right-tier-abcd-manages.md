---
id: itd-117
slug: development-instructions-live-in-the-right-tier-abcd-manages
spec_id: spc-23
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: []
severity: major
impact: additive
---

# A Machine's Shared Conventions Are Declared Once In The User Scope — And Every Managed Repo Inherits Them

## Press Release

> **abcd gains a user-scope rules layer.** The conventions that are yours rather
> than abcd's — your definition-of-done wording, your attribution example, your
> house domains — now live once in `~/.abcd/rules.json` and inject into every
> repo abcd manages on that machine. They layer between abcd's bundled default
> domains and each repo's own `.abcd/rules.json`: the bundled opinions are the
> floor, your machine conventions refine them, and a repo override still wins.
> Nothing is discovered, nothing is grouped, and no directory is occupied: the
> user scope already exists for exactly this kind of machine-local shared state.
> A machine with no user rules file behaves precisely as it does today.

> "I had the same three rules copy-pasted into several repos, and they'd already
> drifted — one spelled the trailer one way, another another," said Carol,
> maintaining a set of sibling projects. "The bundled defaults covered most of
> it, but the places where my house style differs from the tool's were exactly
> the places I was repeating myself. Now the delta lives in one file. When I
> change how we word a rule, I change it once, and the repo that genuinely needs
> to differ still overrides it locally."

## Why This Matters

A three-tier audit of a working machine (user-global harness config, the
directory holding several sibling repos, and each repo) found the shared
conventions duplicated in prose across repos and already drifting — the same
attribution rule stated in several places with divergent examples, the same
git-safety and definition-of-done beats restated repo by repo — while other
repos carried no instruction file at all and inherited none of it.

The honest scope is narrower than that framing suggests, and this is the
correction an adversarial review forced. **abcd's plugin-bundled default
domains already ship most of the duplicated prose**: `COMMITTING` carries the
`Assisted-by` trailer rule, "never `Co-Authored-By` for AI", "never commit or
push unless asked", and "branch + PR"; `PII` carries the privacy conventions.
A repo that installs abcd inherits all of it with no new machinery. So the
unguarded repos are not an argument for a new tier — they are an argument for
installing abcd in them.

What genuinely has no home is the **delta between abcd's opinions and a
particular machine's house style**: the specific wording of a definition of
done, the model-id used in an attribution example, persona names, and any
custom domain a person wants across their own projects but that abcd has no
business bundling for everyone. Today that delta can only be expressed by
editing every repo's `.abcd/rules.json`, which is precisely the duplication
that drifts.

The user scope is the right home, and it already exists. The brief records
exactly two scopes — user (`~/.abcd/`) and repo (in-tree `.abcd/`) — with the
user scope reserved for "state that cannot live in any one repo's tree because
it is shared across every abcd-managed repo on the machine". Machine-wide rule
conventions are that state. This adds a layer to an existing scope rather than
inventing one: **no grouping layer, no workspace registry, no directory
designated or occupied** — the stance shipped intent itd-40 locked, left
intact. The rule loader's existing `Merge` already composes layers, so the
mechanism is `Merge(Merge(defaults, user), repo)`.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a machine with no `~/.abcd/rules.json`, **when** the rule loader
  runs in any repo, **then** the injected rules are byte-identical to today's
  and no new file or directory is created anywhere — absence is the default and
  costs nothing.
- **Given** a `~/.abcd/rules.json` declaring a domain field that a bundled
  default domain also declares, **when** rules inject in a managed repo with no
  repo-level override of that field, **then** the user-scope value is injected
  in place of the bundled default.
- **Given** the same field also overridden in that repo's `.abcd/rules.json`,
  **when** rules inject, **then** the repo value wins — precedence is bundled
  defaults, then user scope, then repo, with each layer overriding per field by
  the same wholesale replacement semantics `mergeDomain` already applies.
- **Given** a `~/.abcd/rules.json` declaring a custom domain abcd does not
  bundle, **when** a prompt recall-matches that domain's keywords in any
  managed repo on the machine, **then** its rules inject there without that
  domain being declared in the repo.
- **Given** a malformed, oversize, or symlinked `~/.abcd/rules.json`, **when**
  the loader reads it, **then** it is refused through the same trust-boundary
  guards the repo-scope file already passes, and the failure is reported
  loudly rather than silently skipped — an unreadable user layer never
  degrades to a silent partial injection.
- **Given** a repo where rules injected, **when** the user runs `abcd rules`,
  **then** each rendered domain shows which layer it came from (bundled, user,
  or repo), so a rule's provenance is visible rather than inferred.
- **Given** the repo kill switch or a `dormant` state set at repo level,
  **when** the user scope declares the same domain active, **then** the repo's
  suppression still wins — the existing kill switch is not weakened by a layer
  above it.

## Open Questions

- **Surface for editing**: is `~/.abcd/rules.json` hand-edited only, or does a
  verb scaffold/validate it (`abcd rules --user`?)? A read-only render of the
  effective merged set may be enough for the first slice.
- **Precedence granularity**: `mergeDomain` replaces a domain's fields
  wholesale (a partial rules-array override is not expressible today). Three
  layers make that coarser grain more visible — is per-field wholesale
  replacement still the right semantics, or does the third layer justify
  finer-grained merging? Recorded as a known limitation, not a blocker.
- **Deferred to a follow-up intent**: detecting that a repo's rules duplicate
  the user layer (so the user can delete the copy). The motivating duplication
  is near-verbatim *with drift*, so detection is semantic-similarity work — a
  host-delegated oracle, not a string match, and too large for this slice.
- **Deferred**: whether conventions currently living in per-project harness
  memory should migrate into the user-scope record.
- **Out of scope, recorded for clarity**: any directory-designated workspace
  or repo-grouping layer. The maintainer chose the user scope precisely to
  avoid reversing itd-40; if machine-wide ever proves too coarse a grain, that
  is a separate intent with its own ADR.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-056068d9886e -->
Fidelity review — receipt rcp-056068d9886e (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:9776608c18287af8c45be5db9e7b5422040542f74fd82352aadad2a98cac5b6f
Input attestations: diff:tree at 4c09b5c749de3dc3d1a1e0ab85d9dcd58ffcb4d5 (main lineage, itd-117 shipped)@-;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Load returns Defaults() unchanged when neither file is present and readUserLayer treats an absent path as no layer; TestUserLayerAbsentChangesNothing asserts DeepEqual with Defaults(), an unchanged repo merge, and an unchanged user-scope tree listing before and after
  evidence: internal/core/rules/rules.go:239 — "if !haveUser && !haveRepo {"
  evidence: internal/core/rules/rules.go:341 — "if refusal == fsutil.DeclarationAbsent && (os.IsNotExist(err)"
  evidence: internal/core/rules/user_layer_test.go:88 — "Load created something in the user scope"
- ac-2 — MET: the user layer merges onto Defaults() through mergeFrom with SourceUser before the repo layer; TestUserLayerOverridesBundled asserts the user rule is injected in place of the bundled one, the source reads user, and unset fields inherit the bundled value
  evidence: internal/core/rules/rules.go:244 — "merged = mergeFrom(merged, user, SourceUser)"
  evidence: internal/core/rules/user_layer_test.go:158 — "func TestUserLayerOverridesBundled(t *testing.T) {"
- ac-3 — MET: the repo layer merges last through the same Merge/mergeDomain path, so per-field wholesale replacement is unchanged; TestRepoLayerOverridesUser asserts the repo's rules win and a field the repo does not set survives from the layer below
  evidence: internal/core/rules/rules.go:255 — "merged = Merge(merged, repo)"
  evidence: internal/core/rules/rules.go:563 — "func mergeDomain(base, over Domain) Domain {"
  evidence: internal/core/rules/user_layer_test.go:183 — "func TestRepoLayerOverridesUser(t *testing.T) {"
- ac-4 — MET: a user-declared custom domain is added by mergeFrom as a new key and recall-matches in a repo with and without a rules file; TestUserCustomDomainInjectsInAnyRepo asserts the match, the injected text and the user-override label
  evidence: internal/core/rules/rules.go:548 — "out.origins[name] = source"
  evidence: internal/core/rules/user_layer_test.go:212 — "func TestUserCustomDomainInjectsInAnyRepo(t *testing.T) {"
- ac-5 — MET: readUserLayer reads through fsutil.ReadDeclaration and turns not-regular, writable-by-others, foreign-owner, oversize, symlinked-leaf and symlinked-~/.abcd into errors naming the file; Load returns an empty set with the error, the hook prints it and injects nothing; TestUserLayerRefusalsAreLoud covers malformed, oversize, symlinked leaf, FIFO, symlinked dir and group-writable, and TestUserLayerRefusalIsNotAPartialSet asserts no domains survive a refusal
  evidence: internal/core/rules/rules.go:335 — "data, refusal, err := fsutil.ReadDeclaration(path, maxRulesFileBytes)"
  evidence: internal/core/rules/rules.go:344 — "rules: ~/.abcd is a symlink (refusing to follow it to %s)"
  evidence: internal/core/rules/rules.go:357 — "exceeds the %d-byte cap"
  evidence: internal/surface/cli/cli.go:1371 — "fmt.Fprintf(cmd.ErrOrStderr(), "abcd %v; injecting nothing\n", err)"
  evidence: internal/core/rules/user_layer_test.go:240 — "func TestUserLayerRefusalsAreLoud(t *testing.T) {"
  evidence: internal/core/rules/user_layer_test.go:364 — "func TestUserLayerRefusalIsNotAPartialSet(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: every resolved domain carries a Source of bundled, user or repo and the rendered heading labels user and repo overrides; TestProvenanceNamesAllThreeLayers asserts the three sources through Lookup and the two labelled headings. Concern: the text render of `abcd rules` shows a bundled domain bare, so the bundled layer is conveyed by the absence of a marker rather than a word; the explicit bundled value is on the --json surface only. spc-23 AC6 specifies labels for overridden domains, so the shape is the spec's, not a drift
  evidence: internal/core/rules/rules.go:978 — "func Label(name, source string) string {"
  evidence: internal/core/rules/rules.go:983 — "return name"
  evidence: internal/surface/cli/cli.go:1948 — "An untouched bundled domain renders bare and carries "source": "bundled""
  evidence: internal/core/rules/user_layer_test.go:382 — "func TestProvenanceNamesAllThreeLayers(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-23-a-machines-shared-conventions-are-declared-once-in-the-user-s.md:82 — "the CLI test asserts the label for a domain overridden at each level"
- ac-7 — MET: the repo layer merges after the user layer so its dormant state replaces the user's active one, and the kill switch is sticky through mergeFrom; TestRepoSuppressionBeatsUserActivation asserts a repo-dormant domain stays dormant and a repo kill switch suppresses the user's custom domain
  evidence: internal/core/rules/rules.go:538 — "out.killedBy = append(out.killedBy, source)"
  evidence: internal/core/rules/user_layer_test.go:411 — "func TestRepoSuppressionBeatsUserActivation(t *testing.T) {"

Gap audit:
- honoured:
  - one user-scope file layers between bundled defaults and the repo file with no new directory occupied
    evidence: internal/core/rules/rules.go:229 — "func Load(repoRoot string) (RuleSet, error) {"
  - a machine with no user rules file behaves precisely as before
    evidence: internal/core/rules/user_layer_test.go:62 — "func TestUserLayerAbsentChangesNothing(t *testing.T) {"
  - an unreadable user layer fails loudly, never as a partial injection
    evidence: internal/core/rules/user_layer_test.go:364 — "func TestUserLayerRefusalIsNotAPartialSet(t *testing.T) {"
- diverged:
  - provenance is visible for every domain: the text render marks user and repo overrides and leaves a bundled domain bare, with the explicit bundled source on --json only
    evidence: internal/core/rules/rules.go:983 — "return name"
- missing: (none)
