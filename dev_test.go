package main

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/donutnomad/gogen/plugin"
	"github.com/fsnotify/fsnotify"
)

func TestBuildGenerateToolArgs(t *testing.T) {
	tests := []struct {
		name   string
		opts   *DevOptions
		pkgDir string
		want   []string
	}{
		{
			name: "verbose default output",
			opts: &DevOptions{
				Verbose: true,
				Output:  "generate.go",
				Async:   true,
			},
			pkgDir: "/repo/models",
			want:   []string{"tool", "gogen", "-v", "-output", "generate.go", "-async=true", "gen", "/repo/models"},
		},
		{
			name: "no output async false",
			opts: &DevOptions{
				NoOutput: true,
				Async:    false,
			},
			pkgDir: "/repo/models",
			want:   []string{"tool", "gogen", "-no-output", "-async=false", "gen", "/repo/models"},
		},
		{
			name: "prebuilt tool args",
			opts: &DevOptions{
				Verbose:  false,
				Output:   "ignored.go",
				Async:    true,
				ToolArgs: []string{"-v", "-output", "$FILE_gen", "-async=false"},
			},
			pkgDir: "/repo/models",
			want:   []string{"tool", "gogen", "-v", "-output", "$FILE_gen", "-async=false", "gen", "/repo/models"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildGenerateToolArgs(tt.opts, tt.pkgDir)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildGenerateToolArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBuildRestartToolArgs(t *testing.T) {
	opts := &DevOptions{
		OriginalArgs: []string{"-v", "dev", "./models/..."},
	}

	got := buildRestartToolArgs(opts)
	want := []string{"tool", "gogen", "-v", "dev", "./models/..."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRestartToolArgs() = %#v, want %#v", got, want)
	}
}

func TestHandleEventRestartsOnModuleFileChange(t *testing.T) {
	restarted := make(chan []string, 1)
	runner := &devRunner{
		opts: &DevOptions{
			OriginalArgs: []string{"dev", "./..."},
			Debounce:     time.Millisecond,
			RestartCommand: func(args []string) error {
				restarted <- append([]string(nil), args...)
				return nil
			},
		},
		ctx:         context.Background(),
		pendingDirs: make(map[string]*time.Timer),
	}

	runner.handleEvent(fsnotify.Event{
		Name: filepath.Join(t.TempDir(), "go.mod"),
		Op:   fsnotify.Rename,
	})

	got := receiveArgs(t, restarted)
	want := []string{"tool", "gogen", "dev", "./..."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restart args = %#v, want %#v", got, want)
	}
}

func TestHandleEventDebouncesModuleFileRestart(t *testing.T) {
	restarted := make(chan []string, 2)
	runner := &devRunner{
		opts: &DevOptions{
			OriginalArgs: []string{"dev", "./..."},
			Debounce:     10 * time.Millisecond,
			RestartCommand: func(args []string) error {
				restarted <- append([]string(nil), args...)
				return nil
			},
		},
		ctx:         context.Background(),
		pendingDirs: make(map[string]*time.Timer),
	}

	moduleFile := filepath.Join(t.TempDir(), "go.sum")
	runner.handleEvent(fsnotify.Event{Name: moduleFile, Op: fsnotify.Write})
	runner.handleEvent(fsnotify.Event{Name: moduleFile, Op: fsnotify.Write})

	receiveArgs(t, restarted)

	time.Sleep(25 * time.Millisecond)
	select {
	case got := <-restarted:
		t.Fatalf("unexpected second restart args: %#v", got)
	default:
	}
}

func TestHandleEventRunsGenerateCommandForAnnotatedGoFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "model.go")
	content := []byte("package model\n\n// @Gsql\n\ntype User struct{}\n")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	generated := make(chan []string, 1)
	runner := &devRunner{
		opts: &DevOptions{
			Output:   "generate.go",
			Async:    true,
			Debounce: time.Millisecond,
			GenerateCommand: func(ctx context.Context, args []string) error {
				generated <- append([]string(nil), args...)
				return nil
			},
		},
		scanner:     plugin.NewScanner(plugin.WithAnnotationFilter("Gsql")),
		ctx:         context.Background(),
		pendingDirs: make(map[string]*time.Timer),
	}

	runner.handleEvent(fsnotify.Event{
		Name: filePath,
		Op:   fsnotify.Write,
	})

	got := receiveArgs(t, generated)
	want := []string{"tool", "gogen", "-output", "generate.go", "-async=true", "gen", tmpDir}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("generate args = %#v, want %#v", got, want)
	}
}

func receiveArgs(t *testing.T, ch <-chan []string) []string {
	t.Helper()

	select {
	case args := <-ch:
		return args
	case <-time.After(time.Second):
		t.Fatal("command was not called")
	}
	return nil
}
