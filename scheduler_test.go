package schedulr_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kehl-gopher/schedulr"
)

func TestSubmitAndExecuteTask(t *testing.T) {
	sch := schedulr.SchedulerInit()
	defer sch.ShutDown()

	done := make(chan bool)
	task := schedulr.NewTask(2*time.Second, func() error {
		fmt.Println("[Task Simple] Running")
		done <- true
		return nil
	}, 5)

	sch.Submit(task)

	select {
	case <-done:
		fmt.Println("[Task Simple] Completed")
	case <-time.After(10 * time.Second):
		t.Error("task did not complete in time")
	}
}

func TestTaskTimeout(t *testing.T) {
	sch := schedulr.SchedulerInit()
	defer sch.ShutDown()

	task := schedulr.NewTask(500*time.Millisecond, func() error {
		time.Sleep(2 * time.Second)
		return nil
	}, 5)

	sch.Submit(task)

	time.Sleep(3 * time.Second)
}

func TestScheduleOnce(t *testing.T) {
	sch := schedulr.SchedulerInit()
	defer sch.ShutDown()

	done := make(chan bool)

	_, err := sch.ScheduleOnce(func() error {
		fmt.Println("[ScheduledOnce] Executed")
		done <- true
		return nil
	}, time.Now().Add(1*time.Second), 3)

	if err != nil {
		t.Fatalf("failed to schedule once: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("scheduled-once task did not execute")
	}
}

func TestScheduleRecurringAndCancel(t *testing.T) {
	sch := schedulr.SchedulerInit()
	defer sch.ShutDown()

	count := 0
	done := make(chan bool)

	id, err := sch.ScheduleRecurring(func() error {
		count++
		fmt.Printf("[Recurring] Run #%d\n", count)
		if count >= 3 {
			done <- true
		}
		return nil
	}, 1*time.Second, 2)

	if err != nil {
		t.Fatalf("failed to schedule recurring: %v", err)
	}

	select {
	case <-done:
		_ = sch.Cancel(id)
	case <-time.After(5 * time.Second):
		t.Error("recurring task did not complete expected runs")
	}
}

func TestCancelScheduledTask(t *testing.T) {
	sch := schedulr.SchedulerInit()
	defer sch.ShutDown()

	executed := false

	id, err := sch.ScheduleOnce(func() error {
		executed = true
		return nil
	}, time.Now().Add(2*time.Second), 1)

	if err != nil {
		t.Fatalf("failed to schedule task: %v", err)
	}

	err = sch.Cancel(id)
	if err != nil {
		t.Fatalf("failed to cancel task: %v", err)
	}

	time.Sleep(3 * time.Second)

	if executed {
		t.Error("task executed after being canceled")
	}
}

func TestPriorityTaskExecutionOrder(t *testing.T) {
	executedOrder := []string{}

	createJob := func(id string) schedulr.JobFunc {
		return func() error {
			executedOrder = append(executedOrder, id)
			return nil
		}
	}

	sch := schedulr.SchedulerInit()
	defer sch.ShutDown()

	taskLow := schedulr.NewTask(2*time.Second, createJob("low"), 1)
	taskMid := schedulr.NewTask(2*time.Second, createJob("mid"), 5)
	taskHigh := schedulr.NewTask(2*time.Second, createJob("high"), 10)

	sch.Submit(taskLow)
	sch.Submit(taskMid)
	sch.Submit(taskHigh)

	time.Sleep(6 * time.Second)

	expected := []string{"high", "mid", "low"}
	if len(executedOrder) != 3 {
		t.Fatalf("expected 3 tasks to run, got %d", len(executedOrder))
	}
	for i := range expected {
		if executedOrder[i] != expected[i] {
			t.Errorf("expected task %s at position %d, got %s", expected[i], i, executedOrder[i])
		}
	}
}

func TestRecurringTaskIsNonBlocking(t *testing.T) {
	sch := schedulr.SchedulerInit()
	defer sch.ShutDown()

	var mu sync.Mutex
	executions := 0
	done := make(chan struct{})

	longJob := func() error {
		mu.Lock()
		executions++
		fmt.Printf("[LongRecurring] Execution #%d\n", executions)
		mu.Unlock()
		time.Sleep(3 * time.Second)
		return nil
	}

	_, err := sch.ScheduleRecurring(longJob, 2*time.Second, 2)
	if err != nil {
		t.Fatalf("Failed to schedule recurring task: %v", err)
	}

	fastJob := func() error {
		fmt.Println("[FastTask] Executed")
		close(done)
		return nil
	}

	_, err = sch.ScheduleOnce(fastJob, time.Now().Add(1*time.Second), 5)
	if err != nil {
		t.Fatalf("Failed to schedule one-time task: %v", err)
	}

	select {
	case <-done:
	case <-time.After(6 * time.Second):
		t.Fatal("Recurring task blocked other tasks from running")
	}
}
