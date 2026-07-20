# safeops-go workspace instructions

- [x] Clarify project requirements: pure Go behavior-oriented rewrite of safeops.
- [x] Scaffold the independent project under `/Users/hemu/safeops-go`.
- [x] Add the initial configuration, model, planner, policy, engine, and CLI packages.
- [x] Implement inspect, check, run, the shared Agent Kernel, and auditable traces.
- [x] No additional VS Code extensions are required.
- [x] Compile and test the initial project.
- [x] Add build tasks through the Makefile.
- [x] Document architecture and the Go learning roadmap.

## Engineering rules

- Keep the project independent from the Python implementation and do not add a Python runtime dependency.
- Prefer the Go standard library; YAML uses `gopkg.in/yaml.v3`.
- Keep `cmd/safeops` thin and place implementation under `internal`.
- Pass `context.Context` across I/O and execution boundaries.
- Construct external commands as argument slices and never invoke a shell.
- Use table-driven tests and run `go test -race ./...` for concurrency-sensitive changes.
- Natural language, templates, RAG, and LLM output must never authorize apply mode.
- Only an explicit operator apply control may request a real change, and all policy gates must still pass.
- Preserve clear, incremental Git commits; do not create releases or tags unless explicitly requested.
