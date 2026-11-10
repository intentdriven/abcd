---
schema_version: 1
id: "iss-2611080057001367"
slug: "a-subscription-machine-cannot-launch-a"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "first live abcd drain --drive"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reviewset.go"
remedy: "When the role's runner is the subscription harness, admit a class's model by the model name the harness takes, without a provider block, so a subscription-only machine's bundled review set runs; and refuse a drive at its start, naming the configuration to add, when its review set has no route that can launch."
resolution: "The subscription harness admits a request's model with no provider block, and a drive refuses at its start a review set no route can launch"
impact: fix
---

On a machine whose agents run only on the subscription harness, with no API provider configured, a drive's reviewers can never launch. The bundled review set names each class as a provider route (anthropic/claude-opus-5-5, anthropic/claude-sonnet-5-5), and the runner admits a model only through a configured provider, so every reviewer is refused before any runner starts and the drive retries every 30 seconds without end. The first live drive's first lane to finish its implementation has waited at review since.
