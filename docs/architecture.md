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

## Planned request flow

```text
CLI / HTTP
    -> Agent service
        -> Planner
        -> Static analysis
        -> Preflight checks
        -> Policy decision
        -> Command preview or execution
        -> Trace persistence
```
