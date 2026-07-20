# safeops-go

`safeops-go` 是 safeops 的纯 Go 行为重写项目，用于系统学习 Go 工程实践。它不依赖 Python，也不是逐行翻译；原 Python 项目只作为功能需求和行为参考。

## 当前状态

学习型重写的完整主链路已经实现：

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
- `safeops goal`：自然语言规划、模板匹配、澄清、分析、检查、审批、执行与验证；
- `--plan-only` 命令预览，不执行 Ansible 变更；
- 每次 goal 运行写入原子 JSON trace，执行时另存日志；
- CLI 与未来 HTTP API 共用的 Agent Kernel service；
- 可选 DeepSeek/OpenAI-compatible Provider，失败时自动回退到确定性规则；
- 无向量数据库依赖的本地 Markdown/TXT lexical RAG；
- `safeops serve`：基于 `net/http` 与 `embed` 的单二进制 Web Console；
- `/api/status`、`/api/goal`、`/api/rag`、`/api/runs` JSON API；
- Web 与 CLI 共用 planner、policy、runner 和 trace，不复制安全逻辑；
- 单元测试、race test、vet 和 GitHub Actions。

项目定位是**能真实运行的 Go 学习与作品集项目**，不是面向生产环境的自动化平台。它有完整安全主链路和测试，但不提供多用户认证、分布式调度、远程密钥托管或高可用能力。

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
go run ./cmd/safeops goal "安全发布 testdata/demo.yml 到 dev" -i testdata/inventory.ini --plan-only
go run ./cmd/safeops serve
```

Web Console 默认监听 `http://127.0.0.1:8765`。可通过 `serve --host 0.0.0.0 --port 9000` 修改地址；暴露到非本机网络前应自行增加反向代理认证。

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

## DeepSeek 与本地 RAG

默认配置不访问外部服务。需要 DeepSeek 时，在配置中设置：

```yaml
api:
	provider: deepseek
	deepseek:
		enabled: true
		base_url: https://api.deepseek.com
		model: deepseek-chat
		timeout: 30
rag:
	enabled: true
	paths:
		- docs/playbooks
	max_documents: 3
	max_chars: 1200
```

API key 推荐通过 `DEEPSEEK_API_KEY` 环境变量提供，不提交到仓库。Provider 只能补充计划提示；路径候选会再次校验，并且 Provider、RAG 文档和自然语言都不能打开 apply 权限。

## 适合作品集展示的技术点

- 用小接口实现依赖倒置：`Runner`、`GoalParser`、`Searcher`、`Sink`；
- 用 `context.Context` 贯穿 HTTP、检查和外部进程取消；
- 用参数切片调用 `exec.CommandContext`，避免 shell 拼接；
- 用保守默认值、独立 policy 和原子 trace 实现可解释安全边界；
- 通过同一 application service 同时支持 CLI 与 Web；
- 使用标准库 HTTP、结构化 JSON、嵌入静态资源和优雅关闭；
- 使用 table-driven test、`httptest`、race detector、vet 和跨平台构建。

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
	- 已完成：本地 planner、模板、澄清门、策略门、共享 service、原子 trace 和运行日志。
4. DeepSeek Provider 与本地 RAG（已完成）。
5. 嵌入式 Web Console（已完成）。
6. race/vet、HTTP 测试和跨平台构建（已完成主项，fuzz 可继续练习）。

详细设计见 `docs/architecture.md`，学习计划见 `docs/go-learning-roadmap.md`。

## Git 策略

每个学习阶段拆分为可验证的提交。当前项目不创建 Release 或 Tag；作品集以源码、测试和文档为主。
