package telemetry

type ErrorHandleFunc func(error)

func (f ErrorHandleFunc) Handle(err error) {
	f(err)
}
