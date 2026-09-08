package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNodeProvisionLayout(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("USERPROFILE", tmp)
	if err := unzipRoot("fake-node.zip", filepath.Join(toolsDir(), "node")); err != nil {
		t.Fatalf("unzip: %v", err)
	}
	exe := portableNodeExe()
	if exe == "" || !strings.HasSuffix(exe, "node.exe") {
		t.Fatalf("node.exe not found, got %q", exe)
	}
	if !nodeInstallComplete() {
		t.Fatal("complete install not detected")
	}
}

func TestNodeProvisionIncomplete(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("USERPROFILE", tmp)
	dir := filepath.Join(toolsDir(), "node")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "node.exe"), []byte("x"), 0755); err != nil {
		t.Fatal(err)
	}
	if nodeInstallComplete() {
		t.Fatal("half install must NOT count as complete")
	}
}
