package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gogithub "github.com/google/go-github/v62/github"
	common "github.com/workflow-sync-poc/common/code"
)

type SyncedRepository struct {
	Identifier  string
	Error       error
	ElapsedTime time.Duration
	PullRequest *gogithub.PullRequest
}

func getTargetRepos() []string {
	reposJsonPath := "repos.json"
	reposJson, err := common.ReadFile(reposJsonPath)
	if err != nil {
		panic(fmt.Errorf("could not read '%s': %v", reposJsonPath, err))
	}

	var repos []string
	err = json.Unmarshal([]byte(reposJson), &repos)
	if err != nil {
		panic(fmt.Errorf("could not parse '%s', expected a JSON formatted list of strings: %v", reposJsonPath, err))
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

	return "✔️"
}

func formatPullRequestStatus(syncedRepo SyncedRepository) string {
	mergedImageUrl := "https://github.com/workflow-sync-poc/component-1/assets/48988185/43f86b74-a8eb-4df3-a2f3-ac8e714784b5"
	openImageUrl := "https://github.com/workflow-sync-poc/component-1/assets/48988185/bc28bc57-f91c-4389-a103-d4524d4f6e39"

	format := "<img align=\"center\" src=\"%s\"/>"

	if syncedRepo.Error != nil {
		return fmt.Sprintf(format, openImageUrl)
	}

	return fmt.Sprintf(format, mergedImageUrl)
}

func formatPullRequest(syncedRepo SyncedRepository) string {
	pullRequestString := "No changes needed."
	if syncedRepo.PullRequest != nil {
		pullRequestString = fmt.Sprintf("%s [**%s**](%s) #%v", formatPullRequestStatus(syncedRepo), *syncedRepo.PullRequest.Title, *syncedRepo.PullRequest.HTMLURL, *syncedRepo.PullRequest.Number)
	} else if syncedRepo.Error != nil {
		pullRequestString = "Could not create."
	}

	return fmt.Sprintf("<ul><li>%s</li></ul>", pullRequestString)
}

func formatTime(syncedRepo SyncedRepository) string {
	return syncedRepo.ElapsedTime.Round(time.Second).String()
}

func WriteSyncedReposSummary(syncedRepos []SyncedRepository) {
	var syncedReposTable []string
	var syncedReposErrors []string

	syncedReposTable = append(syncedReposTable, "| Repository | Success | Pull Request | T-Start |")
	syncedReposTable = append(syncedReposTable, "|:-|:-:|:-|-:|")

	for _, syncedRepo := range syncedRepos {
		if syncedRepo.Error != nil {
			syncedReposErrors = append(syncedReposErrors, fmt.Sprintf("- ❌ %s (%s)", formatRepo(syncedRepo), syncedRepo.Error))
		}

		syncedReposTable = append(syncedReposTable, fmt.Sprintf("| %s | %s | %s | %s |", formatRepo(syncedRepo), formatSuccess(syncedRepo), formatPullRequest(syncedRepo), formatTime(syncedRepo)))
	}

	var summaryLines []string
	summaryLines = append(summaryLines, strings.Join(syncedReposTable, "\r\n"))
	if len(syncedReposErrors) > 0 {
		summaryLines = append(summaryLines, strings.Join(syncedReposErrors, "\r\n"))
	}

	common.WriteJobSummary(strings.Join(summaryLines, "\r\n"))
}

func AnySyncedRepoHasError(syncedRepos []SyncedRepository) bool {
	for _, syncedRepo := range syncedRepos {
		if syncedRepo.Error != nil {
			return true
		}
	}

	return false
}

func syncWorkflows(repos []string, sourceRef string) []SyncedRepository {
	startTime := time.Now()
	syncedRepos := []SyncedRepository{}

	for _, repo := range repos {
		pullRequest, err := common.SyncRepository(repo, sourceRef)
		if err != nil {
			log.Printf("Failed to sync to '%s': %v\n", repo, err)
		}

		syncedRepository := SyncedRepository{
			Identifier:  repo,
			Error:       err,
			ElapsedTime: time.Since(startTime),
			PullRequest: pullRequest,
		}

		syncedRepos = append(syncedRepos, syncedRepository)
	}

	return syncedRepos
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("missing argument for source ref (e.g. \"v1\")")
	}

	sourceRef := os.Args[1]
	targetRepos := getTargetRepos()
	syncedRepos := syncWorkflows(targetRepos, sourceRef)

	WriteSyncedReposSummary(syncedRepos)
	if AnySyncedRepoHasError(syncedRepos) {
		panic(errors.New("one or more repositories were not synced successfully"))
	} else {
		if err := common.AddOrMoveTag("last-synced"); err != nil {
			panic(err)
		}
	}
}
