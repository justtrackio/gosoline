package log

import (
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

const ecsMetadataFileEnv = "ECS_CONTAINER_METADATA_FILE"

type EcsMetadata map[string]interface{}

var ecsLck sync.Mutex
var ecsMetadata EcsMetadata

func ReadEcsMetadata() (EcsMetadata, error) {
	ecsLck.Lock()
	defer ecsLck.Unlock()

	if ecsMetadata != nil {
		return ecsMetadata, nil
	}

	path, ok := os.LookupEnv(ecsMetadataFileEnv)

	if len(path) == 0 || !ok {
		return nil, nil
	}

	metadata := make(EcsMetadata)

	for {
		data, err := ioutil.ReadFile(path)

		if err != nil {
			return nil, errors.Wrap(err, "can not read ecs metadata file")
		}

		metadata = make(EcsMetadata)
		err = json.Unmarshal(data, &metadata)

		if err != nil {
			return nil, errors.Wrap(err, "can not unmarshal ecs metadata")
		}

		if status, ok := metadata["MetadataFileStatus"]; ok {
			if status.(string) == "READY" {
				break
			}
		}

		time.Sleep(1 * time.Second)
	}

	ecsMetadata = metadata

	return ecsMetadata, nil
}
