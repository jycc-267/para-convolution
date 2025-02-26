package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"proj1/png"
)

// ImageTask is used to unmarshal each JSON object from effects.txt
// https://www.youtube.com/watch?v=hzSkpuL2I_Y&t=695s
// https://stackoverflow.com/questions/21197239/decoding-json-using-json-unmarshal-vs-json-newdecoder-decode
type ImageTask struct {
	InPath  string   `json:"inPath"`
	OutPath string   `json:"outPath"`
	Effects []string `json:"effects"`
	DataDir string
}

func RunSequential(config Config) {

	// separate directory names
	dataDirs := strings.Split(config.DataDirs, "+")

	// process each data directory
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

		// an infinite loop to read JSON objects from effects.txt until the end is reached
		for {
			// declare a variable to hold each image task read from the JSON
			var task ImageTask

			/*
			   Decode next JSON entry
			   The decoder reads the entire JSON object and maintains its position in the file
			   subsequent calls to Decode() will start from where the previous call left off
			   When the decoder reaches the end of the file, it returns an io.EOF error to break the loop
			*/
			if err := reader.Decode(&task); err != nil {
				break
			}

			// process current ImageTask
			task.DataDir = dataDir
			processImageTask(task)
		}
	}
}

// Refer to PROJ1/sample/sample.go
func processImageTask(task ImageTask) {
	start := time.Now()
	inPath := fmt.Sprintf("../data/in/%s/%s", task.DataDir, task.InPath)
	outPath := fmt.Sprintf("../data/out/%s_%s", task.DataDir, task.OutPath)

	img, err := png.Load(inPath)
	if err != nil {
		panic(err)
	}
	if len(task.Effects) > 0 {
		img.EffectsApplied = true

		applyEffects(img, task.Effects)

	}
	if err := img.Save(outPath); err != nil {
		panic(err)
	}
	end := time.Since(start).Seconds()
	fmt.Printf("parfiles: %.2f\n", end)
}

func applyEffects(img *png.Image, effects []string) {

	for i, effect := range effects {
		switch effect {
		case "S":
			img.Sharpen()
		case "E":
			img.EdgeDetection()
		case "B":
			img.Blur()
		case "G":
			img.Grayscale()
		default:
			panic("Unknown effect: " + effect)
		}

		// swap buffers between effects except for last one
		if i < len(effects)-1 {
			img.SwapBuffers() // the output of the previous effect becomes the input for the next effect
		}
	}
}
