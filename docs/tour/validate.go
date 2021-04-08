package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

func main() {
	for _, arg := range os.Args[1:] {
		if err := runValidate(arg); err != nil {
			_, err := fmt.Fprintf(os.Stderr, "Failed to process %s: %s\n", arg, err.Error())
			if err != nil {
				panic(err)
			}
			os.Exit(1)
		}
	}
}

func runValidate(inputFile string) error {
	input, err := ioutil.ReadFile(inputFile)

	if err != nil {
		return fmt.Errorf("failed to read %s: %w", inputFile, err)
	}

	parts := parseText(strings.Split(string(input), "\n"))

	for k, v := range parts {
		kPath := path.Join(path.Dir(inputFile), k)
		kFile, err := ioutil.ReadFile(kPath)

		if err != nil {
			return fmt.Errorf("failed to read %s: %w", kPath, err)
		}

		if !sameContent(string(kFile), v) {
			diff, err := getDiff(string(kFile), v)
			if err != nil {
				return fmt.Errorf("file %s did not validate, but failed to produce a diff: %w", kPath, err)
			}

			return fmt.Errorf("file %s did not validate: %s", kPath, diff)
		}
	}

	return nil
}

func sameContent(a string, b string) bool {
	return strings.ReplaceAll(a, "\n", "") == strings.ReplaceAll(b, "\n", "")
}

func getDiff(a string, b string) (diff string, err error) {
	tmp1, err := ioutil.TempFile(os.TempDir(), "diff-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		removeErr := os.Remove(tmp1.Name())
		if removeErr != nil {
			err = multierror.Append(err, fmt.Errorf("failed to delete temp file %s: %w", tmp1.Name(), removeErr))
		}
	}()

	tmp2, err := ioutil.TempFile(os.TempDir(), "diff-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		removeErr := os.Remove(tmp2.Name())
		if removeErr != nil {
			err = multierror.Append(err, fmt.Errorf("failed to delete temp file %s: %w", tmp2.Name(), removeErr))
		}
	}()

	if _, err := tmp1.WriteString(a); err != nil {
		return "", fmt.Errorf("failed to write to file: %w", err)
	}
	if _, err := tmp2.WriteString(b); err != nil {
		return "", fmt.Errorf("failed to write to file: %w", err)
	}

	var buffer bytes.Buffer
	cmd := exec.Command("diff", tmp1.Name(), tmp2.Name())
	cmd.Stdout = &buffer
	cmd.Stderr = &buffer
	if err = cmd.Run(); err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) && exitErr.ExitCode() < 2 {
			// diff returns 0 in case of no difference, 1 in case of a difference and 2 in case of some other error
			return buffer.String(), nil
		}
		return "", fmt.Errorf("failed to run diff: %w %s", err, buffer.String())
	}

	return buffer.String(), nil
}

func parseText(lines []string) map[string]string {
	inCodeBlock := false
	var currentFiles []string
	result := map[string]string{}

	for _, line := range lines {
		references := isSetReferenceLine(line)
		if !inCodeBlock && len(references) > 0 {
			currentFiles = references
			continue
		}

		if toggleCodeBloc(line) {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			for _, currentFile := range currentFiles {
				if currentFile == "-" {
					continue
				}

				if acc, ok := result[currentFile]; ok {
					result[currentFile] = acc + "\n" + line
				} else {
					result[currentFile] = line
				}
			}
		}
	}

	return result
}

const (
	referencePrefix = "[//]: # ("
	referenceSuffix = ")"
)

func isSetReferenceLine(line string) []string {
	if strings.HasPrefix(line, referencePrefix) && strings.HasSuffix(line, referenceSuffix) {
		s := line[len(referencePrefix) : len(line)-len(referenceSuffix)]

		return strings.Split(s, ",")
	}

	return nil
}

func toggleCodeBloc(line string) bool {
	return strings.HasPrefix(line, "```")
}
