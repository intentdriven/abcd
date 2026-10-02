---
schema_version: 1
id: "iss-2610020727129199"
slug: "the-site-s-authorship-tally-counts-the-abcd-label-as-an-ai"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/contributors.go"
remedy: "Count the abcd label in its own bucket beside DeclaredNone (abcd-composed commits, neither assisted nor undeclared), never as a model bar or a vendor, and tie the site's recognition of it to scripts/check-attribution.sh ABCD_RE with a test shaped like TestAssistedByGrammarMatchesTheGate; correct the contributors.go:183 comment to name both non-vendor forms. Grounds: ruling PC1 defines the label as abcd's composition with no model, and the gate's own three-form statement."
---

The site's authorship tally counts the abcd label as an AI model: internal/core/site/contributors.go LoadAuthorship treats every Assisted-by value other than None as assistance, so a commit carrying Assisted-by: abcd:<version> (the label the implement loop's pick and sync commits carry under ruling PC1) is added to AssistedCommits, charted as a model bar named abcd:<version>, and adds abcd to the vendor set, and the comment at contributors.go:183 still calls None the only accepted non-vendor value. The label says abcd composed the text and no model wrote it, so publishing it under a heading that lists assisting models misstates the disclosure. Latent until the first loop commit lands on the default branch.
