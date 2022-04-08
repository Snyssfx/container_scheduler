//go:generate minimock -i container -o ./mock/ -s ".go" -g

package deduplicator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Snyssfx/container_scheduler/internal/containers"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// RequestDeduplicator holds subscriptions to calculations for all incoming requests for a container,
// deduplicate calculations and publish a result for all subscribers.
type RequestDeduplicator struct {
	l           *zap.SugaredLogger
	seed        int
	container   container
	reqID       atomic.Int64
	closeCtx    context.Context
	closeLoopFn context.CancelFunc

	mu                  sync.RWMutex
	inputToSubsriptions map[int]map[int]*subscription
	curInput            int
	cancelCurCalcFn     context.CancelFunc
}

type container interface {
	Calculate(ctx context.Context, input int) (int, error)
	Close() error
}

// NewRequestDeduplicator creates RequestDeduplicator.
func NewRequestDeduplicator(l *zap.SugaredLogger, seed int) (*RequestDeduplicator, error) {
	q, err := containers.NewQual(l.Named("qual"), seed)
	if err != nil {
		return nil, fmt.Errorf("cannot create qual: %w", err)
	}

	ctx, closer := context.WithCancel(context.Background())

	return &RequestDeduplicator{
		l:           l,
		seed:        seed,
		container:   q,
		reqID:       *atomic.NewInt64(0),
		closeCtx:    ctx,
		closeLoopFn: closer,

		mu:                  sync.RWMutex{},
		inputToSubsriptions: make(map[int]map[int]*subscription),
	}, nil
}

// Calculate subscribe user to a calculation, and wait for result.
func (r *RequestDeduplicator) Calculate(ctx context.Context, input int) (int, error) {
	reqID := r.reqID.Inc()
	sub := r.subscribe(input, int(reqID))

	select {

	case <-ctx.Done():
		r.unsubscribe(input, int(reqID))
		return 0, fmt.Errorf("request was canceled: %d, %d", input, reqID)

	case res, opened := <-sub.resultCh:
		if !opened {
			r.l.Panic("unexpected result ch closing")
		}
		return res, nil
	}
}

// Start is an infinite loop when deduplicator choose next input for calculation,
// sends it to the container and publish results.
func (r *RequestDeduplicator) Start() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.closeCtx.Done():
			return
		case <-ticker.C:
			input, result, inputValid, err := r.calculateNextInput()
			if err != nil {
				if inputValid {
					r.l.Errorf("cannot calculate: %s", err.Error())
					r.unsubscribeAll(input)
				}
				continue
			}

			r.publish(input, result)
		}
	}
}

func (r *RequestDeduplicator) calculateNextInput() (input, result int, inputValid bool, err error) {
	r.mu.Lock()

	input, err = r.chooseNextInput()
	if err != nil {
		r.mu.Unlock()
		return 0, 0, false, fmt.Errorf("cannot choose next input: %w", err)
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	r.curInput, r.cancelCurCalcFn = input, cancelFn
	r.mu.Unlock()

	result, err = r.calculateInput(ctx, input)

	r.mu.Lock()
	r.curInput, r.cancelCurCalcFn = 0, nil
	r.mu.Unlock()

	return input, result, true, err
}

func (r *RequestDeduplicator) chooseNextInput() (int, error) {
	if len(r.inputToSubsriptions) == 0 {
		return 0, fmt.Errorf("empty input to subcriptions")
	}

	inputWithMaxSubs, i := 0, 0
	for input, subs := range r.inputToSubsriptions {
		if i == 0 || len(subs) > len(r.inputToSubsriptions[inputWithMaxSubs]) {
			inputWithMaxSubs = input
		}
		i++
	}

	return inputWithMaxSubs, nil
}

func (r *RequestDeduplicator) calculateInput(ctx context.Context, input int) (int, error) {
	result, err := r.container.Calculate(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("cannot calculate input %d: %w", input, err)
	}

	return result, nil
}

// Close stops calculation loop and the container.
func (r *RequestDeduplicator) Close() error {
	r.closeLoopFn()
	err := r.container.Close()
	if err != nil {
		return fmt.Errorf("cannot shutdown container: %w", err)
	}

	r.l.Infof("deduplicator %d closed", r.seed)
	return nil
}

// subscription holds a channel with the result for a user.
type subscription struct {
	resultCh chan int
}

func newSubscription() *subscription {
	return &subscription{
		resultCh: make(chan int),
	}
}

func (s *subscription) close() {
	close(s.resultCh)
}
func (r *RequestDeduplicator) subscribe(input, reqID int) *subscription {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub := newSubscription()
	if len(r.inputToSubsriptions[input]) == 0 {
		r.inputToSubsriptions[input] = make(map[int]*subscription)
	}

	r.inputToSubsriptions[input][reqID] = sub
	return sub
}

func (r *RequestDeduplicator) unsubscribe(input, reqID int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.inputToSubsriptions[input][reqID].close()
	delete(r.inputToSubsriptions[input], reqID)

	if len(r.inputToSubsriptions[input]) == 0 {
		delete(r.inputToSubsriptions, input)
		if r.curInput == input && r.cancelCurCalcFn != nil {
			r.cancelCurCalcFn()
		}
	}
}

func (r *RequestDeduplicator) unsubscribeAll(input int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, sub := range r.inputToSubsriptions[input] {
		sub.close()
	}
	delete(r.inputToSubsriptions, input)
	if r.curInput == input && r.cancelCurCalcFn != nil {
		r.cancelCurCalcFn()
	}
}

func (r *RequestDeduplicator) publish(input, result int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, sub := range r.inputToSubsriptions[input] {
		sub.resultCh <- result
		sub.close()
	}

	delete(r.inputToSubsriptions, input)
}
