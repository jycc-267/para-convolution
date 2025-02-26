# Project \#1: An Image Processing System

The project implements three versions of image editor that apply convolution effects on given images. The sequential version processes images one at a time without parallelism. Each image is fully loaded, effects are applied in-order using sequential convolution operations, and results are saved before moving to the next image. This version serves as the baseline for performance comparisons. The parfiles version spawns goroutines that compete to pull image tasks from a shared task queue protected by a TAS lock. After a goroutine acquires the lock, the image task along with a sequence of effects will then be executed independently. The parslices version processes an individual image by splitting it into slices. This version has each goroutine apply the same effect on their own slices, wait for all slices to be completed between effects, and move on to the next effect instruction together. The parfiles version attempts to maximize hardware utilization when processing many images, while the parslices version tries to accelerate single large image processing.

## Preliminary Instructions

 If you are unfamiliar
with image convolutions then you should read over the following sources before
beginning the assignment:

-   [Two Dimensional
    Convolution](http://www.songho.ca/dsp/convolution/convolution2d_example.html)
-   [Image Processing using
    Convolution](https://en.wikipedia.org/wiki/Kernel_(image_processing))

Instruction on generating performance testing plots, run: ``sbatch benchmark-proj1.sh`` at ``./proj1/benchmark``. The test runs each combination of parallel version, number of threads, ``data_dir`` of image five times, and outputs the results into txt files at benchmark/results.

## Program Usage

The program will read from a series of JSON strings, where each string
represents an image along with the effects that should be applied to that
image. Each string will have the following format,

``` json
{ 
  "inPath": string, 
  "outPath": string, 
  "effects": [string] 
}
```

For example, processing an image of a sky may have the following JSON
string,

``` json
{ 
  "inPath": "sky.png", 
  "outPath": "sky_out.png",  
  "effects": ["S","B","E"]
}
```

where each key-value is described in the table below,

| Key-Value                     | Description |
|-------------------------------|-------------|
| ``"inPath":"sky.png"``        | The ``"inPath"`` pairing represents the file path of the image to read in. Images in  this assignment will always be PNG files. All images are relative to the ``data`` directory inside the ``proj1`` folder. |
| ``"outPath:":"sky_out.png"``  | The ``"outPath"`` pairing represents the file path to save the image after applying the effects. All images are relative to the ``data`` directory inside the ``proj1`` folder. |
| ``"effects":["S"\,"B"\,"E"]`` | The ``"effects"`` pairing  represents the image effects to apply to the image. You must apply these in the order they are listed. If no effects are specified (e.g.\, ``[]``) then the out image is the same as the input image. |

The program will read in the images, apply the effects associated with
an image, and save the images to their specified output file paths. How
the program processes this file is described in the **Program
Specifications** section.

## Image Effects

The sharpen, edge-detection, and blur image effects are required to use
image convolution to apply their effects to the input image.
The size of the input and output image
are fixed (i.e., they are the same). Thus, results around the border
pixels will not be fully accurate since we will need to pad zeros
where inputs are not defined. The grayscale effect uses a
simple algorithm defined below that does not require convolution.

Each effect is identified by a single character that is described below,

| Image Effect | Description |
| -------------|-------------|
| ``"S"`` | Performs a sharpen effect with the following kernel (provided as a flat go array): ``[9]float6 {0,-1,0,-1,5,-1,0,-1,0}``. |
| ``"E"`` | Performs an edge detection effect with the following kernel (provided as a flat go array): ``[9]float64{-1,-1,-1,-1,8,-1,-1,-1,-1}``. |
| ``"B"`` | Performs a blur effect with the following kernel (provided as a flat go array): ``[9]float64{1 / 9.0, 1 / 9, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0, 1 / 9.0}``. |
| ``"G"`` | Performs a grayscale effect on the image. This is done by averaging the values of all three color numbers for a pixel, the red, green and blue, and then replacing them all by that average. So if the three colors were 25, 75 and 250, the average would be 116, and all three numbers would become 116. |

## The `data` Directory

Inside the `proj1` directory, the image `data` can be downloaded here:

-   [Proj 1 Data](https://www.dropbox.com/s/cwse3i736ejcxpe/data.zip?dl=0) :
    There should be a download arrow icon on the left side to download
    the zip folder.
-   Place this directory inside the `proj1` directory that contains the
subdirectories: `editor` and `png`.
-   Here is the structure of the `data` directory:

| Directory/Files | Description  |
|-----------------|--------------|
| ``effects.txt`` |  This is the file that contains the string of JSONS that were described above. This will be the only file used for this program (and also for testing purposes). You must use a relative path to your ``proj1`` directory to open this file. For example, if you open this file from the ``editor.go`` file then you should open as ``../data/effects.txt``. |
|  ``expected`` directory | This directory contains the expected filtered out image for each JSON string provided in the ``effects.txt``. We will only test your program against the images provided in this directory. Your  produced images do not need to look 100% like the provided output. If there are some slight differences based on rounding-error then that's fine for full credit. |
|  ``in`` directory | This directory contains three subdirectories called: ``big``, ``mixture``, and ``small``. The actual images in each of these subdirectories are all the same, with the exception of their *image sizes*. The ``big`` directory has the best resolution of the images, ``small`` has a reduced resolution of the images, and the ``mixture`` directory has a mixture of both big and small sizes for different images. You must use a relative path to your ``proj1`` directory to open this file. For example, if you want to open the ``IMG_2029.png`` from the ``big`` directory from inside the ``editor.go`` file then you should open as ``../data/in/big/IMG_2029.png``. |
| ``out`` directory | This is where the program will place the ``outPath`` images when running the program. |

### Working with Images in Go and Startup Code

As part of the Go standard library, an `image` package is provided that
makes it easy to load,read,and save PNG images. I recommend looking at
the examples from these links:

-   [Go PNG docs](https://golang.org/pkg/image/png/)
-   A [helpful
    tutorial](https://www.devdungeon.com/content/working-images-go)

> **Note**:
> The image package only allows you to read an image data and not modify
> it in-place. You will need to create a separate out buffer to represent
> the modified pixels. We have done this for you already in the `Image`
> struct as follows:

``` go
type Image struct {
  in  *image.RGBA64  // Think about swapping these between effects 
  out *image.RGBA64  // Think about swapping these between effects 
  Bounds  image.Rectangle
  ... 
} 
```

Feel free to reuse or modify this in your implementation. Remember these are
**pointers** so you only need to swap the pointers to make the old out buffer
the new in buffer when applying one effect after another effect.  This process
is less expensive than copying pixel data after apply each effect.

To help you get started, I provide code for loading, saving, performing
the grayscale effect on a png image. You are not required to use this
code and you can modify it as you wish. This code is already inside the
`proj1/sample/sample.go` directory. You can run this sample program by
going into the `proj1/sample` directory typing in the following command:

    $: go run sample.go test_img.png 

## Program Specifications

For this project, You will implement three versions of this image
processing system. The versions will include a sequential version and
two parallel versions.

The running of these various versions have already been setup for you
inside the `proj1/editor/editor.go` file.

The `data_dir` argument will always be either `big`, `small`, or
`mixture` or a combination between them. The program will always read
from the `data/effects.txt` file; however, the `data_dir` argument
specifies which directory to use. The user can also add a `+` to perform
the effects on multiple directories. For example, `big` will apply the
`effects.txt` file on the images coming from the `big` directory. The
argument `big+small` will apply the `effects.txt` file on both the `big`
and `small` directory. The program must always prepend the `data_dir`
identifier to the beginning of the `outPath`. For example, running the
program as follows:

    $: go run editor.go big bsp 4 

will produce inside the `out` directory the following files:

    big_IMG_2020_Out.png 
    big_IMG_2724_Out.png 
    big_IMG_3695_Out.png 
    big_IMG_3696_Out.png 
    big_IMG_3996_Out.png 
    big_IMG_4061_Out.png 
    big_IMG_4065_Out.png
    big_IMG_4066_Out.png 
    big_IMG_4067_Out.png
    big_IMG_4069_Out.png

Here's an example of a combination run:

    $: go run editor.go big+small pipeline 2

will produce inside the `out` directory the following files:

    big_IMG_2020_Out.png 
    big_IMG_2724_Out.png 
    big_IMG_3695_Out.png 
    big_IMG_3696_Out.png 
    big_IMG_3996_Out.png 
    big_IMG_4061_Out.png 
    big_IMG_4065_Out.png
    big_IMG_4066_Out.png 
    big_IMG_4067_Out.png
    big_IMG_4069_Out.png
    small_IMG_2020_Out.png 
    small_IMG_2724_Out.png 
    small_IMG_3695_Out.png 
    small_IMG_3696_Out.png 
    small_IMG_3996_Out.png 
    small_IMG_4061_Out.png 
    small_IMG_4065_Out.png
    small_IMG_4066_Out.png 
    small_IMG_4067_Out.png
    small_IMG_4069_Out.png

We will always provide valid command line arguments so you will only be
given at most 3 specified identifiers for the `data_dir` argument. A
single `+` will always be used to separate the identifiers with no
whitespace.

The `mode` and `number_of_threads` arguments will be used to run one of
the parallel versions. Parts 2 and 3 will discuss these arguments in
more detail. If the `mode` and `number_of_threads` arguments are not
provided then the program will default to running the sequential
version, which is discussed in Part 1.

The scheduling (i.e., running) of the various implementations is handled
by the `scheduler` package defined in `proj1/scheduler` directory. The
`editor.go` program will create a configuration object (similar to
project 1) using the following struct:

``` go
type Config struct {
  DataDirs string //Represents the data directories to use to load the images.
  Mode     string // Represents which scheduler scheme to use
  ThreadCount int // Runs in parallel with this number of threads
}
```

The `Schedule` function inside the `scheduler/scheduler.go` file
will then call the correct version to run based on the `Mode` field of
the configuration value. Each of the functions to begin running the
various implementation will be explained in the following sections.
**You cannot modify any of the code in the
\`\`proj1/scheduler/scheduler.go\`\` or \`\`proj1/editor/editor.go\`\`
file**.

**Additional Assumptions**: No error checking is needed to be done to
the strings coming in from *effects.txt*. You can assume the JSON
strings will contain valid values and provided in the format described
above. We will always provide the correct command line arguments and in
the correct order. The `expected` directory in `proj1/data` is based on
only running the small dataset. Thus, the resolution for mixture and big
modes will make the images appear slightly different. This is okay for
this assignment. We will always run/grade your solutions by going inside
the `proj1/editor` directory so loading in files should be relative to
that directory.

## Part 1: Sequential Implementation

Inside the `proj1/scheduler/sequential.go` file, implement the function:

``` go
func RunSequential(config Config) {

}
```

The sequential version is ran by default when executing the `editor`
program when the `mode` and `number_of_threads` are both not provided.
The sequential program is relatively straightforward. This version
should run through the images specified by the strings coming in from
`effects.txt`, apply their effects and save the modified images to their
output files inside the `data/out` directory. Make sure to prepend the
`data_dir` identifier.

> **Note**:
> You should implement the sequential version first. Make sure your code
> is **modular** enough such that you can potentially reuse functions/data
> structures later in your parallel version. Think about what libraries
> and functions should be created. **We will consider code and design style
> when grading this assignment**.

You may find this code useful:

``` go
effectsPathFile := fmt.Sprintf("../data/effects.txt")
effectsFile, _ := os.Open(effectsPathFile)
reader := json.NewDecoder(effectsFile)
```

## Part 2: Multiple Images in Parallel

The first parallel implementation will process multiple images in parallel,
but each individual image is handled by only one thread. The code should be
implemented as follows:

1.  Create a queue, where each node contains all information about the tasks
    related to an individual image (e.g. input file, output file, effects).
    You can either implement your own queue (e.g. as a linked list), or use an
    existing sequential data structure. The queue can be populated sequentially
    while reading the JSON input strings. It does not matter if your queue is
    FIFO or any other order.

2.  Spawn Go routines. The number of Go routines should be the
    number of threads specified in the command line, or the number of images in
    the queue (whichever is smaller). The Go routines should take image tasks
    from the queue and process them. You must implement your own TAS lock
    to safeguard accesses to the queue, i.e. items can only be taken out of the
    queue by a Go routine that holds the lock. **You cannot use any existing
    thread-safe queue datastructures or locks**.

3.  Go routines should run until all tasks from the queue are processed. The
    main program should wait until all Go routines have terminated. This is
    best implemented using a wait group; Please use the standard implementation
    provided by Go.

## Part 3: Parallelize Each Image

In the second parallel implementation, you will parallelize the processing of
individual images. For now, we assume that only one image is processed at a
time. This should be done as follows:

1.  Iterate over the same queue as in Part 2. For each image, spawn Go routines
    that operate on slices of the image. You will probably want to use slicing
    here, and take inspiration from the examples shown during class to compute the
    start and end index that each Go routine needs to work on.
2.  Let each Go routine apply effects to its own slice of the image.

3.  Only start working on the next image when the current image is fully
    processed. You can use waitgroups for this.

A performance hint about this part:

 -  Take care to minimize the amount of data copying, i.e. pass pointers or
    slices instead of copying chunks of data around all the time.

Convolutions have a slightly larger read set than write set, i.e. they read
from more indices than they write to. If multiple effects E1, E2, ... are
applied and slices are processed in parallel, you need to manage the fact that
the output of E1 in one slice affects the input of E2 in neighboring slices.
A few options come to mind (bonus points for coming up with new ones):

 - Use waitgroups so that nobody works on E2 before everyone has finished E1.
 - Give larger overlapping slices to Go routines, and let the slices shrink
   with each effect so that the output slices have just the right
   non-overlapping size when writing to the shared output.

Your grade will depend on the performance, which is affected by how well you
manage this fact and how well you avoid data copies. 

## Part 4: Performance Measurements and Speedup Graphs

Observation
1. For sequential version:
The main hotspot in the sequential program is the convolution operation, which requires multiple nested loops and kernel calculations for each pixel. File I/O operations (reading/writing PNG files) create sequential bottlenecks since loading and writing large image files create latency.
2. Comparison between two parallel versions:
The parfiles version is faster than the parslices version because unlike the later version, the former one has less synchronization overhead since each image is processed independently. Also, the later one creates sequential bottlenecks while loading and saving large images.
3. Image size impact:
a. Parfiles: Mixture dataset performs worse than both, I suspect there exists overhead from handling varying image sizes?
b. Parslices: Image size doesn’t matter much eventually when we use 12 workers. The program reaches a ~1.8x speedup with 12 threads for all types of dataset.
4. Amdahl’s Law Analysis:
a. For the parfiles version, the theoretical speed-up should be near-linear. This
is due to the fact that each image is processed independently by a single thread/goroutine. Each goroutine handles its own file I/O individually and doesn’t have to wait between image effects. However, the discrepancies from the actual speed-up data could be derived from contention from shared system resource contention (SBATCH --nodes=1). In this sense, memory bandwidth also becomes a bottleneck since all threads share the same memory bus, thus affecting performance when multiple threads access different parts of memory. File I/O is also a bottleneck when multiple threads try to read/write images simultaneously.
b. For the parslices version, the actual speed-ups align closer to theoretical values.
5. Improvement:
For parslices, instead of processing effects sequentially (image → slices → E1 → image → slices → E2…), we can create a pipeline in which multiple slices of the image move through the pipeline concurrently so that the program doesn’t have to wait between effects (multiples slices → E1 → E2 → E3 → E4 → image). And I think this can be done by the second option provided in the project instruction part3.


    ![image](./proj1/benchmark/speedup-parslices.png)
    ![image](./proj1/benchmark/speedup-parfiles.png)
