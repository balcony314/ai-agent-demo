package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ─── 路径安全验证测试 ─────────────────────────────────────────

func TestValidatePath_Valid(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	tests := []struct {
		name string
		path string
	}{
		{"相对路径", "test.txt"},
		{"当前目录", "./test.txt"},
		{"子目录", "subdir/test.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validatePath(tt.path)
			if err != nil {
				t.Fatalf("validatePath(%q) 返回错误: %v", tt.path, err)
			}
			if result == "" {
				t.Error("返回路径不应为空")
			}
		})
	}
}

func TestValidatePath_EmptyPath(t *testing.T) {
	_, err := validatePath("")
	if err == nil {
		t.Error("空路径应返回错误")
	}
}

func TestValidatePath_DotDotTraversal(t *testing.T) {
	tests := []string{
		"../test.txt",
		"subdir/../../test.txt",
		"../outside.txt",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			_, err := validatePath(path)
			if err == nil {
				t.Errorf("路径 %q 应返回错误（包含 ..）", path)
			}
		})
	}
}

func TestValidatePath_AbsolutePathOutside(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 尝试访问工作目录外的绝对路径
	_, err := validatePath("/etc/passwd")
	if err == nil {
		t.Error("绝对路径应返回错误（超出工作目录范围）")
	}
}

// ─── 辅助函数测试 ─────────────────────────────────────────────

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"main.go", true},
		{"script.py", true},
		{"app.js", true},
		{"style.css", true},
		{"data.json", true},
		{"README.md", true},
		{"Makefile", true},
		{"Dockerfile", true},
		{"image.png", false},
		{"binary.exe", false},
		{"archive.zip", false},
		{"video.mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isTextFile(tt.path)
			if got != tt.want {
				t.Errorf("isTextFile(%q) = %v, 期望 %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSize(tt.size)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, 期望 %q", tt.size, got, tt.want)
			}
		})
	}
}

// ─── read_file 测试 ──────────────────────────────────────────

func TestReadFileTool_Success(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	testFile := filepath.Join(workDir, "test.txt")
	content := "Hello\nWorld\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("read_file")

	result, err := tool.Execute(json.RawMessage(`{"path":"test.txt"}`))
	if err != nil {
		t.Fatalf("read_file 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestReadFileTool_NotFound(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("read_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"nonexistent.txt"}`))
	if err == nil {
		t.Error("读取不存在的文件应返回错误")
	}
}

func TestReadFileTool_IsDirectory(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("read_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"."}`))
	if err == nil {
		t.Error("读取目录应返回错误")
	}
}

func TestReadFileTool_NonTextFile(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建二进制文件
	testFile := filepath.Join(workDir, "binary.bin")
	os.WriteFile(testFile, []byte{0x00, 0x01, 0x02}, 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("read_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"binary.bin"}`))
	if err == nil {
		t.Error("读取非文本文件应返回错误")
	}
}

func TestReadFileTool_FileTooLarge(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建大文件（超过 1MB）
	testFile := filepath.Join(workDir, "large.txt")
	data := make([]byte, 1024*1024+1)
	os.WriteFile(testFile, data, 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("read_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"large.txt"}`))
	if err == nil {
		t.Error("读取超大文件应返回错误")
	}
}

// ─── write_file 测试 ─────────────────────────────────────────

func TestWriteFileTool_Success(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("write_file")

	result, err := tool.Execute(json.RawMessage(`{"path":"output.txt","content":"Hello World"}`))
	if err != nil {
		t.Fatalf("write_file 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}

	// 验证文件已创建
	content, err := os.ReadFile(filepath.Join(workDir, "output.txt"))
	if err != nil {
		t.Fatalf("读取写入的文件失败: %v", err)
	}
	if string(content) != "Hello World" {
		t.Errorf("文件内容 = %q, 期望 %q", string(content), "Hello World")
	}
}

func TestWriteFileTool_CreateDirs(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("write_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"subdir/nested/output.txt","content":"Hello"}`))
	if err != nil {
		t.Fatalf("write_file 执行失败: %v", err)
	}

	// 验证文件已创建
	if _, err := os.Stat(filepath.Join(workDir, "subdir", "nested", "output.txt")); err != nil {
		t.Fatalf("文件未创建: %v", err)
	}
}

func TestWriteFileTool_NonTextFile(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("write_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"binary.bin","content":"data"}`))
	if err == nil {
		t.Error("写入非文本文件应返回错误")
	}
}

// ─── edit_file 测试 ──────────────────────────────────────────

func TestEditFileTool_Replace(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	testFile := filepath.Join(workDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello World\nHello Go"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	result, err := tool.Execute(json.RawMessage(`{"path":"test.txt","action":"replace","find":"Hello","replace_with":"Hi"}`))
	if err != nil {
		t.Fatalf("edit_file 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}

	// 验证文件内容
	content, _ := os.ReadFile(testFile)
	expected := "Hi World\nHi Go"
	if string(content) != expected {
		t.Errorf("文件内容 = %q, 期望 %q", string(content), expected)
	}
}

func TestEditFileTool_Append(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件（以换行符结尾）
	testFile := filepath.Join(workDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello\n"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"test.txt","action":"append","content":"World"}`))
	if err != nil {
		t.Fatalf("edit_file 执行失败: %v", err)
	}

	// 验证文件内容
	content, _ := os.ReadFile(testFile)
	expected := "Hello\nWorld"
	if string(content) != expected {
		t.Errorf("文件内容 = %q, 期望 %q", string(content), expected)
	}
}

func TestEditFileTool_Insert(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	testFile := filepath.Join(workDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Line 1\nLine 3"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"test.txt","action":"insert","line":2,"content":"Line 2"}`))
	if err != nil {
		t.Fatalf("edit_file 执行失败: %v", err)
	}

	// 验证文件内容
	content, _ := os.ReadFile(testFile)
	expected := "Line 1\nLine 2\nLine 3"
	if string(content) != expected {
		t.Errorf("文件内容 = %q, 期望 %q", string(content), expected)
	}
}

func TestEditFileTool_Delete(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	testFile := filepath.Join(workDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Line 1\nLine 2\nLine 3"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"test.txt","action":"delete","line":2}`))
	if err != nil {
		t.Fatalf("edit_file 执行失败: %v", err)
	}

	// 验证文件内容
	content, _ := os.ReadFile(testFile)
	expected := "Line 1\nLine 3"
	if string(content) != expected {
		t.Errorf("文件内容 = %q, 期望 %q", string(content), expected)
	}
}

func TestEditFileTool_NotFound(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"nonexistent.txt","action":"replace","find":"a","replace_with":"b"}`))
	if err == nil {
		t.Error("编辑不存在的文件应返回错误")
	}
}

func TestEditFileTool_InvalidLineInsert(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	testFile := filepath.Join(workDir, "test.txt")
	os.WriteFile(testFile, []byte("Line 1"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"test.txt","action":"insert","line":10,"content":"test"}`))
	if err == nil {
		t.Error("行号超出范围应返回错误")
	}
}

func TestEditFileTool_InvalidLineDelete(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	testFile := filepath.Join(workDir, "test.txt")
	os.WriteFile(testFile, []byte("Line 1"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"test.txt","action":"delete","line":0}`))
	if err == nil {
		t.Error("行号小于 1 应返回错误")
	}
}

func TestEditFileTool_MissingParams(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	testFile := filepath.Join(workDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	// replace 缺少 find 参数
	_, err := tool.Execute(json.RawMessage(`{"path":"test.txt","action":"replace"}`))
	if err == nil {
		t.Error("replace 操作缺少 find 参数应返回错误")
	}

	// append 缺少 content 参数
	_, err = tool.Execute(json.RawMessage(`{"path":"test.txt","action":"append"}`))
	if err == nil {
		t.Error("append 操作缺少 content 参数应返回错误")
	}

	// insert 缺少 content 参数
	_, err = tool.Execute(json.RawMessage(`{"path":"test.txt","action":"insert","line":1}`))
	if err == nil {
		t.Error("insert 操作缺少 content 参数应返回错误")
	}

	// 未知操作
	_, err = tool.Execute(json.RawMessage(`{"path":"test.txt","action":"unknown"}`))
	if err == nil {
		t.Error("未知操作应返回错误")
	}
}

func TestEditFileTool_DeleteRange(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	testFile := filepath.Join(workDir, "test.txt")
	os.WriteFile(testFile, []byte("Line 1\nLine 2\nLine 3\nLine 4"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("edit_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"test.txt","action":"delete","line":2,"end_line":3}`))
	if err != nil {
		t.Fatalf("edit_file 执行失败: %v", err)
	}

	content, _ := os.ReadFile(testFile)
	expected := "Line 1\nLine 4"
	if string(content) != expected {
		t.Errorf("文件内容 = %q, 期望 %q", string(content), expected)
	}
}

// ─── list_dir 测试 ───────────────────────────────────────────

func TestListDirTool_Success(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件和目录
	os.MkdirAll(filepath.Join(workDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(workDir, "test.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("list_dir")

	result, err := tool.Execute(json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("list_dir 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestListDirTool_WithPattern(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	os.WriteFile(filepath.Join(workDir, "test.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("list_dir")

	result, err := tool.Execute(json.RawMessage(`{"pattern":"*.go"}`))
	if err != nil {
		t.Fatalf("list_dir 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestListDirTool_NotExists(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("list_dir")

	_, err := tool.Execute(json.RawMessage(`{"path":"nonexistent"}`))
	if err == nil {
		t.Error("列出不存在的目录应返回错误")
	}
}

func TestListDirTool_NotDirectory(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建文件（不是目录）
	os.WriteFile(filepath.Join(workDir, "file.txt"), []byte("test"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("list_dir")

	_, err := tool.Execute(json.RawMessage(`{"path":"file.txt"}`))
	if err == nil {
		t.Error("对文件调用 list_dir 应返回错误")
	}
}

// ─── file_info 测试 ──────────────────────────────────────────

func TestFileInfoTool_Success(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	testFile := filepath.Join(workDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("file_info")

	result, err := tool.Execute(json.RawMessage(`{"path":"test.txt"}`))
	if err != nil {
		t.Fatalf("file_info 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestFileInfoTool_Directory(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("file_info")

	result, err := tool.Execute(json.RawMessage(`{"path":"."}`))
	if err != nil {
		t.Fatalf("file_info 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestFileInfoTool_NotExists(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("file_info")

	_, err := tool.Execute(json.RawMessage(`{"path":"nonexistent.txt"}`))
	if err == nil {
		t.Error("获取不存在文件的信息应返回错误")
	}
}

// ─── delete_file 测试 ────────────────────────────────────────

func TestDeleteFileTool_File(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	testFile := filepath.Join(workDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("delete_file")

	result, err := tool.Execute(json.RawMessage(`{"path":"test.txt"}`))
	if err != nil {
		t.Fatalf("delete_file 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}

	// 验证文件已删除
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("文件应已被删除")
	}
}

func TestDeleteFileTool_Directory(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试目录
	testDir := filepath.Join(workDir, "testdir")
	os.MkdirAll(testDir, 0755)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("delete_file")

	// 尝试删除非空目录（不设置 recursive）
	os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("test"), 0644)
	_, err := tool.Execute(json.RawMessage(`{"path":"testdir"}`))
	if err == nil {
		t.Error("删除非空目录应返回错误（未设置 recursive）")
	}

	// 设置 recursive 删除
	result, err := tool.Execute(json.RawMessage(`{"path":"testdir","recursive":true}`))
	if err != nil {
		t.Fatalf("delete_file 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestDeleteFileTool_EmptyDirectory(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建空目录
	testDir := filepath.Join(workDir, "emptydir")
	os.MkdirAll(testDir, 0755)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("delete_file")

	result, err := tool.Execute(json.RawMessage(`{"path":"emptydir"}`))
	if err != nil {
		t.Fatalf("delete_file 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestDeleteFileTool_NotExists(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("delete_file")

	_, err := tool.Execute(json.RawMessage(`{"path":"nonexistent.txt"}`))
	if err == nil {
		t.Error("删除不存在的文件应返回错误")
	}
}

// ─── search_files 测试 ───────────────────────────────────────

func TestSearchFilesTool_Success(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	os.WriteFile(filepath.Join(workDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(workDir, "test.go"), []byte("package test"), 0644)
	os.WriteFile(filepath.Join(workDir, "readme.md"), []byte("# README"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("search_files")

	result, err := tool.Execute(json.RawMessage(`{"pattern":"*.go"}`))
	if err != nil {
		t.Fatalf("search_files 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestSearchFilesTool_NotExists(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("search_files")

	_, err := tool.Execute(json.RawMessage(`{"pattern":"*.go","path":"nonexistent"}`))
	if err == nil {
		t.Error("搜索不存在的目录应返回错误")
	}
}

func TestSearchFilesTool_NotDirectory(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建文件
	os.WriteFile(filepath.Join(workDir, "file.txt"), []byte("test"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("search_files")

	_, err := tool.Execute(json.RawMessage(`{"pattern":"*.go","path":"file.txt"}`))
	if err == nil {
		t.Error("对文件调用 search_files 应返回错误")
	}
}

// ─── search_content 测试 ─────────────────────────────────────

func TestSearchContentTool_Success(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	testFile := filepath.Join(workDir, "test.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("search_content")

	result, err := tool.Execute(json.RawMessage(`{"query":"func main"}`))
	if err != nil {
		t.Fatalf("search_content 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestSearchContentTool_Regex(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	testFile := filepath.Join(workDir, "test.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("search_content")

	result, err := tool.Execute(json.RawMessage(`{"query":"func \\w+"}`))
	if err != nil {
		t.Fatalf("search_content 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

func TestSearchContentTool_NotExists(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("search_content")

	_, err := tool.Execute(json.RawMessage(`{"query":"test","path":"nonexistent"}`))
	if err == nil {
		t.Error("搜索不存在的目录应返回错误")
	}
}

func TestSearchContentTool_NotDirectory(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建文件
	os.WriteFile(filepath.Join(workDir, "file.txt"), []byte("test"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("search_content")

	_, err := tool.Execute(json.RawMessage(`{"query":"test","path":"file.txt"}`))
	if err == nil {
		t.Error("对文件调用 search_content 应返回错误")
	}
}

func TestSearchContentTool_InvalidRegex(t *testing.T) {
	// 保存原始函数并恢复
	origGetWorkDir := getWorkDir
	defer func() { getWorkDir = origGetWorkDir }()

	workDir := t.TempDir()
	getWorkDir = func() string { return workDir }

	// 创建测试文件
	os.WriteFile(filepath.Join(workDir, "test.txt"), []byte("test content"), 0644)

	reg := NewToolRegistry()
	RegisterFileTools(reg)
	tool, _ := reg.Get("search_content")

	// 无效正则会降级为字面量匹配，不应报错
	result, err := tool.Execute(json.RawMessage(`{"query":"[invalid"}`))
	if err != nil {
		t.Fatalf("search_content 执行失败: %v", err)
	}

	if result == "" {
		t.Error("结果不应为空")
	}
}

// ─── 集成测试 ─────────────────────────────────────────────────

func TestRegisterFileTools(t *testing.T) {
	reg := NewToolRegistry()
	RegisterFileTools(reg)

	expected := []string{
		"read_file", "write_file", "edit_file", "list_dir",
		"file_info", "delete_file", "search_files", "search_content",
	}
	for _, name := range expected {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("文件工具 %q 未注册", name)
		}
	}
}

func TestRegisterBuiltinTools_WithFileTools(t *testing.T) {
	reg := NewToolRegistry()
	RegisterBuiltinTools(reg)

	// 验证原有工具
	originalTools := []string{"calculator", "current_time", "search", "text_transform"}
	for _, name := range originalTools {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("原有工具 %q 未注册", name)
		}
	}

	// 验证文件工具
	fileTools := []string{
		"read_file", "write_file", "edit_file", "list_dir",
		"file_info", "delete_file", "search_files", "search_content",
	}
	for _, name := range fileTools {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("文件工具 %q 未注册", name)
		}
	}
}
