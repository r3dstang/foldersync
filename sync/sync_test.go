package sync

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mockDest is an in-memory Destination for testing.
type mockDest struct {
	objects     map[string]*ObjectMeta
	putCalls    []string
	deleteCalls []string
}

func newMockDest() *mockDest {
	return &mockDest{objects: make(map[string]*ObjectMeta)}
}

func (m *mockDest) Put(_ context.Context, key string, _ io.Reader, size int64, modTime time.Time) error {
	m.putCalls = append(m.putCalls, key)
	m.objects[key] = &ObjectMeta{Size: size, ModTime: modTime.Truncate(time.Second)}
	return nil
}

func (m *mockDest) Stat(_ context.Context, key string) (*ObjectMeta, error) {
	return m.objects[key], nil
}

func (m *mockDest) List(_ context.Context) ([]string, error) {
	keys := make([]string, 0, len(m.objects))
	for k := range m.objects {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockDest) Delete(_ context.Context, key string) error {
	m.deleteCalls = append(m.deleteCalls, key)
	delete(m.objects, key)
	return nil
}

// writeFile creates a file under dir with the given content and returns its os.FileInfo.
func writeFile(t *testing.T, dir, name, content string) os.FileInfo {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func TestSync_uploadsNewFiles(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "a.txt", "hello")
	writeFile(t, src, "b.txt", "world")

	dst := newMockDest()
	if err := Sync(context.Background(), Options{Src: src, Dst: dst}); err != nil {
		t.Fatal(err)
	}

	if len(dst.putCalls) != 2 {
		t.Errorf("expected 2 uploads, got %d: %v", len(dst.putCalls), dst.putCalls)
	}
}

func TestSync_skipsUpToDateFiles(t *testing.T) {
	src := t.TempDir()
	info := writeFile(t, src, "a.txt", "hello")

	dst := newMockDest()
	dst.objects["a.txt"] = &ObjectMeta{
		Size:    info.Size(),
		ModTime: info.ModTime().Truncate(time.Second),
	}

	if err := Sync(context.Background(), Options{Src: src, Dst: dst}); err != nil {
		t.Fatal(err)
	}

	if len(dst.putCalls) != 0 {
		t.Errorf("expected no uploads for up-to-date file, got %v", dst.putCalls)
	}
}

func TestSync_reuploadsWhenMtimeDiffers(t *testing.T) {
	src := t.TempDir()
	info := writeFile(t, src, "a.txt", "hello")

	dst := newMockDest()
	dst.objects["a.txt"] = &ObjectMeta{
		Size:    info.Size(),
		ModTime: info.ModTime().Truncate(time.Second).Add(-time.Hour),
	}

	if err := Sync(context.Background(), Options{Src: src, Dst: dst}); err != nil {
		t.Fatal(err)
	}

	if len(dst.putCalls) != 1 || dst.putCalls[0] != "a.txt" {
		t.Errorf("expected a.txt to be re-uploaded, got %v", dst.putCalls)
	}
}

func TestSync_reuploadsWhenSizeDiffers(t *testing.T) {
	src := t.TempDir()
	info := writeFile(t, src, "a.txt", "hello")

	dst := newMockDest()
	dst.objects["a.txt"] = &ObjectMeta{
		Size:    info.Size() + 1,
		ModTime: info.ModTime().Truncate(time.Second),
	}

	if err := Sync(context.Background(), Options{Src: src, Dst: dst}); err != nil {
		t.Fatal(err)
	}

	if len(dst.putCalls) != 1 || dst.putCalls[0] != "a.txt" {
		t.Errorf("expected a.txt to be re-uploaded, got %v", dst.putCalls)
	}
}

func TestSync_deleteMode(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "keep.txt", "keep")

	dst := newMockDest()
	dst.objects["keep.txt"] = &ObjectMeta{}
	dst.objects["extra.txt"] = &ObjectMeta{}

	if err := Sync(context.Background(), Options{Src: src, Dst: dst, Delete: true}); err != nil {
		t.Fatal(err)
	}

	if len(dst.deleteCalls) != 1 || dst.deleteCalls[0] != "extra.txt" {
		t.Errorf("expected extra.txt to be deleted, got %v", dst.deleteCalls)
	}
	if _, ok := dst.objects["keep.txt"]; !ok {
		t.Error("keep.txt should not have been deleted")
	}
}

func TestSync_dryRunSkipsAllWrites(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "new.txt", "new")

	dst := newMockDest()
	dst.objects["stale.txt"] = &ObjectMeta{}

	if err := Sync(context.Background(), Options{Src: src, Dst: dst, DryRun: true, Delete: true}); err != nil {
		t.Fatal(err)
	}

	if len(dst.putCalls) != 0 {
		t.Errorf("dry-run: expected no uploads, got %v", dst.putCalls)
	}
	if len(dst.deleteCalls) != 0 {
		t.Errorf("dry-run: expected no deletes, got %v", dst.deleteCalls)
	}
}

func TestSync_nestedDirectories(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "a/x.txt", "x")
	writeFile(t, src, "a/b/y.txt", "y")

	dst := newMockDest()
	if err := Sync(context.Background(), Options{Src: src, Dst: dst}); err != nil {
		t.Fatal(err)
	}

	if len(dst.putCalls) != 2 {
		t.Errorf("expected 2 uploads, got %d: %v", len(dst.putCalls), dst.putCalls)
	}
	for _, key := range dst.putCalls {
		for _, c := range key {
			if c == '\\' {
				t.Errorf("key %q contains backslash; S3 keys must use forward slashes", key)
			}
		}
	}
}

func TestSync_invalidSrc(t *testing.T) {
	dst := newMockDest()
	err := Sync(context.Background(), Options{Src: "/nonexistent/path", Dst: dst})
	if err == nil {
		t.Error("expected error for nonexistent source, got nil")
	}
}

func TestSync_srcMustBeDirectory(t *testing.T) {
	f, err := os.CreateTemp("", "foldersync-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })

	dst := newMockDest()
	err = Sync(context.Background(), Options{Src: f.Name(), Dst: dst})
	if err == nil {
		t.Error("expected error when src is a file, got nil")
	}
}
