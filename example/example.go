package main

import (
	"fmt"
	"time"

	"github.com/kehl-gopher/schedulr"
)

// simulate job being run
func sleepTask() {
	fmt.Println("starting job")
	time.Sleep(5 * time.Second)
	fmt.Println("finished sleeping ntoor")
}

func main() {
	task := schedulr.Task{
		Job: func() error {
			sleepTask()
			return nil
		},
	}
	sch, _ := schedulr.SchedulerInit()
	sch.AddJobs(task)

	// ctx, canceVl := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cance]l()
	sch.Start()

	fmt.Println("still running main")

	time.Sleep(7 * time.Second)
	sch.ShutDown()
}
