package file

import (
	"os"
	"path/filepath"
)

// WriteBytesToFile saves content type '[]byte' to file by given path.
// It returns error when fail to finish operation.
func WriteBytesToFile(filePath string, b []byte) (int, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		return 0, err
	}
	fw, err := os.Create(filePath) // #nosec G304 -- 通用文件工具，路径由调用方决定
	if err != nil {
		return 0, err
	}
	defer fw.Close()
	return fw.Write(b)
}

// WriteStringToFile saves content type 'string' to file by given path.
// It returns error when fail to finish operation.
func WriteStringToFile(filePath string, s string) (int, error) {
	return WriteBytesToFile(filePath, []byte(s))
}
