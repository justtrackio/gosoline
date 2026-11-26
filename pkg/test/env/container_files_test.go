package env

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArchiveContainerFiles(t *testing.T) {
	data := []byte{0, 0xff, 0x80, '\n'}
	filename := filepath.Join(t.TempDir(), "binary.gz")
	require.NoError(t, os.WriteFile(filename, data, 0o600))
	archive, err := archiveContainerFiles([]ContainerFile{{Source: filename, Target: "/tmp/fixtures/binary.gz"}})
	require.NoError(t, err)
	reader := tar.NewReader(bytes.NewReader(archive))
	header, err := reader.Next()
	require.NoError(t, err)
	require.Equal(t, "tmp/fixtures/binary.gz", header.Name)
	require.Equal(t, int64(0o644), header.Mode)
	actual, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, data, actual)
	_, err = reader.Next()
	require.ErrorIs(t, err, io.EOF)
}

func TestArchiveContainerFilesRejectsInvalidPaths(t *testing.T) {
	for _, target := range []string{"relative.gz", "/", "/tmp/../response.gz"} {
		t.Run(target, func(t *testing.T) {
			_, err := archiveContainerFiles([]ContainerFile{{Source: "unused", Target: target}})
			require.ErrorContains(t, err, "must be a clean absolute file path")
		})
	}
}

func TestArchiveContainerFilesRejectsMissingSource(t *testing.T) {
	_, err := archiveContainerFiles([]ContainerFile{{Source: filepath.Join(t.TempDir(), "missing.gz"), Target: "/tmp/response.gz"}})
	require.ErrorIs(t, err, os.ErrNotExist)
}
