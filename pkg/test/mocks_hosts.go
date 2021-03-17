package test

func (m *Mocks) ProvideSqsHost(name string) string {
	component := m.components[name].(*snsSqsComponent)
	return component.settings.Host
}

func (m *Mocks) ProvideSnsHost(name string) string {
	component := m.components[name].(*snsSqsComponent)
	return component.settings.Host
}

func (m *Mocks) ProvideCloudwatchHost(name string) string {
	component := m.components[name].(*cloudwatchComponent)
	return component.settings.Host
}

func (m *Mocks) ProvideDynamoDbHost(name string) string {
	component := m.components[name].(*dynamoDbComponent)
	return component.settings.Host
}

func (m *Mocks) ProvideKinesisHost(name string) string {
	component := m.components[name].(*kinesisComponent)
	return component.settings.Host
}

func (m *Mocks) ProvideS3Host(name string) string {
	component := m.components[name].(*s3Component)
	return component.settings.Host
}

func (m *Mocks) ProvideMysqlHost(name string) string {
	component := m.components[name].(*mysqlComponentLegacy)
	return component.settings.Host
}

func (m *Mocks) ProvideRedisHost(name string) string {
	component := m.components[name].(*redisComponent)
	return component.settings.Host
}

func (m *Mocks) ProvideWiremockHost(name string) string {
	component := m.components[name].(*wiremockComponent)
	return component.settings.Host
}
