package suite_test

import (
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

func (s *GatewayTestSuite) TestBase(app suite.AppUnderTest, client *resty.Client) error {
	defer app.WaitDone()
	defer app.Stop()

	response, err := client.R().
		SetBody("this is a test").
		Execute(http.MethodPost, "/reverse")
	if err != nil {
		return err
	}

	s.Equal(http.StatusOK, response.StatusCode())
	s.Equal(funk.Reverse([]byte("this is a test")), response.Body())

	return nil
}
