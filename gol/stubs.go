package gol

var GOLHandler = "GOLOperations.Evolve"

type Response struct {
	FinalBoard [][]byte
	Turn       int
}

type Request struct {
	P     Params
	World [][]byte
}
