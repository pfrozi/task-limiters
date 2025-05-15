# task-limiter: A Golang rate limiter

[![CI State](https://github.com/pfrozi/task-limiters/actions/workflows/go_test.yml/badge.svg?branch=main&event=push)](https://github.com/pfrozi/task-limiters/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/pfrozi/task-limiters)](https://goreportcard.com/report/github.com/pfrozi/task-limiters)
[![Go Reference](https://pkg.go.dev/badge/github.com/pfrozi/task-limiters.svg)](https://pkg.go.dev/github.com/pfrozi/task-limiters)

 `task-limiter` is a package which limit the execution of goroutines in a specific frequency of time.

Semantic example:
> _Execute at most 4 goroutines in a 5 seconds interval._

## Quick Start

```sh
go get github.com/pfrozi/task-limiters
```

```golang
package main

import (
    "context"
    "sync"
    "testing"
    "time"

    "github.com/pfrozi/task-limiters"
)

func main() {

    ctx, cancel := context.WithCancel(context.Background())

    // Execute at most 4 goroutines in a 5 seconds interval
    ws := 4
    wt := 5*time.Second

    // status function to log the interval. Optional
    statsFunction := func(begin time.Time, end time.Time, pids ...int) {
        if len(pids) < ws {
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

        c, cancel := context.WithTimeout(context.Background(), wt)
        defer cancel()

        err := <-limiter.Wait(c, p)

        if err != nil {
            t.Errorf("Task %d timeouted", p)
        } else {
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
    wg.Wait()

    // release context
    cancel()
    time.Sleep(time.Second)
}
```

## Concepts

- **What is a [_rate limiter_](https://en.wikipedia.org/wiki/Rate_limiting)?** A "rate limiter" is a resource that controls how frequently some event is allowed to happen. In this package, the event is a goroutine. To avoid confusion with the network concept name (_rate limiting_), we think the _task limiter_ would make more sense.
- **Window Size (ws)**: Total of tasks allowed to happen in a specific period of time.
- **Window Time (wt)**: The frequency time in which the tasks are release
  > _ws tasks / wt time_
