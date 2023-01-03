package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"text/tabwriter"
	"time"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	SCREENWIDTH    = 1200
	SCREENHEIGHT   = 800
	FRAMETIME      = 16
	G              = 100
	PIXELDECAYRATE = 2
)

var (
	// The array of pixels to give to the SDL renderer
	pixels []byte = make([]byte, SCREENWIDTH*SCREENHEIGHT*4)
	// The color black which is used multiple times for the background
	sdlColorBlack sdl.Color = sdl.Color{0, 0, 0, 255}
	// Some variables for command line flags
	saveFilePath string
	numBodies    int
	// List of bodies to store current frame and next frame
	// This allows for consistent simulations (not changing bodies mid frame)
	// We keep both so the garbage collector does not kill old arrays every frame
	currentBodies []*Body
	nextBodies    []*Body
	// Variables to do with the simulation behavior
	paused        bool    = true
	pixeldecay    bool    = false
	timescale     float64 = 0.25
	zoomscale     float64 = 1
	movescale     float64 = 25
	currentXCoord float64 = 0
	currentYCoord float64 = 0
	// Finally, a writer to print these variables nicely
	tableWriter *tabwriter.Writer = tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
)

// At start of program, process command line flags and allocate some memory for bodies
func init() {
	var helpFlag bool
	flag.StringVar(&saveFilePath, "saveFile", "", "The path to the save file to use.\nIf not specified, use other flags to determine simulation behavior")
	flag.IntVar(&numBodies, "numBodies", 5, "The number of bodies to add to this simulation")
	flag.BoolVar(&helpFlag, "h", false, "Display help on this program, then quit")
	flag.Parse()

	// If the user has selected the help flag, print the help message then quit
	if helpFlag {
		fmt.Println(`
Gravity Simulation
Usage:
	Run from source using "go run ."
	Build from source using "go build ."
	Run from executable using "./gravity_simulation"

Flags:
	When running this program, some flags can be specified to change starting configurations
	--saveFile : The path to the csv file to load into the simulation
		Note if this flag is not set, the simulation will be loaded with a random initial configuration
	--numBodies : An integer to specify the number of bodies to randomly seed when starting this simulation
		Defaults to 5

Controls:
	While the simulation is running you can use the keyboard to control parts of the application. The controls are:

	W : Move view window up
	A : Move view window left
	S : Move view window down
	D : Move view window right
	Q : Zoom out
	E : Zoom in
	
	ArrowKeyDown : Decrease the rate of view window movement
	ArrowKeyUp : Increase the rate of view window movement
	ArrowKeyLeft : Decrease the speed of the simulation
	ArrowKeyRight : Increase the speed of the simulation

	Spacebar : Toggle pause/resume
	X : Toggle particle trails
	C : Advance a single timestep (without unpausing)
	P : Print the current state of the simulation (all bodies + settings)
	O : Save the currect state of the simulation`)
		os.Exit(0)
	}

	// If we were given a file to read from, try it
	if saveFilePath != "" {
		fmt.Println("LOADING FROM FILE ", saveFilePath)
		data, err := os.ReadFile(saveFilePath)
		if err != nil {
			fmt.Println("ERROR: Could not read file ", saveFilePath)
			panic(err)
		}

		// Save files are in csv format, so we can use the encoding/csv to read it out
		r := csv.NewReader(strings.NewReader(string(data)))
		// Comment lines start with #
		// e.g. the first line which details the csv format
		r.Comment = '#'
		records, err := r.ReadAll()
		if err != nil {
			fmt.Println("ERROR: CSV file not correctly formatted")
			panic(err)
		}

		// Now we have read all the bodies in the saved file we can allocate exactly this much memory!
		currentBodies = make([]*Body, len(records))
		nextBodies = make([]*Body, len(records))
		for i, b := range records {
			currentBodies[i] = NewBodyFromStrings(b)
		}
	} else { // If we did not get a save file we will instead create a set of random bodies
		fmt.Println("NO LOAD FILE")
		fmt.Println("USING NUMBODIES = ", numBodies)
		// Seed the creation  with the current time to get new simulations with each run
		rand.Seed(time.Now().UnixMicro())
		// We also know exactly how many bodies we expect so we can allocate this memory
		currentBodies = make([]*Body, numBodies)
		nextBodies = make([]*Body, numBodies)
		for i := 0; i < numBodies; i++ {
			currentBodies[i] = NewRandomBody()
		}
	}

	// Finally, we can save this starting config to a file so the user can run it again if need be
	saveState()
}

// Save the state of the simulation to a file
func saveState() {
	f, err := os.OpenFile("save.csv", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		// However, if we cannot create the file as expected it isn't the end of the world
		// We just return, not panic
		fmt.Println("Cannot create save.csv to save state!")
		return
	}
	fmt.Fprintln(f, "#x, y, xVel, yVel, mass, radius, red, green, blue")
	for _, b := range currentBodies {
		if b != nil {
			fmt.Fprintf(f, "%v,%v,%v,%v,%v,%v,%v,%v,%v\n", b.x, b.y, b.xVel, b.yVel, b.mass, b.radius, b.color.R, b.color.G, b.color.B)
		}
	}
	fmt.Fprintf(f, "\n")
}

// print all of the bodies that are not nil from the currentBodies array
// some extra formatting is added (a line of hyphens, etc)
func printBodies() {
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Fprintf(tableWriter, "Body Index\tx\ty\txVel\tyVel\tmass\tradius\tcolor\n")
	for i, b := range currentBodies {
		if b == nil {
			continue
		}
		fmt.Fprintf(tableWriter, "BODY %v\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%v\t\n",
			i,
			b.x,
			b.y,
			b.xVel,
			b.yVel,
			b.mass,
			b.radius,
			b.color,
		)
	}
	tableWriter.Flush()
}

// print the configuration variables with some formatting
func printConfiguration() {
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Fprintln(tableWriter, "PAUSED\t", paused)
	fmt.Fprintf(tableWriter, "TIMESCALE\t%.2f\n", timescale)
	fmt.Fprintf(tableWriter, "ZOOMSCALE\t%.2f\n", zoomscale)
	fmt.Fprintf(tableWriter, "MOVESCALE\t%.2f\n", movescale)
	fmt.Fprintf(tableWriter, "SCREEN CENTER\t (%.2f, %.2f)\n", currentXCoord, currentYCoord)
	fmt.Fprintf(tableWriter, "SCREEN LIMITS\t X: %v - %v,  Y: %v - %v\n",
		int32(currentXCoord-zoomscale*SCREENWIDTH),
		int32(currentXCoord+zoomscale*SCREENWIDTH),
		int32(currentYCoord-zoomscale*SCREENHEIGHT),
		int32(currentYCoord+zoomscale*SCREENHEIGHT))
	tableWriter.Flush()
}

// set all pixels in the array to a specific color
func setAllPixels(color sdl.Color) {
	for y := 0; y < SCREENHEIGHT; y++ {
		for x := 0; x < SCREENWIDTH; x++ {
			setPixel(int32(x), int32(y), color)
		}
	}
}

// set a specific pixel to a color
func setPixel(x, y int32, c sdl.Color) {
	// This is the index into the pixels array
	// Which is a flattened array of rgb values
	// Hence the extra factor of screenwidth for y
	// and multiplying by the four color channels
	index := (y*SCREENWIDTH + x) * 4

	// The conditional here is just to avoid drawing off the screen
	if index < int32(len(pixels)-4) && index >= 0 {
		pixels[index] = c.R
		pixels[index+1] = c.G
		pixels[index+2] = c.B
	}
}

// Decay a pixel by subtracting a small value from each RGB channel
// When the color channel is below the decay rate (i.e. the next subtraction would be negative)
// instead we set the color channel to zero. A zero value in the color channel will remain at zero
func decayPixel(x, y int32) {
	index := (y*SCREENWIDTH + x) * 4
	if index < int32(len(pixels)-4) && index >= 0 {
		var i int32
		for i = 0; i < 3; i++ {
			if pixels[index+i] < PIXELDECAYRATE {
				pixels[index+i] = 0
				continue
			}
			pixels[index+i] = pixels[index+i] - PIXELDECAYRATE
		}
	}
}

// Handle all the inputs for the application
// This includes quit events (alt+F4, ...) and keyboard events
// SDL also supports other events such as mouse inputs but these are not used
func handleInputs() {
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch t := event.(type) {
		case *sdl.QuitEvent:
			os.Exit(0)
		case *sdl.KeyboardEvent:
			// Ignore released keys
			if t.State == sdl.RELEASED {
				continue
			}

			// If spacebar pressed, pause the simulation
			if t.Keysym.Scancode == sdl.SCANCODE_SPACE && t.Repeat != 1 {
				paused = !paused
			}

			// X makes pixels decay
			if t.Keysym.Scancode == sdl.SCANCODE_X && t.Repeat != 1 {
				pixeldecay = !pixeldecay
			}

			// Pressing c steps one frame
			if t.Keysym.Scancode == sdl.SCANCODE_C {
				timeStep()
			}

			// Pressing Q/E zooms
			if t.Keysym.Scancode == sdl.SCANCODE_Q {
				zoomscale *= 1.2
				setAllPixels(sdlColorBlack)
			}
			if t.Keysym.Scancode == sdl.SCANCODE_E {
				zoomscale /= 1.2
				setAllPixels(sdlColorBlack)
			}

			// Pressing W moves the view up and so on...
			if t.Keysym.Scancode == sdl.SCANCODE_W {
				currentYCoord -= movescale * zoomscale
				setAllPixels(sdlColorBlack)
			}
			if t.Keysym.Scancode == sdl.SCANCODE_S {
				currentYCoord += movescale * zoomscale
				setAllPixels(sdlColorBlack)
			}
			if t.Keysym.Scancode == sdl.SCANCODE_A {
				currentXCoord -= movescale * zoomscale
				setAllPixels(sdlColorBlack)
			}
			if t.Keysym.Scancode == sdl.SCANCODE_D {
				currentXCoord += movescale * zoomscale
				setAllPixels(sdlColorBlack)
			}

			// Pressing up and down scales how quickly we move through space
			if t.Keysym.Scancode == sdl.SCANCODE_UP {
				movescale += 1
			}
			if t.Keysym.Scancode == sdl.SCANCODE_DOWN {
				if movescale > 0 {
					movescale -= 1
				}
			}

			// Pressing left slows down the simulation
			if t.Keysym.Scancode == sdl.SCANCODE_LEFT {
				timescale /= 1.1
			}
			// Pressing right speeds up the simulation
			if t.Keysym.Scancode == sdl.SCANCODE_RIGHT {
				timescale *= 1.1
			}

			// P prints out all bodies
			if t.Keysym.Scancode == sdl.SCANCODE_P {
				fmt.Printf("\n\n\n")
				printBodies()
				printConfiguration()
			}

			// O saves the current state of the simulation to a file
			if t.Keysym.Scancode == sdl.SCANCODE_O {
				fmt.Println("SAVING TO FILE")
				saveState()
			}
		}
	}
}

// Perform a single timestep across the bodies.
func timeStep() {
	for i, body := range currentBodies {
		nextBodies[i] = body.Update()
	}
	// To avoid memory being allocated and collected each frame
	// Simply swap the next (now calculated) array and current array
	temp := currentBodies
	currentBodies = nextBodies
	nextBodies = temp
}

func main() {
	// Start the main method by initializing the SDL framework
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		fmt.Println("Failed to initialize SDL:", err)
		return
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("Gravity Simulation", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		SCREENWIDTH, SCREENHEIGHT, sdl.WINDOW_SHOWN)
	if err != nil {
		fmt.Println("Failed to create window:", err)
		return
	}
	defer window.Destroy()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		fmt.Println("Failed to create renderer:", err)
		return
	}
	defer renderer.Destroy()

	tex, err := renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, SCREENWIDTH, SCREENHEIGHT)
	if err != nil {
		fmt.Println("Failed to create texture:", err)
		return
	}
	defer tex.Destroy()

	// Game loop
	for {
		// At start of each frame, handle any inputs
		handleInputs()

		// If we are not paused, the bodies can be updated
		if !paused {
			timeStep()
		}

		// Before drawing bodies on top, do something (set black or decay) to the background
		for y := 0; y < SCREENHEIGHT; y++ {
			for x := 0; x < SCREENWIDTH; x++ {
				if pixeldecay {
					if !paused {
						decayPixel(int32(x), int32(y))
					}
				} else {
					setPixel(int32(x), int32(y), sdlColorBlack)
				}
			}
		}

		// Then, draw the bodies on top
		for _, bodies := range currentBodies {
			bodies.Draw()
		}

		// Actually draw the pixel array to the window and carry on
		tex.Update(nil, unsafe.Pointer(&pixels[0]), SCREENWIDTH*4)
		renderer.Copy(tex, nil, nil)
		renderer.Present()

		sdl.Delay(FRAMETIME)
	}
}
