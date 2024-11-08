package main

import "uk.ac.bris.cs/gameoflife/util"

var (
	CalculateNextStateHandler = "GOLOperations.CalculateNextState"

	CurrentWorldStateHandler      = "Broker.CurrentWorldState"
	InitialiseBoardAndTurnHandler = "Broker.InitialiseBoardAndTurn"
	ReportAliveCellsHandler       = "Broker.ReportAliveCells"
	PauseHandler                  = "Broker.Pause"
	QuitHandler                   = "Broker.Quit"
	TerminateHandler              = "Broker.Terminate"
	GOLHandler                    = "Broker.Evolve"
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
	P            Params
	World        [][]byte
	ServerNumber int
} //gameboard

type EmptyResponse struct {
}

type EmptyRequest struct {
}

type TickerResponse struct {
	AliveCells []util.Cell
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
