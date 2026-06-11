# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在本仓库中工作提供指引。

## 项目概述

用 Go 实现的教学级 AI Agent，演示 **ReAct (Reasoning + Acting)** 模式。通过 OpenAI 兼容的 function calling 实现带工具调用（计算器、时间、搜索、文本处理、计划创建）的对话式 Agent。使用 cobra 管理 CLI 命令。

支持 **ReAct + Plan 机制**：对于复杂任务，Agent 可以先创建执行计划，再按步骤执行。

## 常用命令

```bash
go run .                                          # 交互式 REPL（Mock 模式）
go run . --api-key sk-xxx --model gpt-4o          # 接 OpenAI
go run . --api-key dummy -u http://localhost:11434/v1 -m qwen2  # 接 Ollama
go run . chat "你好"                              # 单次提问
go run . skill list                               # 查看技能列表
go run . version                                  # 版本信息
go build -o ai-agent .                            # 编译
go test ./...                                     # 运行测试
go test ./... -coverprofile=coverage.out          # 测试+覆盖率
go vet ./...                                      # 静态检查
```

全局 flags：`--api-key`/`-k`、`--base-url`/`-u`、`--model`/`-m`（默认 `gpt-4o`）、`--mock`

## 架构

采用 **Plan + ReAct 两阶段执行模式**：

```
用户消息
   ↓
阶段 1: Plan（planPhase）
   LLM 分析任务复杂度 → 简单任务直接 ReAct，复杂任务生成执行计划
   ↓
阶段 2: Execute
   对计划中的每个步骤执行 ReAct 循环（reactLoop）
   ↓
汇总结果（summarizeResults）→ 返回最终答案
```

```
main.go           → 入口：调用 cmd.Execute()
cmd/
  root.go         → Root 命令 + 全局 flags + 交互式 REPL + 启动连接检查
  chat.go         → chat 子命令：单次提问（非交互式）
  version.go      → version 子命令：版本信息
agent/
  types.go        → 核心类型：Message, ToolCall, ToolDefinition, Config, LLMClient 接口 + ConfigWithModel
  tools.go        → ToolRegistry（map 实现）+ RegisterBuiltinTools() 注册 5 个内置工具
  skill.go        → SkillRegistry + RegisterBuiltinSkills() 注册 5 个内置技能
  llm.go          → OpenAIClient（Ping + /v1/chat/completions）+ MockClient（演示用）
  agent.go        → Agent 核心：Plan + ReAct 两阶段编排 + Skill 切换
```

### agent.go 核心方法

| 方法 | 职责 |
|------|------|
| `Run(userInput)` | 主入口：Plan 阶段 → Execute 阶段 → 汇总结果 |
| `planPhase()` | 阶段 1：调用 LLM 判断任务复杂度，复杂任务生成 Plan，简单任务返回 nil |
| `executeStep(step, stepNum, totalSteps)` | 对单个计划步骤构建提示并调用 reactLoop |
| `reactLoop(taskPrompt)` | ReAct 核心循环：LLM 生成 → 工具执行 → 结果回传，最多 MaxTurns 轮 |
| `summarizeResults(plan, stepResults)` | 汇总所有步骤结果，调用 LLM 生成最终答案 |
| `registerPlanTool()` | 注册 `create_plan` 工具（闭包捕获 Agent 引用） |
| `setPlan(plan)` | 设置当前计划（创建副本，不修改原始数据） |

关键抽象：
- **`LLMClient` 接口**（`Chat` 方法）—— 可在 `OpenAIClient` 和 `MockClient` 之间切换
- **`OpenAIClient.Ping()`** —— 启动时检查 API 可达性，带 Authorization header，失败则退出
- **`ConfigWithModel(model)`** —— 将模型名注入系统提示词，让 LLM 知道自己使用的模型
- **`ToolRegistry`** —— `map[string]Tool`，提供 `Register`/`Get`/`Definitions`；每个 `Tool` 由 JSON Schema 的 `ToolDefinition` + `Execute` 函数组成
- **`Message`** —— role + content + 可选的 tool_calls/tool_call_id，对齐 OpenAI chat 格式

## 添加新工具

在 `agent/tools.go` 的 `RegisterBuiltinTools()` 中注册。提供带 JSON Schema 参数的 `ToolDefinition` 和一个 `Execute` 函数。LLM 根据 description 决定何时调用。

## Skill 系统

Skill 是预定义的 Agent 角色配置，可让同一个 Agent 切换不同"人格"。

内置技能：
- `general` - 通用助手（默认）
- `coder` - 代码助手
- `translator` - 翻译官
- `analyst` - 数据分析师
- `storyteller` - 故事大王

交互命令：
- `skills` - 列出所有可用技能
- `skill <名称>` - 切换到指定技能

添加新技能：在 `agent/skill.go` 的 `RegisterBuiltinSkills()` 中注册。每个 Skill 包含 Name、Description、SystemPrompt 和可选的 Tools 列表。

## Plan 系统

Plan + ReAct 两阶段执行模式，让 Agent 能够处理复杂任务。

核心类型：
- `Plan` — 执行计划，包含 Goal（目标）和 Steps（步骤列表）
- `Step` — 单个步骤，包含 ID、Description、Status

执行流程（`Run` 方法驱动）：

```
1. planPhase()
   ├─ LLM 分析任务复杂度
   ├─ 简单任务 → 返回 nil → 直接进入 reactLoop
   └─ 复杂任务 → 调用 create_plan 工具 → 返回 Plan

2. 逐步骤执行
   └─ 对每个 Step 调用 executeStep() → reactLoop()

3. summarizeResults()
   └─ 汇总所有步骤结果 → LLM 生成最终答案
```

关键设计：
- `planPhase()` 使用临时历史副本，不污染主对话历史
- 简单任务（单步可完成）跳过计划，直接 ReAct 执行
- 步骤执行失败不中断后续步骤，错误信息记入汇总
- `reactLoop` 最多执行 `MaxTurns` 轮（默认 10 轮）

## 代码风格

- 注释和文档为中文（教学项目）
- 标准 Go 错误处理：`fmt.Errorf` + `%w` 包装
- 全程使用 OpenAI API 兼容的请求/响应格式
- 参考规范：
  - [Uber Go 语言风格指南（中文）](https://github.com/xxjwxc/uber_go_guide_cn)
  - [Go 标准项目布局](https://github.com/golang-standards/project-layout)
