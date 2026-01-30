package mdl_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/suite"
)

func TestModelIdStringTestSuite(t *testing.T) {
	suite.Run(t, new(ModelIdStringTestSuite))
}

type ModelIdStringTestSuite struct {
	suite.Suite
}

func (s *ModelIdStringTestSuite) TestString_NotSet() {
	id := mdl.ModelId{
		Name: "test",
	}

	_, err := id.String()
	s.EqualError(err, "model id domain pattern is not set; call PadFromConfig first")
}

func (s *ModelIdStringTestSuite) TestString_HappyPath_Env() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.model_id.domain_pattern").Return("{app.env}", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)

	id := mdl.ModelId{
		Name: "order",
		Env:  "prod",
		App:  "app",
	}
	s.NoError(id.PadFromConfig(config))

	str, err := id.String()
	s.NoError(err)
	s.Equal("prod.order", str)
}

func (s *ModelIdStringTestSuite) TestString_HappyPath_Tags() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.model_id.domain_pattern").Return("{app.tags.project}.{app.env}", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)

	id := mdl.ModelId{
		Name: "event",
		Env:  "test",
		App:  "app",
		Tags: map[string]string{
			"project": "jt",
		},
	}
	s.NoError(id.PadFromConfig(config))

	str, err := id.String()
	s.NoError(err)
	s.Equal("jt.test.event", str)
}

func (s *ModelIdStringTestSuite) TestString_HappyPath_Static() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.model_id.domain_pattern").Return("my.static.domain", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)

	id := mdl.ModelId{
		Name: "m",
		Env:  "env",
		App:  "app",
	}
	s.NoError(id.PadFromConfig(config))

	str, err := id.String()
	s.NoError(err)
	s.Equal("my.static.domain.m", str)
}

func (s *ModelIdStringTestSuite) TestString_Error_MissingName() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.model_id.domain_pattern").Return("{app.env}", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)

	id := mdl.ModelId{
		Name: "", // Empty name
		Env:  "prod",
		App:  "app",
	}
	s.NoError(id.PadFromConfig(config))

	_, err := id.String()
	s.EqualError(err, `failed to format model id with pattern "{app.env}": model name is required`)
}

func (s *ModelIdStringTestSuite) TestString_Error_MissingTag() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.model_id.domain_pattern").Return("{app.tags.project}.{app.tags.family}", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)

	id := mdl.ModelId{
		Name: "event",
		Env:  "env",
		App:  "app",
		Tags: map[string]string{
			"project": "jt",
			// family missing
		},
	}
	s.NoError(id.PadFromConfig(config))

	_, err := id.String()
	s.EqualError(err, `failed to format model id with pattern "{app.tags.project}.{app.tags.family}": missing required tags: family`)
}

func (s *ModelIdStringTestSuite) TestString_Error_EmptyEnv() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.model_id.domain_pattern").Return("{app.env}", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)

	id := mdl.ModelId{
		Name: "x",
		Env:  "temp", // Set initially so PadFromConfig doesn't query config
		App:  "app",
	}
	s.NoError(id.PadFromConfig(config))

	// Manually clear Env to trigger error in String()
	id.Env = ""

	_, err := id.String()
	s.EqualError(err, `failed to format model id with pattern "{app.env}": pattern requires app.env but it is empty`)
}

func (s *ModelIdStringTestSuite) TestString_DomainString() {
	config := mocks.NewConfig(s.T())
	config.EXPECT().GetString("app.model_id.domain_pattern").Return("{app.tags.project}.{app.env}", nil)
	config.EXPECT().GetStringMap("app.tags").Return(map[string]any{}, nil)

	id := mdl.ModelId{
		Name: "event",
		Env:  "test",
		App:  "app",
		Tags: map[string]string{
			"project": "jt",
		},
	}
	s.NoError(id.PadFromConfig(config))

	str, err := id.DomainString()
	s.NoError(err)
	s.Equal("jt.test", str)
}

func (s *ModelIdStringTestSuite) TestString_DomainString_NotSet() {
	id := mdl.ModelId{
		Name: "test",
	}

	_, err := id.DomainString()
	s.EqualError(err, "model id domain pattern is not set; call PadFromConfig first")
}
