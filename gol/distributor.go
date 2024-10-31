package gol

import (
	"flag"
	"log"
	"net/rpc"
	"strconv"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func makeCall(client *rpc.Client, p Params, world [][]byte) *Response {
	request := Request{P: p, World: world}
	response := new(Response)
	err := client.Call(GOLHandler, request, response)

	if err != nil {
		panic(err)
	}

	return response
}

func createInitialBoard(p Params, c distributorChannels) [][]byte {
	// Create a 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	// Request the filename and read the image.
	filename := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	// Populate the world array from the input.
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			world[y][x] = <-c.ioInput
		}
	}
	return world
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	world := createInitialBoard(p, c)

	// client side code
	var server string
	if flag.Lookup("server") == nil {
		serverPtr := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
		flag.Parse()
		server = *serverPtr
	} else {
		server = flag.Lookup("server").Value.String()
	}

	client, err := rpc.Dial("tcp", server)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	defer client.Close()

	response := makeCall(client, p, world)

	// utilise the response
	aliveCells := calculateAliveCells(p, response.FinalBoard)

	// Report the final state using FinalTurnCompleteEvent.
	FinalTurnCompleteEvent := FinalTurnComplete{response.Turn, aliveCells}
	c.events <- FinalTurnCompleteEvent

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{response.Turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	var aliveCells []util.Cell
	IMHT := p.ImageHeight
	IMWD := p.ImageWidth

	for y := 0; y < IMHT; y++ {
		for x := 0; x < IMWD; x++ {
			if world[y][x] == 255 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}
