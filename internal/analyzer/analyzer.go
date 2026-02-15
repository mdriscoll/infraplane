// Package analyzer reads and extracts key infrastructure-relevant files from application codebases.
// It supports both local filesystem paths and git repository URLs.
package analyzer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// maxFileSize is the maximum size of a single file to read (10KB).
const maxFileSize = 10 * 1024

// maxReadmeLines is the maximum number of lines to read from README.md.
const maxReadmeLines = 200

// CodeContext holds extracted file contents from a codebase for LLM analysis.
type CodeContext struct {
	Files   []FileContent `json:"files"`
	Summary string        `json:"summary"`
}

// FileContent represents a single file's path and content.
type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// infraFiles are exact filenames to look for at the project root.
var infraFiles = []string{
	"Dockerfile",
	"docker-compose.yml",
	"docker-compose.yaml",
	"package.json",
	"requirements.txt",
	"Pipfile",
	"go.mod",
	"Gemfile",
	"pom.xml",
	"build.gradle",
	"serverless.yml",
	"serverless.yaml",
	"Procfile",
	".env.example",
	".env.sample",
	"Makefile",
	"README.md",
}

// infraGlobs are glob patterns to match against the project tree.
var infraGlobs = []string{
	"*.tf",
	"*.tfvars",
	"config/database.yml",
	"knexfile.js",
	"prisma/schema.prisma",
	"drizzle.config.ts",
	"drizzle.config.js",
	"k8s/*.yaml",
	"k8s/*.yml",
	"kubernetes/*.yaml",
	"kubernetes/*.yml",
	"deploy/*.sh",
	"deploy/*.yaml",
	"deploy/*.yml",
	"scripts/*.sh",
	".github/workflows/*.yml",
	".github/workflows/*.yaml",
}

// IsGitURL returns true if the source looks like a git repository URL.
func IsGitURL(source string) bool {
	if strings.HasPrefix(source, "git@") {
		return true
	}
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		return strings.HasSuffix(source, ".git") ||
			strings.Contains(source, "github.com") ||
			strings.Contains(source, "gitlab.com") ||
			strings.Contains(source, "bitbucket.org")
	}
	return false
}

// Analyze reads key infrastructure-relevant files from a local path or git URL.
// If sourcePath is a git URL, it performs a shallow clone to a temp directory,
// analyzes the files, and cleans up. If it's a local path, it reads directly.
func Analyze(sourcePath string) (CodeContext, error) {
	if sourcePath == "" {
		return CodeContext{}, fmt.Errorf("source path is empty")
	}

	if IsGitURL(sourcePath) {
		return analyzeGitRepo(sourcePath)
	}

	return analyzeLocalPath(sourcePath)
}

// analyzeGitRepo clones a git repo to a temp dir and analyzes it.
func analyzeGitRepo(url string) (CodeContext, error) {
	tmpDir, err := os.MkdirTemp("", "infraplane-analyze-*")
	if err != nil {
		return CodeContext{}, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Shallow clone for speed
	cmd := exec.Command("git", "clone", "--depth", "1", url, tmpDir)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return CodeContext{}, fmt.Errorf("git clone %s: %w", url, err)
	}

	ctx, err := analyzeLocalPath(tmpDir)
	if err != nil {
		return ctx, err
	}
	ctx.Summary = fmt.Sprintf("Analyzed git repository: %s", url)
	return ctx, nil
}

// analyzeLocalPath reads key files from a local directory.
func analyzeLocalPath(dir string) (CodeContext, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return CodeContext{}, fmt.Errorf("stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		return CodeContext{}, fmt.Errorf("%s is not a directory", dir)
	}

	var files []FileContent

	// Check exact filenames at root
	for _, name := range infraFiles {
		path := filepath.Join(dir, name)
		content, err := readFileLimited(path, name)
		if err != nil {
			continue // File doesn't exist or can't be read — skip
		}
		files = append(files, FileContent{
			Path:    name,
			Content: content,
		})
	}

	// Check glob patterns
	for _, pattern := range infraGlobs {
		fullPattern := filepath.Join(dir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			relPath, err := filepath.Rel(dir, match)
			if err != nil {
				relPath = filepath.Base(match)
			}

			// Skip if we already have this file
			if hasFile(files, relPath) {
				continue
			}

			content, err := readFileLimited(match, relPath)
			if err != nil {
				continue
			}
			files = append(files, FileContent{
				Path:    relPath,
				Content: content,
			})
		}
	}

	summary := fmt.Sprintf("Analyzed local path: %s — found %d infrastructure-relevant files", dir, len(files))

	return CodeContext{
		Files:   files,
		Summary: summary,
	}, nil
}

// readFileLimited reads a file up to maxFileSize bytes.
// For README.md, it limits to maxReadmeLines lines.
func readFileLimited(path string, name string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("is a directory")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	// Truncate large files
	if len(data) > maxFileSize {
		data = data[:maxFileSize]
	}

	content := string(data)

	// Limit README to first N lines
	if strings.EqualFold(name, "README.md") {
		lines := strings.SplitN(content, "\n", maxReadmeLines+1)
		if len(lines) > maxReadmeLines {
			content = strings.Join(lines[:maxReadmeLines], "\n")
		}
	}

	return content, nil
}

// hasFile checks if a file with the given relative path already exists in the list.
func hasFile(files []FileContent, relPath string) bool {
	for _, f := range files {
		if f.Path == relPath {
			return true
		}
	}
	return false
}
