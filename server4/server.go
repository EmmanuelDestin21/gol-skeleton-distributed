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
	terminateServerSignal = make(chan bool)
	clientConnected       = false
	wg                    sync.WaitGroup
	numberOfServers       = 4
)

func (s *GOLOperations) Terminate(req EmptyRequest, res *EmptyResponse) (err error) {
	terminateServerSignal <- true
	return
}

func (s *GOLOperations) CalculateNextState(req Request, res *ServerSliceResponse) (err error) {
	p := req.P
	world := req.World
	serverNumber := req.ServerNumber

	workerHeight := p.ImageHeight / numberOfServers
	startHeight := serverNumber * workerHeight
	endHeight := (serverNumber + 1) * workerHeight
	if serverNumber == numberOfServers-1 {
		endHeight += p.ImageHeight % numberOfServers
	}
	IMWD := p.ImageWidth

	res.Slice = make([][]byte, endHeight-startHeight)
	for i := range res.Slice {
		res.Slice[i] = make([]byte, IMWD)
	}

	for y := startHeight; y < endHeight; y++ {
		for x := 0; x < IMWD; x++ {
			// Calculate sum of 8 neighbors
			up := (y - 1 + p.ImageHeight) % p.ImageHeight
			left := (x - 1 + p.ImageWidth) % p.ImageWidth
			right := (x + 1) % p.ImageWidth
			down := (y + 1) % p.ImageHeight

			sum := int(world[up][left]) +
				int(world[up][x]) +
				int(world[up][right]) +
				int(world[y][left]) +
				int(world[y][right]) +
				int(world[down][left]) +
				int(world[down][x]) +
				int(world[down][right])

			if world[y][x] == 255 {
				// Cell is alive
				if sum < 2*255 || sum > 3*255 {
					res.Slice[y-startHeight][x] = 0 // Underpopulation or overpopulation: cell dies
				} else {
					res.Slice[y-startHeight][x] = 255 // Cell survives
				}
			} else {
				// Cell is dead
				if sum == 3*255 {
					res.Slice[y-startHeight][x] = 255 // Reproduction: cell becomes alive
				} else {
					res.Slice[y-startHeight][x] = 0 // Cell remains dead
				}
			}
		}
	}
	return
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
	pAddr := flag.String("port", "8080", "Port to listen on")
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

	// Channel to signal a new connection
	connChan := make(chan net.Conn)
	// Goroutine to handle accepting new connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
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
