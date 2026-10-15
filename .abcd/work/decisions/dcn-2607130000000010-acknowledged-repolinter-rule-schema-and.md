---
id: dcn-2607130000000010
slug: acknowledged-repolinter-rule-schema-and
date: 2026-07-13
---
2026-07-13 (itd-85 M7): acknowledged repolinter (rule schema) and Conftest
(severity/exit vocabulary) in ACKNOWLEDGEMENTS now, since both are actually
adapted in the shipped audit engine. DEFERRED the SARIF acknowledgement to P3:
the serializer seam is shaped for SARIF but no SARIF is emitted yet, and the
convention is to credit a pattern in the change that lands it, never ahead. Add
the SARIF entry when the --format sarif serializer ships.
