# ADR-0011: BodyRenderer Interface in Domain

## Status

Accepted

## Context

Template rendering is an infrastructure concern (it depends on external libraries like expr-lang and pongo2), but the domain layer needs to trigger rendering during request handling.

## Decision

Define a `BodyRenderer` interface in the domain/match package. Template adapters in the infrastructure layer implement this interface without the domain importing template libraries.

## Consequences

- Domain layer stays clean with zero external dependencies.
- Template engine implementations are pluggable via the interface.
- Follows the Dependency Inversion Principle -- domain defines the contract, infrastructure provides the implementation.
