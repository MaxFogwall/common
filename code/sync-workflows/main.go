package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	files "github.com/MaxFogwall/common/code"
)

func main() {
	data := []byte(files.ReadFile("repos.json"))

	var repositories []string
	err := json.Unmarshal(data, &repositories)
	if err != nil {
		panic(err)
	}

	syncedRepositories := syncWorkflows(repositories)
	syncedRepositoriesTable := "| Repository | Success | T-Start |\r\n"
	syncedRepositoriesTable += "|-:|:-:|:-|\r\n"

	for _, syncedRepository := range syncedRepositories {
		repositoryOwnerNameSlice := strings.Split(syncedRepository.Identifier, "/")
		if len(repositoryOwnerNameSlice) < 2 {
			panic(errors.New("one of the repositories was not in the correct format (i.e. \"owner/name\")"))
		}
		repositoryName := strings.Split(syncedRepository.Identifier, "/")[1]
		repositoryString := fmt.Sprintf("**[`%s`](https://github.com/%s)**", repositoryName, syncedRepository.Identifier)

		successString := "❌"
		if syncedRepository.Success {
			successString = "✅"
		}

		timeString := syncedRepository.ElapsedTime.String()

		syncedRepositoriesTable += fmt.Sprintf("| %s | %s | %s |\r\n", repositoryString, successString, timeString)
	}

	files.MakeSummary("### Synchronization Complete\r\n" + syncedRepositoriesTable)
}

type SyncedRepository struct {
	Identifier  string
	Success     bool
	ElapsedTime time.Duration
}

func syncWorkflows(repositories []string) []SyncedRepository {
	startTime := time.Now()
	syncedRepositories := []SyncedRepository{}

	for _, repository := range repositories {
		// TODO: Actually add the synchronization itself.

		elapsedTime := time.Since(startTime)
		syncedRepository := SyncedRepository{
			Identifier:  repository,
			Success:     true,
			ElapsedTime: elapsedTime,
		}

		syncedRepositories = append(syncedRepositories, syncedRepository)
	}

	return syncedRepositories
}
