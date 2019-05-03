package mon

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

const ecsMetadataFileEnv = "ECS_CONTAINER_METADATA_FILE"

func (l *logger) checkEcsMetadataAvailability() {
	path, available := os.LookupEnv(ecsMetadataFileEnv)

	l.ecsAvailable = available && len(path) > 0
}

func (l *logger) readEcsMetadata() EcsMetadata {
	l.ecsLck.Lock()
	defer l.ecsLck.Unlock()

	if l.ecsAvailable == false || len(l.ecsMetadata) > 0 {
		return l.ecsMetadata
	}

	var metadata EcsMetadata
	path, _ := os.LookupEnv(ecsMetadataFileEnv)

	for {
		data, err := ioutil.ReadFile(path)

		if err != nil {
			l.WithFields(Fields{
				"err": err,
			}).Warn("can not read ecs metadata file")
			os.Exit(1)
		}

		metadata = make(EcsMetadata)
		err = json.Unmarshal(data, &metadata)

		if err != nil {
			l.WithFields(Fields{
				"err": err,
			}).Warn("can not unmarshal ecs metadata")
			os.Exit(1)
		}

		if status, ok := metadata["MetadataFileStatus"]; ok {
			if status.(string) == "READY" {
				break
			}
		}

		l.Info("waiting for ecs metadata being ready")
		time.Sleep(1 * time.Second)
	}

	l.ecsMetadata = metadata

	return metadata
}
