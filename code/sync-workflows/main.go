package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	common "github.com/workflow-sync-poc/common/code"
)

type SyncedRepository struct {
	Identifier  string
	Error       error
	ElapsedTime time.Duration
}

func getTargetRepos() []string {
	data := []byte(common.ReadFile("repos.json"))

	var repos []string
	err := json.Unmarshal(data, &repos)
	if err != nil {
		panic(err)
	}

	return repos
}

func formatRepo(syncedRepo SyncedRepository) string {
	_, name := common.RepoOwnerName(syncedRepo.Identifier)
	return fmt.Sprintf("**[`%s`](https://github.com/%s)**", name, syncedRepo.Identifier)
}

func formatSuccess(syncedRepo SyncedRepository) string {
	if syncedRepo.Error != nil {
		return "❌"
	}

	return "✅"
}

func formatTime(syncedRepo SyncedRepository) string {
	return syncedRepo.ElapsedTime.Round(time.Second).String()
}

func MakeSyncedReposSummary(syncedRepos []SyncedRepository) {
	var syncedReposTable []string
	var syncedReposErrors []string

	syncedReposTable = append(syncedReposTable, "| Repository | Success | T-Start |")
	syncedReposTable = append(syncedReposTable, "|:-|:-:|-:|")

	for _, syncedRepo := range syncedRepos {
		if syncedRepo.Error != nil {
			syncedReposErrors = append(syncedReposErrors, fmt.Sprintf("- ❌ %s (%s)", formatRepo(syncedRepo), syncedRepo.Error))
		}

		syncedReposTable = append(syncedReposTable, fmt.Sprintf("| %s | %s | %s |", formatRepo(syncedRepo), formatSuccess(syncedRepo), formatTime(syncedRepo)))
	}

	var summaryLines []string
	summaryLines = append(summaryLines, "### Overview")
	summaryLines = append(summaryLines, strings.Join(syncedReposTable, "\r\n"))
	if len(syncedReposErrors) > 0 {
		summaryLines = append(summaryLines, "### Errors")
		summaryLines = append(summaryLines, strings.Join(syncedReposErrors, "\r\n"))
	}

	common.MakeSummary(strings.Join(summaryLines, "\r\n"))
}

func AnySyncedRepoHasError(syncedRepos []SyncedRepository) bool {
	for _, syncedRepo := range syncedRepos {
		if syncedRepo.Error != nil {
			return true
		}
	}

	return false
}

func syncWorkflows(repos []string) []SyncedRepository {
	startTime := time.Now()
	syncedRepos := []SyncedRepository{}

	for _, repo := range repos {
		err := common.SyncRepository(repo)
		if err != nil {
			log.Printf("Failed to sync to '%s': %v\n", repo, err)
		}

		syncedRepository := SyncedRepository{
			Identifier:  repo,
			Error:       err,
			ElapsedTime: time.Since(startTime),
		}

		syncedRepos = append(syncedRepos, syncedRepository)
	}

	return syncedRepos
}

func main() {
	targetRepos := getTargetRepos()
	syncedRepos := syncWorkflows(targetRepos)

	MakeSyncedReposSummary(syncedRepos)
	if AnySyncedRepoHasError(syncedRepos) {
		panic(errors.New("one or more repositories were not synced successfully"))
	}
}
