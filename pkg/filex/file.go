// Package filex 文件与目录操作工具。
package filex

import (
	"errors"
	"os"
	"path/filepath"
)

// HasDir 判断 path 是否为已存在的目录（普通文件返回 false）
func HasDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CreateDir 创建单层目录，已存在时返回 nil；同名普通文件已存在时返回错误。
// 不向 stdout 打印（stdout 只输出结构化日志）
func CreateDir(path string) error {
	err := os.Mkdir(path, 0o750)
	if err == nil {
		return nil
	}
	if os.IsExist(err) {
		if ok, _ := HasDir(path); ok {
			return nil
		}
	}
	return err
}

func SelfPath() string {
	path, _ := filepath.Abs(os.Args[0])
	return path
}

// SelfDir gets compiled executable file directory
func SelfDir() string {
	return filepath.Dir(SelfPath())
}

// Basename 返回路径的最后一个元素
func Basename(file string) string {
	return filepath.Base(file)
}

// Dir 返回路径的目录部分
func Dir(file string) string {
	return filepath.Dir(file)
}

func InsureDir(path string) error {
	if IsExist(path) {
		return nil
	}
	return os.MkdirAll(path, 0o750)
}

func Ext(file string) string {
	return filepath.Ext(file)
}

// Rename 重命名文件
func Rename(file string, to string) error {
	return os.Rename(file, to)
}

// Unlink 删除文件
func Unlink(file string) error {
	return os.Remove(file)
}

// IsFile checks whether the path is a file,
// it returns false when it's a directory or does not exist.
func IsFile(filePath string) bool {
	f, e := os.Stat(filePath)
	if e != nil {
		return false
	}
	return !f.IsDir()
}

// IsExist checks whether a file or directory exists.
// It returns false when the file or directory does not exist.
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// SearchFile 依次在 paths 中查找 filename，返回第一个存在的完整路径
func SearchFile(filename string, paths ...string) (fullPath string, err error) {
	for _, path := range paths {
		if fullPath = filepath.Join(path, filename); IsExist(fullPath) {
			return
		}
	}
	err = errors.New(fullPath + " not found in paths")
	return
}

// RealPath 相对路径基于当前工作目录转为绝对路径
func RealPath(file string) (string, error) {
	if filepath.IsAbs(file) {
		return file, nil
	}
	wd, err := os.Getwd()
	return filepath.Join(wd, file), err
}

// FileMTime 返回文件修改时间（秒级时间戳）
func FileMTime(file string) (int64, error) {
	f, e := os.Stat(file)
	if e != nil {
		return 0, e
	}
	return f.ModTime().Unix(), nil
}

// FileSize 返回文件字节数
func FileSize(file string) (int64, error) {
	f, e := os.Stat(file)
	if e != nil {
		return 0, e
	}
	return f.Size(), nil
}

// DirsUnder 列出 dirPath 下的子目录名
func DirsUnder(dirPath string) ([]string, error) {
	if !IsExist(dirPath) {
		return []string{}, nil
	}

	fs, err := os.ReadDir(dirPath)
	if err != nil {
		return []string{}, err
	}

	sz := len(fs)
	if sz == 0 {
		return []string{}, nil
	}

	ret := []string{}
	for i := 0; i < sz; i++ {
		if fs[i].IsDir() {
			name := fs[i].Name()
			if name != "." && name != ".." {
				ret = append(ret, name)
			}
		}
	}

	return ret, nil

}

// FilesUnder 列出 dirPath 下的文件名（不含子目录）
func FilesUnder(dirPath string) ([]string, error) {
	if !IsExist(dirPath) {
		return []string{}, nil
	}

	fs, err := os.ReadDir(dirPath)
	if err != nil {
		return []string{}, err
	}

	sz := len(fs)
	if sz == 0 {
		return []string{}, nil
	}

	ret := []string{}
	for i := 0; i < sz; i++ {
		if !fs[i].IsDir() {
			ret = append(ret, fs[i].Name())
		}
	}

	return ret, nil
}
