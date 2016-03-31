package main

import (
	"fmt"
	"log"

	"bitbucket.org/mattholt/checkup"
)

func main() {
	c := checkup.Checkup{
		Checkers: []checkup.Checker{
			checkup.HTTPChecker{Name: "IP Chicken", URL: "http://ipchicken.com", Attempts: 5},
			checkup.HTTPChecker{Name: "Example", URL: "https://example.com", Attempts: 5},
		},
		Storage: checkup.S3{
			AccessKeyID:     "...",
			SecretAccessKey: "...",
			Region:          "us-east-1",
			Bucket:          "srcgraph-monitor-test",
		},
	}

	results, err := c.Check()
	if err != nil {
		log.Fatal(err)
	}

	for _, result := range results {
		fmt.Printf("== %s (%s)\n", result.Title, result.Endpoint)
		fmt.Printf("     Max: %s\n", result.Times.Max)
		fmt.Printf("     Min: %s\n", result.Times.Min)
		fmt.Printf("  Median: %s\n", result.Times.Median)
		fmt.Printf(" Average: %s\n", result.Times.Average)
		fmt.Printf("     All: %v\n", result.Times.All)
		fmt.Printf("    Down: %v\n\n", result.Down)
	}
}
