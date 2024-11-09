package gol

import "uk.ac.bris.cs/gameoflife/util"

var (
	CalculateNextStateHandler = "GOLOperations.CalculateNextState"
	TerminateServerHandler    = "GOLOperations.Terminate"

	CurrentWorldStateHandler      = "Broker.CurrentWorldState"
	InitialiseBoardAndTurnHandler = "Broker.InitialiseBoardAndTurn"
	ReportAliveCellsHandler       = "Broker.ReportAliveCells"
	PauseHandler                  = "Broker.Pause"
	QuitHandler                   = "Broker.Quit"
	TerminateBrokerHandler        = "Broker.Terminate"
	GOLHandler                    = "Broker.Evolve"
)

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
	Turn       int
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
