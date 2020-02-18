package fixtures

const (
	SettingsOutputWriterMysql = "mysql"
	SettingsInputReaderJson = "json"
)

type InputSettings struct {
	Path string `cfg:"path"`
	Encoding string `cfg:"encoding"`
}

type FixtureLoaderSettings struct {
	Enable bool `cfg:"enable" default:"false"`
	Purge bool `cfg:"purge" default:"true"`
	Output string `cfg:"output"`
	Input InputSettings `cfg:"input"`
}
