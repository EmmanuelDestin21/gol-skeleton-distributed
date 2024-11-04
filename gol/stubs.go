package gol

var GOLHandler = "GOLOperations.Evolve"
var CurrentWorldStateHandler = "GOLOperations.CurrentWorldState"
var InitialiseBoardAndTurnHandler = "GOLOperations.InitialiseBoardAndTurn"
var PauseHandler = "GOLOperations.Pause"
var QuitHandler = "GOLOperations.Quit"

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
