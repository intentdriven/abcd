---
schema_version: 1
id: "iss-2608310912203724"
slug: "a-record-authored-straight-into-resolved-never-invokes-captu"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-179-fidelity-audit"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues"
---

a record authored straight into resolved never invokes capture resolve so the grounds obligation reaches only records that pass through the verb

Found by the itd-179 fidelity audit and verified at integration HEAD.

itd-179 makes `capture resolve|wontfix|promote` refuse without grounds. That
obligation reaches exactly the records that pass through those verbs.

Most do not. Measured at HEAD: **689 of 715 records in `resolved/` carry no
`## Grounds` section, and all 689 carry a `resolution:` frontmatter key** --
authored directly into the terminal folder rather than transitioned into it. The
auditor's scoped measurement is sharper: 95 records entered `resolved/` after
the refusal landed on this branch, and 69 of those carry no grounds.

The distinction that matters: **the gate is not bypassed, it is never invoked.**
A refusal on a verb cannot bind a workflow that does not call the verb, and
authoring a record straight into `resolved/` is a normal thing to do in this
repository -- the branch that shipped the obligation is itself the largest
producer of grounds-less terminal records.

Related but distinct from iss-2608301747006182, which observes that no GATE
requires a terminal record to carry grounds. This record is about why arming
that gate would be a bigger change than it looks: it would refuse the majority
of the existing corpus, and the population it would refuse was created by the
ordinary working practice of this repo.

**RULED 2026-08-31: a forward-only gate.** A terminal-folder record must carry a
grounds entry if it was created after the gate is armed; the existing corpus is
exempt.

The precedent is this cycle's own: the intent-side grounds gate was promoted
forward-only and left 36 intents NOT READY without refusing any of them. The
same shape applies here, and for the same reason — the population that would be
refused was produced by the ordinary working practice of this repository, so
refusing it retroactively punishes the practice rather than changing it.

**The cutover is the ARMING COMMIT, not a date.** A date written in prose is
exactly the fact itd-195 says not to state: it cannot fail, and it decays the
moment anyone misremembers when the gate landed. Whoever arms the gate records
the commit it was armed at, and the gate compares against that.

Rejected, with reasons kept: narrowing the claim and adding no gate at all,
which is honest but leaves roughly three-quarters of new terminal records
escaping an obligation the intent's press release implies is universal; and
forcing the verb to be the only door, which would make the obligation genuinely
universal but needs 689 records migrated and fights how this repo is worked
daily — too large a change for what this finding establishes.

Needs a record home before it is built. It is a capability change on itd-179's
surface, so it is either that intent's follow-up or a new one; that routing is a
decomposition hand-run nobody has done.

