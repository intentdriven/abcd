---
id: prn-no-general-depth-switch
claim_type: criterion
reference: "abcd's command surface"
comparison: "A general switch that asks a command to look harder, compared with a check added to the default run or a flag named for the one check it adds."
evidence: [dcn-2611211021077161, itd-2611211024360819]
---

# No general depth switch

**The rule.** No command takes a general switch for looking harder, and no
command, help text or result says it covered everything. A check a command
could make cheaply and safely belongs in its default run, for everyone. A
check that is slow or risky gets a flag named for exactly what it adds, and
the command's statement of what it checks lists that addition. How hard judged
work looks is stated on the result. Where it can be chosen, the choice is the
project's and is not made again at each command.

**Why.** A general switch promises something no amount of checking delivers.
Checking can show that faults are present and cannot show that none are left,
so a result produced under a switch named for thoroughness reads as an
all-clear it cannot be. The people it is meant to reassure are the ones it
teaches to stop looking.

One word also cannot keep one meaning across unlike commands. On a command
that lists leftovers it would mean more kinds of leftover; on a lint it would
mean stricter rules; on a review it would mean more runs or another model.
Long-lived tools show where that leads: a compiler's all-warnings flag does
not enable all warnings, and a type checker's strict flag is documented as a
list that may change between releases.

An opt-in is the wrong place for assurance. A run nobody watches never types
the switch, so the work that most needs the deeper look would always get the
shallower one, and nothing in the result would say which a run was. A check
worth making is either cheap enough for every run or costly enough to deserve
a name of its own.

The idea, its research and both reviews are recorded in the ideate verdict
[`2026-11-21-ideate-thorough-flag-on-verbs.md`](../research/ideas/488a0aa96ac5de805348635b27036addf15cddc2/2026-11-21-ideate-thorough-flag-on-verbs.md).

**Bounds.**

- Nothing enforces this yet, and the tree does not fully meet it: the help of
  the sweep command says it lists everything a repository has left lying
  around (iss-2611211020368839). The rule binds new work from the day it is
  adopted; what breaks it today is fixed as it is found.
- A flag named for one check is the sanctioned form: its help says what it
  adds and what it costs, and it adds nothing else.
- Depth for judged work is real. By the product thinker's ruling of
  2026-09-23 (dcn-2609230000000008), delivered work states the rung it was
  checked at, and a project whose work can reach strangers declares that
  reach before it may lower that rung. That ruling is why the rule places
  the choice with the project. How a project chooses a rung is a question
  of itd-173, still a draft, and this stance does not answer it; that draft
  records the product thinker's answer of 2026-11-21 that each kind of
  check has a level of its own. The stance rules out the per-command choice
  and the promise of completeness, never the setting.
- A command that cannot afford a check in its default run says so in what it
  reports, as something it does not check. Leaving the check out silently is
  the failure this stance exists to prevent.

**Promotion.** The enabling artefact is the per-command declaration that
[itd-2611211024360819](../intents/planned/itd-2611211024360819-every-result-abcd-gives-says-how-far-it.md)
asks for, which says of each flag what it adds to what is looked at, how it
narrows or redirects that, or that it changes nothing looked at. A test can
fail on a flag the declaration
does not mention. No test can tell whether what the declaration says of a flag
is true, so that part stays a matter for review. Once the test exists, the
part it covers is promoted to a discipline-kind intent, as the README of this
directory describes; the rest stays here.
