package gol

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strconv"
	"sync"
	"time"
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

func makeCall(client *rpc.Client, c distributorChannels, p Params, world [][]byte, keyPresses <-chan rune) *Response {
	var keyPressMutex sync.Mutex
	request := Request{P: p, World: world}
	response := new(Response)
	err1 := client.Call(InitialiseBoardAndTurnHandler, request, response)
	if err1 != nil {
		panic(err1)
	}
	//quit := false
	paused := false
	go func() {
		for {
			select {
			case key := <-keyPresses:
				switch key {
				case 's':
					keyPressMutex.Lock()
					req := new(EmptyRequest)
					currentWorldStateResponse := new(Response)
					err := client.Call(CurrentWorldStateHandler, req, currentWorldStateResponse)
					if err != nil {
						panic(err)
					}
					filename := fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, currentWorldStateResponse.Turn)
					saveImage(p, c, currentWorldStateResponse.FinalBoard, filename)
					fmt.Println()
					c.events <- ImageOutputComplete{currentWorldStateResponse.Turn, filename}
					keyPressMutex.Unlock()
				//case 'q':
				//	quit = true
				//	if paused {
				//		resumeSignal <- true
				//	}
				case 'p':
					req := new(EmptyRequest)
					res := new(EmptyResponse)
					err := client.Call(PauseHandler, req, res)
					if err != nil {
						panic(err)
					}
					currentWorldStateResponse := new(Response)
					err2 := client.Call(CurrentWorldStateHandler, req, currentWorldStateResponse)
					if err2 != nil {
						panic(err2)
					}
					paused = currentWorldStateResponse.Paused
					if paused {
						fmt.Println(currentWorldStateResponse.Turn)
					} else {
						fmt.Println("Continuing")
					}
				}
			}
		}
	}()

	go getCurrentAliveCells(c, p, world, client)
	err2 := client.Call(GOLHandler, request, response)

	if err2 != nil {
		panic(err2)
	}

	return response
}

func getCurrentAliveCells(c distributorChannels, p Params, world [][]byte, client *rpc.Client) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		req := new(EmptyRequest)
		response := new(Response)
		err := client.Call(CurrentWorldStateHandler, req, response)
		if err != nil {
			continue
		}
		// Process the current state
		aliveCells := calculateAliveCells(p, response.FinalBoard)
		AliveCellsCountEvent := AliveCellsCount{response.Turn, len(aliveCells)}
		c.events <- AliveCellsCountEvent
	}
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

func saveImage(p Params, c distributorChannels, world [][]byte, filename string) {
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, keyPresses <-chan rune, c distributorChannels) {
	world := createInitialBoard(p, c)

	// client side code
	var server string
	if flag.Lookup("server") == nil {
		serverPtr := flag.String("server", "localhost:8030", "IP:port string to connect to as server")
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

	c.events <- StateChange{0, Executing}

	response := makeCall(client, c, p, world, keyPresses)

	// utilise the response
	aliveCells := calculateAliveCells(p, response.FinalBoard)

	// Send the filename to write the image in.
	c.ioCommand <- ioOutput
	c.ioFilename <- strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.Turns)

	// Send the output world slice.
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- response.FinalBoard[y][x]
		}
	}

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
