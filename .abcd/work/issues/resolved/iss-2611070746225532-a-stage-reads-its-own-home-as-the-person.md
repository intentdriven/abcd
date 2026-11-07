---
schema_version: 1
id: "iss-2611070746225532"
slug: "a-stage-reads-its-own-home-as-the-person"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "first live abcd drain --drive"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/residual.go"
remedy: "Have scanner.CallerHome read abcdhome.MachineHome, the one resolver that already gives the person's own home under a stage and keeps a test fixture's home, so identity and path sweeps in a stage judge by the person's home."
resolution: "CallerHome reads the machine home, so a stage's own folder is no longer taken as the person's account."
impact: fix
---

Inside a walled stage the scanner reads the stage's own home as the person's, so the home folder's last segment, literally home, becomes the local username. A shell word such as $HOME is then rewritten to $[redacted-user], which fails TestPromoteOrphanRemedyRunsAsPrinted in internal/core/capture. Every lane of abcd drain --drive runs its definition of done in a stage, so every lane's preflight fails whatever its issue, and no lane can land.
