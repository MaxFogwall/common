package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	common "github.com/workflow-sync-poc/common/code"
)

func getRepos() []string {
	reposJsonPath := "repos.json"
	reposJson := common.ReadFile(reposJsonPath)

	var repos []string
	err := json.Unmarshal([]byte(reposJson), &repos)
	if err != nil {
		panic(fmt.Errorf("could not parse '%s', expected a JSON formatted list of strings: %v", reposJsonPath, err))
	}

	return repos
}

func appendBatchAsJsonString(batch []string, batches []string) ([]string, []string) {
	batchJson, err := json.Marshal(batch)
	if err != nil {
		log.Fatalf("could not convert batch '%v' to JSON: %v", batch, err)
	}

	batches = append(batches, string(batchJson))
	batch = []string{}

	return batch, batches
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("missing argument for batch size")
	}

	batchSizeString := os.Args[1]
	batchSize, err := strconv.Atoi(batchSizeString)
	if err != nil {
		log.Fatalf("could not parse batch size %s", batchSizeString)
	}

	repos := getRepos()

	var batches []string
	var batch []string

	for index, repo := range repos {
		batch = append(batch, repo)

		// If full, add batch and start over.
		if (index+1)%batchSize == 0 {
			batch, batches = appendBatchAsJsonString(batch, batches)
		}
	}

	// If we have a partially full batch left over, add it too.
	if len(batch) > 0 {
		_, batches = appendBatchAsJsonString(batch, batches)
	}

	repoBatchesJson, err := json.Marshal(batches)
	if err != nil {
		log.Fatalf("could not convert repo batches to JSON")
	}

	repoBatchesJsonString := fmt.Sprintf("{\"batches\": %s}", string(repoBatchesJson))
	fmt.Printf("converted repo batches to JSON: \"%s\"", repoBatchesJsonString)
	common.WriteOutput(repoBatchesJsonString)
}
