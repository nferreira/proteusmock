# ADR-0001: DDD + Hexagonal Architecture

## Status

Accepted

## Context

ProteusMock needs a clear separation between business logic (scenario matching, predicate evaluation) and infrastructure concerns (HTTP handling, file I/O, rate limiting). Testability is a primary concern since the project relies heavily on unit and integration tests.

## Decision

Adopt Domain-Driven Design with Hexagonal (Ports & Adapters) Architecture. The domain layer (`internal/domain/`) has zero external dependencies. All I/O crosses port interfaces defined in the infrastructure layer.

## Consequences

- Domain logic can be tested in complete isolation without mocking infrastructure.
- New adapters (e.g., a different file format) can be added without touching domain code.
- Slightly more boilerplate due to interface definitions and adapter implementations.
- Clear dependency direction: domain depends on nothing; infrastructure depends on domain.
