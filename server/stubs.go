package main

var GOLHandler = "GOLOperations.Evolve"
var CurrentWorldStateHandler = "GOLOperations.CurrentWorldState"
var InitialiseBoardAndTurnHandler = "GOLOperations.InitialiseBoardAndTurn"
var PauseHandler = "GOLOperations.Pause"
var QuitHandler = "GOLOperations.Quit"
var TerminateHandler = "GOLOperations.Terminate"

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
}

type EmptyResponse struct {
}

type EmptyRequest struct {
}
