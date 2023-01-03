# Gravity Simulation Golang
## Go with [Simple Direct Layer](https://github.com/veandco/go-sdl2)

This is a basic implementation of a particle simulation with gravity. The simulation is crude and mildly inefficient but works as a proof of concept. Some basic interaction is implemented using the keyboard, and the use of SDL2 should make this application portable to almost any device.

## Usage

In all cases, use the `-h` flag to get more help on how to use the application.

Note that building the SDL2 library can take a long time - for me it took close to five minutes. This is a one time compilation however, and subsequent runs will not require this lengthy step.

### Running from Source

`go run .`

### Building from Source 

First, build using

`go build .`

Then run the application with

`./gravity_simulation`

## Future Plans

The main goal of this project was to:
- Use Go for a complex application
- Use a graphics framework in Go

On these fronts, the project has been a resounding success. However, there is still some work that could be done to improve the application.

### Computational inefficiency:

The physics calculations performed at each step is somewhat inefficient. Currently the program calculates the force contribution on a Body from all other Bodies before summing this up and finding the resulting acceleration. Then, a small step is made to simulate a continuous flow of time. This is fine for small simulations (numBodies <= 100) but the computation grows with O(n^2). There are some techniques that could significantly improve performance here:

#### Symmetry
If we note that the force on Body A from Body B is exactly equal but opposite to the force on Body B from Body A we can immediately cut out exactly half of the expensive force calculations. This would provide an immediate speed up by a factor of 2 and may not be very difficult to implement. However, scaling is still O(n^2)

#### Quadtree
A quadtree is a data structure that splits space into quadrants recursively until nothing of interest remains in a leaf. In this application a quadtree is useful as for a body in a quadrant, all bodies outside of a quadrant can act like a single body instead of many individual bodies. This would reduce the number of computations immensely and result in a much more efficient computation at the cost of implementing and building a quadtree at every step.

A quadtree would allow for scaling like O(log(n)) instead of O(n)

### Interactivity:

Current interactivity is rather crude, as only the keyboard is used. It would be nice to make some the interactions use the mouse, or even support other devices for true portability.

### File management:

It may be nice to have a good way to interact with the file system when saving/loading files, instead of hardcoding that saved files are saved to `save.csv`