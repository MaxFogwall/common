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

type Batches struct {
	Batches []string `json:"batches"`
}

type Batch struct {
	Repos  []string `json:"repos"`
	Number int      `json:"number"`
}

func AsJson[T any](contents T) string {
	contentsJson, err := json.Marshal(contents)
	if err != nil {
		log.Fatalf("could not convert repo batches to JSON")
	}

	return string(contentsJson)
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
	batches := Batches{
		Batches: []string{},
	}
	reposInBatch := []string{}
	batchNumber := 1

	for index, repo := range repos {
		batchNumber = int(index/batchSize) + 1
		reposInBatch = append(reposInBatch, repo)

		if (index+1)%batchSize == 0 {
			batches.Batches = append(batches.Batches, AsJson(Batch{
				Repos:  reposInBatch,
				Number: batchNumber,
			}))
			reposInBatch = []string{}
		}
	}

	if len(reposInBatch) > 0 {
		batches.Batches = append(batches.Batches, AsJson(Batch{
			Repos:  reposInBatch,
			Number: batchNumber,
		}))
	}

	batchesJson, err := json.Marshal(batches)
	if err != nil {
		log.Fatalf("could not convert repo batches to JSON")
	}

	fmt.Printf("converted repo batches to JSON: \"%s\"", string(batchesJson))
	common.WriteOutput(string(batchesJson))
}
