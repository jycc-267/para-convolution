// Package png allows for loading png images and applying image flitering effects on them
// Kernels for image operation: https://en.wikipedia.org/wiki/Kernel_(image_processing)
// Refer to png.go and https://www.devdungeon.com/content/working-images-go to see how to apply Image package
package png

import (
	"image/color"
	"sync"
)

type Barrier struct {
	sync.Mutex
	cond       *sync.Cond
	count      int // Number of waiting threads
	threshold  int // Total required participants required to release the barrier
	generation int // Phase counter (prevent spurious wakeups between phases)
}

func NewBarrier(threshold int) *Barrier {
	b := &Barrier{threshold: threshold}
	b.cond = sync.NewCond(&b.Mutex)
	return b
}

func (b *Barrier) Wait() {
	b.Lock()
	defer b.Unlock()

	localGen := b.generation
	b.count++

	if b.count == b.threshold {
		b.count = 0
		b.generation++
		b.cond.Broadcast()
	} else {
		for localGen == b.generation {
			b.cond.Wait()
		}
	}
}

// ImageTask is used to unmarshal each JSON object from effects.txt
// https://www.youtube.com/watch?v=hzSkpuL2I_Y&t=695s
// https://stackoverflow.com/questions/21197239/decoding-json-using-json-unmarshal-vs-json-newdecoder-decode
type ImageTask struct {
	InPath  string   `json:"inPath"`
	OutPath string   `json:"outPath"`
	Effects []string `json:"effects"`
	DataDir string
}

// CNN: https://www.youtube.com/watch?v=FrKWiRv254g&list=PLJV_el3uVTsPy9oCRY30oBPNLCo89yu49&index=19
// Apply a 3x3 convolution kernel to the image
func (img *Image) convolution(kernel [9]float64) {
	bounds := img.Bounds // get the bounds of the input image
	// iterate over each pixel in the image
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// apply kernel to the current pixel, returning new RGB values
			r, g, b := img.convolve(x, y, kernel)
			// retrieve the alpha value (transparency) of the current pixel from the input image
			_, _, _, a := img.In.At(x, y).RGBA()
			// set the new RGB values and original alpha to the output image
			img.Out.Set(x, y, color.RGBA64{r, g, b, uint16(a)})
		}
	}
}

// Convolve applies the convolution kernel to a single pixel
func (img *Image) convolve(x int, y int, kernel [9]float64) (r, g, b uint16) {
	var sumR, sumG, sumB float64 // new sum variables for each color channel
	bounds := img.Bounds

	// iterate over the 3x3 neighborhood of the current pixel
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			neighborX := x + dx
			neighborY := y + dy

			/*
				Image convolution involves applying kernels to each pixel and its surrounding pixels in the image
				For pixels at the edges of the image, some of the surrounding pixels required for the convolution operation don't exist
				To handle this, we use "zero-padding" that imaginary pixels with a value of zero are added around the edges of the image
				The results for these edge pixels will not be as accurate as for pixels in the center of the image
			*/
			// valid coordinates range: [Min.X, Max.X) and [Min.Y, Max.Y)
			if neighborX < bounds.Min.X || neighborX >= bounds.Max.X || neighborY < bounds.Min.Y || neighborY >= bounds.Max.Y {
				continue // skip this pixel: zero padding
			}

			// get the RGB values of the neighboring pixel
			r, g, b, _ := img.In.At(neighborX, neighborY).RGBA()
			// map the 2D kernel coordinates to a 1D array index
			index := (dy+1)*3 + (dx + 1)
			// accumulate the kernel value to each color channel
			sumR += float64(r) * kernel[index]
			sumG += float64(g) * kernel[index]
			sumB += float64(b) * kernel[index]
		}
	}

	return clamp(sumR), clamp(sumG), clamp(sumB)
}

// Sharpen() applies a sharpening effect to a image
func (img *Image) Sharpen() {
	kernel := [9]float64{0, -1, 0, -1, 5, -1, 0, -1, 0}
	img.convolution(kernel)
}

// EdgeDetection() applies an edge detection effect to a image
func (img *Image) EdgeDetection() {
	kernel := [9]float64{-1, -1, -1, -1, 8, -1, -1, -1, -1}
	img.convolution(kernel)
}

// Blur() applies a blur effect to a image
func (img *Image) Blur() {
	kernel := [9]float64{1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9}
	img.convolution(kernel)
}

// Grayscale() applies a grayscale filtering effect to a image
func (img *Image) Grayscale() {

	// Bounds returns defines the dimensions of the image. Always
	// use the bounds Min and Max fields to get out the width
	// and height for the image
	bounds := img.Out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Returns the pixel (i.e., RGBA) value at a (x,y) position
			// Note: These get returned as int32 so based on the math you'll
			// be performing you'll need to do a conversion to float64(..)
			r, g, b, a := img.In.At(x, y).RGBA()

			// Note: The values for r,g,b,a for this assignment will range between [0, 65535].
			// For certain computations (i.e., convolution) the values might fall outside this
			// range so you need to clamp them between those values.
			greyC := clamp(float64(r+g+b) / 3)

			// Note: The values need to be stored back as uint16 (I know weird..but there's valid reasons
			// for this that I won't get into right now).
			img.Out.Set(x, y, color.RGBA64{greyC, greyC, greyC, uint16(a)})
		}
	}
}

////// Below is the BSP version //////

/*
[Apply an effect]
├─ Main thread creates barrier(#workers + 1)
├─ Starts #workers
├─ Main thread calls barrier.Wait()
│  (blocks until all workers + self reach barrier)
│
├─ Workers process convolution on their slices
├─ Each worker calls barrier.Wait() when done
│
[Barrier Release]
└─ All workers + main thread continue
*/
func (img Image) BSPConvolution(kernel [9]float64, numThreads int) {

	// divide image into horizontal slices
	bounds := img.Bounds
	height := bounds.Max.Y - bounds.Min.Y
	sliceHeight := height / numThreads
	barrier := NewBarrier(numThreads + 1)

	for i := 0; i < numThreads; i++ {
		// horizontally set start and end of a slice for each goroutine
		// give larger overlapping slices to goroutines
		start := bounds.Min.Y + i*sliceHeight
		end := start + sliceHeight

		// clamp to image bounds for the last slice
		if i == numThreads-1 {
			end = bounds.Max.Y
		}

		go func(startY, endY int) {
			for y := startY; y < endY; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					r, g, b := img.convolve(x, y, kernel)
					_, _, _, a := img.In.At(x, y).RGBA()
					img.Out.Set(x, y, color.RGBA64{r, g, b, uint16(a)})
				}
			}
			barrier.Wait()
		}(start, end)
	}
	barrier.Wait()
}

// BSPSharpen() parallelly applies a sharpening effect to a image
func (img *Image) BSPSharpen(numThreads int) {
	kernel := [9]float64{0, -1, 0, -1, 5, -1, 0, -1, 0}
	img.BSPConvolution(kernel, numThreads)
}

// BSPEdgeDetection() parallelly applies an edge detection effect to a image
func (img *Image) BSPEdgeDetection(numThreads int) {
	kernel := [9]float64{-1, -1, -1, -1, 8, -1, -1, -1, -1}
	img.BSPConvolution(kernel, numThreads)
}

// BSPBlur() parallelly applies a blur effect to a image
func (img *Image) BSPBlur(numThreads int) {
	kernel := [9]float64{1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9}
	img.BSPConvolution(kernel, numThreads)
}

func (img *Image) BSPGrayscale(numThreads int) {

	bounds := img.Bounds
	height := bounds.Max.Y - bounds.Min.Y
	sliceHeight := height / numThreads
	barrier := NewBarrier(numThreads + 1)

	for i := 0; i < numThreads; i++ {
		start := bounds.Min.Y + i*sliceHeight
		end := start + sliceHeight
		if i == numThreads-1 {
			end = bounds.Max.Y
		}
		go func(startY, endY int) {

			for y := startY; y < endY; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					r, g, b, a := img.In.At(x, y).RGBA()
					grey := clamp(float64(r+g+b) / 3)
					img.Out.Set(x, y, color.RGBA64{grey, grey, grey, uint16(a)})
				}
			}
			barrier.Wait()
		}(start, end)
	}
	barrier.Wait()
}
