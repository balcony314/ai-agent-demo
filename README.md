# AI Agent Demo

用 Go 实现的教学级 AI Agent，演示 **ReAct (Reasoning + Acting)** 模式。

通过 OpenAI 兼容的 function calling，实现带工具调用的对话式 Agent。使用 cobra 管理 CLI 命令。

## 🚀 快速开始

```bash
# Mock 模式（无需 API key，内置模拟响应）
go run .

# 使用 OpenAI API
go run . --api-key sk-your-key --model gpt-4o

# 使用 Ollama（本地运行）
go run . --api-key dummy --base-url http://localhost:11434/v1 --model qwen2

# 单次提问
go run . chat "什么是 AI Agent"

# 查看版本
go run . version
```

启动后进入交互式 REPL，输入问题即可对话。输入 `quit` 退出。

### 交互命令

- `skills` — 列出所有可用技能
- `skill <名称>` — 切换技能（如 `skill coder`）
- `tools` — 列出所有可用工具
- `history` — 查看对话历史
- `reset` — 重置对话
- `help` — 显示帮助信息
- `quit` / `exit` / `q` — 退出程序

### 使用 Taskfile 构建

项目集成了 [Task](https://taskfile.dev/) 作为构建工具，提供标准化命令：

```bash
task build          # 编译到 build/ai-agent
task run            # Mock 模式运行
task test           # 运行测试
task lint           # 静态分析
task clean          # 清理构建产物
task --list         # 查看所有可用命令
```

## 📁 项目结构

```
ai-agent-demo/
├── main.go              # 入口：调用 cmd.Execute()
├── cmd/
│   ├── root.go          # Root 命令 + 全局 flags + 交互式 REPL + 启动连接检查
│   ├── chat.go          # chat 子命令：单次提问
│   └── version.go       # version 子命令：版本信息
├── agent/
│   ├── types.go         # 核心类型：Message, ToolCall, ToolDefinition, Config, LLMClient 接口
│   ├── tools.go         # 工具注册表 + 5 个内置工具
│   ├── skill.go         # 技能注册表 + 5 个内置技能
│   ├── llm.go           # LLM 客户端（OpenAI Ping + Chat / Mock）
│   └── agent.go         # ReAct 编排循环 + 技能切换
├── Taskfile.yml         # Task 构建配置
└── build/               # 编译输出（gitignore）
```

## 🏗️ 架构设计

```
用户输入 → [Agent] → LLM 推理 → 工具调用 → 结果回传 → LLM 推理 → ... → 最终回答
              ↑                                               ↑
         Skill 系统（角色切换）                        Plan 系统（任务规划）
```

核心是 **ReAct 循环**：LLM 生成文本或工具调用 → Agent 执行工具 → 把结果喂回 LLM → 循环（最多 10 轮），直到 LLM 输出最终文本回答。

### ReAct + Plan 机制

对于复杂任务，Agent 支持 **Plan 机制**：

1. LLM 判断任务复杂度，决定是否需要创建计划
2. 使用 `create_plan` 工具生成执行步骤
3. 按步骤逐一执行，完成后给出最终总结

```
简单任务: 用户输入 → ReAct 循环 → 回答
复杂任务: 用户输入 → 创建计划 → 按步骤执行 → 总结回答
```

### 关键抽象

- **`LLMClient` 接口**（`Chat` 方法）— 可在 `OpenAIClient` 和 `MockClient` 之间切换
- **`ToolRegistry`** — `map[string]Tool`，提供 `Register`/`Get`/`Definitions`
- **`SkillRegistry`** — `map[string]Skill`，提供 `Register`/`Get`/`List`
- **`Message`** — role + content + 可选的 tool_calls/tool_call_id，对齐 OpenAI chat 格式
- **`Plan`/`Step`** — 执行计划和步骤定义

## 🔧 内置工具

| 名称 | 描述 | 示例触发 |
|------|------|----------|
| `search` | 模拟搜索引擎 | "帮我搜索 Go 并发编程" |
| `calculator` | 安全数学表达式计算 | "计算 123 * 456 + 789" |
| `current_time` | 获取当前日期时间 | "现在几点了？" |
| `text_transform` | 文本处理（大小写/反转/长度） | "把 hello 转大写" |
| `create_plan` | 为复杂任务创建执行计划 | "帮我分析 Go 和 Python 的区别" |

## 🎭 技能系统

技能（Skill）是预定义的 Agent 角色配置，让同一个 Agent 切换不同"人格"。

| 技能 | 描述 | 切换命令 |
|------|------|----------|
| `general` | 通用助手（默认） | `skill general` |
| `coder` | 代码助手 | `skill coder` |
| `translator` | 翻译官 | `skill translator` |
| `analyst` | 数据分析师 | `skill analyst` |
| `storyteller` | 故事大王 | `skill storyteller` |

## ⚙️ 配置

### CLI 参数

| 参数 | 短参数 | 默认值 | 描述 |
|------|--------|--------|------|
| `--api-key` | `-k` | `""` | API Key（空则使用 Mock 模式） |
| `--base-url` | `-u` | `""` | API 地址（默认 OpenAI） |
| `--model` | `-m` | `gpt-4o` | 模型名称 |
| `--mock` | | `false` | 强制使用 Mock 模式 |

### 子命令

| 命令 | 描述 |
|------|------|
| `chat [问题]` | 单次提问（非交互式） |
| `version` | 显示版本信息 |

### 扩展工具

在 `agent/tools.go` 的 `RegisterBuiltinTools()` 中添加新工具：

```go
registry.Register(Tool{
    Definition: ToolDefinition{
        Type: "function",
        Function: FunctionSchema{
            Name:        "my_tool",
            Description: "工具描述（LLM 根据这个决定何时调用）",
            Parameters:  json.RawMessage(`{"type": "object", "properties": {...}}`),
        },
    },
    Execute: func(args json.RawMessage) (string, error) {
        // 解析参数、执行逻辑、返回结果
        return "result", nil
    },
})
```

### 扩展技能

在 `agent/skill.go` 的 `RegisterBuiltinSkills()` 中添加新技能：

```go
registry.Register(Skill{
    Name:         "my_skill",
    Description:  "技能描述",
    SystemPrompt: "你是一个自定义角色...",
    Tools:        []string{"search", "calculate"}, // 可选：限制可用工具
})
```

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行测试并生成覆盖率报告
go test ./... -coverprofile=coverage.out

# 查看覆盖率详情
go tool cover -func=coverage.out

# 生成 HTML 覆盖率报告
go tool cover -html=coverage.out -o coverage.html

# 当前覆盖率：85.9%（agent 包 97.3%）
```

## 📄 License

MIT
