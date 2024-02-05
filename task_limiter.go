package tasklimiters

import (
	"context"
	"sync/atomic"
	"time"
)

const (
	StatusStarted = 0
	StatusRunning = 1
	StatusStopped = 2
)

type StatsFunc func(time.Time, time.Time, ...int)

type TaskLimiter struct {
	ctx            context.Context
	ws             int
	wt             time.Duration
	timer          *time.Timer
	allowedCounter atomic.Uint32
	errorCounter   atomic.Uint32
	processed      chan int
	pids           atomic.Value
	tokens         chan int
	ratio          float64
	sf             StatsFunc
	status         Status
}

// NewTaskLimiter returns a new instance of TaskLimiter
func NewTaskLimiter(c context.Context, ws int, wt time.Duration, statsFunc StatsFunc) *TaskLimiter {
	p := &TaskLimiter{
		ctx:            c,
		ws:             ws,
		wt:             wt,
		sf:             statsFunc,
		timer:          time.NewTimer(wt),
		processed:      make(chan int, ws),
		tokens:         make(chan int, ws),
		ratio:          float64(wt) / float64(ws),
		allowedCounter: atomic.Uint32{},
		errorCounter:   atomic.Uint32{},
		status:         Started,
	}
	return p
}

// This function starts the proccess of refresh tokens limited by the timer in the background.
func (p *TaskLimiter) Start() {
	if p.sf != nil {
		go p.getProcessed()
	}
	p.status = Running

	go func() {
		for {
			// fill the tokens channel
			for i := 0; i < p.ws-len(p.tokens); i++ {
				p.tokens <- i
				if i < p.ws-len(p.tokens)-1 {
					time.Sleep(time.Duration(p.ratio))
				}
			}

			var begin time.Time
			// If the stats function is defined then it will reset the stats and the beginning time
			if p.sf != nil {
				p.resetStats()
				begin = time.Now()
			}
			p.timer.Reset(p.wt)
			select {
			case <-p.ctx.Done():
				p.timer.Stop()
				close(p.processed)
				close(p.tokens)
				p.status = Stopped
				return
			case <-p.timer.C:
				// If the stats function is defined then it will call the stats function
				if p.sf != nil {
					end := time.Now()
					s := p.pids.Load().([]int)
					p.sf(begin, end, s...)
				}
			}
		}
	}()
}

// Read the processed channel and append the id to the pids slice
func (p *TaskLimiter) getProcessed() {
	for e := range p.processed {
		a := p.pids.Load().([]int)
		a = append(a, e)
		p.pids.Store(a)
	}
}

// Reset the stats
func (p *TaskLimiter) resetStats() {
	p.allowedCounter.Store(0)
	p.errorCounter.Store(0)

	p.pids.Store([]int{})
}

func (p *TaskLimiter) wait(c context.Context, id int) error {

	select {
	case <-p.tokens:
		if p.sf != nil {
			p.processed <- id
		}
		p.allowedCounter.Add(1)
		return nil
	case <-c.Done():
		p.errorCounter.Add(1)
		return c.Err()
	}
}

// This function wait for the limiter to allow the task to be processed.
// The channel returns nil if the task has been released to exec. Otherwise,
// the channel will return the error of the closed context. When the context
// is closed, it returns an error always.
func (p *TaskLimiter) Wait(c context.Context, id int) chan interface{} {
	done := make(chan interface{})
	go func() {
		done <- p.wait(c, id)
	}()
	return done
}

// Get status of the limiter
func (p *TaskLimiter) Status() Status {
	return p.status
}