package db_repo

type noopLogger struct{}

func (*noopLogger) Print(v ...interface{}) {}
