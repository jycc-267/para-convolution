package scheduler

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// TASLock implements test-and-set lock
type TASLock struct {
	state int32 // 0 = unlocked, 1 = locked
}

func (l *TASLock) Lock() {
	for !atomic.CompareAndSwapInt32(&l.state, 0, 1) {
		// spin until acquired
	}
}

func (l *TASLock) Unlock() {
	atomic.StoreInt32(&l.state, 0)
}

func RunParallelFiles(config Config) {
	var tasks []ImageTask // we are actually working with a header pointing to the underlying array, a length, and a capacity
	dataDirs := strings.Split(config.DataDirs, "+")

	for _, dataDir := range dataDirs {
		// open effects.txt file
		effectsPath := "../data/effects.txt"
		effectsFile, err := os.Open(effectsPath)
		if err != nil {
			panic(err)
		}
		defer effectsFile.Close()

		// create JSON decoder for effects.txt
		// os.File type implements the io.Reader interface through its Read() method
		reader := json.NewDecoder(effectsFile)
		for {
			var task ImageTask
			if err := reader.Decode(&task); err != nil {
				break
			}
			task.DataDir = dataDir
			tasks = append(tasks, task)
		}
	}

	if len(tasks) == 0 {
		return
	}

	// #Goroutines = min(#Threads specified in the command line, #Images in the queue)
	numThreads := config.ThreadCount
	if numThreads > len(tasks) {
		numThreads = len(tasks)
	}

	// initialize TASLock for critical section
	// Go routines should run until all tasks from the queue are processed
	var tasLock TASLock
	var wg sync.WaitGroup
	wg.Add(numThreads)

	// start goroutines
	for i := 0; i < numThreads; i++ {

		go func() {
			defer wg.Done()
			for {
				////// Critical Section: each goroutine attempts to lock queue -> take task -> unlock //////
				//////
				// repeatedly checks for available tasks in the queue
				tasLock.Lock()

				// if the queue is empty, the (last) goroutine terminated
				if len(tasks) == 0 {
					tasLock.Unlock()
					return
				}

				// dequeue a task
				task := tasks[0]
				tasks = tasks[1:]
				tasLock.Unlock()
				//////
				////// Critical Section: lock queue -> take task -> unlock //////

				// process task: see sequential.go
				processImageTask(task)
			}
		}()
	}

	// wait until all goroutines have terminated
	wg.Wait()

}
