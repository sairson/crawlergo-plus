package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func ReadFile(filePath string) []string {
	var filePaths []string
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		defer f.Close()
		rd := bufio.NewReader(f)
		for {
			line, err := rd.ReadString('\n')
			if err != nil || io.EOF == err {
				break
			}
			filePaths = append(filePaths, line)
		}
	}
	return filePaths
}
