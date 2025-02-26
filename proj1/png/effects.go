// Package png allows for loading png images and applying image flitering effects on them
// Kernels for image operation: https://en.wikipedia.org/wiki/Kernel_(image_processing)
// Refer to png.go and https://www.devdungeon.com/content/working-images-go to see how to apply Image package
package png

import (
	"image/color"
	"sync"
)

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
			_, _, _, a := img.in.At(x, y).RGBA()
			// set the new RGB values and original alpha to the output image
			img.out.Set(x, y, color.RGBA64{r, g, b, uint16(a)})
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
			r, g, b, _ := img.in.At(neighborX, neighborY).RGBA()
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
	bounds := img.out.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Returns the pixel (i.e., RGBA) value at a (x,y) position
			// Note: These get returned as int32 so based on the math you'll
			// be performing you'll need to do a conversion to float64(..)
			r, g, b, a := img.in.At(x, y).RGBA()

			// Note: The values for r,g,b,a for this assignment will range between [0, 65535].
			// For certain computations (i.e., convolution) the values might fall outside this
			// range so you need to clamp them between those values.
			greyC := clamp(float64(r+g+b) / 3)

			//Note: The values need to be stored back as uint16 (I know weird..but there's valid reasons
			// for this that I won't get into right now).
			img.out.Set(x, y, color.RGBA64{greyC, greyC, greyC, uint16(a)})
		}
	}
}

// parallelConvolution applies convolution in parallel slices
func (img Image) paraConvolution(kernel [9]float64, numThreads int) {

	// divide image into horizontal slices
	bounds := img.Bounds
	height := bounds.Max.Y - bounds.Min.Y
	sliceHeight := height / numThreads

	var wg sync.WaitGroup
	wg.Add(numThreads)

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
			defer wg.Done()
			// intialize a local buffer for each goroutine to work on its expanded slice
			// tempOut := image.NewRGBA64(img.out.Bounds())

			for y := startY; y < endY; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					r, g, b := img.convolve(x, y, kernel)
					_, _, _, a := img.in.At(x, y).RGBA()
					img.out.Set(x, y, color.RGBA64{r, g, b, uint16(a)})
				}
			}
		}(start, end)
	}
	wg.Wait()
}

// ParaSharpen() parallelly applies a sharpening effect to a image
func (img *Image) ParaSharpen(numThreads int) {
	kernel := [9]float64{0, -1, 0, -1, 5, -1, 0, -1, 0}
	img.paraConvolution(kernel, numThreads)
}

// ParaEdgeDetection() parallelly applies an edge detection effect to a image
func (img *Image) ParaEdgeDetection(numThreads int) {
	kernel := [9]float64{-1, -1, -1, -1, 8, -1, -1, -1, -1}
	img.paraConvolution(kernel, numThreads)
}

// ParaBlur() parallelly applies a blur effect to a image
func (img *Image) ParaBlur(numThreads int) {
	kernel := [9]float64{1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9, 1.0 / 9}
	img.paraConvolution(kernel, numThreads)
}

// parallelGrayscale applies grayscale in parallel
// No pixel-neighborhood dependencies (unlike convolution)
func (img *Image) ParaGrayscale(numThreads int) {

	bounds := img.Bounds
	height := bounds.Max.Y - bounds.Min.Y
	sliceHeight := height / numThreads

	var wg sync.WaitGroup
	wg.Add(numThreads)

	for i := 0; i < numThreads; i++ {
		start := bounds.Min.Y + i*sliceHeight
		end := start + sliceHeight
		if i == numThreads-1 {
			end = bounds.Max.Y
		}

		go func(startY, endY int) {
			defer wg.Done()
			for y := startY; y < endY; y++ {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					r, g, b, a := img.in.At(x, y).RGBA()
					grey := clamp(float64(r+g+b) / 3)
					img.out.Set(x, y, color.RGBA64{grey, grey, grey, uint16(a)})
				}
			}
		}(start, end)
	}
	wg.Wait()
}
