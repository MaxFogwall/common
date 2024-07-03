package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	files "github.com/MaxFogwall/common"
)

func main() {
	data := []byte(files.ReadFile("repositories-to-sync.json"))

	var repositories []string
	err := json.Unmarshal(data, &repositories)
	if err != nil {
		panic(err)
	}

	syncedRepositories := syncWorkflows(repositories)
	syncedRepositoriesTable := "| Repository | Success | T-Start |\n"
	syncedRepositoriesTable += "|-:|:-:|:-|\n"

	for _, syncedRepository := range syncedRepositories {
		repositoryName := strings.Split(syncedRepository.Identifier, "/")[1]
		repositoryString := fmt.Sprintf("**[`%s`](https://github.com/%s)**", repositoryName, syncedRepository.Identifier)

		successString := "❌"
		if syncedRepository.Success {
			successString = "✅"
		}

		timeString := fmt.Sprintf("%s", syncedRepository.ElapsedTime)

		syncedRepositoriesTable += fmt.Sprintf("| %s | %s | %s |\n", repositoryString, successString, timeString)
	}

	files.MakeSummary("### Synchronization Complete\n" + syncedRepositoriesTable)
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
