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
}

var (
	currentWorld            [][]byte
	currentTurn             int
	imageWidth, imageHeight int
	allServers              []*rpc.Client
	evolveMutex             sync.Mutex
	pauseMutex              sync.Mutex
	clientConnectionMutex   sync.Mutex
	pauseBool               bool
	resumeSignal            chan bool
	quitSignal              chan bool
	terminateSignal         chan bool
	terminateBrokerSignal   chan bool
	quitHappened            = false
	terminateHappened       = false
	clientConnected         = false
	wg                      sync.WaitGroup
	numberOfServers         = 4
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
	terminateBrokerSignal = make(chan bool)

	// Channel to signal a new connection
	connChan := make(chan net.Conn)
	// Goroutine to handle accepting new connections
	go func() {
		for {
			conn, _ := clientListener.Accept()
			connChan <- conn
		}
	}()

	for {
		select {
		case <-terminateBrokerSignal:
			// Gracefully shut down the server
			fmt.Println("Waiting for client to shut down")
			wg.Wait()
			for _, server := range allServers {
				err := server.Call(TerminateServerHandler, new(EmptyRequest), new(EmptyResponse))
				if err != nil {
					log.Fatal(err)
				}
			}
			fmt.Println("Terminate signal received. Shutting down server...")
			return
		case connection := <-connChan:
			go handleClientConnection(connection, broker)
		}
	}
}

func handleClientConnection(connection net.Conn, server *rpc.Server) {
	if clientConnected {
		fmt.Println("A client is already connected. Waiting for space.")
	}
	clientConnectionMutex.Lock()
	wg.Add(1)
	clientConnected = true // Mark client as connected
	defer func() {
		fmt.Println("Client connection closed")
		clientConnected = false // Mark client as disconnected when done
		connection.Close()
		wg.Done()
		clientConnectionMutex.Unlock()
	}()
	// Serve the connected client.
	server.ServeConn(connection)
	fmt.Println("Client connected")
}

func calculateAliveCells() []util.Cell {
	var aliveCells []util.Cell

	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {
			if currentWorld[y][x] == 255 {
				aliveCells = append(aliveCells, util.Cell{X: x, Y: y})
			}
		}
	}
	return aliveCells
}

func (b *Broker) ReportAliveCells(req EmptyRequest, res *TickerResponse) (err error) {
	evolveMutex.Lock()
	res.AliveCells = calculateAliveCells()
	res.Turn = currentTurn
	evolveMutex.Unlock()
	return
}

func (b *Broker) InitialiseBoardAndTurn(req Request, res *EmptyResponse) (err error) {
	pauseBool = false
	quitHappened = false
	terminateHappened = false
	currentWorld = req.World
	currentTurn = 0
	imageWidth = req.P.ImageWidth
	imageHeight = req.P.ImageHeight
	return
}

func (b *Broker) CurrentWorldState(req EmptyRequest, res *Response) (err error) {
	evolveMutex.Lock()
	res.FinalBoard = currentWorld
	res.Turn = currentTurn
	res.Paused = pauseBool
	evolveMutex.Unlock()
	return
}

func sendWork(p Params, resultsChannel chan<- [][]byte, server *rpc.Client, serverNumber int) {
	req := Request{P: p, World: currentWorld, ServerNumber: serverNumber}
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
	terminateBrokerSignal <- true
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
	for currentTurn < p.Turns {
		evolveMutex.Lock()
		// send work to servers
		for i, server := range allServers {
			go sendWork(p, resultsChannel[i], server, i)
		}

		currentWorld = assembleNewWorld(resultsChannel, p)
		currentTurn++
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
	res.Turn = currentTurn
	res.FinalBoard = currentWorld

	return
}
