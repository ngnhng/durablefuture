package future

// BlockingFutureError is thrown in a panic if a Future is not ready when Get is called.
// The concept is similar to the yield mechanism in coroutines, this will pause the workflow and resume it when the Future is ready.
type BlockingFutureError struct{}

func (e BlockingFutureError) Error() string {
	return "blocking_future"
}
