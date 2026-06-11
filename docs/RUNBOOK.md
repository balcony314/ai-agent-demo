# 运维手册

## 部署

### 构建

```bash
# 使用 Task 构建
task build

# 或直接使用 Go
go build -o build/ai-agent .

# 交叉编译（Linux）
GOOS=linux GOARCH=amd64 go build -o build/ai-agent-linux .

# 交叉编译（macOS）
GOOS=darwin GOARCH=arm64 go build -o build/ai-agent-mac .
```

### 运行

```bash
# 交互模式（Mock）
./build/ai-agent

# 交互模式（OpenAI）
./build/ai-agent --api-key sk-xxx --model gpt-4o

# 单次提问
./build/ai-agent chat "什么是 AI Agent"

# 查看版本
./build/ai-agent version
```

### 环境变量

| 变量 | 描述 | 示例 |
|------|------|------|
| `OPENAI_API_KEY` | OpenAI API Key | `sk-proj-xxx` |
| `OPENAI_BASE_URL` | 自定义 API 地址 | `http://localhost:11434/v1` |

## 监控

### 日志

程序运行时会输出以下信息：

```
🤖 LLM 模式: 真实 API
   模型: gpt-4o
   地址: https://api.openai.com/v1

🔗 检查 API 连接... ✅

📝 用户: 你好
🔄 推理轮次 1/10
🛠️  LLM 请求调用 1 个工具
✅ Agent 回复完成 (共 1 轮推理)
```

### 关键指标

- **推理轮次**：每次对话的 ReAct 循环次数（最大 10）
- **工具调用**：工具执行成功/失败次数
- **响应时间**：端到端对话延迟

## 常见问题

### 1. 启动时连接检查失败

**症状**：
```
🔗 检查 API 连接... ❌

❌ 无法连接到 API: ...
```

**解决**：
- 检查 API Key 和 base-url 是否正确
- 确认域名可达：`ping <hostname>`
- 如果 ping 通但 Go 报 `no such host`，尝试：`GODEBUG=netdns=cgo ./build/ai-agent ...`
- 如果返回 401，检查 API Key 是否有效

### 2. API Key 无效

**症状**：
```
❌ 错误: LLM API 调用失败: 401 Unauthorized
```

**解决**：
- 检查 API Key 是否正确
- 验证 API Key 是否有足够额度
- 确认 API 地址是否正确

### 3. DNS 解析失败（Go 特有）

**症状**：
```
❌ 无法连接到 API: ... dial tcp: lookup xxx: no such host
```

但 `ping` 命令可以正常解析该域名。

**原因**：Go 默认使用纯 Go DNS 解析器，与系统解析器行为不一致。

**解决**：
```bash
# 方法 1：使用系统 DNS 解析器
GODEBUG=netdns=cgo ./build/ai-agent --api-key ... --base-url ...

# 方法 2：在 /etc/hosts 中添加域名映射
echo "1.2.3.4 your-api-host.com" | sudo tee -a /etc/hosts
```

### 4. 连接超时

**症状**：
```
❌ 错误: LLM API 调用失败: context deadline exceeded
```

**解决**：
- 检查网络连接
- 确认 API 地址可达
- 检查防火墙设置

### 5. Ollama 连接失败

**症状**：
```
❌ 错误: connection refused
```

**解决**：
- 确认 Ollama 正在运行：`ollama serve`
- 检查端口：`curl http://localhost:11434`
- 确认模型已下载：`ollama pull qwen2`

### 6. 工具执行失败

**症状**：
```
🛠️  执行工具 calculator 失败: ...
```

**解决**：
- 检查工具参数格式
- 查看工具实现代码
- 验证 JSON Schema 定义

## 回滚

### Git 回滚

```bash
# 查看提交历史
git log --oneline

# 回滚到指定提交
git reset --hard <commit-hash>

# 回滚并保留更改
git revert <commit-hash>
```

### 版本回退

```bash
# 检出旧版本
git checkout v1.0.0

# 重新构建
task build

# 运行旧版本
./build/ai-agent
```

## 性能调优

### LLM 响应慢

1. 使用更快的模型（如 `gpt-4o-mini`）
2. 减少工具数量
3. 简化系统提示

### 内存占用高

1. 定期重置对话历史（`reset` 命令）
2. 减少历史消息保留数量
3. 使用单次提问模式（`chat`）

## 安全

### API Key 管理

- **不要**硬编码 API Key
- **使用**环境变量或配置文件
- **定期**轮换 API Key

### 输入验证

- 所有用户输入都会经过验证
- 工具参数使用 JSON Schema 校验
- 错误信息不包含敏感数据

## 扩展

### 添加新 LLM 后端

1. 实现 `LLMClient` 接口
2. 在 `cmd/root.go` 的 `initLLMClient()` 中添加判断
3. 如果需要启动时连接检查，实现 `Ping()` 方法并在 `checkAPIConnection()` 中处理
4. 测试连接

### 添加新工具

1. 在 `agent/tools.go` 注册
2. 实现 `Execute` 函数
3. 添加测试用例

### 添加新技能

1. 在 `agent/skill.go` 注册
2. 定义 `SystemPrompt`
3. 可选：限制可用工具

## 联系方式

- Issue: GitHub Issues
- 文档: `docs/` 目录
- 代码: `agent/` 和 `cmd/` 目录
