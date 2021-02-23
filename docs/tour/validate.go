package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

func main() {
	for _, arg := range os.Args[1:] {
		valid, err := runValidate(arg)

		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "Failed to process %s: %s\n", arg, err.Error())
			if err != nil {
				panic(err)
			}
		}

		if !valid {
			os.Exit(1)
		}
	}
}

func runValidate(inputFile string) (bool, error) {
	input, err := ioutil.ReadFile(inputFile)

	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", inputFile, err)
	}

	parts := parseText(strings.Split(string(input), "\n"))

	for k, v := range parts {
		kPath := path.Join(path.Dir(inputFile), k)
		kFile, err := ioutil.ReadFile(kPath)

		if err != nil {
			return false, fmt.Errorf("failed to read %s: %w", kPath, err)
		}

		if !sameContent(string(kFile), v) {
			return false, nil
		}
	}

	return true, nil
}

func sameContent(a string, b string) bool {
	return strings.ReplaceAll(a, "\n", "") == strings.ReplaceAll(b, "\n", "")
}

func parseText(lines []string) map[string]string {
	inCodeBlock := false
	var currentFile *string
	result := map[string]string{}

	for _, line := range lines {
		references := isSetReferenceLine(line)
		if !inCodeBlock && references != nil {
			currentFile = references
			continue
		}

		if toggleCodeBloc(line) {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock && currentFile != nil {
			if acc, ok := result[*currentFile]; ok {
				result[*currentFile] = acc + "\n" + line
			} else {
				result[*currentFile] = line
			}
		}
	}

	return result
}

const (
	referencePrefix = "[//]: # ("
	referenceSuffix = ")"
)

func isSetReferenceLine(line string) *string {
	if strings.HasPrefix(line, referencePrefix) && strings.HasSuffix(line, referenceSuffix) {
		s := line[len(referencePrefix) : len(line)-len(referenceSuffix)]

		return &s
	}

	return nil
}

func toggleCodeBloc(line string) bool {
	return strings.HasPrefix(line, "```")
}
