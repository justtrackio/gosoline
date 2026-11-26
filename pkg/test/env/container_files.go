package env

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
)

func archiveContainerFiles(files []ContainerFile) ([]byte, error) {
	if len(files) == 0 {
		return nil, nil
	}

	var buffer bytes.Buffer
	archive := tar.NewWriter(&buffer)
	for _, file := range files {
		target := file.Target
		if !path.IsAbs(target) || target == "/" || path.Clean(target) != target {
			return nil, fmt.Errorf("container file target %q must be a clean absolute file path", target)
		}

		data, err := os.ReadFile(file.Source)
		if err != nil {
			return nil, fmt.Errorf("could not read container file %q: %w", file.Source, err)
		}

		if err = archive.WriteHeader(&tar.Header{
			Name: strings.TrimPrefix(target, "/"),
			Mode: 0o644,
			Size: int64(len(data)),
		}); err != nil {
			return nil, fmt.Errorf("could not archive container file %q: %w", target, err)
		}
		if _, err = archive.Write(data); err != nil {
			return nil, fmt.Errorf("could not archive container file content %q: %w", target, err)
		}
	}
	if err := archive.Close(); err != nil {
		return nil, fmt.Errorf("could not close container file archive: %w", err)
	}

	return buffer.Bytes(), nil
}
