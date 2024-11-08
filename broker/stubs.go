package main

import "uk.ac.bris.cs/gameoflife/util"

var (
	GOLHandler                = "GOLOperations.Evolve"
	CurrentWorldStateHandler  = "GOLOperations.CurrentWorldState"
	CalculateNextStateHandler = "GOLOperations.CalculateNextState"

	ReceiveServerAddressHandler   = "Broker.ReceiveServerAddress"
	InitialiseBoardAndTurnHandler = "Broker.InitialiseBoardAndTurn"
	ReportAliveCellsHandler       = "Broker.ReportAliveCells"
	PauseHandler                  = "Broker.Pause"
	QuitHandler                   = "Broker.Quit"
	TerminateHandler              = "Broker.Terminate"
)

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type Response struct {
	FinalBoard [][]byte
	Turn       int
	Paused     bool
	Quit       bool
	Terminated bool
}

type Request struct {
	P     Params
	World [][]byte
} //gameboard

type EmptyResponse struct {
}

type EmptyRequest struct {
}

type TickerResponse struct {
	aliveCells []util.Cell
}

type KeyPressed struct {
	Key rune
}

type ServerSliceResponse struct {
	Slice [][]byte
}

type ServerAddress struct {
	Address string
}

type Test struct {
	Worked bool
}
