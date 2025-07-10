package schedulr_test

import (
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kehl-gopher/schedulr"
)

func TestSchedulrStart(t *testing.T) {
	s, _ := schedulr.SchedulerInit()
	s.Start()
	defer s.ShutDown()

	done := make(chan struct{})

	job := func() error {
		fmt.Println("Hello from test job")
		close(done)
		return nil
	}

	task := schedulr.NewTask(job, "a job", time.Time{}, 0)
	s.AddJobs(task)

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("job did not run in time")
	}
}

func TestWriteToFileJob(t *testing.T) {
	s, _ := schedulr.SchedulerInit()
	s.Start()
	defer s.ShutDown()

	done := make(chan struct{})

	job := func() error {
		f, err := os.CreateTemp("", "jobtest-*.txt")
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString("ran at: " + time.Now().String())
		close(done)
		return err
	}
	task := schedulr.NewTask(job, "a job", time.Time{}, 0)
	s.AddJobs(task)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("file job did not finish")
	}
}

func TestComputePrimeJob(t *testing.T) {
	s, _ := schedulr.SchedulerInit()
	s.Start()
	defer s.ShutDown()

	done := make(chan struct{})

	job := func() error {
		count := 0
		for i := 2; i < 10000; i++ {
			isPrime := true
			for j := 2; j*j <= i; j++ {
				if i%j == 0 {
					isPrime = false
					break
				}
			}
			if isPrime {
				count++
			}
		}
		t.Log("prime count:", count)
		close(done)
		return nil
	}

	task := schedulr.NewTask(job, "a job", time.Time{}, 0)
	s.AddJobs(task)

	select {
	case <-done:
		t.Log("job completed on schedule")
	case <-time.After(2 * time.Second):
		t.Fatal("job timed out")
	}
}

func TestSchedulrScheduledAt(t *testing.T) {
	s, _ := schedulr.SchedulerInit()

	s.Start()
	defer s.ShutDown()

	done := make(chan struct{})
	job := func() error {
		t.Log("Scheduled job ran at", time.Now())
		close(done)
		return nil
	}
	runAt := time.Now().Add(1 * time.Second)

	task := schedulr.NewTask(job, "a job", runAt, 0)
	s.ScheduledAt(task)

	select {
	case <-done:
		t.Log("job completed on schedule")
	case <-time.After(2 * time.Second):
		t.Fatal("❌ scheduled job did not run in time")
	}
}

func TestSchedulrScheduledTck(t *testing.T) {
	s, _ := schedulr.SchedulerInit()

	s.Start()
	defer s.ShutDown()

	var count int32
	job := func() error {
		atomic.AddInt32(&count, 1)
		t.Log("interval job ran")
		return nil
	}

	task := schedulr.NewTask(job, "a job", time.Time{}, 300*time.Millisecond)
	s.SheduleInterval(task)

	// Wait up to 1.2s (expect at least 3 runs)
	time.Sleep(1250 * time.Millisecond)

	if atomic.LoadInt32(&count) < 3 {
		t.Fatalf("expected at least 3 runs, got %d", count)
	}
	t.Logf("interval job ran %d times", count)
}
func TestDelayedJob(t *testing.T) {
	s, _ := schedulr.SchedulerInit()
	s.Start()
	defer s.ShutDown()

	done := make(chan struct{})

	job := func() error {
		fmt.Println("delayed job ran at", time.Now())
		close(done)
		return nil
	}

	task := schedulr.NewTask(job, "a job", time.Time{}, 0)
	go func() {
		time.Sleep(1 * time.Second)
		s.AddJobs(task)
	}()

	select {
	case <-done:
		t.Log("job completed on schedule")
	case <-time.After(2 * time.Second):
		t.Fatal("delayed job did not run")
	}
}

// test error jobs
// func TestFlakyJob(t *testing.T) {
// 	s, _ := schedulr.SchedulerInit()
// 	s.Start()
// 	defer s.ShutDown()

// 	done := make(chan struct{})

// 	job := func() error {
// 		defer close(done)
// 		if rand.Intn(2) == 0 {
// 			return fmt.Errorf("flaky failure")
// 		}
// 		fmt.Println("job succeeded")
// 		return nil
// 	}

// 	s.AddJobs(schedulr.Task{Job: job})

// 	select {
// 	case <-done:
// 	case <-time.After(2 * time.Second):
// 		t.Fatal("flaky job did not return")
// 	}
// }
