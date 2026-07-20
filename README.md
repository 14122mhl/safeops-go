# safeops-go

`safeops-go` 是 safeops 的纯 Go 行为重写项目，用于系统学习 Go 工程实践。它不依赖 Python，也不是逐行翻译；原 Python 项目只作为功能需求和行为参考。

## 当前阶段

前两个阶段已经实现：

- 独立 Go module 与分层包结构；
- `safeops doctor`；
- `safeops config init/show`；
- `safeops version` 和帮助输出；
- YAML 默认配置、校验与密钥脱敏；
- 核心领域模型；
- 确定性的 goal planner 和 apply policy；
- Ansible dry-run/apply 命令构造器；
- `safeops inspect`：解析 play、task、FQCN 模块并计算整体风险；
- `safeops check`：检查文件、环境、静态风险、syntax、host 和 task list；
- `safeops run`：预检查后默认 dry-run，支持超时和取消；
- 显式 `--apply --approve` 与生产 `--confirm PROD` 策略门禁；
- JSON check report；
- 单元测试、race test、vet 和 GitHub Actions。

完整 `goal` 工作流、trace、DeepSeek、RAG 与 Web Console 将按学习路线逐阶段实现。

## 安全边界

自然语言、模板、RAG 和 LLM 只能表达意图，不能授权真实执行。计划中的 `Apply` 只能来自显式操作员控制；未来真实执行还必须通过预检查、审批、置信度和生产确认门禁。

## 环境

- Go 1.24+
- 可选：Ansible，用于后续 playbook 校验和执行阶段

## 快速开始

```bash
go mod download
go run ./cmd/safeops help
go run ./cmd/safeops doctor
go run ./cmd/safeops --config ./config.yaml config init
go run ./cmd/safeops --config ./config.yaml config show
go run ./cmd/safeops inspect testdata/demo.yml
go run ./cmd/safeops check testdata/demo.yml -i testdata/inventory.ini --env dev
go run ./cmd/safeops run testdata/demo.yml -i testdata/inventory.ini --env dev
```

`run` 默认向 Ansible 添加 `--check --diff`。非生产真实执行必须显式添加 `--apply --approve`；生产环境还必须添加 `--confirm PROD`。

```bash
go run ./cmd/safeops run deploy.yml -i inventory.ini --env dev --apply --approve
go run ./cmd/safeops run deploy.yml -i inventory.ini --env prod --apply --approve --confirm PROD
```

构建与验证：

```bash
make build
make test
make race
make vet
```

生成的二进制位于 `bin/safeops`。

## 项目结构

```text
cmd/safeops/                 CLI 进程入口
internal/cli/                参数解析与输出
internal/config/             YAML 配置
internal/model/              领域模型
internal/analysis/           Playbook 静态分析
internal/check/              环境与预检查
internal/engine/             命令构造与执行
internal/agent/planner/      目标规范化
internal/agent/policy/       独立执行策略
internal/agent/service/      Agent Kernel 编排边界
internal/agent/template/     变更模板
internal/agent/rag/          本地文档检索
internal/provider/deepseek/  可选 LLM Provider
internal/trace/              审计记录
internal/web/                HTTP API 与嵌入式控制台
testdata/                    测试 fixture
docs/                        架构和学习记录
```

## 重写路线

1. 基础工程、配置、模型和 CLI。
2. `inspect/check/run` 与安全命令执行（已完成）。
3. Agent Kernel、trace 和端到端测试。
4. DeepSeek Provider 与本地 RAG。
5. 嵌入式 Web Console。
6. 行为对照、race/fuzz 测试和跨平台构建。

详细设计见 `docs/architecture.md`，学习计划见 `docs/go-learning-roadmap.md`。

## Git 策略

每个学习阶段拆分为可验证的提交。当前项目不会创建 Release 或 Tag，直到完整重写完成并明确决定发布。
