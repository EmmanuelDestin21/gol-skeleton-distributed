package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type GOLOperations struct {
	CurrentWorld [][]byte
	CurrentTurn  *int
}

var mutex sync.Mutex
var pauseMutex sync.Mutex
var pauseBool bool
var resumeSignal chan bool
var quitHappened = false
var clientConnected = false

func (s *GOLOperations) InitialiseBoardAndTurn(req Request, res *Response) (err error) {
	if quitHappened {
		pauseBool = false
		resumeSignal = make(chan bool)
	} else {
		s.CurrentWorld = req.World
		initialTurn := 0
		s.CurrentTurn = &initialTurn
		pauseBool = false
		resumeSignal = make(chan bool)
	}
	return
}

func (s *GOLOperations) Quit(req EmptyRequest, res *EmptyResponse) (err error) {
	quitHappened = true
	return
}

func (s *GOLOperations) Evolve(req Request, res *Response) (err error) {
	p := req.P

	// Execute all turns of the Game of Life.
	for *s.CurrentTurn < p.Turns {
		mutex.Lock()
		s.CurrentWorld = calculateNextState(p, s.CurrentWorld)
		*s.CurrentTurn++
		mutex.Unlock()
		pauseMutex.Lock()
		if quitHappened {
			return
		} else if pauseBool {
			pauseMutex.Unlock()
			<-resumeSignal
		} else {
			pauseMutex.Unlock()
		}
	}

	// Allow turn number and final board to be used by client
	res.Turn = *s.CurrentTurn
	res.FinalBoard = s.CurrentWorld

	return
}

func (s *GOLOperations) Pause(req EmptyRequest, res *EmptyResponse) (err error) {
	pauseMutex.Lock()
	pauseBool = !pauseBool
	pauseMutex.Unlock()
	if !pauseBool {
		resumeSignal <- true
	}
	return
}

func (s *GOLOperations) CurrentWorldState(req EmptyRequest, res *Response) (err error) {
	mutex.Lock()
	res.FinalBoard = s.CurrentWorld
	res.Turn = *s.CurrentTurn
	res.Paused = pauseBool
	mutex.Unlock()
	return
}

func calculateNextState(p Params, world [][]byte) [][]byte {
	newWorld := make([][]byte, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}

	IMHT := p.ImageHeight
	IMWD := p.ImageWidth

	for y := 0; y < IMHT; y++ {
		for x := 0; x < IMWD; x++ {
			// Calculate sum of 8 neighbors
			sum := int(world[(y+IMHT-1)%IMHT][(x+IMWD-1)%IMWD]/255) +
				int(world[(y+IMHT-1)%IMHT][(x+IMWD)%IMWD]/255) +
				int(world[(y+IMHT-1)%IMHT][(x+IMWD+1)%IMWD]/255) +
				int(world[(y+IMHT)%IMHT][(x+IMWD-1)%IMWD]/255) +
				int(world[(y+IMHT)%IMHT][(x+IMWD+1)%IMWD]/255) +
				int(world[(y+IMHT+1)%IMHT][(x+IMWD-1)%IMWD]/255) +
				int(world[(y+IMHT+1)%IMHT][(x+IMWD)%IMWD]/255) +
				int(world[(y+IMHT+1)%IMHT][(x+IMWD+1)%IMWD]/255)

			if world[y][x] == 255 {
				// Cell is alive
				if sum < 2 || sum > 3 {
					newWorld[y][x] = 0 // Underpopulation or overpopulation: cell dies
				} else {
					newWorld[y][x] = 255 // Cell survives
				}
			} else {
				// Cell is dead
				if sum == 3 {
					newWorld[y][x] = 255 // Reproduction: cell becomes alive
				} else {
					newWorld[y][x] = 0 // Cell remains dead
				}
			}
		}
	}
	return newWorld
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GOLOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
