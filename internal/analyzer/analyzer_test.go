package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"https://github.com/org/repo.git", true},
		{"https://github.com/org/repo", true},
		{"git@github.com:org/repo.git", true},
		{"https://gitlab.com/org/repo", true},
		{"https://bitbucket.org/org/repo", true},
		{"/home/user/projects/my-app", false},
		{"./relative/path", false},
		{"", false},
		{"https://example.com/not-a-repo", false},
		{"https://example.com/something.git", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsGitURL(tt.input)
			if got != tt.want {
				t.Errorf("IsGitURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAnalyze_EmptySource(t *testing.T) {
	_, err := Analyze("")
	if err == nil {
		t.Fatal("expected error for empty source path")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected 'empty' in error, got: %s", err.Error())
	}
}

func TestAnalyze_NonexistentPath(t *testing.T) {
	_, err := Analyze("/nonexistent/path/12345")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestAnalyze_NotADirectory(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "infraplane-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	_, err = Analyze(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error for file (not directory)")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' in error, got: %s", err.Error())
	}
}

func TestAnalyze_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "infraplane-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	ctx, err := Analyze(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(ctx.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(ctx.Files))
	}
	if !strings.Contains(ctx.Summary, "0 infrastructure-relevant files") {
		t.Errorf("expected summary to mention 0 files, got: %s", ctx.Summary)
	}
}

func TestAnalyze_DetectsRootFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "infraplane-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some infrastructure files
	files := map[string]string{
		"Dockerfile":       "FROM golang:1.22\nWORKDIR /app\nCOPY . .\nRUN go build -o main .\nCMD [\"./main\"]",
		"go.mod":           "module example.com/myapp\n\ngo 1.22\n\nrequire github.com/lib/pq v1.10.9",
		"docker-compose.yml": "services:\n  db:\n    image: postgres:16\n    environment:\n      POSTGRES_DB: myapp\n",
		".env.example":     "DATABASE_URL=postgres://localhost:5432/myapp\nREDIS_URL=redis://localhost:6379",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	ctx, err := Analyze(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(ctx.Files) != len(files) {
		t.Errorf("expected %d files, got %d", len(files), len(ctx.Files))
	}

	// Verify each file was found
	for name := range files {
		found := false
		for _, f := range ctx.Files {
			if f.Path == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("file %q not found in results", name)
		}
	}
}

func TestAnalyze_DetectsGlobFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "infraplane-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Terraform files
	os.WriteFile(filepath.Join(tmpDir, "main.tf"), []byte(`provider "aws" { region = "us-east-1" }`), 0644)
	os.WriteFile(filepath.Join(tmpDir, "variables.tf"), []byte(`variable "region" { default = "us-east-1" }`), 0644)

	// Create k8s directory with yaml
	k8sDir := filepath.Join(tmpDir, "k8s")
	os.MkdirAll(k8sDir, 0755)
	os.WriteFile(filepath.Join(k8sDir, "deployment.yaml"), []byte("apiVersion: apps/v1\nkind: Deployment"), 0644)

	// Create prisma schema
	prismaDir := filepath.Join(tmpDir, "prisma")
	os.MkdirAll(prismaDir, 0755)
	os.WriteFile(filepath.Join(prismaDir, "schema.prisma"), []byte("datasource db {\n  provider = \"postgresql\"\n}"), 0644)

	ctx, err := Analyze(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(ctx.Files) != 4 {
		t.Errorf("expected 4 files, got %d", len(ctx.Files))
		for _, f := range ctx.Files {
			t.Logf("  found: %s", f.Path)
		}
	}

	// Check specific paths
	expectedPaths := []string{"main.tf", "variables.tf", "k8s/deployment.yaml", "prisma/schema.prisma"}
	for _, expected := range expectedPaths {
		found := false
		for _, f := range ctx.Files {
			if f.Path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find %q", expected)
		}
	}
}

func TestAnalyze_FileSizeLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "infraplane-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file larger than maxFileSize
	bigContent := strings.Repeat("x", maxFileSize+1000)
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(bigContent), 0644)

	ctx, err := Analyze(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(ctx.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(ctx.Files))
	}

	if len(ctx.Files[0].Content) > maxFileSize {
		t.Errorf("file content should be truncated to %d bytes, got %d", maxFileSize, len(ctx.Files[0].Content))
	}
}

func TestAnalyze_ReadmeLinesLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "infraplane-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a README with more than maxReadmeLines lines
	lines := make([]string, maxReadmeLines+50)
	for i := range lines {
		lines[i] = "This is a line of README content."
	}
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte(strings.Join(lines, "\n")), 0644)

	ctx, err := Analyze(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(ctx.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(ctx.Files))
	}

	readmeLines := strings.Split(ctx.Files[0].Content, "\n")
	if len(readmeLines) > maxReadmeLines {
		t.Errorf("README should be limited to %d lines, got %d", maxReadmeLines, len(readmeLines))
	}
}

func TestAnalyze_IgnoresNonInfraFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "infraplane-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create non-infrastructure files that should be ignored
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "utils.js"), []byte("export default {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "styles.css"), []byte("body { color: red; }"), 0644)

	// Create one infra file
	os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM node:20"), 0644)

	ctx, err := Analyze(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(ctx.Files) != 1 {
		t.Errorf("expected 1 file (only Dockerfile), got %d", len(ctx.Files))
		for _, f := range ctx.Files {
			t.Logf("  found: %s", f.Path)
		}
	}

	if ctx.Files[0].Path != "Dockerfile" {
		t.Errorf("expected Dockerfile, got %s", ctx.Files[0].Path)
	}
}

func TestAnalyze_NoDuplicates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "infraplane-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Dockerfile is in both infraFiles and could be matched by a glob
	os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM alpine"), 0644)

	ctx, err := Analyze(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	count := 0
	for _, f := range ctx.Files {
		if f.Path == "Dockerfile" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected Dockerfile to appear once, got %d times", count)
	}
}
