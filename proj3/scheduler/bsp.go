package scheduler

// iterate over the image queue, and for each image, split it into slices for parallel processing handled by goroutines
/*
For each effect:
	split image into slices
	each goroutine applies effect to their own slices
	wait for others to finish
	swap in and out buffers
*/

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"proj3/deque"
	"proj3/png"
)

// https://medium.com/@nathanbcrocker/building-a-multithreaded-work-stealing-task-scheduler-in-go-843861b878be
type Worker struct {
	id    int
	deque *deque.Deque
}

func NewWorker(id int) *Worker {
	return &Worker{
		id:    id,
		deque: deque.NewDeque(),
	}
}

func stealWork(thief *Worker, victims []*Worker) *png.ImageTask {
	for _, victim := range victims {
		if victim.id != thief.id {
			if task, ok := victim.deque.Steal(); ok {
				return task
			}
		}
	}
	return nil
}

// RunBSP processes images one at a time with intra-image parallelism
func RunBSP(config Config) {
	var tasks []png.ImageTask // queue
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
			var task png.ImageTask
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

	for _, task := range tasks {
		processImageBSP(task, numThreads)
	}

}

func RunBSPSteal(config Config) {

	dataDirs := strings.Split(config.DataDirs, "+")
	// #Goroutines = min(#Threads specified in the command line, #Images in the queue)
	numWorkers := config.ThreadCount
	workers := make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = NewWorker(i)
	}

	workerIndex := 0
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
			var task png.ImageTask
			if err := reader.Decode(&task); err != nil {
				break
			}
			task.DataDir = dataDir
			workers[workerIndex].deque.Push(&task)
			workerIndex = (workerIndex + 1) % numWorkers
		}
	}

	if len(workers) == 0 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for _, worker := range workers {
		go func(w *Worker) {
			defer wg.Done()
			for {
				task, ok := w.deque.Pop()
				if !ok {
					if task = stealWork(w, workers); task == nil {
						break
					}
				}
				processImageBSP(*task, numWorkers)
			}
		}(worker)
	}
	wg.Wait()
}

// processImageSlice handles one image with parallel effect processing
func processImageBSP(task png.ImageTask, numThreads int) {
	inPath := fmt.Sprintf("../data/in/%s/%s", task.DataDir, task.InPath)
	outPath := fmt.Sprintf("../data/out/%s_%s", task.DataDir, task.OutPath)

	img, err := png.Load(inPath)
	if err != nil {
		panic(err)
	}

	if len(task.Effects) > 0 {
		img.EffectsApplied = true
		start := time.Now()
		applyEffectsSliced(img, task.Effects, numThreads)
		end := time.Since(start).Seconds()
		fmt.Printf("parslices: %.2f\n", end)
	}
	if err := img.Save(outPath); err != nil {
		panic(err)
	}
}

// applyEffectsSliced applies effects with parallel slices
func applyEffectsSliced(img *png.Image, effects []string, threads int) {
	for i, effect := range effects {
		switch effect {
		case "S":
			img.BSPSharpen(threads)
		case "E":
			img.BSPEdgeDetection(threads)
		case "B":
			img.BSPBlur(threads)
		case "G":
			img.BSPGrayscale(threads)
		default:
			panic("unknown effect")
		}

		if i < len(effects)-1 {
			img.SwapBuffers()
		}
	}
}
