package gol

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strconv"
	"sync"
	"time"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

var (
	closeTickerRoutine = make(chan bool)
	ensureOneTestMutex sync.Mutex
	keyPressMutex      sync.Mutex
	wg                 sync.WaitGroup
)

func makeCall(broker *rpc.Client, c distributorChannels, p Params, world [][]byte, keyPresses <-chan rune) *Response {
	request := Request{P: p, World: world}
	err1 := broker.Call(InitialiseBoardAndTurnHandler, request, new(EmptyResponse))
	if err1 != nil {
		panic(err1)
	}

	initialBoardRequest := new(EmptyRequest)
	initialBoardResponse := new(Response)
	err := broker.Call(CurrentWorldStateHandler, initialBoardRequest, initialBoardResponse)
	if err != nil {
		panic(err)
	}

	c.events <- StateChange{0, Executing}

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
					err := broker.Call(CurrentWorldStateHandler, req, currentWorldStateResponse)
					if err != nil {
						panic(err)
					}
					filename := fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, currentWorldStateResponse.Turn)
					saveImage(p, c, currentWorldStateResponse.FinalBoard, filename)
					c.ioCommand <- ioCheckIdle
					<-c.ioIdle
					c.events <- ImageOutputComplete{currentWorldStateResponse.Turn, filename}
					keyPressMutex.Unlock()
				case 'q':
					req := new(EmptyRequest)
					res := new(EmptyResponse)
					err := broker.Call(QuitHandler, req, res)
					if err != nil {
						panic(err)
					}
					return
				case 'k':
					// outputs final pgm image and shuts both client and server
					req := new(EmptyRequest)
					currentWorldStateResponse := new(Response)
					err := broker.Call(CurrentWorldStateHandler, req, currentWorldStateResponse)
					if err != nil {
						panic(err)
					}
					req = new(EmptyRequest)
					res := new(EmptyResponse)
					broker.Call(TerminateBrokerHandler, req, res)
					return
				case 'p':
					req := new(EmptyRequest)
					res := new(EmptyResponse)
					currentWorldStateResponse := new(Response)
					err2 := broker.Call(CurrentWorldStateHandler, req, currentWorldStateResponse)
					if err2 != nil {
						panic(err2)
					}
					paused = currentWorldStateResponse.Paused
					if !paused {
						err := broker.Call(PauseHandler, req, res)
						if err != nil {
							panic(err)
						}
						err2 := broker.Call(CurrentWorldStateHandler, req, currentWorldStateResponse)
						if err2 != nil {
							panic(err2)
						}
						c.events <- StateChange{currentWorldStateResponse.Turn, Paused}
						fmt.Println(currentWorldStateResponse.Turn)
					} else {
						c.events <- StateChange{currentWorldStateResponse.Turn, Executing}
						fmt.Println("Continuing")
						err := broker.Call(PauseHandler, req, res)
						if err != nil {
							panic(err)
						}
					}
				}
			}
		}
	}()

	go getCurrentAliveCells(c, p, world, broker)
	finalStateRequest := Request{P: p, World: world}
	finalStateResponse := new(Response)
	err2 := broker.Call(GOLHandler, finalStateRequest, finalStateResponse)

	if err2 != nil {
		panic(err2)
	}
	return finalStateResponse
}

func getCurrentAliveCells(c distributorChannels, p Params, world [][]byte, broker *rpc.Client) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Process the current state
			res := new(TickerResponse)
			req := Request{P: p, World: world}
			broker.Call(ReportAliveCellsHandler, req, res)
			AliveCellsCountEvent := AliveCellsCount{res.Turn, len(res.AliveCells)}
			c.events <- AliveCellsCountEvent
		case <-closeTickerRoutine:
			return
		}
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

func distributor(p Params, keyPresses <-chan rune, c distributorChannels) {
	ensureOneTestMutex.Lock()
	defer ensureOneTestMutex.Unlock()
	wg.Wait()
	world := createInitialBoard(p, c)

	// client side code
	var brokerAddress string
	if flag.Lookup("broker") == nil {
		brokerPtr := flag.String("broker", "localhost:8030", "IP:port string to connect to as broker")
		flag.Parse()
		brokerAddress = *brokerPtr
	} else {
		brokerAddress = flag.Lookup("broker").Value.String()
	}

	broker, err := rpc.Dial("tcp", brokerAddress)
	wg.Add(1)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer func() {
		broker.Close()
		wg.Done()
	}()

	response := makeCall(broker, c, p, world, keyPresses)

	if response.Quit || response.Terminated {
		req := new(EmptyRequest)
		res := new(Response)
		broker.Call(CurrentWorldStateHandler, req, res)
		filename := fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, res.Turn)
		saveImage(p, c, res.FinalBoard, filename)
		c.ioCommand <- ioCheckIdle
		<-c.ioIdle
		c.events <- ImageOutputComplete{res.Turn, filename}
		c.events <- StateChange{res.Turn, Quitting}
		closeTickerRoutine <- true
		close(c.events)
		return
	}

	// utilise the response
	res := new(TickerResponse)
	err = broker.Call(ReportAliveCellsHandler, new(EmptyRequest), res)
	if err != nil {
		panic(err)
	}
	aliveCells := res.AliveCells

	// Send the filename to write the image in.
	req2 := new(EmptyRequest)
	res2 := new(Response)
	err = broker.Call(CurrentWorldStateHandler, req2, res2)
	if err != nil {
		return
	}
	saveImage(p, c, res2.FinalBoard, fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, res2.Turn))

	// Report the final state using FinalTurnCompleteEvent.
	FinalTurnCompleteEvent := FinalTurnComplete{response.Turn, aliveCells}
	c.events <- FinalTurnCompleteEvent

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{response.Turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	closeTickerRoutine <- true
	close(c.events)
}
