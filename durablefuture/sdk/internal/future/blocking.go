package future

type BlockingFutureError struct{}

func (e BlockingFutureError) Error() string {
	return "blocking_future"
}
