package main

var GOLHandler = "GOLOperations.Evolve"
var CurrentAliveCellsHandler = "GOLOperations.CurrentWorldState"

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
