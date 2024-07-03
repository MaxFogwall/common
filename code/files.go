package files

import (
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func WriteFile(filePath string, contents string) bool {
	f, err := os.Create(filePath)
	check(err)

	defer f.Close()

	bytesWritten, err := f.WriteString(contents)
	check(err)

	f.Sync()

	return bytesWritten > 0
}

func ReadFile(filePath string) string {
	data, err := os.ReadFile(filePath)
	check(err)

	return string(data)
}

func MakeSummary(contents string) {
	WriteFile("summary.md", contents)
}
