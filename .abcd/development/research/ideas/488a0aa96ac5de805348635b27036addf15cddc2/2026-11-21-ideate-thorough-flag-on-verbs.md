# Ideate verdict — thorough-flag-on-verbs

**Verdict: reframed.** Recorded on 2026-11-21 by abcd's idea-admission protocol —
primary-source research, a grill against the existing record, and an
independent adversarial review. This record exists so the idea is not
re-litigated: it stands whether the idea lived or died.

## The idea

Add a --thorough flag to sweep and to other verbs. With it a verb goes into depth and tries to uncover everything, and it never assumes: it tests or verifies instead. The aim is that the product thinker and the technical facilitator can be certain abcd covers everything and leaves no stone unturned in doing its job.

## Where the idea came from

No source: the idea entered the gauntlet with nothing behind it but its own words.

## Leg 1 — Primary-source research

Every load-bearing claim checked against its primary source, never a
secondary citation. A claim marked quoted was checked only against the
source the idea came from, which proves the quotation and not the claim.

| Claim | Primary source | Finding |
|---|---|---|
| A pass that looks harder finds more than one default pass. In a benchmark of model code review, running several reviews and aggregating them raised the F1 score by up to 43.67 per cent. The same paper says current systems underperform (the best pairing reached an F1 of 19.38 per cent), names low precision and a high false-positive rate as the main limit, and gives no precision figure for the aggregated runs. | https://arxiv.org/abs/2509.01494 | verified |
| A deeper mode can make a person certain that nothing was missed. The source says the opposite about checking by testing: it can be a very effective way to show that faults are present and is hopelessly inadequate for showing that they are absent. | https://www.cs.utexas.edu/~EWD/transcriptions/EWD03xx/EWD340.html | falsified |
| Agents left to themselves stop early or verify badly often enough to matter. Across 1642 annotated runs of multi-agent systems, premature termination appeared in 6.20 per cent, no or incomplete verification in 8.20 per cent and incorrect verification in 9.10 per cent. | https://arxiv.org/abs/2503.13657 | verified |
| Adding more verification is enough to close that gap. The same study says the presence of a verifier is not a silver bullet: one added high-level verification step improved task success by 15.6 per cent in one system, and not all failure modes were resolved. | https://arxiv.org/abs/2503.13657 | falsified |
| A flag named for completeness delivers completeness. A long-lived compiler's all-warnings flag does not: its manual says some warning flags are not implied by it, and that many must be enabled one by one. | https://gcc.gnu.org/onlinedocs/gcc/Warning-Options.html | falsified |
| A strictness flag keeps one stable meaning over time. A widely used type checker's manual says the exact list of checks its strict flag enables may change over time. | https://mypy.readthedocs.io/en/stable/command_line.html | falsified |
| The models that do abcd's judged work expose one setting that trades cost for thoroughness. The vendor's page defines its effort setting that way and calls it a behavioural signal, not a strict budget. For one named model it says most of the gain is already present at the high level and that gains above it are uneven and depend on the task. The page nowhere says a higher level gives complete results. | https://platform.claude.com/docs/en/build-with-claude/effort | verified |
| How hard to look is the organisation's own risk decision, and what makes a result trustworthy is a report of the level attempted, the scope, the requirements checked and the reason for anything left out. The standard says merely running an automated tool without thorough testing is insufficient for certification. | https://raw.githubusercontent.com/OWASP/ASVS/master/5.0/en/0x04-Assessment_and_Certification.md | verified |
| sweep today covers less than its own words promise. The record that shipped it says it shows everything a repository has left lying around and then names worktrees, local branches, reservations and waiting inbox reports; neither that record nor the sweep code mentions stashes or lock files. This was read from the record, the help text and the code. It was not established by a run. | itd-2610222214476887 | verified |

## Leg 2 — Record grill

Does the brief, an intent, an ADR, or a principle already cover,
contradict, or supersede this idea? Every hit is cited by record id, and
every id resolved in this repository when the verdict was recorded.

| Record | Relation | Note |
|---|---|---|
| itd-173 | covered | The product thinker already picks how hard verification looks, per project and by what is at stake (the ruling of 2026-08-29 recorded in it); the idea asks for the same choice per command. It is a draft and is not built. The principle less-but-better, which has no id of its own, bears on the same point: a design that demands a decision at every use moves work onto whoever uses it. |
| itd-176 | covered | Whatever ships says how hard anyone looked at it. This is the nearest thing in the record to the assurance the idea wants. It is a draft, it is not built, and it speaks of shipped work, not of what one run of a verb covered. |
| dcn-2609230000000008 | covered | Delivered work states the verification rung it was checked at. It names the two drafts above as its carriers. |
| itd-197 | contradicted | Reading a source instead of recalling it is already a standing rule for every assertion that enters a record or grounds a decision, so it is not something a flag switches on. The rule also bounds itself: it calls verify-everything paralysis rather than rigour. The principle prefer-the-experiment-to-the-inference, which has no id of its own, sets the same default with a cheapness bound and says no mechanical check tells a tested claim from a reasoned one. |
| itd-195 | covered | A claim about how the code behaves is backed by something that runs, or it is not made. Verify-first is the default here, not a mode. |
| itd-196 | covered | A capability is rehearsed end to end before it ships, over a real tree. |
| itd-2610301750162657 | covered | Planned spot-checks say plainly what a clean result rules out and what it cannot. That is the reporting shape the reframing asks of every verb. |
| itd-2610222214476887 | covered | sweep already refuses to assume. It acts only on what is provably spent, and a check that could not be made holds every item it could have cleared and says so once, at the top. The principle enforcement-claims-are-facts, which has no id of its own, bears here: a signal that reports green while measuring a proxy withdraws vigilance more than a missing check does. |

### Searches

| Terms | Places | Count |
|---|---|---|
| thorough | .abcd/development/brief/; .abcd/development/intents/drafts/; .abcd/development/intents/planned/; .abcd/development/intents/shipped/; .abcd/development/intents/disciplines/; .abcd/development/intents/superseded/; .abcd/development/decisions/adrs/; .abcd/development/principles/; .abcd/development/research/notes/; .abcd/development/research/ideas/; .abcd/work/decisions/; the private ideas store on this machine | 11 |
| never assume; always test; always verify; does not assume; unverified assumption | .abcd/development/brief/; .abcd/development/intents/drafts/; .abcd/development/intents/planned/; .abcd/development/intents/shipped/; .abcd/development/intents/disciplines/; .abcd/development/intents/superseded/; .abcd/development/decisions/adrs/; .abcd/development/principles/; .abcd/development/research/notes/; .abcd/development/research/ideas/; .abcd/work/decisions/; the private ideas store on this machine | 7 |
| exhaustive; no stone; uncover everything; covers everything; nothing left; nothing missed; completeness | .abcd/development/brief/; .abcd/development/intents/drafts/; .abcd/development/intents/planned/; .abcd/development/intents/shipped/; .abcd/development/intents/disciplines/; .abcd/development/intents/superseded/; .abcd/development/decisions/adrs/; .abcd/development/principles/; .abcd/development/research/notes/; .abcd/development/research/ideas/; .abcd/work/decisions/; the private ideas store on this machine | 68 |
| effort level; effort tier; --depth; --deep; --effort; --strict; --paranoid; --exhaustive; depth of review; depth of verification | .abcd/development/brief/; .abcd/development/intents/drafts/; .abcd/development/intents/planned/; .abcd/development/intents/shipped/; .abcd/development/intents/disciplines/; .abcd/development/intents/superseded/; .abcd/development/decisions/adrs/; .abcd/development/principles/; .abcd/development/research/notes/; .abcd/development/research/ideas/; .abcd/work/decisions/; the private ideas store on this machine | 14 |
| assurance level; assurance ladder; verification ladder; verification rung; verification level; rung | .abcd/development/brief/; .abcd/development/intents/drafts/; .abcd/development/intents/planned/; .abcd/development/intents/shipped/; .abcd/development/intents/disciplines/; .abcd/development/intents/superseded/; .abcd/development/decisions/adrs/; .abcd/development/principles/; .abcd/development/research/notes/; .abcd/development/research/ideas/; .abcd/work/decisions/; the private ideas store on this machine | 100 |
| what was not checked; not checked; could not be checked; could not be made; could not be verified | .abcd/development/brief/; .abcd/development/intents/drafts/; .abcd/development/intents/planned/; .abcd/development/intents/shipped/; .abcd/development/intents/disciplines/; .abcd/development/intents/superseded/; .abcd/development/decisions/adrs/; .abcd/development/principles/; .abcd/development/research/notes/; .abcd/development/research/ideas/; .abcd/work/decisions/; the private ideas store on this machine | 39 |
| flag definitions named deep, full, strict, verify, all, audit, thorough, exhaustive, depth, effort, paranoid, narrow, wide, quick, fast or shallow | internal/surface/cli/ | 8 |
| stash; .lock; dogfood; transcripts; scratch | internal/core/sweep/ (code outside its tests) | 0 |
| sweep | .abcd/work/issues/open/ | 49 |

## Leg 3 — Adversarial review

Conducted fresh-context and off-policy by an evaluator that did not carry
out the research and received the idea as an artefact of unknown
authorship — the evaluator-outside-the-loop principle applied to ideas.
Each reviewer's attempts, verdict and sources are kept under its own label.

Current state: ignored. The reviewers judged the idea as an ambition for
the product this repository builds: a gap between the idea and what the
product ships today was out of scope as a kill, and reads as what the idea
implies building.

### Reviewer reviewer-a — reframed

- **fatal** — Uncovering everything cannot be built as a mode. A deterministic verb finds only the kinds of thing its code lists: sweep lists five and never looks at stashes, a thorough run cannot find a kind nobody coded, and once someone codes it the flag has nothing left to do. For the judged verbs the best code-review pairing in the benchmark found under a quarter of real problems, and aggregating runs did not approach completeness. Certainty that nothing was missed is out of reach at any setting.
- **fatal** — The flag gives false assurance. Having run with it measures how hard the tool tried, not what it covered, which is the proxy signal that makes people stop watching; the stated aim is to let both roles stop worrying, so the harm is larger here than usual. Verification itself fails too: incorrect verification appeared in 9.1 per cent of traces in the multi-agent study.
- **partial** — Opt-in depth is wrong for deterministic verbs. An extra check sweep could make is either safe and cheap, in which case leaving it out of the default is a coverage defect for everyone who omits the flag, or slow or risky, in which case it belongs behind a narrow flag naming exactly what it adds. Neither calls for a general depth switch, and sweep already declines to assume by default. First judged fatal; re-judged partial under the rule that a gap with what ships today is out of scope, because the argument removes the flag from deterministic verbs only and leaves the judged verbs untouched.
- **partial** — One flag cannot mean one thing across unlike verbs: more effort or more runs on a judged verb, more kinds of leftover on sweep, stricter rules on lint. Umbrella flags drift this way, as the compiler and type-checker manuals show. Once a per-verb list of what it covers exists, the list carries the meaning and the flag is only a name.
- **partial** — Never assuming cannot be a mode. Every check trusts something, such as git's output, the toolchain and the verifier, so the rule regresses without end, and the binary cannot tell whether an agent checked a thing or reasoned its way to it. A bounded form is possible, where each assertion written into a record shows what was run, but that is a per-claim rule, not a mode.
- **partial** — A per-run flag puts the choice in the wrong place. How hard to look depends on what is at stake in the project, not on the moment someone runs a command. Autonomous runs never type the flag, so they would always run at the lower level. A project-level setting with the level stated on every result gives the same depth without either problem.
- **partial** — The ground is already claimed by the verification ladder, the how-hard-anyone-looked disclosure, the decision that delivered work states its rung, and the spot-check intent. First judged survived, because none of those is built; re-judged partial under the rule that a gap with what ships today is out of scope. Judged as an ambition the need is real, but those records already hold it, and the idea adds only the flag shape and the claim of certainty.

What the idea implies building:

- A coverage statement in every verb's output: what it checked, what it did not, and what it could not check.
- The kinds of leftover sweep does not handle yet (stashes, stray lock files, the per-machine stores), added to its default run.
- A project-level choice of how hard to check, made by what is at stake (the ladder of itd-173).
- The depth used, stated on every delivered result (itd-176 and dcn-2609230000000008).
- The mechanics each level of the judged verbs needs: repeated runs, a second model, a different provider, and one record shape that an outside audit's results land in.
- Spot-checks that say what a clean result rules out and what it cannot (itd-2610301750162657).

Sources consulted: https://platform.claude.com/docs/en/build-with-claude/effort, https://arxiv.org/abs/2509.01494, itd-173, itd-176, itd-195, itd-196, itd-197, itd-2610301750162657, itd-2610222214476887, dcn-2609230000000008.

### Reviewer reviewer-b — reframed

- **partial** — Uncovering everything is unachievable, and a flag that promises it makes a false claim: testing shows faults are present, never that they are absent, and added verification still leaves failures. This lands on the stated purpose and on the name, not on every possible build of the idea.
- **fatal** — Never assuming and always verifying has no end, so as worded it cannot be specified or built. The only workable form is a bounded list of what gets verified per verb, and then the list carries the value, not the flag. On judged verbs it would be a hint to a model that cannot be tested to mean the same thing twice.
- **fatal** — One flag cannot mean one thing across unlike verbs: more kinds checked on sweep, something else on lint and launch, more runs or models on a review. Those are unlike operations with different costs and false-positive profiles, and a bundled strictness word comes to mean a drifting, unstated set.
- **partial** — The flag stands in for a gap that should simply be closed. First judged fatal; re-judged partial once the reviewer was given the rule that a gap with what ships today is out of scope. What sweep lacks today was set aside as something to build. What stays is that hiding depth behind an opt-in leaves the default answer silently incomplete, which is an argument against the ambition itself.
- **partial** — A per-command flag duplicates the planned verification ladder and bypasses the person's per-project choice, giving two depth controls that do not agree. The ladder is not built, so nobody can use it today, which weakens the point.
- **partial** — Opt-in is the wrong place for assurance: it is skipped in exactly the unattended runs where assurance matters most, nothing shows either role which runs used it, and a thorough run could not prove it verified where it only reasoned. This lands unless the result states what was and was not checked.
- **survived** — A cost-for-depth setting has no legitimate use at all. This fails: spending more on a review run buys measurable if modest recall, and stated levels are accepted practice. A named depth option on the verbs that call a model, with its cost and what it rules out stated, is not refuted. The problem is the promise of certainty, not the existence of a setting.

What the idea implies building:

- Coverage in sweep of the kinds of leftover it does not check yet (stashes, lock files, the other per-machine stores), or a plain statement in its output that they are left out.
- A stated-coverage line in each verb's result: what was checked, what was not, and why.
- The verification ladder and its per-project setting (itd-173).
- Delivered work that states the rung it was checked at (itd-176 and dcn-2609230000000008).
- A defined meaning of deeper for each judged verb, such as more review runs or a second model, with its cost stated.
- A way for a run to show whether a claim was tested or only reasoned, which nothing mechanical provides today.

Sources consulted: itd-173, itd-176, itd-195, itd-196, itd-197.

The reviewers agree, so no ruling was needed.

## Rejected alternatives

- **The flag as posed: one --thorough on sweep and other verbs, promising that everything is uncovered.** — Certainty that nothing was missed is not attainable by any amount of checking; one word cannot keep one meaning across unlike verbs; and an opt-in leaves the default silently weaker, most of all in autonomous runs, which never type it.
- **A graded scale per run (low to maximum) in place of a yes-or-no flag.** — It keeps the choice at each command, where it is made by whoever happens to type it. The gain above the high level is uneven and depends on the task, and a level still does not say what was covered.
- **Run every verb at its deepest level always, with no switch.** — The cost has no bound, and the standing read-not-recall rule calls verify-everything paralysis rather than rigour. For judged work more passes also bring more false positives for a person to sort, since low precision is the measured limit.
- **Add nothing and wait for the verification ladder and the how-hard-anyone-looked disclosure as they are drafted.** — Both are unbuilt drafts, and both speak of shipped work, not of what a single run of a verb covered. The need behind the idea is real and unmet today.

## The reframing

No general flag and no promise of certainty. First, every verb's result says what it covered, what it did not cover and what it could not check, so the person sees the edges of each answer. Second, where a deterministic verb such as sweep misses a kind of thing it could cheaply check, that is fixed in the default run and not put behind a switch; a check that is slow or risky gets a narrow flag that names exactly what it adds. Third, how hard the judged verbs look (more runs, more models, more effort) is set once per project by what is at stake, through the verification ladder, and every result states the depth it was checked at.

## What follows

The idea as posed does not survive, but the reframing recorded above
does. Any graduation to a draft intent carries the reframing, not the
original wording.
