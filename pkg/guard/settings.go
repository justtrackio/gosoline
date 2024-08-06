package guard

type auditSettings struct {
	LogGrants     bool `cfg:"log_grants"     default:"false"`
	LogRejections bool `cfg:"log_rejections" default:"true"`
}
