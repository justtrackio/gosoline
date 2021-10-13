package instancemetadataservice

import (
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/pkg/errors"
)

const metadataFileEnv = "ECS_CONTAINER_METADATA_FILE"

type Metadata map[string]interface{}

var (
	metadataLck sync.Mutex
	metadata    Metadata
)

func ReadEcsMetadata() (Metadata, error) {
	metadataLck.Lock()
	defer metadataLck.Unlock()

	if metadata != nil {
		return metadata, nil
	}

	path, ok := os.LookupEnv(metadataFileEnv)

	if len(path) == 0 || !ok {
		return nil, nil
	}

	m := make(Metadata)

	for {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "can not read ecs metadata file")
		}

		m = make(Metadata)
		err = json.Unmarshal(data, &m)

		if err != nil {
			return nil, errors.Wrap(err, "can not unmarshal ecs metadata")
		}

		if status, ok := m["MetadataFileStatus"]; ok {
			if status.(string) == "READY" {
				break
			}
		}

		time.Sleep(1 * time.Second)
	}

	metadata = m

	return m, nil
}
