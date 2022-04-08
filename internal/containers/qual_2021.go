package containers

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Qual is a container that start docker container for a qualification,
// wait for initialization, send calculations to it and stops it after the
// given time.
type Qual struct {
	l       *zap.SugaredLogger
	d       *docker
	port    int
	client  *http.Client
	closeFn context.CancelFunc

	stateMu         sync.Mutex
	state           state
	lastCalculation time.Time
}

type state int

const (
	initState state = iota
	readyState
	stoppedState
)

// NewQual creates new Qual.
func NewQual(l *zap.SugaredLogger, seed int) (*Qual, error) {
	port, err := getFreePort()
	if err != nil {
		return nil, fmt.Errorf("cannot get free port: %w", err)
	}

	name := fmt.Sprintf("qual_%d_seed_%d", port, seed)
	stopAfter := 120 * time.Second
	ctx, cancelFn := context.WithCancel(context.Background())

	q := &Qual{
		l: l,
		d: newDocker(
			l.Named("d"),
			"quay.io/milaboratory/qual-2021-devops-server", "latest",
			port,
			name,
			[][]string{{"SEED", strconv.Itoa(seed)}},
		),
		client:  &http.Client{Timeout: 130 * time.Second},
		port:    port,
		closeFn: cancelFn,

		stateMu: sync.Mutex{},
		state:   initState,
	}

	go q.stopAfter(ctx, stopAfter)

	return q, nil
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}

	err = l.Close()
	if err != nil {
		return 0, err
	}

	return l.Addr().(*net.TCPAddr).Port, nil
}

// Calculate starts the container if it is stopped, and send a request for a calculation.
func (q *Qual) Calculate(ctx context.Context, input int) (int, error) {
	q.stateMu.Lock()
	if q.state != readyState {
		err := q.start()
		if err != nil {
			q.stateMu.Unlock()
			return 0, fmt.Errorf("cannot start a container: %w", err)
		}
		q.state = readyState
	}
	q.stateMu.Unlock()

	q.l.Infof("get request for input %d", input)
	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/calculate/%d", q.port, input), nil)
	if err != nil {
		return 0, fmt.Errorf("cannot create request: %w", err)
	}
	req = req.WithContext(ctx)

	resp, err := q.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("cannot do request: %w", err)
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("cannot read body: %w", err)
	}

	result, err := strconv.Atoi(string(bytes))
	if err != nil {
		return 0, fmt.Errorf("cannot parse body %q: %w", string(bytes), err)
	}

	q.lastCalculation = time.Now()
	return result, nil
}

// Close closes underlying Docker container and stops the lifecycle loop.
func (q *Qual) Close() error {
	q.l.Debugf("try to stop %s", q.d.name)

	q.closeFn()

	err := q.d.stop()
	if err != nil {
		return fmt.Errorf("cannot stop docker container: %w", err)
	}

	q.l.Infof("qual %s closed", q.d.name)
	return nil
}

// start runs the container and waits for full initialization.
func (q *Qual) start() error {
	q.lastCalculation = time.Now()

	err := q.d.run()
	if err != nil {
		return fmt.Errorf("cannot run docker container: %w", err)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			resp, err := q.client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", q.port))
			if err != nil {
				q.l.Debugf("%s: /health: %s", q.d.name, err.Error())
				break
			}

			if resp.StatusCode == 200 {
				q.l.Debugf("%s was initialized.", q.d.name)
				return nil
			}

		case <-time.After(130 * time.Second):
			return fmt.Errorf("container was intializing for too long")
		}
	}
}

// stopAfter waits given time after last calculation and stops the container.
func (q *Qual) stopAfter(ctx context.Context, after time.Duration) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Since(q.lastCalculation) > after {
				q.stateMu.Lock()
				if q.state == readyState {
					q.l.Debugf("try to stop in loop %s", q.d.name)
					err := q.d.stop()
					if err != nil {
						q.stateMu.Unlock()
						q.l.Errorf("cannot stop the container: %s", err.Error())
						continue
					}
				}

				q.state = stoppedState
				q.stateMu.Unlock()
			}

		case <-ctx.Done():
			return
		}
	}
}
