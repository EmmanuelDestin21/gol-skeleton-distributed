package stubs

var GOLHandler = "GOLOperations.Evolve"

type Response struct {
	FinalBoard [][]byte
}

type Request struct {
	InitialBoard [][]byte
	Turns        int
	Threads      int
	ImageWidth   int
	ImageHeight  int
}
