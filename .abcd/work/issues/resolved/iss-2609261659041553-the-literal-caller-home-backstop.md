---
schema_version: 1
id: "iss-2609261659041553"
slug: "the-literal-caller-home-backstop"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/residual.go"
resolution: "SweepCallerHome and the user-segment rewrite behind SurvivingCallerHome now collect their spans through backstopSpans, which reads the text as written and the scanner's decoded views of each line carrying a backslash or a percent sign (the percent pre-pass and every JSON-escape layer), mapping each view hit back to whole escape units. The solidus escape, the u002f form in either case, a second JSON layer, the percent-encoded separator and a control escape before a single-segment home are swept to the tilde as the literal home is; an occurrence right after an odd backslash run is judged by the view, so the backstop no longer leaves a dangling backslash or sweeps a single-segment home under another root. Tests: TestSweepCallerHomeReadsEscapedSpellings, TestSweepCallerHomeKeepsEveryOtherLineIntact, TestSurvivingCallerHomeReadsEscapedSpellings, and TestSweepCallerHomeWorkIsLinear for the cost class."
impact: fix
resolved_by:
  commit: "33715d8e"
---

The literal caller-home backstop (SweepCallerHome and SurvivingCallerHome, internal/adapter/scanner/residual.go) reads the home only as written, so an escaped spelling of it is invisible to the one stage the store-before-commit redactors keep independent of the detector: the solidus escape (\/ between segments), its \u002f form, and a separator written as an escape run. Stage one and the stage-two rescan read those spellings through the scanner's JSON-escape views since fix/drain-scanner-identity, so a leak needs both detector passes to miss before the backstop matters; but the backstop exists for exactly that case, and against an escaped home it is not a backstop. Detector: with HOME under the /Users root, SweepCallerHome collapses the solidus-escaped and the \u002f spelling of the home to the tilde, as it does the literal one.

## Grounds

- pursued: every escaped spelling the detector reads through its views is also swept by the literal backstop; a home spelling that ScanText finds through a view but SweepCallerHome leaves verbatim would show it wrong
