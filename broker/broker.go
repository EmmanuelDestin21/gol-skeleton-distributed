package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"strings"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type Broker struct {
	CurrentWorld [][]byte
	CurrentTurn  *int
}

var (
	server1               *rpc.Client
	server2               *rpc.Client
	server3               *rpc.Client
	server4               *rpc.Client
	allServers            []*rpc.Client
	serverMutex           sync.Mutex
	evolveMutex           sync.Mutex
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
	numberOfServers       = 4
)

func main() {
	serverAddresses := flag.String("serverAddresses", "localhost:8050", "server addresses to call")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	pClientAddr := flag.String("clientPort", "8030", "Port to listen for clients on")
	flag.Parse()

	// Create an RPC broker instance
	broker := rpc.NewServer()
	err := broker.Register(&Broker{})
	if err != nil {
		panic(err)
	}

	addresses := strings.Fields(*serverAddresses)
	// dial all servers
	for i, addr := range addresses {
		server, err := rpc.Dial("tcp", addr)
		if err != nil {
			panic(fmt.Sprintf("Failed to dial server %d: %v", i+1, err))
		}
		allServers = append(allServers, server)
	}
	fmt.Println("All servers connected")

	clientListener, err := net.Listen("tcp", ":"+*pClientAddr)
	fmt.Println("Ready to accept client")
	if err != nil {
		panic(err)
	}
	defer clientListener.Close()

	resumeSignal = make(chan bool)
	quitSignal = make(chan bool)
	terminateSignal = make(chan bool)
	terminateServerSignal = make(chan bool)

	// Channel to signal a new connection
	connChan := make(chan net.Conn)
	// Goroutine to handle accepting new connections
	go func() {
		for {
			conn, err := clientListener.Accept()
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
			for _, server := range allServers {
				err := server.Call(TerminateHandler, new(EmptyRequest), new(EmptyResponse))
				if err != nil {
					log.Fatal(err)
				}
			}
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
				go handleClientConnection(conn, broker)
			}
		}
	}
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

func (b *Broker) ReportAliveCells(req Request, res *TickerResponse) (err error) {
	serverMutex.Lock()
	res.AliveCells = calculateAliveCells(req.P, b.CurrentWorld)
	serverMutex.Unlock()
	return
}

func (b *Broker) InitialiseBoardAndTurn(req Request, res *Response) (err error) {
	if quitHappened {
		pauseBool = false
		quitHappened = false
		terminateHappened = false
	} else {
		b.CurrentWorld = req.World
		initialTurn := 0
		b.CurrentTurn = &initialTurn
		pauseBool = false
	}

	return
}

func (b *Broker) CurrentWorldState(req EmptyRequest, res *Response) (err error) {
	serverMutex.Lock()
	res.FinalBoard = b.CurrentWorld
	res.Turn = *b.CurrentTurn
	res.Paused = pauseBool
	serverMutex.Unlock()
	return
}

func sendWork(p Params, currentWorld [][]byte, resultsChannel chan<- [][]byte, server *rpc.Client) {
	req := Request{P: p, World: currentWorld}
	res := new(ServerSliceResponse)
	err := server.Call(CalculateNextStateHandler, req, res)
	if err != nil {
		log.Fatal(err)
	}
	resultsChannel <- res.Slice
}

func assembleNewWorld(resultsChannel []chan [][]byte, p Params) [][]byte {
	newWorld := make([][]byte, 0, p.ImageHeight)
	for i := 0; i < numberOfServers; i++ {
		newWorld = append(newWorld, <-resultsChannel[i]...)
	}
	return newWorld
}

func (b *Broker) Quit(req KeyPressed, res *EmptyResponse) (err error) {
	quitHappened = true
	quitSignal <- true
	return
}

func (b *Broker) Terminate(req KeyPressed, res *EmptyResponse) (err error) {
	terminateHappened = true
	terminateSignal <- true
	// handle terminating servers
	return
}

func (b *Broker) Pause(req KeyPressed, res *EmptyResponse) (err error) {
	pauseMutex.Lock()
	pauseBool = !pauseBool
	pauseMutex.Unlock()
	if !pauseBool {
		resumeSignal <- true
	}
	return
}

func (b *Broker) Evolve(req Request, res *Response) (err error) {
	p := req.P

	resultsChannel := make([]chan [][]byte, 4)
	for i := 0; i < 4; i++ {
		resultsChannel[i] = make(chan [][]byte)
	}

	// Execute all turns of the Game of Life.
	for *b.CurrentTurn < p.Turns {
		evolveMutex.Lock()
		// send work to servers
		for i, server := range allServers {
			go sendWork(p, b.CurrentWorld, resultsChannel[i], server)
		}

		b.CurrentWorld = assembleNewWorld(resultsChannel, p)
		*b.CurrentTurn++
		evolveMutex.Unlock()
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
	res.Turn = *b.CurrentTurn
	res.FinalBoard = b.CurrentWorld

	return
}
