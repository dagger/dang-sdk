package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRendersCliCaseNameAsDangType(t *testing.T) {
	dir := t.TempDir()
	templateDir := filepath.Join(dir, "template")
	outDir := filepath.Join(dir, "out")

	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "main.dang.tmpl"), []byte("type {{ .ModuleType }} {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"my-module", templateDir, outDir}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(outDir, "main.dang"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "type MyModule {") {
		t.Fatalf("rendered template mismatch:\n%s", got)
	}
}
