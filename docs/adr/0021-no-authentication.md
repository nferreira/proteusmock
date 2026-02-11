# ADR-0021: No Authentication

## Status

Accepted

## Context

Some mock server tools include authentication mechanisms for the admin API or scenario management endpoints.

## Decision

ProteusMock does not include any authentication. It is a development and testing tool, not a production API gateway.

## Consequences

- Simpler setup and configuration for developers.
- Must not be exposed to untrusted networks.
- Appropriate for local development and CI environments where network access is controlled.
