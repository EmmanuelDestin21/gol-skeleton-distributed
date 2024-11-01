package gol

var GOLHandler = "GOLOperations.Evolve"
var CurrentAliveCellsHandler = "GOLOperations.CurrentWorldState"

type Response struct {
	FinalBoard [][]byte
	Turn       int
}

type Request struct {
	P     Params
	World [][]byte
}
