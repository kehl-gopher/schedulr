package schedulr

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// --------------------------------------- task queue ---------------------------------

type taskQueue []*task

func (tsk *taskQueue) Push(ts interface{}) {
	t := ts.(*task)
	*tsk = append(*tsk, t)
}

func (tsk *taskQueue) Pop() interface{} {
	taskLen := tsk.Len()
	if taskLen == 0 {
		return nil
	}

	oldTsk := *tsk
	item := oldTsk[taskLen-1]
	oldTsk[taskLen-1] = nil
	*tsk = oldTsk[:taskLen-1]
	return item
}

func (tsk *taskQueue) Len() int {
	return len(*tsk)
}

func (tsk taskQueue) Less(i, j int) bool {
	return tsk[i].priority > tsk[j].priority
}

func (tsk taskQueue) Swap(i, j int) {
	tsk[i], tsk[j] = tsk[j], tsk[i]
}

// -------------------------------------- Job run ----------------------------------------
type JobFunc func() error

type Scheduler interface {
	scaleWorker()
	ShutDown() error
	ScheduleOnce(job JobFunc, at time.Time, priority int) (string, error)
	ScheduleRecurring(job JobFunc, interval time.Duration, priority int) (string, error)
	Submit(t task) string
	Cancel(id string) error
	dequeueAndRun()
	executeTask(n int)
}

type schedulr struct {
	schedulrLock  *sync.Mutex
	queueTask     taskQueue
	tasks         map[string]task
	jobQueue      chan task
	stopChan      chan struct{}
	currentWorker int32
	wg            *sync.WaitGroup
}

type task struct {
	id          string
	job         JobFunc
	taskTimeOut time.Duration
	priority    int
	cancel      context.CancelFunc
}

func SchedulerInit() Scheduler {
	sch := &schedulr{
		schedulrLock:  new(sync.Mutex),
		tasks:         make(map[string]task),
		jobQueue:      make(chan task, 100),
		queueTask:     make(taskQueue, 0),
		stopChan:      make(chan struct{}),
		currentWorker: 0,
		wg:            new(sync.WaitGroup),
	}
	go sch.dequeueAndRun()
	go sch.scaleWorker()
	return sch
}

func (s *schedulr) dequeueAndRun() {
	for {
		s.schedulrLock.Lock()
		if s.queueTask.Len() > 0 {
			task := heap.Pop(&s.queueTask).(*task)
			s.schedulrLock.Unlock()
			s.jobQueue <- *task
		} else {
			s.schedulrLock.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
		select {
		case <-s.stopChan:
			return
		default:
		}
	}
}

func (sch *schedulr) scaleWorker() {
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			sch.schedulrLock.Lock()
			workers := calculateWorkerPertTask(sch.queueTask.Len())
			if workers > int(sch.currentWorker) {
				w := workers - int(sch.currentWorker)
				sch.schedulrLock.Unlock()
				sch.executeTask(w)
			} else {
				sch.schedulrLock.Unlock()
			}
		case <-sch.stopChan:
			return
		}
	}
}

func (sch *schedulr) executeTask(n int) {
	for i := 0; i < n; i++ {
		atomic.AddInt32(&sch.currentWorker, 1)
		sch.wg.Add(1)
		go func() {
			defer sch.wg.Done()
			for {
				select {
				case task, ok := <-sch.jobQueue:
					if ok {
						fmt.Printf("[Task %s] Starting with priority %d\n", task.id, task.priority)
						var done = make(chan error)
						ctx, cancel := context.WithTimeout(context.Background(), task.taskTimeOut)
						defer cancel()
						go func() {
							done <- task.job()
						}()
						select {
						case err := <-done:
							if err != nil {
								fmt.Printf("[Task %s] failed executing err: %s", task.id, err.Error())
							}
						case <-ctx.Done():
							// will handle task retries later on
							fmt.Printf("task timeout before finishing execution")
						}
					}
				case <-sch.stopChan:
					return
				}
			}
		}()
	}
}

func NewTask(timeOut time.Duration, job JobFunc, priority int) task {
	return task{id: generateId(), taskTimeOut: timeOut, job: job, priority: priority}
}

func (s *schedulr) ScheduleOnce(job JobFunc, at time.Time, priority int) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	id := generateId()

	t := task{
		id:     id,
		job:    job,
		cancel: cancel,
		priority: priority,
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
	s.schedulrLock.Lock()
	heap.Push(&s.queueTask, &t)
	s.schedulrLock.Unlock()
	return t.id
}

func (sch *schedulr) ScheduleRecurring(job JobFunc, interval time.Duration, priority int) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	id := generateId()

	t := task{
		id:       id,
		job:      job,
		cancel:   cancel,
		priority: priority,
	}
	sch.schedulrLock.Lock()
	sch.tasks[id] = t
	sch.schedulrLock.Unlock()

	tick := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-tick.C:
				newTask := NewTask(interval, job, priority) // create an instance of each task and add to queue
				sch.Submit(newTask)
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

	task, exist := sch.tasks[id]
	if !exist {
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
