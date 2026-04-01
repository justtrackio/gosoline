package schema_registry_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/justtrackio/gosoline/pkg/exec"
	schemaRegistry "github.com/justtrackio/gosoline/pkg/kafka/schema-registry"
	"github.com/justtrackio/gosoline/pkg/kafka/schema-registry/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/suite"
	"github.com/twmb/franz-go/pkg/sr"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type ServiceTestSuite struct {
	suite.Suite

	ctx     context.Context
	client  *mocks.Client
	service schemaRegistry.Service
}

func (s *ServiceTestSuite) SetupTest() {
	s.ctx = s.T().Context()
	s.client = mocks.NewClient(s.T())
	s.service = schemaRegistry.NewServiceWithInterfaces(
		s.client,
		exec.NewExecutor(
			logMocks.NewLoggerMock(logMocks.WithTestingT(s.T()), logMocks.WithMockUntilLevel(log.PriorityWarn)),
			&exec.ExecutableResource{Type: "schema-registry", Name: "test"},
			&exec.BackoffSettings{InitialInterval: 0, MaxInterval: 0, MaxAttempts: 3},
			[]exec.ErrorChecker{exec.CheckConnectionError},
		),
	)
}

func (s *ServiceTestSuite) TestGetSubjectSchemaId_LookupSuccess() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeAvro}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{ID: 13}, nil).Once()

	id, err := s.service.GetSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Avro)

	s.NoError(err)
	s.Equal(13, id)
}

func (s *ServiceTestSuite) TestGetOrCreateSubjectSchemaId_LookupSuccessWithoutCreate() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeJSON}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{ID: 17}, nil).Once()

	id, err := s.service.GetOrCreateSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Json)

	s.NoError(err)
	s.Equal(17, id)
}

func (s *ServiceTestSuite) TestGetSubjectSchemaId_LookupMissReturnsError() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeAvro}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusNotFound,
			ErrorCode:  sr.ErrSchemaNotFound.Code,
			Message:    "schema not found",
		},
	).Once()

	id, err := s.service.GetSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Avro)

	s.Zero(id)
	s.EqualError(err, "failed to lookup subject schema: schema not found")
}

func (s *ServiceTestSuite) TestGetOrCreateSubjectSchemaId_CreatesOnLookupMiss() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeJSON}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusNotFound,
			ErrorCode:  sr.ErrSchemaNotFound.Code,
			Message:    "schema not found",
		},
	).Once()
	s.client.EXPECT().CreateSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{ID: 23}, nil).Once()

	id, err := s.service.GetOrCreateSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Json)

	s.NoError(err)
	s.Equal(23, id)
}

func (s *ServiceTestSuite) TestGetOrCreateSubjectSchemaId_CreateConflictRetriesLookup() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeAvro}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusNotFound,
			ErrorCode:  sr.ErrSchemaNotFound.Code,
			Message:    "schema not found",
		},
	).Once()
	s.client.EXPECT().CreateSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusConflict,
			ErrorCode:  sr.ErrUnknown.Code,
			Message:    "schema already registered",
		},
	).Once()
	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{ID: 29}, nil).Once()

	id, err := s.service.GetOrCreateSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Avro)

	s.NoError(err)
	s.Equal(29, id)
}

func (s *ServiceTestSuite) TestGetOrCreateSubjectSchemaId_CreateIncompatibleReturnsError() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeProtobuf}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusNotFound,
			ErrorCode:  sr.ErrSchemaNotFound.Code,
			Message:    "schema not found",
		},
	).Once()
	s.client.EXPECT().CreateSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusConflict,
			ErrorCode:  sr.ErrIncompatibleSchema.Code,
			Message:    sr.ErrIncompatibleSchema.Description,
		},
	).Once()

	id, err := s.service.GetOrCreateSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Protobuf)

	s.Zero(id)
	s.EqualError(err, "failed to create subject schema: Schema is incompatible with an earlier schema")
}

func (s *ServiceTestSuite) TestGetOrCreateSubjectSchemaId_RetriesCreateOnTransientConnectionError() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeJSON}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusNotFound,
			ErrorCode:  sr.ErrSchemaNotFound.Code,
			Message:    "schema not found",
		},
	).Once()
	s.client.EXPECT().CreateSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{}, io.EOF).Once()
	s.client.EXPECT().CreateSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{ID: 31}, nil).Once()

	id, err := s.service.GetOrCreateSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Json)

	s.NoError(err)
	s.Equal(31, id)
}

func (s *ServiceTestSuite) TestGetOrCreateSubjectSchemaId_RetriesCreateUntilConflictThenLooksUp() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeAvro}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusNotFound,
			ErrorCode:  sr.ErrSchemaNotFound.Code,
			Message:    "schema not found",
		},
	).Once()
	s.client.EXPECT().CreateSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{}, io.EOF).Once()
	s.client.EXPECT().CreateSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusConflict,
			ErrorCode:  sr.ErrUnknown.Code,
			Message:    "schema already registered",
		},
	).Once()
	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{ID: 37}, nil).Once()

	id, err := s.service.GetOrCreateSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Avro)

	s.NoError(err)
	s.Equal(37, id)
}

func (s *ServiceTestSuite) TestGetOrCreateSubjectSchemaId_ReturnsRetryableErrorWhenAttemptsExhausted() {
	registrySchema := sr.Schema{Schema: `{"type":"string"}`, Type: sr.TypeJSON}

	s.client.EXPECT().LookupSchema(matcher.Context, "test-subject", registrySchema).Return(
		sr.SubjectSchema{},
		&sr.ResponseError{
			StatusCode: http.StatusNotFound,
			ErrorCode:  sr.ErrSchemaNotFound.Code,
			Message:    "schema not found",
		},
	).Once()
	s.client.EXPECT().CreateSchema(matcher.Context, "test-subject", registrySchema).Return(sr.SubjectSchema{}, io.EOF).Times(3)

	id, err := s.service.GetOrCreateSubjectSchemaId(s.ctx, "test-subject", registrySchema.Schema, schemaRegistry.Json)

	s.Zero(id)
	s.Error(err)
	s.True(errors.Is(err, io.EOF))
	s.EqualError(err, "failed to create subject schema: EOF")
}
