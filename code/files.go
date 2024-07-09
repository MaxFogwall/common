package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
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

func CreateDirectory(dir string) error {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("could not create directory '%s': %w", dir, err)
	}

	return nil
}

func DeleteDirectory(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("could not delete directory '%s': %w", dir, err)
	}

	return nil
}

func DeleteSpecificFiles(dir string, shouldDelete func(os.FileInfo) bool) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && shouldDelete(info) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file '%s': %w", path, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not delete files from '%s': %v", dir, err)
	}

	return nil
}

func CopySpecificFiles(sourceDir string, targetDir string, shouldCopy func(os.FileInfo) bool) error {
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && shouldCopy(info) {
			sourceFile, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("could not open source file '%s': %w", path, err)
			}
			defer sourceFile.Close()

			sourceFilePath := filepath.Join(sourceDir, info.Name())
			targetFilePath := filepath.Join(targetDir, info.Name())

			targetFile, err := os.Create(targetFilePath)
			if err != nil {
				return fmt.Errorf("could not create target file '%s': %w", targetFilePath, err)
			}
			defer targetFile.Close()

			_, err = io.Copy(targetFile, sourceFile)
			if err != nil {
				return fmt.Errorf("could not copy data from '%s' to '%s': %w", sourceFilePath, targetFilePath, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not copy files from '%s' to '%s': %v", sourceDir, targetDir, err)
	}

	return nil
}

func WriteOutput(output string) {
	err := ExecInDir(getEnv("$GITHUB_WORKSPACE"), func() error {
		WriteFile("go-output.txt", output)

		return nil
	})
	if err != nil {
		log.Fatalf("could not write `go-output.txt`: %v", err)
	}
}
