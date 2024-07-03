package files

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func writeFile(filePath string, contents string) bool {
	f, err := os.Create(filePath)
    check(err)

	defer f.Close()

	bytesWritten, err := f.WriteString(contents)
    check(err)

	f.Sync()

	return bytesWritten > 0
}

func readFile(filePath string) string {
	data, err := os.ReadFile(filePath)
    check(err)

    return string(data)
}

func makeSummary(contents string) {
	writeFile("summary.md", contents)
}