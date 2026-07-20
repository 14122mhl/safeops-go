# Go learning roadmap

## Stage 1 — foundation (implemented)

- modules and package boundaries;
- structs, constants, methods, and errors;
- standard `flag` CLI;
- YAML encoding and configuration validation;
- table-driven tests;
- deterministic planner and policy interfaces.

## Stage 2 — safe automation core (implemented)

- parse plays and tasks with `yaml.v3` and normalize FQCN modules;
- implement `inspect`, `check`, and `run`;
- execute argv safely with `context.Context` and `os/exec`;
- capture output without invoking a shell;
- enforce apply, approval, confidence, checks, and production confirmation gates;
- add timeout, preflight, CLI, and risk-analysis tests.

## Stage 3 — Agent Kernel (implemented)

- implement a shared service used by CLI and future HTTP adapters;
- add release, rollback, and incident templates;
- add clarification, confidence, and approval gates;
- atomically persist JSON traces and execution logs;
- add service, trace, template, and CLI end-to-end tests.

## Stage 4 — LLM and retrieval (implemented)

- define a provider interface;
- implement a DeepSeek-compatible HTTP client;
- retrieve local operational documents;
- keep reasoning isolated from authorization.

## Stage 5 — Web Console (implemented)

- expose JSON APIs with `net/http`;
- embed static assets with `embed`;
- add graceful shutdown and HTTP adapter tests.

## Stage 6 — engineering finish (implemented baseline)

- run race tests, vet, formatting checks, and API tests;
- add cross-platform compile checks;
- document learning value, scope, and production differences.

## Optional exercises

- add fuzz tests for YAML and JSON boundaries;
- replace the lexical scorer with a pluggable embedding retriever;
- add Server-Sent Events for live execution output;
- add authentication before allowing a non-loopback deployment;
- implement another automation engine through the existing interfaces.
