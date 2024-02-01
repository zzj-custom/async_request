package async_request

import (
	"context"
	"github.com/pkg/errors"
	"sync"
)

type AsyncRequest struct {
	worker int // 需要的工作数
	finish int // 已经完成的工作数
	ch     chan struct{}
	err    error
	mux    sync.Mutex
}

type AsyncRequestOptions struct {
	worker int
}

type AsyncRequestWorker func(*AsyncRequestOptions)

func NewAsyncRequest(options ...AsyncRequestWorker) *AsyncRequest {
	aro := &AsyncRequestOptions{worker: 4}
	for _, option := range options {
		option(aro)
	}

	return &AsyncRequest{
		worker: aro.worker,
		ch:     make(chan struct{}),
	}
}

func WorkerOption(worker int) AsyncRequestWorker {
	return func(aro *AsyncRequestOptions) {
		if worker > 0 {
			aro.worker = worker
		}
	}
}

func (r *AsyncRequest) getError() error {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.err
}

func (r *AsyncRequest) hasFree() bool {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.finish < r.worker
}

func (r *AsyncRequest) launch() {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.finish++
}

func (r *AsyncRequest) WaitForWorker(ctx context.Context) error {
	for {
		r.checkContextDone(ctx)

		if err := r.getError(); err != nil {
			return err
		}

		if r.hasFree() {
			break
		}

		r.wait(ctx)
	}
	r.launch()
	return r.getError()
}

func (r *AsyncRequest) setError(err error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if err == nil {
		return
	}

	if r.err == nil {
		r.err = err
	} else {
		r.err = errors.Wrap(r.err, err.Error())
	}
}

func (r *AsyncRequest) checkContextDone(ctx context.Context) {
	select {
	case <-ctx.Done():
		r.setError(ctx.Err())
	default:

	}
}

func (r *AsyncRequest) wait(ctx context.Context) {
	select {
	case <-r.ch:
	case <-ctx.Done():
		r.setError(ctx.Err())
	}
}

func (r *AsyncRequest) CompleteWorker(err error) {
	r.setError(err)
	r.land()
	r.ch <- struct{}{}
}

func (r *AsyncRequest) land() {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.finish--
}

func (r *AsyncRequest) hasFinishStanding() bool {
	return r.finish > 0
}

func (r *AsyncRequest) Flush(ctx context.Context) error {
	for {
		r.checkContextDone(ctx)
		if err := r.getError(); err != nil {
			return err
		}

		if !r.hasFinishStanding() {
			break
		}
		r.wait(ctx)
	}
	return r.getError()
}

func Request(ctx context.Context, requests []IRequest, responses []IResponse, options ...AsyncRequestWorker) error {
	asyncRequestRepository := NewAsyncRequest(options...)
	worker := func(ctx context.Context, i int) error {
		err := Do(requests[i], responses[i])
		return err
	}

	for i := range requests {
		err := asyncRequestRepository.WaitForWorker(ctx)
		if err != nil {
			return errors.Wrapf(err, "request %d error", i)
		}
		go func(i int) {
			asyncRequestRepository.CompleteWorker(worker(ctx, i))
		}(i)
	}
	return asyncRequestRepository.Flush(ctx)
}
