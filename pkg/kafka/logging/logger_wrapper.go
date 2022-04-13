package logging

type LoggerWrapper func(template string, values ...interface{})

func (log LoggerWrapper) Printf(t string, v ...interface{}) {
	log(t, v...)
}
