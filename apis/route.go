package apis

import "net/rpc"

const ESMiddlewareName = "es"

func RegisterESMiddleware(name string, handler ESMiddlewareInterface) error {
	return rpc.RegisterName(name, handler)
}
