# Incident Template — remora

Use this template to document and coordinate response to incidents affecting remora deployments or operations. Link to this issue from the applicable postmortem if an incident review is warranted.

## Incident information

- **Date/time started:** (UTC)
- **Date/time resolved:** (UTC)
- **Severity:** (SEV1 / SEV2 / SEV3 / SEV4)
- **Component(s):** (e.g., package resolution, image build, bootc switch, timer, installation)
- **Affected users/systems:** (number and characteristics)

## Summary

<!-- One-paragraph summary of what happened. State the user-facing impact: what changed, what stopped working, or what became incorrect. Be specific. -->

## Root cause

<!-- Measured cause, not a hypothesis. Reference specific logs, version numbers, reproduction steps, or error messages. If not yet known, say so. Do not speculate. -->

## Timeline

<!-- Start from the earliest sign of the problem. Use UTC times. Include:
- When the problem first appeared (measured, not estimated)
- When it was detected and by whom
- Key actions taken
- When the impact ended
- When the fix was applied (if different)

Format: HH:MM UTC — observed or action -->

## Impact

- **Duration:** (total time users were affected)
- **Scope:** (specific systems, users, base images, configurations, or package sets)
- **What failed or changed:** (be specific: package resolution failed, lockfile became stale, bootc switch hung, binary corrupted, timer failed, etc.)

## Resolution

<!-- What was done to stop the ongoing impact. If automated, describe the automation. If manual, describe the steps and who performed them. Reference [rollback-a-bad-remora-build.md](rollback-a-bad-remora-build.md) if a rollback was used. -->

## Follow-up

- [ ] Postmortem scheduled (if SEV1/2, or if root cause analysis is incomplete)
- [ ] Tracking item created: (link)
- [ ] Monitoring/alerting added to prevent recurrence
- [ ] Related issues or PRs filed

## Notes

<!-- Anything else relevant to this incident: unusual base image behavior, package manager-specific issues, timing patterns, etc. -->
