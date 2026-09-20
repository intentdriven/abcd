---
schema_version: 1
id: "iss-2609061432214212"
slug: "abcd-launch-cannot-set-up-the-release-flow-for-a-managed-rep"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "adopting the release flow in a managed macOS app repo, 2026-09-06"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "This asks abcd to set up the release flow for a repository it manages, which is a capability rather than a defect. What the release flow should assume about a managed artefact that is not a plugin, what it should scaffold, and what it should refuse to guess are product questions, and the record lists them as open. One symptom is fixed in this cut: a repository declaring no plugin manifest no longer refuses at an unconditional manifest read, so the changelog verb gives an honest verdict where it previously died. The rest wants the capability designed rather than inferred."
found_at: "internal (launch, changelog, scaffold)"
promoted_to: itd-2609150819432059
---

abcd launch cannot set up the release flow for a managed repo that is not a plugin. Observed adopting abcd in a managed Go macOS app (own tag-driven release workflow building on a macOS runner with minisign, no CHANGELOG.md, no .claude-plugin/plugin.json): 'abcd launch --dry-run' fails with 'include config not found: .abcd/config/launch-payload.json'; 'abcd changelog' and 'abcd launch ship' fail reading .claude-plugin/plugin.json; 'abcd launch scaffold' writes the generic ubuntu Go template (verify/build/publish) that replaces the repo's own release workflow, and nothing creates the pieces the flow presupposes: CHANGELOG.md with its empty [Unreleased] anchor, the launch payload include config, a version location for a non-plugin artefact, and (where detectors are configured) the release-gate manifest and receipts directory. Needed: an adoption step (ahoy install or launch scaffold --init) that lays these down for a managed repo, a way to declare a non-plugin version location and payload, and a template extension point for platform-specific build/publish steps so scaffold parity does not fight a macOS build. Until then a managed repo has to hand-port the template.

## Grounds

- pursued: we expect a managed artefact that is not a plugin to need a declared artefact kind and a release flow shaped by it, because the current flow assumes a plugin at three places and a repository can only say what it is not by failing; it is shown wrong if a single generic flow serves every artefact kind without a declaration

**Corroboration (2026-09-18, Gropius managed-repo session gropiusllm-56, relayed
to abcd-17).** Second managed repository, same wall, at v0.9.0: `abcd launch
--dry-run` and `launch ship` refuse with the same `include config not found:
.abcd/config/launch-payload.json`, and `launch scaffold` does not lay that
include. The cost is now measured: that repository cut three releases in one
day (its 0.6.0, 0.7.0 and 0.7.1) entirely by hand, each as a dated CHANGELOG
heading on a release branch, a script writing `deferred_after` and
`deferral_reason` into the open major records, a pull request, and the tag from
the repository's own auto-release workflow. The release-cut gate (refuse on an
open major captured since the anchor tag unless deferred) therefore never ran
there; the session enforced it by reading the ledger. The ask is the one this
record already carries: the launch verbs for a managed repository that is not a
plugin, or a `launch scaffold` that lays the missing include and says so.
