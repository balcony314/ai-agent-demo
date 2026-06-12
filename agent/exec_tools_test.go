package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestRegisterExecTools(t *testing.T) {
	registry := NewToolRegistry()
	RegisterExecTools(registry)

	// 验证工具已注册
	tools := registry.Names()
	expectedTools := []string{"execute_command", "manage_process"}

	for _, name := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("工具 %q 未注册", name)
		}
	}
}

func TestExecuteCommandTool_BasicExecution(t *testing.T) {
	// 保存并恢复原始函数
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	// 设置环境变量避免审计日志
	os.Setenv("EXEC_AUDIT_LOG", "")
	defer os.Unsetenv("EXEC_AUDIT_LOG")

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, ok := registry.Get("execute_command")
	if !ok {
		t.Fatal("工具未找到")
	}

	args := json.RawMessage(`{"command": "echo hello"}`)
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestExecuteCommandTool_BlockedCommand(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("execute_command")

	args := json.RawMessage(`{"command": "rm -rf /"}`)
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("危险命令应被拒绝")
	}
}

func TestExecuteCommandTool_SensitiveCommand(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("execute_command")

	// 不确认
	args := json.RawMessage(`{"command": "rm file.txt", "confirm": false}`)
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if result == "" {
		t.Error("应返回确认消息")
	}

	// 确认后执行
	args = json.RawMessage(`{"command": "echo confirmed", "confirm": true}`)
	result, err = tool.Execute(args)
	if err != nil {
		t.Fatalf("确认后执行失败: %v", err)
	}
}

func TestManageProcessTool_List(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("manage_process")

	args := json.RawMessage(`{"action": "list"}`)
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestManageProcessTool_InvalidAction(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("manage_process")

	args := json.RawMessage(`{"action": "invalid"}`)
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("无效操作应返回错误")
	}
}

func TestManageProcessTool_StatusWithoutPID(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("manage_process")

	args := json.RawMessage(`{"action": "status"}`)
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("status 操作无 pid 应返回错误")
	}
}

func TestCloseExecutor(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	// 重置单例
	CloseExecutor()

	// 获取共享执行器（会创建新的）
	executor := getSharedExecutor()
	if executor == nil {
		t.Fatal("执行器不应为 nil")
	}

	// 关闭
	CloseExecutor()

	// 再次获取应创建新实例
	executor2 := getSharedExecutor()
	if executor2 == nil {
		t.Fatal("新执行器不应为 nil")
	}

	// 清理
	CloseExecutor()
}

func TestExecuteCommandTool_BackgroundExecution(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	// 重置单例以清理旧进程
	CloseExecutor()
	defer CloseExecutor()

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("execute_command")

	// 后台执行命令
	args := json.RawMessage(`{"command": "sleep 0.5", "background": true}`)
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("后台执行失败: %v", err)
	}
	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestManageProcessTool_KillWithoutPID(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("manage_process")

	args := json.RawMessage(`{"action": "kill"}`)
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("kill 操作无 pid 应返回错误")
	}
}

func TestManageProcessTool_StatusNotFound(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	// 重置单例
	CloseExecutor()
	defer CloseExecutor()

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("manage_process")

	args := json.RawMessage(`{"action": "status", "pid": 99999}`)
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("不存在的进程应返回错误")
	}
}

func TestManageProcessTool_KillNotFound(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	// 重置单例
	CloseExecutor()
	defer CloseExecutor()

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	tool, _ := registry.Get("manage_process")

	args := json.RawMessage(`{"action": "kill", "pid": 99999}`)
	_, err := tool.Execute(args)
	if err == nil {
		t.Error("不存在的进程应返回错误")
	}
}

func TestManageProcessTool_StatusRunning(t *testing.T) {
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	tmpDir := t.TempDir()
	getWorkDir = func() string { return tmpDir }

	// 重置单例以清理旧进程
	CloseExecutor()
	defer CloseExecutor()

	registry := NewToolRegistry()
	RegisterExecTools(registry)

	// 后台启动一个 sleep
	execTool, _ := registry.Get("execute_command")
	bgArgs := json.RawMessage(`{"command": "sleep 5", "background": true}`)
	_, err := execTool.Execute(bgArgs)
	if err != nil {
		t.Fatalf("后台执行失败: %v", err)
	}

	// 获取进程管理器并查询状态
	pm := getSharedExecutor().GetProcessManager()
	procs := pm.List()
	if len(procs) == 0 {
		t.Fatal("应有后台进程")
	}

	// 查询状态
	tool, _ := registry.Get("manage_process")
	statusArgs := fmt.Sprintf(`{"action": "status", "pid": %d}`, procs[0].PID)
	result, err := tool.Execute(json.RawMessage(statusArgs))
	if err != nil {
		t.Fatalf("查询状态失败: %v", err)
	}
	if !strings.Contains(result, "运行中") {
		t.Errorf("应包含运行中状态: %s", result)
	}
}
