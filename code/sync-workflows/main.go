package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	common "github.com/MaxFogwall/common/code"
)

func main() {
	sourceRepo := "MaxFogwall/common"
	data := []byte(common.ReadFile("repos.json"))

	var targetRepos []string
	err := json.Unmarshal(data, &targetRepos)
	if err != nil {
		panic(err)
	}

	syncedRepos := syncWorkflows(sourceRepo, targetRepos)
	var syncedReposTable []string
	syncedReposTable = append(syncedReposTable, "| Repository | Success | T-Start |")
	syncedReposTable = append(syncedReposTable, "|:-|:-:|-:|")

	var syncedReposErrors []string
	anyRepoHadErrors := false

	for _, syncedRepo := range syncedRepos {
		_, name := common.RepoOwnerName(syncedRepo.Identifier)
		repoString := fmt.Sprintf("**[`%s`](https://github.com/%s)**", name, syncedRepo.Identifier)

		successString := "✅"
		if syncedRepo.Error != nil {
			successString = "❌"
			syncedReposErrors = append(syncedReposErrors, fmt.Sprintf("- ❌ %s (%s)", repoString, syncedRepo.Error))
			anyRepoHadErrors = true
		}

		timeString := syncedRepo.ElapsedTime.Round(time.Second).String()

		syncedReposTable = append(syncedReposTable, fmt.Sprintf("| %s | %s | %s |", repoString, successString, timeString))
	}

	var summaryLines []string
	summaryLines = append(summaryLines, "### Overview")
	summaryLines = append(summaryLines, strings.Join(syncedReposTable, "\r\n"))
	if anyRepoHadErrors {
		summaryLines = append(summaryLines, "### Errors")
		summaryLines = append(summaryLines, strings.Join(syncedReposErrors, "\r\n"))
	}
	common.MakeSummary(strings.Join(summaryLines, "\r\n"))

	if anyRepoHadErrors {
		panic(errors.New("one or more repositories were not synced successfully"))
	}
}

type SyncedRepository struct {
	Identifier  string
	Error       error
	ElapsedTime time.Duration
}

func syncWorkflows(sourceRepo string, targetRepos []string) []SyncedRepository {
	startTime := time.Now()
	syncedRepos := []SyncedRepository{}

	sourceRepoDir := "SourceRepo"
	common.CloneRepository(sourceRepo, sourceRepoDir)

	for _, targetRepo := range targetRepos {
		err := common.SyncRepository(targetRepo, sourceRepoDir)
		if err != nil {
			log.Printf("Failed to sync to '%s': %v\n", targetRepo, err)
		}

		elapsedTime := time.Since(startTime)
		syncedRepository := SyncedRepository{
			Identifier:  targetRepo,
			Error:       err,
			ElapsedTime: elapsedTime,
		}

		syncedRepos = append(syncedRepos, syncedRepository)
	}

	return syncedRepos
}
