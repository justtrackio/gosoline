package share

type Settings struct {
	TableName struct {
		Owner  string `cfg:"owner" default:"owners"`
		Policy string `cfg:"policy" default:"policies"`
		Share  string `cfg:"share" default:"shares"`
	} `cfg:"table_name"`
}
