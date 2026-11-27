---
id: adr-2611262044127943
slug: the-project-s-committed-facilitator-mode
status: accepted
date: 2026-11-26
supersedes: null
superseded_by: null
reverses: [adr-2610150603484802, dcn-2611252040395990]
refines: [adr-2609151528057260]
related_intents: [itd-2611252049031692, itd-2610150602547271, itd-97, itd-2611252117557299]
related_rfcs: []
related_adrs: [adr-2610150603484802, adr-2609151528057260, adr-2611011332427073]
---

# ADR-2611262044127943: The project's committed facilitator mode decides, then the person's global setting, and nothing else

Typed links: `reverses` in part
[adr-2610150603484802](2610150603484802-a-fresh-install-s-facilitator-is-a.md)
(two of its clauses: that `human` from any layer wins over `auto`, and that the
declaration lives in `.abcd/config.json` alone), and `reverses` in part
[dcn-2611252040395990](../../../work/decisions/dcn-2611252040395990-the-product-thinker-ruled-on-2026-11-25.md)
(its per-repository personal setting, and its leaving "human from any layer
wins" untouched); `refines`
[adr-2609151528057260](2609151528057260-three-roles-who-each-artefact-addresses.md)
(its technical facilitator is a mode a role can be in; this says who decides
the mode, and that auto is the whole role).

## Context

[adr-2610150603484802](2610150603484802-a-fresh-install-s-facilitator-is-a.md)
made a person the facilitator of every repository until it says `auto`, read
`facilitator.mode` from the layered configuration, and let `human` from any
layer win. So one committed line spoke for everyone who works in a project,
the computer's file could only take the role back, and a project had no way to
leave the choice to each contributor.

On 2026-11-25 the product thinker ruled that the mode is a person's own
setting and that the status line says whose it is (dcn-2611252040395990), and
then how a project and a person stand to each other (dcn-2611252132144530): a
project enforces auto, prevents it, or leaves it to each contributor; what the
project sets is above what the person set; and an agent must not be able to
change the project's value. The draft of itd-2611252049031692 read that as
three places: the project's value, a person's own entry for one project, and
the computer's file as the person's default.

dcn-2611252040395990 had held the mode as a personal setting for each
repository, and had said that adr-2610150603484802's "human from any layer
wins" was left untouched.

On 2026-11-26 the product thinker ruled again, and more simply. Their own
words, so that no later reader has to take the reversal on trust:

- Asked whether two clauses of adr-2610150603484802 go, "human from any layer
  wins over auto" and the setting living only in `.abcd/config.json`: "yes,
  both go".
- On the per-repository personal setting: "I think we should remove the
  personal per-repo option -- it's global and enforced per repo only; simpler;
  if we ever need a third option, we can revisit. For now, we don't".
- On abcd never merging a change to the repository's committed mode line, the
  person merging it on the forge by hand (dcn-2611252138203673): "confirmed".
- On whether a person's own `auto` merges a colleague's pull request: "only
  your own".
- On the plan of itd-2611252049031692: "yes, I sign the plan off".

## Decision

We will decide the technical facilitator's mode in one order, through one
resolver that every reader asks:

1. **What the repository enforces**, `facilitator.mode` in `.abcd/config.json`
   as committed on its default branch: `auto` enforces auto for everyone who
   works in the project, and `human` prevents it for everyone. This beats
   everything else. A branch or a working tree that holds either value changes
   nothing; the default branch's tip is read as the object store holds it.
2. **Otherwise the person's global setting**, `facilitator.mode` in
   `~/.abcd.noindex/config.json`, read only under the home the account database
   names for the account. Under any other `HOME`, a hosted session's among
   them, a person has no setting.
3. **With neither, a person is the facilitator** (`human`), as
   adr-2610150603484802 has it for a fresh install.

There is no setting of a person's for one project alone, and nothing else (a
flag, an environment variable, a file in a checkout) sets `auto` or decides the
mode. abcd never arms or merges a pull request that would change the
repository's committed `facilitator.mode` line, in any mode and whatever
approvals it has; the person merges it on the forge's page by hand
(dcn-2611252138203673). Setting the person's global value is the person's own
act: the guard refuses `abcd facilitator auto|human` to an agent, and a shell
line or a host file tool that writes, moves or removes the global settings
file. A person's own `auto` merges only pull requests their own account
opened.

What stands of adr-2610150603484802: a person is the facilitator until told
otherwise, and abcd's own repository commits `auto`. What is reversed: `human`
from any layer no longer wins (an enforced `auto` holds over a person's own
`human`), and the declaration is no longer the repository's file alone. Of
dcn-2611252040395990, the setting held for each repository is reversed, and so
is its leaving "human from any layer wins" untouched; that the mode is a
person's own setting and that the status line says whose it is both stand.

Ruled by the product thinker, 2026-11-26 ("yes, both go").

## Alternatives Considered

- **Two places, the project's then the person's global** (chosen): one order
  a person can say in a sentence, one file of their own to set, and nothing
  per repository to keep in step. Cost: a person who wants `auto` in one
  project and `human` in another cannot have it unless the project commits
  one of them.
- **Three places, with a person's entry for each project** between the two,
  selected by the identity record and matched on the project's first commit
  and forge address: what the draft built first. Rejected by the product
  thinker as more than is needed now ("For now, we don't"); it brought an identity check, a fork rule
  and a kept set of project facts, each a way for the row and the gate to
  disagree. It can be revisited if a third option is ever needed.
- **`human` from any layer wins**, as before: rejected on 2026-11-25, since a
  project could not enforce `auto` for everyone who works in it.
- **The project's file alone**: rejected on 2026-11-25, since a project that
  says nothing would then leave no one any choice.

## Consequences

- The merge gate, the guard's judgement of a merge command, the prompt line,
  the board and the status row read the mode through the one resolver, and a
  test fails on a second reader of `facilitator.mode` beside it.
- A project that commits `auto` is auto for a person whose global setting is
  `human`, and under a home abcd refuses; that is the breaking part of
  itd-2611252049031692.
- The merge gate holds both rules beside the order. A person's global `auto`
  arms only a pull request the forge account abcd runs as opened, and judges
  anyone else's as `human` judges it, waiting for an approval of its head or
  for its author to land it (dcn-2611262202295309, superseding
  dcn-2611262048349309). A pull request whose diff changes the committed
  `facilitator.mode` line is armed in no mode, and the person merges it on the
  forge's page (dcn-2611262202306256). The shell guard does not yet hold
  either rule for a merge command an agent types, and an arming of abcd's own
  that already stands at such a head is not yet taken back off; both are
  spc-2611262159212693.
- The verb itd-2611252049031692 plans, `abcd facilitator auto|human`, is to
  write only `~/.abcd.noindex/config.json` and take no flag that chooses where.
  It is not built: until it is, a person sets `facilitator.mode` in that file
  by hand, and a project's in its committed `.abcd/config.json`.
- [adr-2611011332427073](2611011332427073-abcd-resolves-every-layered-setting-by.md),
  proposed, lists `facilitator.mode` among the keys a repository only
  tightens; it takes an exception for this key before it is accepted.
- The brief's configuration chapter and its invariants say this order, and no
  page describes a setting of a person's for one project.
