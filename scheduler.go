package schedulr

import (
	"context"
	"fmt"
	"time"
)

type JobFunc func() error

type Scheduler interface {
	// start a task to run immediately
	Start()

	// shutdown scheduler and throw error if not shutting down
	ShutDown() error

	// add jobs to scheduler channels for task
	AddJobs(task task)

	// stop task preferable use with defer
	Stop()

	// Schedule a task to run at a specific time
	ScheduledAt(t task)
	// Schedule a task to run every X interval
	SheduleInterval(t task) error
}

type schedulr struct {
	shutDownCtx  context.Context
	jobs         chan task
	runAt        chan time.Timer    // run schedulr at the specific time provided
	delay        chan time.Timer    // delay schedulr run at a specific time
	tickJobs     chan time.Duration // run schedulr every minute, seconds or daily
	stopTime     chan time.Time
	cancelJobCtx context.Context // cancel jobs within context timeout
	cancelJobs   context.CancelFunc
	// cancelJobId  string // cancel job using job id
	cancel chan struct{}
}

type task struct {
	id       string
	Job      JobFunc
	tag     string
	runAt    time.Time     // schedule when task should run...
	interval time.Duration // run task every specified interval
}


func NewTask(job JobFunc, tag string, runAt time.Time, interval time.Duration) task{
	return task{
		id: "", 
		Job: job,
		tag: tag,
		runAt: runAt,
		interval: interval,
	}
}
// instantiate scheduler
func SchedulerInit() (Scheduler, error) {
	ctx, cancel := context.WithCancel(context.Background())
	sch := &schedulr{
		jobs:         make(chan task, 100),
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

// start static number of workers to run task for now
func (s *schedulr) Start() {
	for i := 0; i < 5; i++ {
		go func(workerId int) {
			for {
				select {
				case t := <-s.jobs:
					//TODO: handle error properly... with channels
					if err := t.Job(); err != nil {
						fmt.Println(err.Error()) 
					}
				case <-s.cancel:
					return
				}
			}
		}(i)
	}
}

// add user provided task to chan
func (s *schedulr) AddJobs(task task) {
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

func (s *schedulr) ScheduledAt(t task) {
	go func() {
		<-time.After(time.Until(t.runAt))
		s.jobs <- t
	}()
}

func (s *schedulr) SheduleInterval(t task) error {
	var err error
	tick := time.NewTicker(t.interval)
	go func() {
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				if err := t.Job(); err != nil {
					fmt.Println(err.Error())
				}
			case <-s.cancel:
				return
			}
		}
	}()
	return err
}
