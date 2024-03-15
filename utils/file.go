package utils

import (
	"fmt"
	"os"
)

func CreateDir(path string) {
	_exist, _err := HasDir(path)
	if _err != nil {
		fmt.Printf("获取文件夹异常： %v\n", _err)
		return
	}
	if _exist {
		return
	} else {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			fmt.Printf("创建目录异常： %v\n", err)
		} else {
			fmt.Println("创建目录成功! [" + path + "]")
		}
	}
}

func HasDir(path string) (bool, error) {
	_, _err := os.Stat(path)
	if _err == nil {
		return true, nil
	}
	if os.IsNotExist(_err) {
		return false, nil
	}
	return false, _err
}

func ReadFileLineByLine() {
	var (
		filePath string
	)
	_, err := os.Stat(localFilePath)
	if err != nil {
		filePath = srvFilePath
	} else {
		filePath = localFilePath
	}
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("无法打开文件{WeakPassword.txt}", err)
		return
	}
	defer file.Close()

	weakPassWorkList := make(map[string]struct{})

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		weakPassWorkList[scanner.Text()] = struct{}{}
	}
}
