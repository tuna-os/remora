# Postmortem Template — remora

Use this template to conduct a structured review of incidents affecting remora. The goal is to understand what happened, why, and how to prevent recurrence. Postmortems are blameless: they focus on systemic factors and process, not individual actions.

Reference the related incident issue for details on timeline, impact, and resolution.

## Postmortem information

- **Incident issue:** (link)
- **Date conducted:** (UTC)
- **Attendees:** (list of participants)
- **Facilitator:** (who led the review)
- **Postmortem lead:** (primary author)

## Incident summary

<!-- One paragraph. Link to the incident issue for full details. State what happened and the impact. -->

## Root cause analysis

### Primary cause

<!-- What was the direct technical cause? Be specific and evidence-based. If multiple causes combine, list them in order. -->

### Contributing factors

<!-- Systemic or process factors that enabled or worsened the incident. Examples:
- Testing gap (e.g., invariant not checked, configuration not validated)
- Documentation gap (e.g., procedure unclear, runbook missing)
- Monitoring gap (e.g., alert missing, threshold wrong)
- Dependency gap (e.g., external service changed, package regression)
- Tooling gap (e.g., release process weakness, safety check missing)
-->

## Timeline

<!-- Detailed timeline from incident issue, with added context on decisions and their rationale. -->

## Impact review

- **Duration:** (how long users were affected)
- **Scope:** (what fraction of users, systems, or configurations were affected)
- **Detectability:** (how was it discovered, could it have been detected sooner)

## Lessons learned

### What went well

<!-- Actions or safeguards that contained or mitigated the incident. -->

### What could be improved

<!-- Process, tooling, testing, documentation, or monitoring changes that would prevent or reduce the impact of a similar incident. -->

## Action items

<!-- Specific, measurable steps to address root causes and contributing factors. Each item should:
- State the problem being solved
- Propose a specific solution
- Assign an owner (or "unassigned")
- Estimate effort (small / medium / large)
- Set a target date for completion

Example:
- [small, 2026-11-01] Package resolution failures should fail the build immediately, not silently. (tuna-os/remora#NNN)
- [medium, 2026-11-15] Add integration test for lockfile generation. (tuna-os/remora#NNN)
-->

## Sign-off

- **Reviewed by:** (stakeholder, maintainer, or operations)
- **Date:** (UTC)

## Related resources

- [Incident template](incident-template.md)
- [Rollback procedure](rollback-a-bad-remora-build.md)
- Related issue(s): (links)
