package main

import (
	"fmt"
	"regexp"
	"strconv"

	common "github.com/workflow-sync-poc/common/code"
)

func nextMajorVersionForTag(tag string) int {
	majorVersionPattern := regexp.MustCompile(`v(?P<MajorVersion>\d+)`)
	majorVersionSubmatches := majorVersionPattern.FindStringSubmatch(tag)
	majorVersionSubmatchIndex := majorVersionPattern.SubexpIndex("MajorVersion")
	majorVersionString := majorVersionSubmatches[majorVersionSubmatchIndex]
	majorVersion, err := strconv.Atoi(majorVersionString)
	if err != nil {
		panic(fmt.Errorf("could not parse major version from tag '%s': %v", tag, err))
	}

	return majorVersion + 1
}

func hasAnySyncedWorkflowChanged() bool {
	isNoSyncedWorkflowChanged, err := common.IsLastCommitClean(".github/workflows/synced_*")
	if err != nil {
		panic(err)
	}

	return !isNoSyncedWorkflowChanged
}

func hasSyncedReposListChangedSince(sinceTag string) bool {
	isSyncedReposListSame, err := common.IsTaggedCommitClean("repos.json", sinceTag)
	if err != nil {
		panic(err)
	}

	return !isSyncedReposListSame
}

func hasAnySyncedWorkflowChangedSince(sinceTag string) bool {
	isSyncedWorkflowsSame, err := common.IsTaggedCommitClean(".github/workflows/synced_*", sinceTag)
	if err != nil {
		panic(err)
	}

	return !isSyncedWorkflowsSame
}

func shouldIncrementTag() bool {
	return hasAnySyncedWorkflowChanged()
}

func shouldSyncWorkflows() bool {
	lastSyncedTag := "last-synced"

	hasEverSynced, err := common.TagExists(lastSyncedTag)
	if err != nil {
		panic(err)
	}

	if !hasEverSynced {
		return true
	}

	return hasAnySyncedWorkflowChangedSince(lastSyncedTag) || hasSyncedReposListChangedSince(lastSyncedTag)
}

func main() {
	common.SetupGitHubUser()

	tag, err := common.GetLatestVersionTag()
	if err != nil {
		panic(err)
	}

	if tag == "" {
		tag = "v1"
		if err := common.AddTag(tag); err != nil {
			panic(err)
		}
		common.WriteJobSummary(fmt.Sprintf("### 🏷️ Tag `%s` Created", tag))
	} else if shouldIncrementTag() {
		nextMajorVersion := nextMajorVersionForTag(tag)
		nextTag := fmt.Sprintf("v%v", nextMajorVersion)
		if err := common.AddTag(nextTag); err != nil {
			panic(err)
		}
		common.WriteJobSummary(fmt.Sprintf("### 🏷️ Tag `%s` Created", nextTag))
	} else {
		if err := common.MoveTag(tag); err != nil {
			panic(err)
		}
		common.WriteJobSummary(fmt.Sprintf("### 🏷️ Tag `%s` Updated", tag))
	}

	common.WriteOutput(fmt.Sprintf("%v", shouldSyncWorkflows()))
}
