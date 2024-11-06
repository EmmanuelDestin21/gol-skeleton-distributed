package main

import (
	"flag"
	"fmt"
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

var (
	mutex                 sync.Mutex
	pauseMutex            sync.Mutex
	pauseBool             bool
	resumeSignal          chan bool
	quitSignal            chan bool
	terminateSignal       chan bool
	terminateServerSignal chan bool
	quitHappened          = false
	terminateHappened     = false
	clientConnected       = false
	wg                    sync.WaitGroup
)

func (s *GOLOperations) InitialiseBoardAndTurn(req Request, res *Response) (err error) {
	if quitHappened {
		pauseBool = false
		quitHappened = false
		terminateHappened = false
	} else {
		s.CurrentWorld = req.World
		initialTurn := 0
		s.CurrentTurn = &initialTurn
		pauseBool = false
	}

	return
}

func (s *GOLOperations) Quit(req EmptyRequest, res *EmptyResponse) (err error) {
	quitHappened = true
	quitSignal <- true
	return
}

func (s *GOLOperations) Terminate(req EmptyRequest, res *EmptyResponse) (err error) {
	terminateHappened = true
	terminateSignal <- true
	terminateServerSignal <- true
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
		if terminateHappened {
			res.Terminated = true
			<-terminateSignal
			pauseMutex.Unlock()
			return
		} else if quitHappened {
			res.Quit = true
			<-quitSignal
			pauseMutex.Unlock()
			return
		} else if pauseBool {
			pauseMutex.Unlock()
			select {
			case <-resumeSignal:
				continue
			case <-quitSignal:
				res.Quit = true
				return
			case <-terminateSignal:
				res.Terminated = true
				return
			}
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

func handleClientConnection(conn net.Conn, server *rpc.Server) {
	clientConnected = true // Mark client as connected
	defer func() {
		fmt.Println("Client connection closed")
		clientConnected = false // Mark client as disconnected when done
		conn.Close()
		wg.Done()
	}()

	// Serve the connected client.
	server.ServeConn(conn)
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	// Create an RPC server instance
	server := rpc.NewServer()
	server.Register(&GOLOperations{})

	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	resumeSignal = make(chan bool)
	quitSignal = make(chan bool)
	terminateSignal = make(chan bool)
	terminateServerSignal = make(chan bool)

	// Channel to signal a new connection
	connChan := make(chan net.Conn)
	// Goroutine to handle accepting new connections
	go func() {
		for {
			conn, _ := listener.Accept()
			wg.Add(1)
			connChan <- conn
		}
	}()

	for {
		select {
		case <-terminateServerSignal:
			// Gracefully shut down the server
			fmt.Println("Waiting for client to shut down")
			wg.Wait()
			fmt.Println("Terminate signal received. Shutting down server...")
			return
		case conn := <-connChan:
			// Check if a client is already connected
			if clientConnected {
				// Print error and close the connection
				fmt.Println("A client is already connected. Rejecting new connection attempt.")
				conn.Close()
			} else {
				// Handle client connection
				fmt.Println("Client connected")
				go handleClientConnection(conn, server)
			}
		}
	}
}
