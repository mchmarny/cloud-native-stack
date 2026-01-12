## Summary

<!-- What changed? Keep it short. -->

## Motivation / Context

<!-- Why is this change needed? Link issues/discussions if applicable. -->

Fixes: <!-- #123 -->
Related: <!-- #123 -->

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation
- [ ] Refactoring
- [ ] Build/CI

## Component(s) Affected
- [ ] CLI (`cmd/cnsctl`, `pkg/cli`)
- [ ] API server (`cmd/cnsd`, `pkg/api`, `pkg/server`)
- [ ] Recipe engine / data (`pkg/recipe`)
- [ ] Bundlers (`pkg/bundler`)
- [ ] Collectors / snapshotter (`pkg/collector`, `pkg/snapshotter`)
- [ ] Docs/examples (`docs/`, `examples/`)
- [ ] Other: ____________

## Implementation Notes

<!-- Key decisions, trade-offs, and any non-obvious behavior changes. -->

## Testing

<!-- Paste command(s) and summarize results. Prefer `make qualify` when relevant. -->

- [ ] `make test`
- [ ] `make lint`
- [ ] `make scan` (if applicable)
- [ ] `make qualify` (optional but preferred for non-trivial changes)

## Risk / Rollout

<!-- What could break? What’s the blast radius? Any backwards compatibility notes? -->

- Risk level: Low / Medium / High
- Rollout/upgrade notes (if any):

## Checklist

- [ ] I ran the tests relevant to this change and they pass
- [ ] I did not disable or skip tests to make CI green
- [ ] I added/updated tests where it makes sense
- [ ] I updated docs/examples if user-facing behavior changed
- [ ] I kept changes focused and consistent with existing patterns
- [ ] (If required) commits are signed off (DCO): `git commit -s`
