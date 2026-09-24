---
schema_version: 1
id: "iss-168"
slug: "abcd-s-presence-should-be-visible-in-the-host-harness-s-stat"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-105 grill session"
found_at: "commands/abcd"
related_intents: [itd-20, itd-200]
related_issues: [iss-164, iss-165]
resolution: "Delivered by itd-200 (spc-70): the presence badge leads the host's status line in a managed repository, in three states read from the per-repo mode store; ahoy install offers and wires the line, the bare board carries the same state, and the palette ruling on this record is what ships."
impact: additive
resolved_by:
  intent: "itd-200"
  spec: "spc-70"
---

abcd's presence should be visible in the host harness's status line / command line: a managed repo shows an 'abcd-managed' indicator (and possibly guard health) so the user can tell at a glance whether the current session is under abcd management without running a command. Needs a per-host adapter (e.g. a statusline hook) with the usual basics-built-in stance.

Design direction (maintainer, 2026-08-10), stated in harness-neutral terms: two layers over one fact. A host-agnostic default render lives in core — a single canonical line that both layers format, so no front door improvises its own words — and it surfaces ON DEMAND ONLY, at the `/abcd` board (the bare render and its `status` positional alias, `commands/abcd.md`). It is never streamed to stdout on every command and never injected into ordinary session output. Some harnesses show a persistent status bar and some offer no such surface at all; where one exists, ambient continuous display is exclusively its job, and where none exists the default layer stands alone rather than degrading to nothing. The default layer's contract is that a user who asks gets the answer, not that the answer is permanently on screen. The enhancement is an automatic option — self-detected, never configured. Adapters land one harness at a time, in a maintainer-set roster order (first the harness abcd already ships lifecycle hooks for, then opencode, then others much later). `/abcd:status` is not a separate command and must not become one: `status` is already a positional alias for the same bare render, and itd-20 owns that board's expansion, which is why this issue's `found_at` is `commands/abcd`.

Detection is INSTALL-time, not render-time. Where a harness has a status surface, it works by invoking a command and displaying its stdout, so by render time the harness has identified itself by calling — the enhancement therefore lives in the wiring, not in the print path, and the render can take the host as a flag or read whatever the harness supplies on stdin. What `ahoy install` must self-detect is which harness configurations are present, and therefore which wirings to write. Detection fails closed: no recognised harness means the default layer alone, and `unknown` is a legitimate terminal answer rather than a guess. Absence of a harness-supplied environment variable proves nothing — the plugin-root ladder is `ABCD_PLUGIN_ROOT` -> a harness-specific plugin-root variable -> executable-ancestor fallback (`internal/core/ahoy/store.go:85`), so a dev shim or an explicit `ABCD_PLUGIN_ROOT` yields no harness variable while running inside that very harness. Detection rests on positive evidence, never on a missing variable, and core has no host abstraction today to hang it on.

Reusable already: `internal/core/ahoy/guard_health.go` computes the guard-health half (`PluginRootResolved`, `HookInstalled`, `BinaryReachable`, with reasons) and `detect.go` classifies `managed-repo`, so the fact is largely assembled — what is missing is the one-line canonical render and the per-host wiring. Constraint to settle once: adapter ids in code and config are not user-facing prose and are unaffected, but any `docs/` page naming a harness trips the harness-naming blocker (`harness/claude-code`, `.abcd/docs-lint.json:14`) and needs generic phrasing or the `docs-lint: allow` escape.

Extended 2026-08-29 by the role work (adr-2609151528057260). The line has a second thing worth carrying besides managed-repo state and guard health: who the loop is currently waiting on. Three states cover it, running with nobody waiting, waiting on the facilitator, and waiting on the product thinker, and the last is the one that earns the space. Under adr-2609151528057260 the agents run unattended and stop only to obtain a verdict, and under adr-2609151528057131 a product thinker answers on a surface of their own rather than at a terminal, so a stop parked on them is invisible to the person sitting in front of the repository. The status line is where that becomes visible, and it is a facilitator surface throughout: the product thinker never sees it, so the marker is not addressed to them, it reports that they owe an answer.

This also happens to be the one host surface where colour survives. A host-rendered selection escapes terminal colour codes rather than interpreting them (iss-2608291009106041), so an addressee distinction there rests on its word and its glyph alone; a status line takes the text abcd supplies and generally renders its colour, which makes it the first place the role palette actually shows. The accessibility rule still holds and matters more here rather than less, since a single line under width pressure is exactly where a designer is tempted to drop the word and keep the colour.

Two constraints carry over unchanged. Where a harness offers no status surface the on-demand board stands alone rather than degrading to nothing, so the waiting-on state must be answerable there too. And the line is contended: managed-repo state, guard health, the mascot and now this are competing for one row whose width the host owns, which is a design problem rather than an implementation detail.

Settled by demonstration 2026-08-29, in a throwaway wiring of a real status line rather than on paper. The marker goes at the START of the line, not the end. Two findings drove it. The payload a harness hands the status-line command carries no terminal width at all, and the command runs with no controlling terminal, so the usual ways of asking return a default rather than the truth: width is unknowable, right alignment is guesswork, and anything abcd puts on that line must be designed for truncation rather than for layout. Since the end of the line is what a narrow terminal cuts first, the rightmost slot is the worst place for the one thing that must not be missed, and leading the line is what survives.

The cost is real and was visible immediately: the marker displaces the orientation cues a reader consults constantly, the working directory and the branch. A variant that keeps the quiet state on the right and moves the badge left only when somebody owes an answer was considered and set aside, since position that changes meaning is harder to learn than position that does not.

The badge renders as inverse video with its word inside it rather than as a coloured dot, which keeps it legible where the palette is unavailable. An animated pulse was tried and abandoned: a status line is re-rendered on events rather than on a clock, so a time-based frame changes only when something else already happened, which is exactly when a pulse is not needed. Indefinite attention-grabbing motion would also have to be switchable off to meet the accessibility bar the addressee register sets, and the inverse badge reaches the same prominence without that obligation.

Refined 2026-08-29. The neutral state and the presence indicator are the same slot, not two competing for one row. When nobody is being waited on, the marker reads as the managed-repo presence this issue already asks for, so the line carries one thing that changes meaning rather than a presence indicator plus a separate idle word. Three states in one place: managed and nobody waiting, waiting on the facilitator, waiting on the product thinker.

The presence half keeps its own precondition. It claims management only where the repository is actually managed, and an unmanaged repository shows nothing rather than an indicator saying it is not managed, since a marker that renders in both cases stops carrying information. That also keeps the fail-closed reading intact: absence means either unmanaged or undetected, and neither is a claim worth making on one row.

The presence badge is configurable and the role badges are not, and the line between them is what each one is for. A role badge says whose question is waiting, and recognition without reading only works if it looks the same in every repository, so those stay fixed with an accessibility override as an accommodation rather than a preference. The presence badge says abcd is here, which is ambient rather than a signal to act on, and nothing breaks when it differs between projects. It therefore ships with a sensible default and is a setting.

Which settings it belongs with follows from who reads it. The status line is a facilitator surface and the product thinker never sees it, so the presence badge is configured where the facilitator's own settings live, not in the register the product thinker owns. A product thinker configuring a surface they never look at would be a setting nobody exercises.

The default is a neutral fill carrying a warm word: a gold text on a dark grey, measured at 6.94 to 1. It was chosen against a plain white on the same grey at 9.74 to 1 and a pure yellow at 9.07 to 1, all three clearing the bar, so the choice was what the idle state should say rather than which was safe. White reads as quiet presence and yellow reads as caution, which is the wrong signal for a state meaning nothing needs anyone; the gold sits between them and marks abcd's presence without asking for attention.

The one collision in the set is that gold is the nearest hue to the amber the product thinker badge uses. The polarity difference holds them apart, since the presence badge is light text on a dark fill and the product thinker badge is dark text on a light one, and the words differ regardless. Any configured pair is held to the same contrast bar the role badges meet: measured against its own background, not chosen by eye.

## Grounds

- pursued: the status bar is the visible face of abcd, so a managed repository should look managed; we expect facilitators to keep the line on after living with it, and if they switch it off in their own settings that shows this was the wrong call

---

- pursued: a managed repository now looks managed and a parked stop is visible before any output is read; facilitators switching the line off in their own settings would show this was the wrong call

## Colour ruling, 2026-09-15

The presence pair shipped is livery's house yellow `#f0c052` on `#444444`,
measuring **5.74:1**, not the 6.94:1 recorded above.

This record states ratios and no hex. Building the render, the hexes were solved
back from the three recorded figures — gold 6.94, white 9.74, pure yellow 9.07
against one grey — which gives `#444444` for the grey and `#ffd700` for the gold,
all three reproducing to two decimals. But `#ffd700` is not in abcd's palette;
the house yellow is `#f0c052`. Put to the maintainer as record-versus-palette,
the ruling was the palette.

**What survives.** The reasoning above is about HUE: a gold between white's quiet
presence and yellow's caution, because the idle state should say that nothing
needs anyone. `#f0c052` is that same gold, so the argument holds and only the
measurement moved. The pair still clears the 4.5:1 bar, with less margin.

**What changes, and it is worth naming.** This record says gold "is the nearest
hue to the amber the product thinker badge uses", held apart by polarity. The
product thinker badge uses `#f0c052`. So the two are no longer the nearest hue,
they are the SAME hue, and polarity is now the only colour-borne discriminator:
presence is light-on-dark, product thinker is dark-on-light. The words differ
regardless (`abcd` against `waiting: product thinker`), and the word is what
carries the meaning with colour only reinforcing, so the badge stays legible with
every escape stripped — which is a test, not an assurance. But the collision this
paragraph called near is now exact, and a future change to either pair should
know that polarity is carrying it alone.
