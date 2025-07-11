package main

import (
	"fmt"
	"time"
)

// simulate job being run
func sleepTask() {
	fmt.Println("starting job")
	time.Sleep(5 * time.Second)
	fmt.Println("finished sleeping ntoor")
}

func sleepTask2() {
	fmt.Println("starting second job")
	time.Sleep(3 * time.Second)
	fmt.Println("finished second job")
}

func main() {
}
