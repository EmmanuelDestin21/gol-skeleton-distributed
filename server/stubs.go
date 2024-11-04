package main

var GOLHandler = "GOLOperations.Evolve"
var CurrentWorldStateHandler = "GOLOperations.CurrentWorldState"
var InitialiseBoardAndTurnHandler = "GOLOperations.InitialiseBoardAndTurn"

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type Response struct {
	FinalBoard [][]byte
	Turn       int
}

type Request struct {
	P     Params
	World [][]byte
}
