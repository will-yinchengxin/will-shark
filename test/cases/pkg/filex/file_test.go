package filex

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFileHelpers(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a", "b", "x.txt")
	if n, err := WriteStringToFile(p, "hello"); err != nil || n != 5 {
		t.Fatalf("write n=%d err=%v", n, err)
	}
	if s, err := ReadFileToString(p); err != nil || s != "hello" {
		t.Fatalf("read %q %v", s, err)
	}
	if !IsFile(p) || IsFile(dir) || !IsExist(p) || IsExist(filepath.Join(dir, "nope")) {
		t.Fatal("exist checks")
	}
	if sz, err := FileSize(p); err != nil || sz != 5 {
		t.Fatal("size")
	}
	if _, err := FileMTime(p); err != nil {
		t.Fatal(err)
	}
	if Basename(p) != "x.txt" || Ext(p) != ".txt" || Dir(p) != filepath.Join(dir, "a", "b") {
		t.Fatal("path helpers")
	}
	if err := InsureDir(filepath.Join(dir, "c", "d")); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(filepath.Join(dir, "c")); info.Mode().Perm()&0o007 != 0 {
		t.Fatalf("dir must not be world accessible: %v", info.Mode())
	}
	if dirs, _ := DirsUnder(dir); !slices.Equal(dirs, []string{"a", "c"}) {
		t.Fatalf("dirs = %v", dirs)
	}
	if files, _ := FilesUnder(filepath.Join(dir, "a", "b")); !slices.Equal(files, []string{"x.txt"}) {
		t.Fatalf("files = %v", files)
	}
	if full, err := SearchFile("x.txt", dir, filepath.Join(dir, "a", "b")); err != nil || full != p {
		t.Fatalf("search %s %v", full, err)
	}
	if _, err := SearchFile("nope", dir); err == nil {
		t.Fatal("search miss")
	}
	q := filepath.Join(dir, "y.txt")
	if err := Rename(p, q); err != nil || !IsFile(q) {
		t.Fatal("rename")
	}
	if err := Unlink(q); err != nil || IsExist(q) {
		t.Fatal("unlink")
	}
	if abs, _ := RealPath("rel"); !filepath.IsAbs(abs) {
		t.Fatal("realpath")
	}
	if ok, err := HasDir(dir); !ok || err != nil {
		t.Fatal("hasdir")
	}
	if SelfDir() == "" {
		t.Fatal("selfdir")
	}
}

func TestCreateDirAndHasDir(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "new")
	if err := CreateDir(d); err != nil {
		t.Fatal(err)
	}
	if err := CreateDir(d); err != nil {
		t.Fatalf("existing dir must not be an error: %v", err)
	}
	f := filepath.Join(dir, "file")
	_, _ = WriteStringToFile(f, "x")
	if ok, err := HasDir(f); ok || err != nil {
		t.Fatal("HasDir must be false for a regular file")
	}
	if err := CreateDir(f); err == nil {
		t.Fatal("CreateDir over a regular file must fail")
	}
	if err := InsureDir(d); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFileToString(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing file")
	}
}

// os.Stat 返回非"不存在"的错误（无权限）时 HasDir 应返回该错误
func TestHasDirPermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	locked := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if ok, err := HasDir(filepath.Join(locked, "child")); ok || err == nil {
		t.Fatalf("ok=%v err=%v, want permission error", ok, err)
	}
}
