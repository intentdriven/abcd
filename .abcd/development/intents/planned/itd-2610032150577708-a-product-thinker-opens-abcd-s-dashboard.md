---
id: itd-2610032150577708
slug: a-product-thinker-opens-abcd-s-dashboard
spec_id: spc-2610040741034208
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610031214560142]
related_adrs: [adr-2610032150581128]
severity: minor
origin: researcher-authored
production_mode: hand-written
supersedes: [itd-139, itd-2610032150580455]
impact: additive
---

# The product thinker's dashboard, in a browser through Tailscale

## Press Release

> A product thinker opens abcd's dashboard on their phone, iPad or computer, wherever they are, through their own private Tailscale network. They see in plain words what waits on them, what is being built now and what comes next, with the whole private record behind it. They read the brief laid out for reading, mark a line and leave a note, and abcd brings the note up at their next session.

_The product thinker's accepted text (2026-10-04, decision 15)._

## Why This Matters

_Facilitator-written._

The board shows the product thinker where things stand, but only in a Terminal or a Claude Code session, and never on a phone. The public record site shows the record, but not what waits on them, and none of the private work. This dashboard is the private side of the record site, for the product thinker first, with the brief at its centre: they read it where they already are, mark what they question, and everything they touch is earmarked for the technical facilitator's review, so the brief is shaped together rather than after the fact. Being reachable on a network is new for abcd; the rule for it is adr-2610032150581128.

Typed links: builds on itd-2610031214560142 (the board: the same picture, from the same read of the project); supersedes itd-139 (the static team site) and itd-2610032150580455 (reach through Tailscale, folded in by decision 9); later additions are drafts itd-2610040740108331, itd-2610040740122709 and itd-2610040740135705 (see decision 18). The state-of-the-art passes of 2026-10-03 (serving a local dashboard safely) and 2026-10-04 (how a product person engages with a brief) and the two reviews of 2026-10-03 fed the interview.

## Decisions

1. 2026-10-03, the product thinker, confirming the routing (itd-84, hand-run): split. This draft is the dashboard on the home network; who may open it and what it may change is a standing rule (adr-2610032150581128, proposed); reach from anywhere through Tailscale is a later draft (itd-2610032150580455).
2. 2026-10-03, the product thinker, asked how the dashboard relates to itd-139 (a readable static site of the record for a team; keep both; the dashboard replaces it; decide later): the dashboard replaces it. The product thinker accepted the cost shown: no published site for a team.
3. 2026-10-03, the product thinker, asked where the dashboard is reachable in its first version (home network first; Tailscale only; both from the start; decide later), with the cost of each shown (a home-network page is only as private as the Wi-Fi; Tailscale needs an app on each device; both is the biggest first version): both from the start. Reach through Tailscale (itd-2610032150580455) is therefore part of the first version, planned and built with this draft rather than later.
4. 2026-10-03, the product thinker, asked how a new device is let in (scan a one-time code; type a passcode; a code on the home network or, on Tailscale, the person's own devices let in already; decide later): a code on the home network, or Tailscale. The product thinker accepted the cost shown: anyone on their Tailscale sees it. Consequence (facilitator): the tailnet identity is verified by Tailscale's own lookup of the connecting device, never by a header a local program could write; this overrides the security review's "pairing gates even on the tailnet".
5. 2026-10-04, the product thinker, asked how long a device stays let in on the home network (30 days; 7 days; until it stops; decide later): 7 days, revocable from the computer. Carried to the home-network draft by decision 9.
6. 2026-10-04, the product thinker, asked whether the dashboard can answer abcd's questions or only be read (read only; answer on Tailscale; decide later): read only. Revised by decision 11: notes and "Still right" on the brief are the only things done from it.
7. 2026-10-04, the product thinker, asked how much of each item it shows (headline; headline and story; everything; decide later), answered in their own words: "essentially a 'dogfooding' playground the part of the public dashboard (https://abcdev.app/record/) that offers everything the developers need in addition to the public facing website content."
8. 2026-10-04, the product thinker, asked who it is mainly for (everyone on the team; me first, team behind; developers only; decide later), answered in their own words: "me first, intially as product thinker: Later additions: Rewrite the brief; approve intents; for the techincal facilitator, for other team embers (configurable)." The later additions are filed as drafts (decision 18).
9. 2026-10-04, the product thinker, asked when the dashboard switches itself off on the home network, answered in their own words: "revert my decision: tailscale only first". This revises decision 3: the first version is reached through Tailscale only. Facilitator consequence (decided, not asked; it follows from the revision): itd-2610032150580455 (reach through Tailscale) is folded into this intent and superseded by it; the home network without Tailscale, with decision 4's code and decision 5's seven days, is the later draft itd-2610040740108331.
10. 2026-10-04, the product thinker, asked what the first version shows besides "waiting on you, now, next" (summary and public pages; add problems without notes; everything private too; decide later), answered in their own words: "everything private, with a focus on the brief and how to allow the PT to engage with it".
11. 2026-10-04, the product thinker, asked what engaging with the brief means in the first version (read it well; mark lines to discuss; read, then research; decide later): "combine 2 & 3": mark a line and leave a note, shaped by a state-of-the-art pass on how product people engage with a brief, which ran the same day.
12. 2026-10-04, the product thinker, asked how it runs through Tailscale (started by hand; starts with the computer; decide later): started by hand, running until stopped, never starting by itself.
13. 2026-10-04, the product thinker wearing the facilitator's hat, asked what becomes of the discipline itd-140's promise that the record site works for any project, whose proof was itd-139's sample project (the dashboard's tests prove it; give up the promise; a separate record for the sample; decide later): give up the promise. The record site and the dashboard are promised for abcd's own record only; itd-140 carries the ruling.
14. 2026-10-04, the product thinker, asked whether a note picks a kind (just a note; optional kind; decide later): an optional kind: question (the default), disagree, out of date.
15. 2026-10-04, the product thinker accepted the press release above as quoted.
16. 2026-10-04, the product thinker, asked whether each brief chapter shows "changed / last confirmed by you" with a one-tap "Still right" (yes, in the first version; dates only; decide later): yes, in the first version.
17. 2026-10-04, the product thinker, unprompted mid-walk, verbatim: "there must be a function in v1 of this that what the PT touches (i.e. highlights/comments on/etc) must be 'earmarked' for a later conversation with the TF (or send to TF for review in the console)". Asked where an earmarked item goes (a joint conversation; the facilitator's console; both; decide later): the facilitator's console. Criterion 6 was rewritten to it and accepted.
18. 2026-10-04, the facilitator (decided, not asked: one split by what each needs): the later additions are three drafts: the home network without Tailscale (itd-2610040740108331); acting from the dashboard, rewriting the brief and approving intents (itd-2610040740122709); the dashboard for more people, the technical facilitator's view and other team members, configurable (itd-2610040740135705).
19. 2026-10-04, the product thinker accepted the eight criteria below one at a time (criterion 6 as rewritten), and, wearing the facilitator's hat, the technical checks as one list.
20. 2026-10-04, the facilitator (decided, not asked: a new capability, nothing removed): impact additive.
21. 2026-10-04, the product thinker wearing the facilitator's hat, told by the spec writer that Tailscale Serve hides which device connects, so the identity check cannot be met behind it, and asked for the route (listen on the computer's own Tailscale addresses with Tailscale's certificate; embed Tailscale as its own device, a new dependency; keep Serve and trust its header; decide later): the computer's own Tailscale addresses. The first technical check is reworded to it, and no new dependency is added.
22. 2026-10-04, the facilitator (decided, not asked: a rule is in force when the code that enforces it lands): adr-2610032150581128 is accepted in the change that lands the spec's step 1, with brief invariant 7's inbound clause.
23. 2026-10-04, the product thinker, asked to explain and decide now, told that a certificate (the browser's padlock) publishes the computer's Tailscale name in a public register for good while Tailscale already encrypts the connection without one, and asked which for the first version (no certificate; certificate after renaming the computer; certificate as is; decide later): no certificate. The first and fifth technical checks are reworded to it; the cost accepted is the browser's "Not secure" label and the older form of cross-origin protection. A certificate stays possible later, with consent to the publication.

## Mechanism

We expect the product thinker to engage with the brief at least weekly once it is on the devices they already carry, because the brief is today reachable only through the tools the facilitator uses, and because every mark they make reaches the facilitator's next session rather than waiting to be remembered. The product thinker's falsifiers (recorded under Grounds): a month with no notes or "Still right" taps, or the product thinker still learning of decisions about the brief only after they are made.

## Scope Conditions

- The product thinker's devices and the computer running the dashboard are on the same Tailscale network, and the computer is on while they look. <!-- cond: cond-2610040741030720 -->
- The technical facilitator opens a Terminal or Claude Code session in the project often enough for earmarked items to be reviewed. <!-- cond: cond-2610040741033160 -->
- The project is abcd's own record; other projects are not promised (decision 13). <!-- cond: cond-2610040741037826 -->

## Acceptance Criteria

_The product thinker's checks, accepted one at a time on 2026-10-04:_

- Given the dashboard is running at the computer and my phone is on my Tailscale, when I open its address, then I see first what waits on me, then what is being built now and the next few items, in plain words, with no record numbers or commands.
- Given a phone or computer that is not on my Tailscale, when it tries the dashboard's address, then nothing answers, and nothing of the project is visible.
- Given I tap an item, when its page opens, then I see the whole private record behind it: its announcement, its checklist of what done means, its decisions, the problems linked to it and the work notes, readable on a phone.
- Given I open the brief, when I read a chapter, then it is one page laid out for reading on a phone, headed 'changed <date> · last confirmed by you <date or never>', with how many ideas and decisions rely on it, folded away until I tap.
- Given I select a sentence of the brief and write a note, optionally marked question, disagree or out of date, when I save it, then it stays attached to that sentence even after the brief is reworded; if the sentence is gone, the note is shown with the old words struck through, never lost.
- Given I marked, noted or confirmed anything on the dashboard, when the facilitator next opens a Terminal or Claude Code session, then each item appears there for review one at a time, quoting what I touched; their proposal comes back to my dashboard and next session, and nothing in the brief changes until I confirm.
- Given I read a brief chapter and it is still right, when I tap 'Still right', then the chapter shows today as 'last confirmed by you', and the tap goes to the facilitator's review like any note.
- Given someone starts the dashboard at the computer, when it starts, then it says in one line where to open it and who can; when they stop it, the address answers nothing until it is started again, and it never starts by itself.

_Technical checks, accepted as one list on 2026-10-04 (facilitator's hat):_

- Given the dashboard runs, when its listeners are enumerated, then it listens on this computer's own Tailscale addresses only, never through Serve or Funnel, and requests no certificate, and a test proves one package alone opens a listener.
- Given a request arrives through Tailscale, when the dashboard decides who it is from, then it asks Tailscale's own lookup of the connecting device and never trusts a header a local program could write.
- Given any request, when it is served, then only expected host names are answered, server timeouts and size caps apply, every viewer shares one snapshot of the project, and live-update connections are capped.
- Given any page, when it is drawn, then it comes from the record site's one renderer with records looked up by id, no file is served from a requested path, the content-security policy is self only with no framing, and a test proves nothing is loaded from off the host.
- Given a note or a "Still right" tap, when it is written, then it is accepted only from a device on the person's Tailscale that the dashboard let in, with cross-origin protection.
- Given a note, when it is stored, then it is anchored by quote, offsets and commit, and a note whose sentence is gone is kept as an orphan.
- Given notes and stamps, when they are stored, then they live in the checkout's local tier until reviewed.

## Open Questions

_Answered at the interview of 2026-10-04 (decisions 4 to 18). Two remain open for the product thinker, below; until they are answered, the dashboard refuses both, the facilitator's safe default (`.abcd/work/DECISIONS.md`, 2026-10-05). For the spec (facilitator): how the dashboard serves the record site's pages without the minutes-long site build (the record review's finding 4), and how earmarked items reach the facilitator's session (the session-start seam the rules loader uses)._

- **Open: may a device someone else shared into your Tailscale network open the dashboard?** Another person can share one of their own devices into your Tailscale network. It then shows up among your devices, but it belongs to them, not to you. "Anyone on your Tailscale" (decision 4) did not say whether that includes such a device. Until you answer, it is refused: it gets no answer at all, as if the dashboard were not there. Letting it in would show your project to the person who shared it.
- **Open: may the dashboard be opened through another of your devices that passes it on?** Another of your own devices can pass the dashboard on, using Tailscale's Serve (to people on your Tailscale network) or Funnel (to anyone on the internet). The dashboard then sees only your own device asking, not who is really behind it. Until you answer, such a passed-on request is refused and nothing is shown, when it carries the marks a passing-on device adds, which Tailscale's Serve and Funnel always add for a web page. One kind cannot be caught: a device that passes on the raw connection without reading it adds no mark and looks exactly like that device opening the dashboard itself, so it is let in like that device.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect the product thinker to engage with the brief at least weekly once it is on the devices they carry, with every touch reaching the facilitator; shown wrong if a month passes with no notes or Still-right taps, or if the product thinker still learns of decisions about the brief only after they are made
