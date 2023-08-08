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
	err := apis.RegisterESMiddleware(apis.ESMiddlewareName, new(apis.Handler))
	if err != nil {
		log.Err(err)
		return
	}

	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Err(err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Err(err)
			return
		}

		go rpc.ServeConn(conn)
	}
}
