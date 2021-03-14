package server

type RPCArgs string

type RPCReply struct {
	Success bool
	Message string
}

type RPC struct{}

func (r *RPC) Load(args *RPCArgs, reply *RPCReply) error {
	reply.Message = "ok, return"
	return nil
}
