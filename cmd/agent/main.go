package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/nireo/mci/agent"
	"github.com/nireo/mci/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	port := flag.Int("port", 8001, "the port to listen on")
	coreAddr := flag.String("core", "localhost:8080", "the address of the core server")
	flag.Parse()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("error starting listener")
	}
	defer ln.Close()

	conn, err := grpc.Dial(*coreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()
	log.Printf("Successfully connected to server at %s", *coreAddr)

	c := pb.NewCoreClient(conn)

	jobAgent := agent.NewAgent(c)
	agentSrv := agent.NewAgentServer(jobAgent)

	s := grpc.NewServer()
	pb.RegisterAgentServer(s, agentSrv)

	if err := s.Serve(ln); err != nil {
		log.Fatalf("failed to serve grpc")
	}
}
