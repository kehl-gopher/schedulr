package schedulr

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type JobFunc func() error

type Scheduler interface {
	Start()
	ShutDown() error
	ScheduleOnce(job JobFunc, at time.Time) (string, error)
	ScheduleRecurring(job JobFunc, interval time.Duration) (string, error)
	Submit(t task) string
	Cancel(id string) error
}

type schedulr struct {
	schedulrLock *sync.Mutex
	tasks        map[string]task
	jobQueue     chan task
	stopChan     chan struct{}
	wg           *sync.WaitGroup
}

type task struct {
	id          string
	job         JobFunc
	taskTimeOut time.Duration
	cancel      context.CancelFunc
}

func SchedulerInit() Scheduler {
	sch := &schedulr{
		schedulrLock: new(sync.Mutex),
		tasks:        make(map[string]task),
		jobQueue:     make(chan task, 100),
		stopChan:     make(chan struct{}),
		wg:           new(sync.WaitGroup),
	}

	return sch
}

func (s *schedulr) Start() {
	for i := 1; i <= 5; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				select {
				case task, ok := <-s.jobQueue:
					if ok {
						var done = make(chan error)
						ctx, cancel := context.WithTimeout(context.Background(), task.taskTimeOut)
						defer cancel()
						go func() {
							done <- task.job()
						}()
						select {
						case <-ctx.Done():
							fmt.Printf("task with %s timeout", task.id)
						case err := <-done:
							if err != nil {
								fmt.Printf("task id %s: failed to run task: %s", task.id, err.Error())
							}
						}
					}
				case <-s.stopChan:
					fmt.Println("scheduled task stop")
					return
				}
			}
		}()
	}
}

func NewTask(timeOut time.Duration, job JobFunc) task {
	return task{id: generateId(), taskTimeOut: timeOut, job: job}
}

func (s *schedulr) ScheduleOnce(job JobFunc, at time.Time) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	id := generateId()

	t := task{
		id:     id,
		job:    job,
		cancel: cancel,
	}

	delay := time.Until(at)

	s.schedulrLock.Lock()
	s.tasks[id] = t
	s.schedulrLock.Unlock()

	after := time.NewTimer(delay)
	go func() {
		select {
		case <-after.C:
			s.Submit(t)
		case <-ctx.Done():
			after.Stop()
		}
	}()
	return id, nil
}

func (s *schedulr) Submit(t task) string {
	s.jobQueue <- t
	return t.id
}

func (sch *schedulr) ScheduleRecurring(job JobFunc, interval time.Duration) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	id := generateId()

	t := task{
		id:     id,
		job:    job,
		cancel: cancel,
	}
	sch.schedulrLock.Lock()
	sch.tasks[id] = t
	sch.schedulrLock.Unlock()

	tick := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-tick.C:
				sch.Submit(t)
			case <-ctx.Done():
				tick.Stop()
				return
			}
		}
	}()
	return id, nil
}

func (sch *schedulr) Cancel(id string) error {
	sch.schedulrLock.Lock()
	defer sch.schedulrLock.Unlock()

	task, ok := sch.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}
	task.cancel()
	delete(sch.tasks, id)
	return nil
}

func (sch *schedulr) ShutDown() error {
	close(sch.stopChan)
	sch.wg.Wait()
	sch.schedulrLock.Lock()
	for _, t := range sch.tasks {
		if t.cancel != nil {
			t.cancel()
		}
	}
	sch.schedulrLock.Unlock()
	close(sch.jobQueue)
	return nil
}
