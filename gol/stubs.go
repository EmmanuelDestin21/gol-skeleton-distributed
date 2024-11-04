package gol

var GOLHandler = "GOLOperations.Evolve"
var CurrentWorldStateHandler = "GOLOperations.CurrentWorldState"
var InitialiseBoardAndTurnHandler = "GOLOperations.InitialiseBoardAndTurn"

type Response struct {
	FinalBoard [][]byte
	Turn       int
}

type Request struct {
	P     Params
	World [][]byte
}
