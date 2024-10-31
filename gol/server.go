package gol

import (
	"errors"
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"
)

type GOLOperations struct{}

func (s *SecretStringOperations) FastReverse(req stubs.Request, res *stubs.Response) (err error) {
	if req.Message == "" {
		err = errors.New("A message must be specified")
		return
	}

	res.Message = ReverseString(req.Message, 2)
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GOLOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
