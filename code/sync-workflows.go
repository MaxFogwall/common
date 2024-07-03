package main

import (
    "fmt"
	"github.com/MaxFogwall/common/files"
	"time"
)

func main() {
	files.readFile("repositories-to-sync.json")

	var repositories []string
    err = json.Unmarshal(data, &repositories)
    if err != nil {
        panic(e)
    }

    syncedRepositories := syncWorkflows(repositories)
	syncedRepositoriesTable := "| Repository | Success | T-Start |\n"
	syncedRepositoriesTable += "|-:|:-:|:-|\n"

	for index, syncedRepository := range syncedRepositories {
		repositoryName := strings.Split(syncedRepository.Identifier, "/")[-1]
		repositoryString := fmt.Sprintf("**[`%s`](https://github.com/%s)**", repositoryName, syncedRepository.Identifier)

		successString := "❌"
		if syncedRepository.Success {
			successString = "✅"
		}

		timeString := fmt.Sprintf(,syncedRepository.ElapsedTime)

		syncedRepositoriesTable += fmt.Sprintf("| %s | %s | %s |\n", repositoryString, successString, timeString)
	}

	files.makeSummary(
		"### Synchronization Complete" +
		syncedRepositoriesTable
	)
}

type SyncedRepository struct {
	Identifier string
	Success bool
	ElapsedTime time.Duration
}

func syncWorkflows(repositories []string) []string {
	startTime := time.Now()
	syncedRepositories := []SyncedRepository{}

	for index, repository := range repositories {
		// TODO: Actually add the synchronization itself.

		elapsedTime := time.Since(startTime)
		syncedRepository = SyncedRepository{
			Identifier: repository,
			Success: true,
			ElapsedTime: elapsedTime
		}
	}

    return syncedRepositories
}