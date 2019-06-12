package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"log"
	"net"

	pb "../proto"
)

var port = flag.Int("port", 10000, "the port to serve on")

type server struct{}

func clientIP(ctx context.Context) (string, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("[getClinetIP] invoke FromContext() failed")
	}
	if pr.Addr == net.Addr(nil) {
		return "", fmt.Errorf("[getClientIP] peer.Addr is nil")
	}

	return pr.Addr.String(), nil
}

func getClientIP(ctx context.Context) string {
	ip, err := clientIP(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	return ip
}

func (s *server) Receive(ctx context.Context, in *pb.Request) (*pb.Reply, error) {
	ip := getClientIP(ctx)
	log.Println("Login service called from: ", ip)
	name := in.Name
	return &pb.Reply{Msg: "Hello " + name}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Server is listening at %v\n", lis.Addr())

	s := grpc.NewServer()

	pb.RegisterEchoServer(s, &server{})

	reflection.Register(s)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
