# 贡献指南

## 开发环境

### 前置要求

- Go 1.23+
- [Task](https://taskfile.dev/)（可选，用于标准化构建命令）

### 获取代码

```bash
git clone https://github.com/your-username/ai-agent-demo.git
cd ai-agent-demo
go mod download
```

## 开发流程

### 1. 创建分支

```bash
git checkout -b feature/your-feature
# 或
git checkout -b fix/your-fix
```

### 2. 开发

项目采用 **测试驱动开发 (TDD)**：

```bash
# 1. 编写测试
# 2. 运行测试（预期失败）
go test ./...

# 3. 编写实现代码
# 4. 运行测试（预期通过）
go test ./...

# 5. 重构并确保测试仍然通过
```

### 3. 代码检查

```bash
# 静态分析
go vet ./...

# 运行所有测试
go test ./...

# 检查覆盖率（目标：80%+）
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### 4. 提交

遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>: <description>

[可选正文]
```

类型：
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `test`: 测试相关
- `refactor`: 重构
- `chore`: 构建/工具相关

示例：
```bash
git commit -m "feat: 添加新的搜索工具"
git commit -m "fix: 修复 calculator 的除零错误"
git commit -m "docs: 更新 README 的快速开始部分"
```

## 项目结构

```
ai-agent-demo/
├── main.go              # 入口
├── cmd/                 # CLI 命令（cobra）
│   ├── root.go          # 根命令 + 全局 flags
│   ├── chat.go          # chat 子命令
│   └── version.go       # version 子命令
├── agent/               # 核心逻辑
│   ├── types.go         # 类型定义
│   ├── tools.go         # 工具注册表
│   ├── skill.go         # 技能注册表
│   ├── llm.go           # LLM 客户端
│   └── agent.go         # ReAct 编排
├── docs/                # 文档
├── Taskfile.yml         # 构建配置
└── build/               # 编译输出
```

## 添加新功能

### 添加工具

1. 在 `agent/tools.go` 的 `RegisterBuiltinTools()` 中注册
2. 提供 `ToolDefinition`（含 JSON Schema）和 `Execute` 函数
3. 在 `agent/tools_test.go` 中添加测试

### 添加技能

1. 在 `agent/skill.go` 的 `RegisterBuiltinSkills()` 中注册
2. 提供 `Name`、`Description`、`SystemPrompt` 和可选的 `Tools` 列表
3. 在 `agent/skill_test.go` 中添加测试

### 添加 CLI 命令

1. 在 `cmd/` 目录创建新文件
2. 使用 cobra 定义命令
3. 在 `init()` 中注册到 `rootCmd`
4. 添加对应的测试文件

## 测试规范

### 单元测试

- 文件命名：`*_test.go`
- 函数命名：`TestXxx(t *testing.T)`
- 使用 `testing.T` 报告错误
- 测试覆盖率目标：80%+

### 测试示例

```go
func TestMyFunction(t *testing.T) {
    // 准备
    input := "test"

    // 执行
    result, err := MyFunction(input)

    // 验证
    if err != nil {
        t.Fatalf("MyFunction 失败: %v", err)
    }
    if result != "expected" {
        t.Errorf("预期 'expected'，实际 '%s'", result)
    }
}
```

## 代码风格

- 注释和文档使用中文（教学项目）
- 错误处理：`fmt.Errorf` + `%w` 包装
- 函数长度：< 50 行
- 文件长度：< 800 行

## 提交 PR

1. 确保所有测试通过：`go test ./...`
2. 确保静态检查通过：`go vet ./...`
3. 更新相关文档
4. 填写 PR 描述，说明变更内容和原因
5. 关联相关 Issue

## 常见问题

### Q: 如何在 Mock 模式下测试？

```bash
go run . --mock
# 或
go run .  # 没有 --api-key 时自动使用 Mock
```

### Q: 如何连接本地 Ollama？

```bash
go run . --api-key dummy --base-url http://localhost:11434/v1 --model qwen2
```

### Q: 如何查看测试覆盖率？

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out  # 在浏览器中查看
```
