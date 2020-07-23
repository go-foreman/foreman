package errors

type ResponseError struct {
	error
	status int
}

//Status returns http status code
func (e ResponseError) Status() int {
	return e.status
}

func NewResponseError(status int, err error) ResponseError {
	return ResponseError{status: status, error: err}
}
