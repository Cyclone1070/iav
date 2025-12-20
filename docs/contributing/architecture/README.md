# Architecture Guidelines

This directory contains the Go Architecture & Design Guidelines, split into focused documents for easy reference.

## Documents

| #   | Document                                               | Description                              |
| --- | ------------------------------------------------------ | ---------------------------------------- |
| 0   | [Preface & Checklist](00-preface.md)                   | Core principles and pre-commit checklist |
| 1   | [Package Design](01-package-design.md)                 | Small, focused, reusable components      |
| 2   | [Dependency Injection](02-dependency-injection.md)     | Explicit, testable dependencies          |
| 3   | [Interfaces](03-interfaces.md)                         | Consumer-defined interfaces              |
| 4   | [Structs & Encapsulation](04-structs-encapsulation.md) | Control state and enforce invariants     |
| 5   | [Validation](05-validation.md)                         | Constructor vs runtime validation        |
| 6   | [Testing](06-testing.md)                               | Deterministic, isolated tests            |
| 7   | [Error Handling](07-error-handling.md)                 | Shared error domain and sentinels        |
