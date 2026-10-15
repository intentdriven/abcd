---
schema_version: 1
id: "iss-2610142234581003"
slug: "kill-port-given-no-literal-operand-is"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security review of feat/guard-port-blocks for v0.15.0"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Block kill-port with no literal operand: read kill-port as the command xargs runs (operands may arrive on stdin), add a flags form of the entry for -p and --port (attached or separate), or drop min_operands and exempt --help and -h; add each spelling to the kill-port fixtures and the adversarial corpus at block."
---

kill-port given no literal operand is allowed: a port read from standard input (echo 3000 | xargs npx kill-port, xargs -n1 npx kill-port < ports) or given as an attached flag (npx kill-port --port=3000, -p=3000) passes the guard, because the kill-port entry asks for at least one operand and xargs is not read as handing it one. Each line stops whatever holds the port, as npx kill-port 3000 does, which blocks.
