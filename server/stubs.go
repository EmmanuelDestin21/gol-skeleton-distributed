package main

var GOLHandler = "GOLOperations.Evolve"
var CurrentWorldStateHandler = "GOLOperations.CurrentWorldState"
var InitialiseBoardAndTurnHandler = "GOLOperations.InitialiseBoardAndTurn"
var PauseHandler = "GOLOperations.Pause"
var QuitHandler = "GOLOperations.Quit"

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
}

type Request struct {
	P     Params
	World [][]byte
}

type EmptyResponse struct {
}

type EmptyRequest struct {
}
