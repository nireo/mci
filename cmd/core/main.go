package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/nireo/mci/core"
	"github.com/nireo/mci/pb"
	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 8001, "the port to listen on")
	agentServers := flag.String("agents", "", "list of agent servers comma separated")
	flag.Parse()

	addrs := strings.Split(*agentServers, ",")

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("error starting listener")
	}
	defer ln.Close()

	agentPool, err := core.NewAgentPool(addrs)
	if err != nil {
		log.Fatalf("failed to create agent pool: %s", err)
	}
	defer agentPool.CloseConnections()
	jm := core.NewJobManager()

	s := grpc.NewServer()
	pb.RegisterCoreServer(s, &core.Server{})

	go func() {
		if err := s.Serve(ln); err != nil {
			log.Fatalf("failed to serve grpc")
		}
	}()

	srv := core.NewServer("localhost:8070", 10*time.Second, jm, agentPool)
	srv.HttpServer.ListenAndServe()
}
