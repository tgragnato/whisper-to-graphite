package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLimitNoTrigger(t *testing.T) {
	t.Parallel()

	rl := &rateLimiter{
		pointsPerSecond: 10,
		currentPoints:   0,
		full:            make(chan bool, 1), // use buffered channel to avoid blocking
		lock:            new(sync.Mutex),
		enabled:         true,
	}

	rl.limit(5)
	if rl.currentPoints != 5 {
		t.Errorf("expected currentPoints to be 5, got %d", rl.currentPoints)
	}
	select {
	case v := <-rl.full:
		t.Errorf("unexpected event from full channel: %v", v)
	default:
	}
}

func TestLimitTrigger(t *testing.T) {
	t.Parallel()

	rl := &rateLimiter{
		pointsPerSecond: 10,
		currentPoints:   0,
		full:            make(chan bool, 1), // buffered channel
		lock:            new(sync.Mutex),
		enabled:         true,
	}

	rl.limit(6)
	if rl.currentPoints != 6 {
		t.Errorf("expected currentPoints to be 6, got %d", rl.currentPoints)
	}

	rl.limit(5)
	if rl.currentPoints != 0 {
		t.Errorf("expected currentPoints to be reset to 0, got %d", rl.currentPoints)
	}

	select {
	case v := <-rl.full:
		if v != true {
			t.Errorf("expected true from full channel, got %v", v)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected event in full channel, but none was received")
	}
}

func TestLimitWhenDisabled(t *testing.T) {
	t.Parallel()

	rl := &rateLimiter{
		pointsPerSecond: 10,
		currentPoints:   0,
		full:            make(chan bool, 1),
		lock:            new(sync.Mutex),
		enabled:         false,
	}

	rl.limit(100)
	if rl.currentPoints != 0 {
		t.Errorf("expected currentPoints to remain 0 when disabled, got %d", rl.currentPoints)
	}

	select {
	case v := <-rl.full:
		t.Errorf("expected no event in full channel when disabled, but got %v", v)
	default:
	}
}

func TestConvertFilename_Valid(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir() + "/" + t.Name()
	filePath := filepath.Join(baseDir, "sub", "dir", "metric.wsp")
	expected := "sub.dir.metric"
	metric, err := convertFilename(filePath, baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metric != expected {
		t.Errorf("expected metric %q, got %q", expected, metric)
	}

	filePath2 := filepath.Join(baseDir, "simple.wsp")
	expected2 := "simple"
	metric2, err := convertFilename(filePath2, baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if metric2 != expected2 {
		t.Errorf("expected metric %q, got %q", expected2, metric2)
	}
}

func TestConvertFilename_Invalid(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir() + "/" + t.Name()
	invalidPath := filepath.Join(string(filepath.Separator), "not", "in", "basedir", "metric.wsp")
	metric, err := convertFilename(invalidPath, baseDir)
	if err == nil {
		t.Fatalf("expected error for file %q not being under base %q, got metric %q", invalidPath, baseDir, metric)
	}
	var expErr = "path for whisper file does not live in BasePath"
	if !strings.Contains(err.Error(), expErr) {
		t.Errorf("expected error containing %q, got %v", expErr, err)
	}
}

func TestConvertFilename_AbsError(t *testing.T) {
	t.Parallel()

	path, err := convertFilename("", "")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

func TestNewRateLimiter_Enabled(t *testing.T) {
	t.Parallel()

	rl := newRateLimiter(100)
	if rl == nil {
		t.Fatal("expected non-nil rateLimiter")
	}
	if !rl.enabled {
		t.Error("expected rateLimiter to be enabled")
	}
	if rl.pointsPerSecond != 100 {
		t.Errorf("expected pointsPerSecond to be 100, got %d", rl.pointsPerSecond)
	}
	if rl.currentPoints != 0 {
		t.Errorf("expected currentPoints to be 0, got %d", rl.currentPoints)
	}
	if rl.lock == nil {
		t.Error("expected non-nil lock")
	}
	if rl.full == nil {
		t.Error("expected non-nil full channel")
	}
}

func TestNewRateLimiter_Disabled(t *testing.T) {
	t.Parallel()

	rl := newRateLimiter(0)
	if rl == nil {
		t.Fatal("expected non-nil rateLimiter")
	}
	if rl.enabled {
		t.Error("expected rateLimiter to be disabled")
	}
	if rl.pointsPerSecond != 0 {
		t.Errorf("expected pointsPerSecond to be 0, got %d", rl.pointsPerSecond)
	}
	if rl.currentPoints != 0 {
		t.Errorf("expected currentPoints to be 0, got %d", rl.currentPoints)
	}
	if rl.lock == nil {
		t.Error("expected non-nil lock")
	}
	if rl.full == nil {
		t.Error("expected non-nil full channel")
	}
}

func TestFindWhisperFiles(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir() + "/" + t.Name()
	testFiles := []string{
		filepath.Join(baseDir, "test1.wsp"),
		filepath.Join(baseDir, "subdir", "test2.wsp"),
		filepath.Join(baseDir, "subdir", "test3.wsp"),
		filepath.Join(baseDir, "test4.txt"), // Not a whisper file
	}

	if err := os.MkdirAll(filepath.Join(baseDir, "subdir"), 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	for _, file := range testFiles {
		dir := filepath.Dir(file)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", file, err)
		}
	}

	ch := make(chan string, 10)
	quit := make(chan int)

	go findWhisperFiles(ch, quit, baseDir)

	var foundFiles []string
	done := false
	for !done {
		select {
		case file := <-ch:
			foundFiles = append(foundFiles, file)
		case <-quit:
			done = true
		case <-time.After(20 * time.Second):
			t.Fatal("timed out waiting for findWhisperFiles to complete")
		}
	}

	if len(foundFiles) != 3 {
		t.Errorf("expected to find 3 whisper files, but found %d: %v", len(foundFiles), foundFiles)
	}

	for _, file := range foundFiles {
		if !strings.HasSuffix(file, ".wsp") {
			t.Errorf("found file without .wsp extension: %s", file)
		}
	}
}

func TestFindWhisperFilesEmptyDirectory(t *testing.T) {
	t.Parallel()

	emptyDir := t.TempDir() + "/" + t.Name()

	ch := make(chan string, 10)
	quit := make(chan int)

	go findWhisperFiles(ch, quit, emptyDir)

	select {
	case <-quit:
	case file := <-ch:
		t.Errorf("unexpected file found in empty directory: %s", file)
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for findWhisperFiles to complete")
	}

	select {
	case file, ok := <-ch:
		if ok {
			t.Errorf("unexpected file found: %s", file)
		}
	default:
	}
}
