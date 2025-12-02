package internal

// errorBlockingFuture is thrown in a panic if a Future is not ready when Get is called.
// The concept is similar to the yield mechanism in coroutines, this will pause the workflow and resume it when the Future is ready.
type errorBlockingFuture struct{}

func (e errorBlockingFuture) Error() string {
	return "blocking_future"
}
