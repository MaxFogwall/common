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

	var repoBatches [][]string
	var repoBatch []string

	for index, repo := range repos {
		repoBatch = append(repoBatch, repo)

		// If full, add batch and start over.
		if (index+1)%batchSize == 0 {
			repoBatches = append(repoBatches, repoBatch)
			repoBatch = []string{}
		}
	}

	// If we have a partially full batch left over, add it too.
	if len(repoBatch) > 0 {
		repoBatches = append(repoBatches, repoBatch)
	}

	repoBatchesJson, err := json.Marshal(repoBatches)
	if err != nil {
		log.Fatalf("could not convert repo batches to JSON")
	}

	fmt.Printf("converted repo batches to JSON: \"%s\"", string(repoBatchesJson))
	common.WriteFile("repo-batches.json", string(repoBatchesJson))
}
