# ADR-0018: Rate/Burst Update on Access

## Status

Accepted

## Context

After a hot reload, a scenario's rate limit configuration may have changed. The system needs to apply the new rate/burst values without losing the current token state.

## Decision

Update the limiter's rate and burst values in-place on access if they differ from the current configuration. This preserves the existing token state.

## Consequences

- Seamless rate limit updates after hot reload without resetting token buckets.
- No need to recreate limiters on configuration change.
- Slightly more complex access logic, but avoids disruptive rate limit resets.
