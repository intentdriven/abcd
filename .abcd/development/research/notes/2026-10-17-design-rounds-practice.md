# Visual design rounds in practice: what two labs taught the procedure

Research note. Dated 2026-10-17. Written for
[itd-2610171228577260](../../intents/drafts/itd-2610171228577260-abcd-has-no-established-procedure-for.md),
the design-round procedure, so that every repository abcd manages starts from
what has already been learned rather than from memory. It rests on two labs in
this repository's lab store, cited by id: lab-261016142621-a46cdaa (the brief
session page for abcdesign.app, three rounds, 2026-10-16) and
lab-261017071130-6fd3032 (the phone and tablet answer screen, four rounds,
2026-10-17), and on the product thinker's rulings of 2026-10-17, restated in
plain words. An earlier control-panel lab in another managed repository was
the process reference for both and is not cited further.

**This exercise served two things at once.**

1. **It designed a product screen.** The answer-screen rounds are designing
   the phone and tablet web app where the product thinker and the technical
   facilitator answer the questions that would otherwise stop an autonomous
   run (the answer screen of the session-checklist intent, held on another
   records branch).
2. **It designed the design stage of abcdesign.app itself.** The comparison
   tool, its behaviours and the round procedure are the working prototype of
   the stage abcdesign.app will host after the brief and before intents. Each
   behaviour the product thinker asked of the tool (section 3) is therefore a
   requirement on that stage, not a lab convenience.

Status words used below: **held** (used in both labs, or kept after the
product thinker saw it), **added** (introduced during a lab, kept since), and
**candidate** (a design default suggested by these rounds, not yet settled).

## 1. The round procedure as it ran

The procedure in both labs, in order:

1. **Brief.** The coordinating session writes a round brief from the record
   (the intent, the product thinker's interview answers, the current page if
   one exists) and saves it beside the mocks as `brief-round-N.md`. Each later
   brief says which earlier briefs still apply and states only the changes.
2. **Five directions.** Round 1 asks for five distinct directions, one
   self-contained HTML file each. The brief suggests a spread (for the answer
   screen: native list, card stack, timeline, tablet tiles, conversation) and
   lets the agent change it.
3. **Render.** The generating agent renders its own files with a headless
   renderer (a small script that runs headless Chromium on a local file and
   writes a PNG for a given width, height and URL query), looks at every image,
   and fixes what it sees before reporting.
4. **Show.** The product thinker sees the round either in a gallery page or
   by opening the files in a browser, then in the comparison tool's overview.
5. **Pick.** The product thinker names a base (or two) and the changes they
   want, in their own words.
6. **Variations.** The next round makes five variations of the pick, each
   carrying every change asked for, and varying one thing on purpose (for the
   answer screen: how the cards expand on the tablet).
7. **Pairwise comparison.** The comparison tool shows the variations in pairs;
   the product thinker chooses left, right or too close, and the tool ranks
   them and picks finalists.
8. **Synthesis.** One design that keeps the winner and takes the named
   strengths of the runners-up, with the named faults fixed.

How each lab actually went:

- **Brief session page (three rounds).** Round 1: five directions; the
  product thinker found the guided-cards direction most promising and asked
  for alignment fixes. Round 2: five more in the current Apple design
  language plus a refined guided-cards; a revision (2b) then applied three
  changes to all of them (toggleable left and right sidebars, a cross-platform
  look of its own, Desktop / Tablet / Phone views). After round 2 the product
  thinker picked a default and an alternative, both to be built. Round 3
  revised the alternative to six named changes and was accepted. An accent
  comparison (six colours side by side) chose cobalt. **The pairwise step was
  not used:** rounds plus direct review converged first.
- **Answer screen (four rounds).** Round 1: five directions, called a great
  start; the card stack was taken as the base. Round 2: five variations of
  the card stack with three changes (cards that expand on the iPad, a floating
  button for the main task, a role switch), then later changes in the same
  round (a Timeline tab, the role switch moved into settings, short phone
  labels). Round 3: the separate floating buttons were disliked, so all five
  took one floating palette after round 1's floating pill, and were ranked in
  the pairwise tool. Round 4: one synthesis of the winner, the runner-up's
  role distinction and the third's tablet layout, with the floating-bar fault
  fixed and a settings sheet from the role badge.

Wall clock and agents, where the records give them:

- The answer-screen lab opened at 07:11 UTC; round 3's assessment was
  exported at 12:26 UTC, so three rounds and one full assessment took about
  five hours of elapsed time, interleaved with other work.
- Round 3's assessment, from the tool's own timing: criteria 76 minutes (the
  step was left open while the product thinker did other things), overview
  about 2 hours 36 minutes (likewise interleaved), pairs 10.5 minutes for 13
  pairs, finalists 8 minutes. The pairs themselves are fast; looking is slow.
- Agent counts are not recorded per round. The brief-session briefs address
  several generating agents; the answer-screen briefs address one agent that
  writes all five files of a round. **Gap:** the procedure should record, per
  round, the agents used and the minutes from brief to files ready.

Done by hand, in one lab or both:

- The brief, render script, gallery and comparison tool were copied from the
  previous lab each time, and the tool was patched in the lab copy.
- In the brief-session lab the product thinker opened the files by hand (the
  session had lost access to the screen), and the agents could not see their
  output at all until the render script was added mid-lab.
- The tool's result was exported as a file and handed back by hand
  (`results/round-3-assessment.json` in the answer-screen lab).

The product thinker's rulings on how this runs from now on, restated:

- **Side-by-side comparison** is for the obvious, taste-based choice between
  distinct directions. After that, a round makes **variations of one specific
  element** (the navigation, say), so the person compares like with like.
- **After the person's feedback, the next round is built autonomously:** abcd
  synthesises what won, fixes what was named, verifies it and shows it,
  without waiting to be told to continue.
- **Results are never downloaded.** Each judgement is saved into the record as
  it is made: on the person's own computer first, later on abcdesign.app.
- **Best practice is recorded over time,** so the choices that change least
  become the defaults abcd ships (section 7).

## 2. The round brief's hard rules

Rules that held in both labs:

- **One self-contained HTML file per mock:** inline CSS and JS; no external
  script, stylesheet, font, image or CDN; system font stack; inline SVG for
  icons (and a fake QR pattern where one is needed).
- **Mock data only:** a fictional project (Harbour, by example-org), people
  named Alice, Bob and Carol, every person referred to as they/them, short
  bird or tree names for agents; no real names, paths, hostnames or IPs.
- **Plain words:** the people using these screens are not necessarily
  technical, so the brief lists the words that must not appear on screen
  and the plain words to use instead.
- **Light and dark both designed** (`prefers-color-scheme`), and
  `prefers-reduced-motion` respected.
- **Phone and tablet** (and desktop where the page has one), each a genuine
  layout, not a scaled copy; works from 360 px wide with no sideways scroll.
- **Accessible controls:** keyboard reachable, a visible `:focus-visible`
  ring, real buttons, touch targets of at least 44 px; no `alert`, `confirm`
  or `prompt`, confirmations inline.
- **No option styled or ordered as recommended.**
- **The generating agent never opens a browser, a server, a tab or an app.**
  It writes only the files its task names, checks first that each does not
  exist, and stops rather than overwrite.
- **Every screen is reachable**, through a small, clearly mock-only screens
  strip that is not part of the design.

Rules added on the way, each kept since:

- **URL-selectable state** (added in the brief-session round 2, extended in
  the answer-screen rounds): the mock opens the screen, device, theme and role
  named in its query (`?screen=N&device=phone|tablet&theme=&role=`), so every
  state can be rendered and linked directly.
- **Embed mode** (`embed=1`): the mock hides its own screens strip when the
  comparison tool shows it, because the tool has its own controls.
- **Verify with the renderer before reporting** (replaced round 1's
  read-the-HTML check): render every screen on every device, in light and in
  the key dark screens, look at each image, fix, repeat; report the number of
  passes. Images go to the agent's own temporary folder, never the run folder.
- **Alignment rules** (after the first round's misaligned buttons): one
  height per control size from a shared variable; rows of controls centred;
  cards side by side put their main action on the same line at the card's
  foot; edges on a 4 px scale; primary and secondary actions in the same place
  on every screen.
- **A cross-platform look:** follow current platform interaction patterns
  (sidebars, sheets, popovers, floating toolbars) without imitating one
  operating system's chrome, icons or device frames.
- **Every overlay has a visible Close** in the same place, and Escape closes
  it.
- **Each brief states the previous round's verdict first,** in plain words,
  and which earlier briefs still apply, so a fresh agent needs nothing else.

## 3. The comparison tool: requirements for abcdesign.app's design stage

Each behaviour below was asked for by the product thinker while using the lab
tool and is now a requirement on the design stage abcdesign.app will host.

- **One shared screens bar** switches every design on the page to the same
  screen at once, pinned under the header while the grid scrolls.
- **Device and theme come from the page's own top bar**, passed into every
  design, so all of them are compared at the same size and in the same theme.
- **A design's own controls are hidden when it is embedded** and shown again
  only in the close-up view.
- **Keys 1 to 9 pick the screen** in every design, on the overview and on the
  pairs.
- **Home is the starting screen** of every comparison.
- **Look closer fills the window** with the design at real size; the page
  behind is washed out and does not scroll wherever the pointer is; a click
  outside or Escape closes it.
- **First-impression tags sit on top of each design; its letter and a
  magnifier (look closer) sit inside the frame's corners,** so the design
  keeps the space.
- **The left / right / too close choices sit at the top,** above the pair.
- **A reason is asked in an overlay after a left or right choice**
  (Enter saves, Escape or Skip records the choice without one), **and never
  for too close**, which is recorded at once.
- **Every new pair starts at the first screen,** so each pair is judged from
  the same start.
- **Keyboard hints sit behind an iOS-style switch** in the top bar, off by
  default.

Behaviours the tool already had, carried forward: criteria the person can add
to (the product thinker added one during round 3); arrow keys for left, right
and too close; undo; a flip mode that shows one design at a time; pinned notes
on a design; designs run in sandboxed frames that cannot reach the page; a
ranking (Bradley-Terry with bootstrap ranges), repeat agreement and circular
triads; finalists rated against the criteria; and an export in the
`abcd.design-assessment/1` shape. On abcdesign.app the export becomes a save
into the record as each judgement is made.

## 4. Design defaults that emerged (candidates)

All of these are **candidates**: they held for one product and one person.
They become defaults only through the learning loop in section 7.

- **iPhone in portrait, iPad in landscape** by default.
- **One floating palette** (a capsule floating above the content at the
  bottom, translucent, current tab filled in the accent) that holds the tabs
  and, where a screen has one, the screen's main task as its own accent
  segment. Separate floating buttons and heavy button rows were rejected.
- **The palette never covers navigation or controls:** the scrolling area
  ends above the palette's clear zone and pagers sit above their stack. On the
  tablet the tabs move to a slim side rail and the palette carries the main
  task only.
- **Short labels on the phone** (one word, or an unmistakable icon with an
  accessible name); **fuller labels on the tablet** where there is room. On
  desktop, labels collapse smoothly to icons in a stated order as the window
  narrows, never cut mid-word, never wrapping.
- **The role switch lives in settings,** with a small indicator at the top
  right that opens them; from that badge, settings open as a sheet with Done
  at the top right that returns the person exactly where they were.
- **Two half-overlapping circles** stand for "both roles".
- **With both roles, Product and Tech are told apart** by colour, label and
  placement (purple and blue in the answer screen's synthesis).
- **Cobalt accent** (#2457d6 light, #7aa2ff dark), chosen for abcdesign.app
  and reused for the answer screen so the family resemblance shows.
- **Cards that expand on the tablet** to fill the screen (a grid that uses the
  width, opening into one focus card with the next ones peeking), rather than
  a phone column in an empty frame.
- **Guided cards with large text and plainly labelled areas** were the
  brief-session page's preferred direction; a "what happens next" element was
  later removed from every screen of the alternative design at the product
  thinker's request.
- **A new criterion to start every round with:** visually pleasing, not
  technical, reassuring (added by the product thinker in round 3).

## 5. Faults found, as lessons for the brief template

- **Misaligned main actions** (the first brief-session round: two cards'
  buttons at different heights). Lesson: the alignment rules in section 2
  belong in the round-1 brief, not after the first complaint.
- **The floating bar covered important navigation on every iPhone design**
  (round 3 of the answer screen: card pagers and a screen's last controls sat
  under it). Lesson: the brief requires a clear zone the scrolling area never
  enters, and the verify step renders each screen scrolled to the bottom.
- **A non-wrapping header on the answering screen widened the column** past
  the phone's width. Lesson: headers may truncate or wrap, never set the
  column's width; the verify step renders at the narrowest width.
- **Leftover bottom padding** kept space for a palette that a screen no
  longer showed. Lesson: when a screen has no palette, its padding goes too.
- **Content drawn under the status bar.** Lesson: the mock's frame reserves
  the status bar and home-indicator areas, and the verify step looks at the
  top edge as well as the bottom.
- **The comparison page's mouse wheel scrolled the mock behind the close-up
  view** (and the page behind was not washed out). Fixed in the tool: the page
  cannot scroll while the close-up is open. Lesson: the tool, not each mock,
  owns scrolling while an overlay is open.
- **Agents that cannot see their output ship visual faults.** Before the
  render script, alignment faults reached the product thinker; after it, they
  were caught first. Lesson: rendering and looking is part of every brief.

These faults were found by the product thinker or by the verify step during
the answer-screen rounds; the lab's findings file records the floating-bar
fault and the tool fix, and the others were fixed in the round files as they
were found.

## 6. What the measures showed

- **First lab: the person chose without the pairwise step.** Rounds plus
  direct review converged on a default and an alternative; the comparison
  tool was not needed for the decision.
- **Second lab, round 3:** 13 pairs judged, **8 of them too close**. The
  top-ranked design (3 wins, no losses, 2 too close) was also the one the
  product thinker chose and the only one tagged good at first sight, so the
  ranking did not contradict the pick.
- **Repeat agreement was 1 of 3** (two repeated pairs flipped), below the
  lab's own bar of two in three; there were no circular triads.
- **The reasons said more than the ranking.** Every win of the top design
  cited one feature (its Away button, three times) and every win of the
  runner-up cited one other (its Product / Tech distinction, twice). The round
  4 synthesis was built from those reasons and the finalists' notes, not from
  the order.
- **Conclusion:** pairwise comparison discriminates weakly between close
  variations of one design. Keep it for choosing between distinct directions,
  let the person skip it, and for variation rounds rely on the reasons and the
  side-by-side view.

## 7. What abcd should ship

- **The round-brief template:** the hard rules of section 2 (including those
  added on the way), the faults of section 5 as checks, a slot for the
  previous verdict and the changes asked for, and the verify-before-report
  step.
- **The renderer:** the headless script that renders a local file at a
  width, height and URL query to a PNG, with no network and no browser
  window, usable by the generating agent.
- **The comparison tool,** with every behaviour in section 3, saving each
  judgement into the record as it is made; later the design stage on
  abcdesign.app.
- **A defaults file** holding the candidate defaults of section 4, each with
  how many rounds, projects and people it has held for.
- **The learning loop:** each round records what the person kept, changed or
  rejected; across rounds, projects and people, a choice that stops changing
  is promoted into the defaults file, and a choice that keeps changing stays a
  question in the next brief.
