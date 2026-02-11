# ADR-0002: Domain Types Separate from YAML Types

## Status

Accepted

## Context

Scenarios are defined in YAML files, but the domain model should not be coupled to any specific serialization format. Mixing YAML tags into domain structs would leak infrastructure concerns into the domain layer.

## Decision

Maintain separate types: `scenario.Scenario` (domain) and `yamlScenario` (infrastructure). The infrastructure layer is responsible for mapping between the two.

## Consequences

- Domain model remains clean and format-agnostic.
- Supporting a new format (e.g., JSON, TOML) requires only a new adapter, not domain changes.
- Requires a mapping/conversion step between YAML and domain types.
