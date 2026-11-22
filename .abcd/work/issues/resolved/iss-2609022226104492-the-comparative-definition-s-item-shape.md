---
schema_version: 1
id: "iss-2609022226104492"
slug: "the-comparative-definition-s-item-shape"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "the comparative ingest of 2026-09-02; refiled from the parked phase-9 clone, 2026-11-22"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/cold-reading-comparative.md"
remedy: "Reword the item shape in agents/cold-reading-comparative.md so `criterion` takes the declared criterion's name alone, the text before the dash and not the gloss, and say that the criteria are declared as a name, a dash and a gloss on one line; bump the definition's prompt_version by a patch. The ingest needs no change."
resolution: "The comparative definition's item shape asks for the declared criterion's name alone, the text before the dash and not the gloss, and says the criteria are declared as a name, a dash and a gloss on one line. The ingest was the right half and is unchanged: it always matched the declared name. The definition moves to prompt_version 0.1.5, and TestComparativeItemShapeAsksForTheCriterionNameAlone holds the wording."
impact: fix
---

The comparative definition's item shape tells the reader that `criterion` is the declared criterion quoted from the passed material. The criteria discipline (itd-191) declares each criterion as its name, a dash and a gloss on one line, so a reader who quotes the passed material returns the whole line. The ingest's undeclared-criterion check accepts the declared name alone: `declaredCriteria` in internal/core/reading/criteria.go keeps the text before the dash, and the check in internal/core/reading/ingest_regime.go looks the item's value up among those names. The two halves of the instrument disagree, and the first comparative run of 2026-09-02 was refused whole on it. Until the wording moves, a comparative run passes only where the host's prompt states the field's form, which the definition should state itself.

The definition's wording is the half to move. A branch that never reached the default branch made that change; the definition that ships still carries the old wording.
