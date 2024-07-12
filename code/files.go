package common

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func WriteFile(filePath string, contents string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err = file.WriteString(contents); err != nil {
		return err
	}

	file.Sync()

	return nil
}

func ReadFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(data), nil
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

func CopyFile(sourcePath string, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("could not open source file '%s': %w", sourcePath, err)
	}
	defer sourceFile.Close()

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("could not create target file '%s': %w", targetPath, err)
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return fmt.Errorf("could not copy data from '%s' to '%s': %w", sourcePath, targetPath, err)
	}

	return nil
}

func ForSpecificFiles(dir string, isSpecific func(os.FileInfo) bool, doSomething func(string, os.FileInfo) error) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isSpecific(info) {
			if err := doSomething(path, info); err != nil {
				return fmt.Errorf("could not do something with a specific file '%s': %v", path, err)
			}
		}

		return nil
	})
}

func DeleteSpecificFiles(dir string, shouldDelete func(os.FileInfo) bool) error {
	return ForSpecificFiles(dir, shouldDelete, func(path string, info os.FileInfo) error {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("could not delete file: %v", err)
		}

		return nil
	})
}

func CopySpecificFiles(sourceDir string, targetDir string, shouldCopy func(os.FileInfo) bool) error {
	return ForSpecificFiles(sourceDir, shouldCopy, func(path string, info os.FileInfo) error {
		if err := CopyFile(path, filepath.Join(targetDir, info.Name())); err != nil {
			return fmt.Errorf("could not copy file to '%s': %v", targetDir, err)
		}

		return nil
	})
}

func ModifySpecificFiles(dir string, shouldModify func(os.FileInfo) bool, modifyContents func(string) string) error {
	return ForSpecificFiles(dir, shouldModify, func(path string, info os.FileInfo) error {
		contents, err := ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read from file: %v", err)
		}

		if err := WriteFile(path, modifyContents(contents)); err != nil {
			return fmt.Errorf("could not write to file: %v", err)
		}

		return nil
	})
}

func WriteOutput(key string, value string) {
	keyValuePair := fmt.Sprintf("%s=%s", key, value)
	if err := WriteFile(getEnv("GITHUB_OUTPUT"), keyValuePair); err != nil {
		log.Fatalf("could not write '%s' to GITHUB_OUTPUT: %v", keyValuePair, err)
	}
}
