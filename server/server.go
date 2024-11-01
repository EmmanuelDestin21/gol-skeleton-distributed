package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"
)

type GOLOperations struct {
	CurrentWorld [][]byte
	CurrentTurn  *int
}

func (s *GOLOperations) Evolve(req Request, res *Response) (err error) {
	p := req.P
	s.CurrentWorld = req.World

	initialTurn := 0
	s.CurrentTurn = &initialTurn

	// Execute all turns of the Game of Life.
	for *s.CurrentTurn < p.Turns {
		s.CurrentWorld = calculateNextState(p, s.CurrentWorld)
		*s.CurrentTurn++
	}

	// Allow turn number and final board to be used by client
	res.Turn = *s.CurrentTurn
	res.FinalBoard = s.CurrentWorld

	return
}

func (s *GOLOperations) CurrentWorldState(world [][]byte, res *Response) (err error) {
	// Ensure FinalBoard is initialized before responding
	// Remember to add mutex lock
	if s.CurrentWorld == nil {
		s.CurrentWorld = world
	} else if s.CurrentTurn == nil {
		initialTurn := 0
		s.CurrentTurn = &initialTurn
	}

	res.FinalBoard = s.CurrentWorld
	res.Turn = *s.CurrentTurn
	return nil
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
