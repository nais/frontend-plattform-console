package unleash

type UnleashError struct {
	Reason string
	Err    error
}

func (e *UnleashError) Error() string {
	return e.Reason
}
