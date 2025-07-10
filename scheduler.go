package schedulr

import (
	"context"
	"fmt"
	"time"
)

type JobFunc func() error

type Scheduler interface {
	// run jobs that are been scheduled...
	Start()
	ShutDown() error
	AddJobs(task Task)
	Stop()
}

type schedulr struct {
	shutDownCtx  context.Context
	jobs         chan Task
	runAt        chan time.Timer    // run job at the specific time provided
	delay        chan time.Timer    // delay job run at a specific time
	tickJobs     chan time.Duration // run job every minute, seconds or daily
	stopTime     chan time.Time
	cancelJobCtx context.Context // cancel jobs within context timeout
	cancelJobs   context.CancelFunc
	cancelJobId  string // cancel job using job id
	cancel       chan struct{}
}

type Task struct {
	id   string
	Job  func() error
	Tags string
}

// instantiate scheduler
func SchedulerInit() (Scheduler, error) {
	ctx, cancel := context.WithCancel(context.Background())
	sch := &schedulr{
		jobs:         make(chan Task, 100),
		shutDownCtx:  ctx,
		runAt:        make(chan time.Timer),
		delay:        make(chan time.Timer),
		stopTime:     make(chan time.Time),
		cancelJobCtx: ctx,
		cancelJobs:   cancel,
		cancel:       make(chan struct{}),
	}
	// start jobs scheduler
	return sch, nil
}

// start number of workers to run task
func (s *schedulr) Start() {
	for i := 0; i < 5; i++ {
		go func(workerId int) {
			for {
				select {
				case t := <-s.jobs:
					err := t.Job()
					if err != nil {
						fmt.Println("error whoops")
					}
					fmt.Println("finished run task...")
				case <-s.cancel:
					return
				}
			}
		}(i)
	}
}

// add user provided task to chan
func (s *schedulr) AddJobs(task Task) {
	s.jobs <- task
}

// handle stop jobs
func (s *schedulr) Stop() {

}

// TODO: handle shutdown error
func (s *schedulr) ShutDown() error {
	select {
	case <-s.shutDownCtx.Done():
		close(s.cancel)
	default:
	}
	return nil
}
