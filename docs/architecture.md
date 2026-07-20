# safeops-go architecture

## Rewrite strategy

safeops-go is a behavior-oriented rewrite, not a line-by-line Python translation. The Python project remains a requirements reference only; the Go binary has no Python runtime dependency.

## Boundaries

- `cmd/safeops`: process entry point only.
- `internal/cli`: argument parsing and presentation.
- `internal/config`: YAML configuration, defaults, validation, and masking.
- `internal/model`: stable domain contracts.
- `internal/analysis`: playbook static analysis.
- `internal/check`: local and Ansible preflight checks.
- `internal/engine`: shell-free command construction and context-aware execution.
- `internal/agent/planner`: deterministic goal normalization.
- `internal/agent/policy`: execution authorization independent of LLM output.
- `internal/agent/service`: orchestration boundary shared by CLI and Web.
- `internal/provider/deepseek`: optional reasoning provider.
- `internal/trace`: audit persistence.
- `internal/web`: standard-library HTTP API and embedded console.

## Safety invariant

Semantic input is untrusted. Natural language, templates, retrieved documents, and LLM output may improve a plan but cannot authorize mutation.

An apply operation is possible only when:

1. the operator explicitly requests apply mode;
2. preflight checks pass;
3. the operator approves the plan;
4. confidence meets policy;
5. production requests include the required confirmation.

## Request flow

```text
CLI / HTTP
    -> Agent service
        -> Local RAG context (optional)
        -> DeepSeek goal hints (optional, validated)
        -> Planner
        -> Static analysis
        -> Preflight checks
        -> Policy decision
        -> Command preview or execution
        -> Trace persistence
```

## External execution

The engine passes an argument slice directly to `exec.CommandContext`. It never joins user input into a shell command. Timeouts map to exit code 124 and cancellation maps to 130, providing stable evidence across platforms.

## Agent Kernel

`internal/agent/service` is the application boundary shared by presentation adapters. It accepts trusted operator controls separately from semantic goal text, then executes this sequence:

1. deterministic planning and template matching;
2. missing-field clarification;
3. static risk analysis and preflight checks;
4. independent policy evaluation;
5. plan-only preview or context-aware execution;
6. atomic trace and log persistence on every terminal path.

The CLI and HTTP handlers implement only parsing and output adaptation. Both call the same service rather than duplicating policy logic.

## Intelligence and fallback

The DeepSeek client implements a narrow `GoalParser` contract and requests structured JSON. Provider output is clamped and candidate paths are validated against the current workspace. Network, credential, or decoding failures become trace notes and fall back to local planning.

Local RAG scans bounded Markdown/TXT content and ranks token overlap, including Chinese bigrams. This keeps the retrieval path deterministic and understandable without introducing a vector database into a learning project.

## Web adapter

The embedded console uses only `net/http`, `embed`, and browser-native JavaScript. Its API defaults goal requests to plan-only, limits JSON bodies, rejects unknown fields, masks configuration secrets, exposes bounded recent trace summaries, and shuts down through the process context.

The default loopback bind is intentional. Authentication, TLS termination, RBAC, distributed jobs, and secret management are explicitly outside this portfolio project's scope.
