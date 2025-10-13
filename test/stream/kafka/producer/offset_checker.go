package producer

import (
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/stretchr/testify/assert"
)

func CheckExpectedKafkaEndOffset(s suite.TestingSuite, app suite.AppUnderTest, expectedOffset int64) {
	app.WaitDone()

	client := s.Env().Kafka("default").AdminClient()
	offsets, err := client.ListEndOffsets(s.Env().Context(), "gosoline-test-test-grp-testEvent")
	assert.NoError(s.T(), err)

	offset, ok := offsets.Lookup("gosoline-test-test-grp-testEvent", 0)
	assert.True(s.T(), ok)

	assert.Equal(s.T(), expectedOffset, offset.Offset)
}
