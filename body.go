package main

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"

	"github.com/veandco/go-sdl2/sdl"
)

type Body struct {
	// The coordinates of this body
	x float64
	y float64
	// The velocity of this body in cartesian directions
	xVel float64
	yVel float64
	// The mass of this body, directionally proportional to
	// acceleration effect on other bodies
	mass float64
	// The radius of this body - for rendering
	radius float64
	// Color of this body - for rendering
	// Note the alpha channel is unused
	color sdl.Color
}

// Method for converting mass to radius for consistency
func massToRadius(mass float64) float64 {
	return math.Sqrt(mass)
}

// Create a body from a set of strings that map to the body parameters.
// If only some strings are supplied, parameters can be randomly generated.
// Notice all strings are parsed to floats, so the strings MUST be float-y
//
// If 5 or more strings are supplied, the first five strings are mapped to
// - x, y, xVel, yVel, mass
// The remaining parameters are randomly generated (except radius which is calculated using massToRadius function)
//
// If 9 (or more) strings are supplied then all parameters are  set from these strings
func NewBodyFromStrings(bodyParams []string) *Body {
	// Start by converting all params to floats
	// This could be redone in future if none numeric fields are needed
	// Notice that even if color channels are present it will be okay to
	// Temporarily make these floats
	var floatParams []float64
	for i := 0; i < len(bodyParams); i++ {
		convertedParam, err := strconv.ParseFloat(bodyParams[i], 64)
		if err != nil {
			fmt.Println("ERROR: Cannot convert ", bodyParams[i], " to float")
			panic(err)
		}
		floatParams = append(floatParams, convertedParam)
	}

	// If given more than nine params we have the five basic params
	// x,y,xVel, yVel, mass
	// AND the additional four params
	// radius, red, green, blue
	if len(floatParams) >= 9 {
		return &Body{
			x:      floatParams[0],
			y:      floatParams[1],
			xVel:   floatParams[2],
			yVel:   floatParams[3],
			mass:   floatParams[4],
			radius: floatParams[5],
			color:  sdl.Color{uint8(floatParams[6]), uint8(floatParams[7]), uint8(floatParams[8]), 255},
		}
	}

	// If given five options, this is in form of
	// x,y,xVel, yVel, mass
	// Other properties can be inferred (radius) or randomized
	if len(floatParams) >= 5 {
		return &Body{
			x:      floatParams[0],
			y:      floatParams[1],
			xVel:   floatParams[2],
			yVel:   floatParams[3],
			mass:   floatParams[4],
			radius: massToRadius(floatParams[4]),
			color:  sdl.Color{uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), 255},
		}
	}

	// If we don't even have five params we can't do anything!
	panic("NOT ENOUGH PARAMS! Need at least five params to create Body!")
}

// Create a new body with totally random parameters
// Notice some limits are placed on parameter values (e.g. a max speed and mass)
func NewRandomBody() *Body {
	const velocityLimit float64 = 1
	const massLimit float64 = 10
	mass := rand.Float64()*massLimit + 1
	return &Body{
		x:      rand.Float64()*float64(SCREENWIDTH) - float64(SCREENWIDTH)/2,
		y:      rand.Float64()*float64(SCREENHEIGHT) - float64(SCREENHEIGHT)/2,
		xVel:   rand.Float64()*velocityLimit - velocityLimit/2,
		yVel:   rand.Float64()*velocityLimit - velocityLimit/2,
		mass:   mass,
		radius: massToRadius(mass),
		color:  sdl.Color{uint8(rand.Intn(255)), uint8(rand.Intn(255)), uint8(rand.Intn(255)), 255},
	}
}

// Extracted method for finding the squared distance between the centers of two bodies
func distSquared(a, b *Body) float64 {
	return math.Pow(a.x-b.x, 2.0) + math.Pow(a.y-b.y, 2.0)
}

// Associated method to update this body
// Note this method is rather inefficient - it is O(n) for each body and therefore O(n^2) over all in implementation
// This could be improved by:
//   - Noticing that the effect of a->b is the exact opposite of b->a, halving the number of calculations to be done (reducing work by a factor of 2, but still O(n^2))
//   - Using a different method of calculating force (e.g. a quadtree) which reduces calculations for each body from O(n) to O(log(n)) roughly
//
// These have not been implemented because this is a proof of concept and a toy model only - but the options are open in future!
//
// This method handles updating a bodies x,y coordinates based on velocity, and the x,y velocities based on the effects of all other bodies in the simulation
// This simulation uses very crude particle models with simple discrete timesteps. If these timesteps are small enough the simulation is roughly accurate.
// Collisions are modelled as inelastic - the two colliding bodies have their masses added together, velocities set to the solution of the conservation of momentum equations, and coordinates placed at the center of mass
//
// To aide in memory management, two arrays of bodies are used (and swapped at each frame). Therefore, this method has to return a *body to be placed into the next array
// Notice that if a collision occurs, the larger body is kept (updated) and the smaller body returns nil
func (b *Body) Update() *Body {
	// If a body is nil, it has already been consumed
	if b == nil {
		return nil
	}

	newBody := *b

	newBody.x += newBody.xVel * timescale
	newBody.y += newBody.yVel * timescale
	total_acc_x := 0.0
	total_acc_y := 0.0
	for _, other := range currentBodies {
		if other == nil {
			continue
		}

		currDistSquared := distSquared(b, other)
		// If Distance is zero (or close to it) we are ontop one another!
		// Do nothing...
		if currDistSquared < 1 {
			continue
		}

		// If we are too close (touching) then:
		if currDistSquared < math.Pow(b.radius+other.radius, 2) {
			// Merge bodies together!!
			// Smaller mass gets eaten
			if newBody.mass < other.mass {
				return nil
			}

			// Larger mass gets added to
			newBody.x = (newBody.x*newBody.mass + other.x*other.mass) / (newBody.mass + other.mass)
			newBody.y = (newBody.y*newBody.mass + other.y*other.mass) / (newBody.mass + other.mass)
			newBody.xVel = (newBody.xVel*newBody.mass + other.xVel*other.mass) / (newBody.mass + other.mass)
			newBody.yVel = (newBody.yVel*newBody.mass + other.yVel*other.mass) / (newBody.mass + other.mass)
			newBody.radius = massToRadius(newBody.mass + other.mass)
			newBody.mass = (newBody.mass + other.mass)
			return &newBody
		}

		acc_magnitude := -1 * G * other.mass / (currDistSquared)
		angle := math.Atan2(newBody.y-other.y, newBody.x-other.x)
		total_acc_x += acc_magnitude * math.Cos(angle)
		total_acc_y += acc_magnitude * math.Sin(angle)

	}
	newBody.xVel += total_acc_x * timescale
	newBody.yVel += total_acc_y * timescale

	return &newBody
}

// Draw the body to the screen
func (b *Body) Draw() {
	if b == nil {
		return
	}

	// If the ball is already off screen, don't bother doing any loops!
	if (b.x+b.radius) < float64(currentXCoord)-float64(zoomscale*SCREENWIDTH/2) ||
		(b.x-b.radius) > float64(currentXCoord)+float64(zoomscale*SCREENWIDTH/2) ||
		(b.y+b.radius) < float64(currentYCoord)-float64(zoomscale*SCREENHEIGHT/2) ||
		(b.y-b.radius) > float64(currentYCoord)+float64(zoomscale*SCREENHEIGHT/2) {
		return
	}

	for y := -b.radius; y < b.radius; y += zoomscale {
		if b.y+y < float64(currentYCoord)-float64(zoomscale*SCREENHEIGHT/2) ||
			b.y+y >= float64(currentYCoord)+float64(zoomscale*SCREENHEIGHT/2) {
			continue
		}
		for x := -b.radius; x < b.radius; x += zoomscale {
			if b.x+x < float64(currentXCoord)-float64(zoomscale*SCREENWIDTH/2) ||
				b.x+x >= float64(currentXCoord)+float64(zoomscale*SCREENWIDTH/2) {
				continue
			}

			if x*x+y*y < b.radius*b.radius {
				renderX := int32((b.x+x-currentXCoord)/zoomscale + SCREENWIDTH/2)
				renderY := int32((b.y+y-currentYCoord)/zoomscale + SCREENHEIGHT/2)
				setPixel(renderX, renderY, b.color)
			}
		}
	}
}
