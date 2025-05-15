package tasklimiters

import (
	"context"
	"sync"
	"testing"
	"time"
)

// This function will test the NewTaskLimiter function.
func TestNewTaskLimiter(t *testing.T) {
	type args struct {
		c         context.Context
		ws        int
		wt        time.Duration
		statsFunc StatsFunc
	}
	tests := []struct {
		name string
		args args
		want *TaskLimiter
	}{
		{
			name: "Test with valid inputs",
			args: args{
				c:         context.Background(),
				ws:        10,
				wt:        time.Second,
				statsFunc: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTaskLimiter(tt.args.c, tt.args.ws, tt.args.wt, tt.args.statsFunc)
			if got == nil || got.Status() != Started {
				t.Errorf("NewTaskLimiter() = Error to initialize - not started")
			}
		})
	}
}

// This function will test the Start function.
// The test ueses a limited context by the cancel call,
// that will stop the Tasklimiter execution at the end.
func TestTaskLimiter_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	limiter := NewTaskLimiter(ctx, 10, time.Second, nil)
	if limiter == nil || limiter.Status() != Started {
		t.Errorf("NewTaskLimiter() = Error to initialize - isn't started")
	}

	go limiter.Start()
	time.Sleep(time.Second)
	if limiter.Status() != Running {
		t.Errorf("NewTaskLimiter() = Error after start - isn't running")
	}

	cancel()
	time.Sleep(time.Second)
	if limiter.Status() != Stopped {
		t.Errorf("NewTaskLimiter() = Error after cancel() - isn't stopped")
	}
}

// Test the Start function with a StatsFunction.
// The test uses a limited context by the cancel call,
// that will stop the Tasklimiter execution at the end.
func TestTaskLimiter_StartWithStats(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	latency := time.Millisecond

	statsFunction := func(begin time.Time, end time.Time, pids ...int) {
		if end.Sub(begin) < time.Second {
			t.Logf("StatsFunc() = %s to %s - duration < 1s: %v",
				begin.Format("15:04:05.0"),
				end.Format("15:04:05.0"),
				end.Sub(begin))
		}
		if end.Sub(begin) > time.Second && end.Sub(begin)-time.Second > latency {
			t.Logf("StatsFunc() = %s to %s - duration > 1s: %v",
				begin.Format("15:04:05.0"),
				end.Format("15:04:05.0"),
				end.Sub(begin))
		}

		if len(pids) > 0 {
			t.Logf("StatsFunc() = %s to %s - pids > 0: %v",
				begin.Format("15:04:05.0"),
				end.Format("15:04:05.0"),
				pids)
		}
	}

	limiter := NewTaskLimiter(ctx, 10, time.Second, statsFunction)

	go limiter.Start()
	time.Sleep(time.Second * 2)
	cancel()
	time.Sleep(time.Second)
}

// Test the Wait function without a "timeouted" context.
func TestTaskLimiter_WaitWithoutTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ws := 1
	wt := time.Second

	statsFunction := func(begin time.Time, end time.Time, pids ...int) {
		if len(pids) < ws {
			t.Logf("StatsFunc() = %s to %s - pids < ws: %v",
				begin.Format("15:04:05.0"),
				end.Format("15:04:05.0"),
				pids)
		}
	}

	limiter := NewTaskLimiter(ctx, ws, time.Second, statsFunction)
	wg := &sync.WaitGroup{}

	process := func(p int) {
		wg.Add(1)
		defer wg.Done()

		c, cancel := context.WithTimeout(context.Background(), 2*wt)
		defer cancel()

		err := <-limiter.Wait(c, p)

		if err != nil {
			t.Errorf("Task %d timeouted", p)
		} else {
			return
		}
	}

	go limiter.Start()

	go process(1)
	go process(2)
	time.Sleep(time.Second * 2)
	wg.Wait()
	cancel()
	time.Sleep(time.Second)
}

// Test the Wait function with a "timeouted" context.
func TestTaskLimiter_WaitWithTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ws := 1
	wt := time.Second

	statsFunction := func(begin time.Time, end time.Time, pids ...int) {
		if len(pids) < ws {
			t.Logf("StatsFunc() = %s to %s - pids < ws: %v",
				begin.Format("15:04:05.0"),
				end.Format("15:04:05.0"),
				pids)
		}
	}

	limiter := NewTaskLimiter(ctx, ws, time.Second, statsFunction)
	wg := &sync.WaitGroup{}

	process := func(p int) {
		wg.Add(1)
		defer wg.Done()

		c, cancel := context.WithTimeout(context.Background(), 1*wt)
		defer cancel()

		err := <-limiter.Wait(c, p)

		if err != nil {
			if p == 3 {
				t.Logf("Task %d is correctly timeouted: %v", p, err)
			} else {
				t.Errorf("Task %d is wrongly timeouted: %v", p, err)
				return
			}
		} else {
			if p == 3 {
				t.Errorf("Task %d isn't timeouted", p)
			} else {
				return
			}
		}
	}

	go limiter.Start()

	go process(1)
	time.Sleep(time.Millisecond * 200)
	go process(2)
	time.Sleep(time.Millisecond * 200)
	go process(3)
	wg.Wait()
	cancel()
	time.Sleep(time.Second)

}
func TestTaskLimiter_WithPressure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Execute at most 4 goroutines in a 5 seconds interval
	ws := 2
	wt := 1 * time.Second

	// status function to log the interval. Optional
	statsFunction := func(begin time.Time, end time.Time, pids ...int) {
		if len(pids) > 0 {
			t.Logf("StatsFunc() = %s to %s - pids < ws: %v",
				begin.Format("15:04:05.0"),
				end.Format("15:04:05.0"),
				pids)
		}
	}

	limiter := NewTaskLimiter(ctx, ws, wt, statsFunction)
	wg := &sync.WaitGroup{}

	process := func(p int) {
		wg.Add(1)
		defer wg.Done()

		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := <-limiter.Wait(c, p)

		if err != nil {
			t.Errorf("Task %d timeouted", p)
		} else {
			t.Logf("Task %d processed!", p)
			time.Sleep(time.Second)
			return
		}
	}

	go limiter.Start()

	go process(0)
	go process(1)
	go process(2)
	go process(3)
	go process(4)
	go process(5)
	go process(6)
	go process(7)
	go process(8)
	go process(9)

	// waiting until all processes are done
	time.Sleep(time.Second)
	wg.Wait()

	// release context
	time.Sleep(5 * time.Second)
	cancel()
}

// func TestNewTaskLimiter(t *testing.T) {
// 	type args struct {
// 		c         context.Context
// 		ws        int
// 		wt        time.Duration
// 		statsFunc StatsFunc
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want *TaskLimiter
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := NewTaskLimiter(tt.args.c, tt.args.ws, tt.args.wt, tt.args.statsFunc); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("NewTaskLimiter() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestTaskLimiter_Start(t *testing.T) {
// 	tests := []struct {
// 		name string
// 		p    *TaskLimiter
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			tt.p.Start()
// 		})
// 	}
// }

// func TestTaskLimiter_getProcessed(t *testing.T) {
// 	tests := []struct {
// 		name string
// 		p    *TaskLimiter
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			tt.p.getProcessed()
// 		})
// 	}
// }

// func TestTaskLimiter_resetStats(t *testing.T) {
// 	tests := []struct {
// 		name string
// 		p    *TaskLimiter
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			tt.p.resetStats()
// 		})
// 	}
// }

// func TestTaskLimiter_wait(t *testing.T) {
// 	type args struct {
// 		c  context.Context
// 		id int
// 	}
// 	tests := []struct {
// 		name    string
// 		p       *TaskLimiter
// 		args    args
// 		wantErr bool
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if err := tt.p.wait(tt.args.c, tt.args.id); (err != nil) != tt.wantErr {
// 				t.Errorf("TaskLimiter.wait() error = %v, wantErr %v", err, tt.wantErr)
// 			}
// 		})
// 	}
// }

// func TestTaskLimiter_Wait(t *testing.T) {
// 	type args struct {
// 		c  context.Context
// 		id int
// 	}
// 	tests := []struct {
// 		name string
// 		p    *TaskLimiter
// 		args args
// 		want chan interface{}
// 	}{
// 		// TODO: Add test cases.
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := tt.p.Wait(tt.args.c, tt.args.id); !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("TaskLimiter.Wait() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
