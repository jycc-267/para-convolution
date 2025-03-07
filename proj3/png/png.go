// Package png allows for loading png images and applying
// image flitering effects on them
package png

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// The Image represents a structure for working with PNG images.
// You are allowed to update this and change it as you wish!
type Image struct {
	In             *image.RGBA64   //The original pixels before applying the effect (input buffer)
	Out            *image.RGBA64   //The updated pixels after applying teh effect (output buffer)
	Bounds         image.Rectangle //The size of the image
	EffectsApplied bool
	Chunks         []ImageChunk
}

type ImageChunk struct {
	In      *image.RGBA64
	Out     *image.RGBA64
	Bounds  image.Rectangle
	Overlap int
	YStart  int
	YEnd    int
}

type SubImager interface {
	image.Image
	SubImage(r image.Rectangle) image.Image
}

//
// Public functions
//

// Load returns a Image that was loaded based on the filePath parameter
// You are allowed to modify and update this as you wish
func Load(filePath string) (*Image, error) {

	inReader, err := os.Open(filePath)

	if err != nil {
		return nil, err
	}
	defer inReader.Close()

	inOrig, err := png.Decode(inReader)

	if err != nil {
		return nil, err
	}

	bounds := inOrig.Bounds()

	outImg := image.NewRGBA64(bounds)
	inImg := image.NewRGBA64(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := inOrig.At(x, y).RGBA()
			inImg.Set(x, y, color.RGBA64{uint16(r), uint16(g), uint16(b), uint16(a)})
		}
	}
	task := &Image{}
	task.In = inImg
	task.Out = outImg
	task.Bounds = bounds
	task.EffectsApplied = false
	return task, nil
}

// Save saves the image to the given file
// You are allowed to modify and update this as you wish
func (img *Image) Save(filePath string) error {

	// use pointer to avoid copying the entire image data when assigning to saveImg
	var saveImg *image.RGBA64

	outWriter, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outWriter.Close()

	// if no effect applied then save the original image
	if img.EffectsApplied {
		saveImg = img.Out
	} else {
		saveImg = img.In
	}

	err = png.Encode(outWriter, saveImg)
	if err != nil {
		return err
	}
	return nil
}

// clamp will clamp the comp parameter to zero if it is less than zero or to 65535 if the comp parameter
// is greater than 65535.
func clamp(comp float64) uint16 {
	return uint16(math.Min(65535, math.Max(0, comp)))
}

// avoid data copying
func (img *Image) SwapBuffers() {
	img.In, img.Out = img.Out, img.In
}
