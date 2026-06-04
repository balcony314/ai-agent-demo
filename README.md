# AI Agent Demo

用 Go 实现的教学级 AI Agent，演示 **ReAct (Reasoning + Acting)** 模式。

通过 OpenAI 兼容的 function calling，实现带工具调用的对话式 Agent。零第三方依赖，仅用标准库。

## 🚀 快速开始

```bash
# Mock 模式（无需 API key，内置模拟响应）
go run .

# 使用 OpenAI API
go run . -api-key sk-your-key -model gpt-4o

# 使用 Ollama（本地运行）
go run . -api-key dummy -base-url http://localhost:11434/v1 -model qwen2

# 查看版本
go run . -version
```

启动后进入交互式 REPL，输入问题即可对话。输入 `quit` 退出。

### 交互命令

- `skills` — 列出所有可用技能
- `skill <名称>` — 切换技能（如 `skill coder`）
- `quit` / `exit` / `q` — 退出程序

### 使用 Taskfile 构建

项目集成了 [Task](https://taskfile.dev/) 作为构建工具，提供标准化命令：

```bash
task build          # 编译到 build/ai-agent
task run            # Mock 模式运行
task test           # 运行测试
task lint           # 静态分析
task clean          # 清理构建产物
task docker:build   # 构建 Docker 镜像
task docker:run     # Docker 容器运行（交互式）
task --list         # 查看所有可用命令
```

## 📁 项目结构

```
ai-agent-demo/
├── main.go              # CLI 入口 + 交互式 REPL
├── agent/
│   ├── types.go         # 核心类型：Message, ToolCall, ToolDefinition, Config
│   ├── tools.go         # 工具注册表 + 4 个内置工具
│   ├── skill.go         # 技能注册表 + 5 个内置技能
│   ├── llm.go           # LLM 客户端（OpenAI / Mock）
│   └── agent.go         # ReAct 编排循环 + 技能切换
├── Taskfile.yml         # Task 构建配置
├── Dockerfile           # 多阶段 Docker 构建
└── build/               # 编译输出（gitignore）
```

## 🏗️ 架构设计

```
用户输入 → [Agent] → LLM 推理 → 工具调用 → 结果回传 → LLM 推理 → ... → 最终回答
              ↑
         Skill 系统（角色切换）
```

核心是 **ReAct 循环**：LLM 生成文本或工具调用 → Agent 执行工具 → 把结果喂回 LLM → 循环（最多 10 轮），直到 LLM 输出最终文本回答。

### 关键抽象

- **`LLMClient` 接口**（`Chat` 方法）— 可在 `OpenAIClient` 和 `MockClient` 之间切换
- **`ToolRegistry`** — `map[string]Tool`，提供 `Register`/`Get`/`Definitions`
- **`SkillRegistry`** — `map[string]Skill`，提供 `Register`/`Get`/`GetNames`/`Definitions`
- **`Message`** — role + content + 可选的 tool_calls/tool_call_id，对齐 OpenAI chat 格式

## 🔧 内置工具

| 名称 | 描述 | 示例触发 |
|------|------|----------|
| `search` | 模拟搜索引擎 | "帮我搜索 Go 并发编程" |
| `calculate` | 安全数学表达式计算 | "计算 123 * 456 + 789" |
| `get_current_time` | 获取当前日期时间 | "现在几点了？" |
| `string_process` | 文本统计（字数/字符/反转/大小写） | "统计这段文字的字数" |

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

| 参数 | 默认值 | 描述 |
|------|--------|------|
| `-api-key` | `""` | API Key（空则使用 Mock 模式） |
| `-base-url` | `https://api.openai.com/v1` | API 地址 |
| `-model` | `gpt-4o` | 模型名称 |
| `-mock` | `false` | 强制使用 Mock 模式 |
| `-version` | `false` | 显示版本号 |

### 扩展工具

在 `agent/tools.go` 的 `RegisterBuiltinTools()` 中添加新工具：

```go
registry.Register(Tool{
    Definition: ToolDefinition{
        Name:        "my_tool",
        Description: "工具描述（LLM 根据这个决定何时调用）",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "input": map[string]interface{}{
                    "type":        "string",
                    "description": "参数说明",
                },
            },
            "required": []string{"input"},
        },
    },
    Execute: func(args map[string]interface{}) (string, error) {
        // 工具逻辑
        return "结果", nil
    },
})
```

### 扩展技能

在 `agent/skill.go` 的 `RegisterBuiltinSkills()` 中添加新技能：

```go
registry.Register(Skill{
    Name:        "my_skill",
    Description: "技能描述",
    SystemPrompt: "你是一个自定义角色...",
    Tools:       []string{"search", "calculate"}, // 可选：限制可用工具
})
```

## 📄 License

MIT
