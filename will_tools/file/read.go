package file

import "io/ioutil"

// ReadFileToBytes reads data type '[]byte' from file by given path.
// It returns error when fail to finish operation.
func ReadFileToBytes(filePath string) ([]byte, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return []byte(""), err
	}
	return b, nil
}

// ReadFileToString reads data type 'string' from file by given path.
// It returns error when fail to finish operation.
func ReadFileToString(filePath string) (string, error) {
	b, err := ReadFileToBytes(filePath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
