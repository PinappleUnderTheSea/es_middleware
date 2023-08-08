package main

import (
	"es_middleware/apis"
	"es_middleware/models"
	"github.com/rs/zerolog/log"
	"net"
	"net/rpc"
)

func main() {
	// Init es client
	models.Init()

	// Register
	apis.RegisterESMiddleware(apis.ESMiddlewareName, new(apis.Handler))

	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Err(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Err(err)
		}

		go rpc.ServeConn(conn)
	}
}
