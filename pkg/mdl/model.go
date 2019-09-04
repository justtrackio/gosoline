package mdl

import (
	"fmt"
	"strings"
)

type ConfigProvider interface {
	GetString(string) string
}

type ModelId struct {
	Project     string
	Environment string
	Family      string
	Application string
	Name        string
}

func (m *ModelId) String() string {
	return fmt.Sprintf("%v.%v.%v.%v", m.Project, m.Family, m.Application, m.Name)
}

func (m *ModelId) PadFromConfig(config ConfigProvider) {
	if len(m.Project) == 0 {
		m.Project = config.GetString("app_project")
	}

	if len(m.Environment) == 0 {
		m.Environment = config.GetString("env")
	}

	if len(m.Family) == 0 {
		m.Family = config.GetString("app_family")
	}

	if len(m.Application) == 0 {
		m.Application = config.GetString("app_name")
	}
}

type Identifiable interface {
	GetId() *uint
}

type Identifier struct {
	Id *uint `json:"id" binding:"required"`
}

func WithIdentifier(id *uint) *Identifier {
	return &Identifier{
		Id: id,
	}
}

func UuidWithDashes(uuid *string) *string {
	if strings.Contains(*uuid, "-") {
		return uuid
	}

	dashed := fmt.Sprintf("%s-%s-%s-%s-%s", (*uuid)[0:8], (*uuid)[8:12], (*uuid)[12:16], (*uuid)[16:20], (*uuid)[20:32])

	return &dashed
}

type Resource interface {
	GetResourceName() string
}
