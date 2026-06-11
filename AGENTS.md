# IaV Agent Guide

Instructions for AI agents working in this codebase.

## 3. Strict TDD Workflow
Always run the validation script:
```bash
./test_suite.sh
```


## 2. Read Package Comments Before Grep/Find
Every internal package has a `// Package ...` doc comment explaining its purpose
and boundaries. Before grepping or searching in a package, read its doc comment
first (via `go doc ./internal/<pkg>`).
