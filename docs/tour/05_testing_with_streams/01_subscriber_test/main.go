package main

import (
	"github.com/applike/gosoline/docs/tour/05_testing_with_streams/01_subscriber_test/subscriber"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/test/suite"
	"testing"
)

type SubscriberTestSuite struct {
	suite.Suite
}

func (s *SubscriberTestSuite) SetupSuite() []suite.Option {
	subs := mdlsub.NewSubscriberFactory(subscriber.Transformers)

	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithSharedEnvironment(),
		suite.WithConfigFile("./config.dist.yml"),
		suite.WithModuleFactory(subs),
	}
}

func (s *SubscriberTestSuite) TestSubscriber() (suite.SubscriberTestCase, error) {
	return suite.DdbTestCase(suite.DdbSubscriberTestCase{
		Name:          "book",
		SourceModelId: "gosoline.subscriber-test.book-store.book",
		TargetModelId: "gosoline.subscriber-test.book-api.book",
		Input: &subscriber.BookInput{
			Name:   "Romeo and Juliet",
			Author: "William Shakespeare",
			Content: `
Two households, both alike in dignity,
In fair Verona, where we lay our scene,
From ancient grudge break to new mutiny,
Where civil blood makes civil hands unclean.
From forth the fatal loins of these two foes
A pair of star-cross’d lovers take their life;
Whose misadventured piteous overthrows
Do with their death bury their parents’ strife.
The fearful passage of their death-mark’d love,
And the continuance of their parents’ rage,
Which, but their children’s end, nought could remove,
Is now the two hours’ traffic of our stage;
The which if you with patient ears attend,
What here shall miss, our toil shall strive to mend.
`,
		},
		Assert: func(t *testing.T, fetcher *suite.DdbSubscriberFetcher) {
			sub := &subscriber.Book{}
			fetcher.ByHashAndRange("Romeo and Juliet", "William Shakespeare", sub)

			expected := &subscriber.Book{
				Name:   "Romeo and Juliet",
				Author: "William Shakespeare",
				Length: 643,
			}

			s.Equal(expected, sub)
		},
	})
}

func TestSubscriberTest(t *testing.T) {
	suite.Run(t, new(SubscriberTestSuite))
}

func main() {
	testing.Main(func(pat, str string) (bool, error) {
		return true, nil
	}, []testing.InternalTest{
		{
			Name: "SubscriberTest",
			F:    TestSubscriberTest,
		},
	}, nil, nil)
}
