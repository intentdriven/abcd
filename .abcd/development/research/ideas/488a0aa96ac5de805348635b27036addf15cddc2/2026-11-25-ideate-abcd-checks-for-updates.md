# Ideate verdict — abcd-checks-for-updates

**Verdict: reframed.** Recorded on 2026-11-25 by abcd's idea-admission protocol —
primary-source research, a grill against the existing record, and an
independent adversarial review. This record exists so the idea is not
re-litigated: it stands whether the idea lived or died.

## The idea

let abcd check for updates on its own to ensure consistency across the CLI binary, claude code etc.

## Where the idea came from

No source: the idea entered the gauntlet with nothing behind it but its own words.

## Leg 1 — Primary-source research

Every load-bearing claim checked against its primary source, never a
secondary citation. A claim marked quoted was checked only against the
source the idea came from, which proves the quotation and not the claim.

| Claim | Primary source | Finding |
|---|---|---|
| Copies of abcd on one computer can be at different versions and stay so: an update changes one scope's block at a time. | https://code.claude.com/docs/en/plugins/troubleshooting.md | verified |
| Something outside the host must update the plugin, because the host cannot. | https://code.claude.com/docs/en/plugins/loading.md | falsified |
| The host's own plugin auto-update is off by default for a third-party marketplace, and only a user or an admin turns it on. | https://code.claude.com/docs/en/discover-plugins | verified |
| A named scope's plugin can be updated from the command line. | https://code.claude.com/docs/en/plugins/cli-reference.md | verified |
| A project-scope update reaches the copy a session loads; users report that it does not, with no maintainer's reply. | https://github.com/anthropics/claude-code/issues/92121 | unverifiable |
| Which of two scopes' copies loads can be predicted from the documentation. | https://code.claude.com/docs/en/plugins/loading.md | unverifiable |
| A running hook can tell which installed version it belongs to, by its root. | https://code.claude.com/docs/en/plugins-reference | verified |
| No implicit path of abcd's reaches the network today: the bootstrap's resolve of the latest release for a fresh plugin root, from a hook, is one that does. | itd-132 | falsified |
| Updating the parts one at a time yields one version: abcd update refuses a plugin root. | itd-130 | falsified |
| Comparable tools check on their own by default, as a notice: gh asks at most daily, on a terminal only, with an off switch. | https://cli.github.com/manual/gh_help_environment | verified |
| Comparable tools install a newer copy of themselves from an unrelated command by default. | https://raw.githubusercontent.com/jdx/mise/main/settings.toml | falsified |
| Signed provenance makes a release safe to take unasked. | https://slsa.dev/spec/v1.2/threats | falsified |
| An artefact attestation is a guarantee that the artefact is secure. | https://docs.github.com/en/actions/concepts/security/artifact-attestations | falsified |
| A monotonic version mark removes rollback, and only expiring signed metadata detects a freeze. | https://theupdateframework.github.io/specification/latest/ | verified |
| Guidance favours automatic patching and also names its risk: automation is advised, and a patch could itself be compromised. | https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-40r4.pdf | verified |
| Update channels of developer tools are attacked in practice: a stolen publish token shipped an update for about eight hours. | https://github.com/cline/cline/security/advisories/GHSA-9ppg-jx86-fqw7 | verified |
| A version check can itself carry an attack, where it installs what it checks. | https://github.com/nrwl/nx/security/advisories/GHSA-cxm3-wv7p-598c | verified |

## Leg 2 — Record grill

Does the brief, an intent, an ADR, or a principle already cover,
contradict, or supersede this idea? Every hit is cited by record id, and
every id resolved in this repository when the verdict was recorded.

| Record | Relation | Note |
|---|---|---|
| adr-38 | contradicted | Accepted 2026-08-15. Names automatic update discovery and says it needs superseding in the open. Brief invariant 7 states the same. Its recorded reasons are privacy, consent, a bound on what may arrive, and testability; supply-chain compromise, a hostile network, and an update that changes what judges an agent are not among them. |
| itd-130 | contradicted | Ambient discovery in any form is out, unless adr-38 is superseded in the open. |
| itd-111 | covered | Staleness detected from disk only; it does not compare the copies on one computer. |
| itd-105 | contradicted | The host's own auto-update is the person's choice; abcd does not switch it. |
| adr-2610050711287503 | contradicted | abcd writes one entry in the host's user settings and no other. |
| itd-2610031026190632 | contradicted | A daily check at session start for the host's version was put and declined on 2026-10-03. The calibration note records it twice. |
| adr-2611041815445203 | contradicted | abcd looking online and updating a repository's tools by itself was rejected; proposed. |
| adr-2610241057124151 | contradicted | A tool abcd installs updates only when abcd does. |
| itd-132 | covered | The accepted gap: a release with no plugin update is never noticed. |
| itd-2611062149308723 | covered | A drive takes each release on the yes given at its start, and nothing else does. |
| adr-2611062149324491 | covered | The drive's exception; still proposed. |
| iss-2611251742484774 | covered | The same request in the ledger. |
| iss-2610290709017008 | covered | Whether a session start may ask for the newest release: an owed ruling. |
| iss-2611251739422396 | covered | Name every installed copy and which one runs. |
| iss-2610241228359748 | covered | No stated precedence among the copies. |

### Searches

| Terms | Places | Count |
|---|---|---|
| adr-38 | .abcd/development/; .abcd/work/ | 54 |
| auto-update | .abcd/development/; .abcd/work/ | 9 |
| version check; update check | .abcd/development/; .abcd/work/ | 38 |
| self-updat | .abcd/development/; .abcd/work/ | 17 |
| precedence | .abcd/development/; .abcd/work/ | 43 |
| plugin scope; user scope; project scope | .abcd/development/; .abcd/work/ | 85 |
| supply chain | .abcd/development/; .abcd/work/ | 11 |

## Leg 3 — Adversarial review

Conducted fresh-context and off-policy by an evaluator that did not carry
out the research and received the idea as an artefact of unknown
authorship — the evaluator-outside-the-loop principle applied to ideas.
Each reviewer's attempts, verdict and sources are kept under its own label.

Current state: ignored. The reviewers judged the idea as an ambition for
the product this repository builds: a gap between the idea and what the
product ships today was out of scope as a kill, and reads as what the idea
implies building.

### Reviewer reviewer-a (security and trust) — reframed

- **survived** — The check itself runs code, as one tool's version check did: reading release metadata alone runs none.
- **survived** — A notice in an agent's context is an instruction to update: the host has a channel to the person alone.
- **partial** — Disclosure is part of any check made on its own, and invariant 7 says abcd asks the network nothing on its own; a standing yes answers both.
- **survived** — A frozen answer reads as current: detectable with signed, expiring release metadata.
- **partial** — A check alone ensures nothing, and the comparison needs no network.
- **partial** — No control removes a bad change built by the genuine workflow, and applying unasked drops the person's act as the second checkpoint.
- **partial** — The host decides when a new copy loads, and its auto-update switch is the person's.
- **partial** — A release that runs and judges wrongly passes any proof that does not test that rule.
- **partial** — Consent stored on the account can be written by anything running as the account.
- **survived** — Rollback: a mark that never goes down refuses an older release.
- **survived** — Every new trigger widens the reach of a fault in the update path: answered by a check that shares no code with applying.
- **survived** — Against the status quo: only the latest release is fixed, so not updating is a risk of its own.
- **partial** — abcd cannot ensure anything for copies it does not own; there the word can only mean report.
- **partial** — The updater ships inside what it updates: one bad release can switch off every later one.

What the idea implies building:

- a disk-only comparison of every copy that says which one runs
- a bootstrap that provisions the release its own plugin version names
- adr-38 and invariant 7 superseded or refined in the open, and adr-2611062149324491 accepted or revised
- notices on the channel to the person alone
- the consent switches held to the person
- verification in the bootstrap that rests on no signed-in tool
- a verified route and a mark for the plugin copy
- freeze detection, or at least the age of the last check that worked
- a minimum age, with a withdrawal inside it honoured
- a proof before a new binary judges, a kept previous copy, and a fallback to it
- a second checkpoint on the publishing side
- a receipt naming the release that judged each session
- one transport without overrides for every check
- one line for copies abcd does not own

Sources consulted: https://code.claude.com/docs/en/hooks, https://code.claude.com/docs/en/plugins/security.md, https://slsa.dev/spec/v1.2/threats, https://theupdateframework.github.io/specification/latest/, https://github.com/nrwl/nx/security/advisories/GHSA-cxm3-wv7p-598c, adr-38, adr-2611062149324491, adr-2609231048308186, itd-2610031026190632.

### Reviewer reviewer-b (necessity and fit) — reframed

- **partial** — A check finds a newer release; agreement comes from comparing and bringing every copy to one version, so the wording names a means for an end.
- **partial** — The host owns the plugin copies and the reload: a limit on the ambition.
- **partial** — With one copy there is nothing to agree with, so the ambition is currency only.
- **partial** — The field's terminal-only gate would silence it: abcd has a channel the field lacks, at a session's start.
- **partial** — Applying a release replaces the guard that judges a running agent, though the drive already takes releases on a yes.
- **partial** — The host's updater converges on the plugin half; it never updates the binary on PATH or a source build, so agreement across owners stays abcd's.
- **partial** — A checker is kept for ever.
- **survived** — What only the network can say: a terminal-only install whose person never asks.
- **partial** — Coherence with invariant 7: the words contradict it, so the idea forces a ruling.
- **survived** — Is the need real: itd-111's founding incident and the drive's record say it is.
- **partial** — The reach of the last words is unstated: the plugin copy, the host's own version, other hosts, or a team's machines.
- **partial** — A package manager and the host may own every copy in time; what lasts is knowing whether they agree.

What the idea implies building:

- an inventory of every copy on the computer that names which one answered
- a notice when a session is on a superseded plugin root, with the reload step
- a bootstrap pinned to its plugin's own version, with the cache keyed by tag
- a surface-against-binary comparison that works on version-named roots (iss-206)
- one verb that brings every copy abcd owns to a chosen release and prints the host's step for the rest
- install docs with the scope and reload steps and the host's auto-update switch
- a standing-consent surface, an interval, an off switch, and gates, if currency is wanted unprompted
- a boundary rule for applying outside a drive
- a minimum abcd version a repository declares, if the ambition reaches a team

Sources consulted: https://code.claude.com/docs/en/plugins/loading.md, https://code.claude.com/docs/en/discover-plugins, https://code.claude.com/docs/en/plugins/cli-reference.md, https://raw.githubusercontent.com/cli/cli/trunk/internal/update/update.go, adr-38, adr-2611062149324491, itd-105, itd-111, itd-130, itd-132, iss-206.

The reviewers agree, so no ruling was needed.

## Rejected alternatives

- **A check on by default, with an off switch.** — Disclosure is part of it, and adr-38 rejects the pattern by name; a standing yes answers both.
- **One yes that covers both checking and applying.** — The two carry different threats: a check's can be answered, and applying's only reduced.
- **Applying a release mid-run, as soon as it is found.** — It replaces what judges a running agent.
- **abcd switches on the host's auto-update.** — The setting is the person's (itd-105, adr-2610050711287503).
- **Refusing to run where the copies differ.** — A guard that refuses either fails open or stops all work, with the remedy in the host.
- **Downloading ahead and applying on a person's act.** — It needs the same discovery, and leaves a staged copy on an account an agent's stages share.
- **Documentation alone.** — The fault could not be seen from inside the session.

## The reframing

Two ambitions, worded apart. First, abcd knows every copy of itself on a computer, says which one answered, and says when they disagree, with the one step that mends each; it compares from disk alone, at a session's start and where abcd reports its version, and needs no change to adr-38. Second, every copy is current without the person coordinating them: abcd brings the copies it owns to one release and names the owner's own step for the rest. Looking for a newer release unprompted is the only part that touches adr-38 and invariant 7, and it takes a person's ruling: on a standing yes the person gave and can withdraw, abcd asks at most daily, reads release metadata alone, and tells the person alone. Applying is a second, separate yes: one verified release, at a boundary, after a minimum age and a passing proof, keeping the previous copy. It must never install or execute anything as part of a check; put a notice or a remedy command into the model's context; run in the per-command hook; send a version, an identifier, or a token; take either yes from a repository file, an environment variable, or an agent; switch the host's auto-update itself; vouch for bytes it did not verify; apply mid-run, a pre-release, or anything older than the mark; or report current when the check failed. On the security question: every threat of a check can be answered or reduced to a wrong notice; those of applying can be reduced and not removed (a bad change built by the genuine workflow, a release that judges wrongly, and a bad release that disables its own successor stay).

## What follows

The idea as posed does not survive, but the reframing recorded above
does. Any graduation to a draft intent carries the reframing, not the
original wording.
