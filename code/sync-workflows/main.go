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
	syncedReposTable := "| Repository | Success | T-Start |\r\n"
	syncedReposTable += "|:-|:-:|-:|\r\n"
	shouldWorkflowSucceed := true

	for _, syncedRepo := range syncedRepos {
		repositoryOwnerNameSlice := strings.Split(syncedRepo.Identifier, "/")
		if len(repositoryOwnerNameSlice) < 2 {
			panic(errors.New("one of the repositories was not in the correct format (i.e. \"owner/name\")"))
		}
		repoName := strings.Split(syncedRepo.Identifier, "/")[1]
		repoString := fmt.Sprintf("**[`%s`](https://github.com/%s)**", repoName, syncedRepo.Identifier)

		successString := "✅"
		if syncedRepo.Error != nil {
			successString = fmt.Sprintf("❌ %v", err)
			shouldWorkflowSucceed = false
		}

		timeString := syncedRepo.ElapsedTime.String()

		syncedReposTable += fmt.Sprintf("| %s | %s | %s |\r\n", repoString, successString, timeString)
	}

	common.MakeSummary("### Synchronization Complete\r\n" + syncedReposTable)

	if !shouldWorkflowSucceed {
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
