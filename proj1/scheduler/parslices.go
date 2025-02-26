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
	"time"

	"proj1/png"
)

// RunParallelSlice processes images one at a time with intra-image parallelism
func RunParallelSlices(config Config) {
	var tasks []ImageTask
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

	for _, task := range tasks {
		processImageSlice(task, numThreads)
	}

}

// processImageSlice handles one image with parallel effect processing
func processImageSlice(task ImageTask, numThreads int) {
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
			img.ParaSharpen(threads)
		case "E":
			img.ParaEdgeDetection(threads)
		case "B":
			img.ParaBlur(threads)
		case "G":
			img.ParaGrayscale(threads)
		default:
			panic("unknown effect")
		}

		if i < len(effects)-1 {
			img.SwapBuffers()
		}
	}
}
