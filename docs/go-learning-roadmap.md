# Go learning roadmap

## Stage 1 — foundation (implemented)

- modules and package boundaries;
- structs, constants, methods, and errors;
- standard `flag` CLI;
- YAML encoding and configuration validation;
- table-driven tests;
- deterministic planner and policy interfaces.

## Stage 2 — safe automation core

- parse playbooks with `yaml.v3`;
- implement `inspect`, `check`, and `run`;
- execute commands with `context.Context` and `os/exec`;
- stream output without invoking a shell;
- add timeout and cancellation tests.

## Stage 3 — Agent Kernel

- implement the shared service layer;
- add templates, clarification, and confidence;
- persist JSON traces and execution logs;
- add CLI end-to-end tests.

## Stage 4 — LLM and retrieval

- define a provider interface;
- implement a DeepSeek-compatible HTTP client;
- retrieve local operational documents;
- keep reasoning isolated from authorization.

## Stage 5 — Web Console

- expose JSON APIs with `net/http`;
- embed static assets with `embed`;
- add graceful shutdown and concurrent request tests.

## Stage 6 — engineering finish

- run race tests and fuzz parsers;
- add cross-platform builds;
- document behavior parity and remaining differences.
