package main

import (
	"flag"
	"log"
	"strings"

	"github.com/storozhukBM/go-fsm-generator/generator"
)

func main() {
	verbose := flag.Bool("v", false, "verbose output from generator")
	typeNames := flag.String("type", "", "comma-separated list of type names; must be set")
	var dirName string
	flag.StringVar(&dirName, "dir", ".", "working directory; must be set")

	flag.Parse()
	if len(*typeNames) == 0 {
		log.Fatalf("the flag -type must be set")
	}
	if len(dirName) == 0 {
		log.Fatalf("the flag -dir must be set")
	}
	types := strings.Split(*typeNames, ",")
	generator.RunGeneratorForTypes(dirName, types, *verbose)
}
