package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq"
	"github.com/rakyll/statik/fs"
	"github.com/techschool/simplebank/api"
	db "github.com/techschool/simplebank/db/sqlc"
	_ "github.com/techschool/simplebank/doc/statik"
	"github.com/techschool/simplebank/gapi"
	"github.com/techschool/simplebank/pb"
	"github.com/techschool/simplebank/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

// TODO https://github.com/golangci/golangci-lint
func main() {
	config, err := util.LoadConfig(".", "app")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	// conn, err := sql.Open(config.DBDriver, config.DBSource)
	conn, err := openDBConnection(config)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	db.RunDBMigration(config.MigrationURL, config.DBSource)

	store := db.NewStore(conn)
	// TODO test
	// go runGatewayServer(config, store)
	// runGrpcServer(config, store)
	runGinServer(config, store)
}

// https://medium.com/@kelseyhightower/12-fractured-apps-1080c73d481c
// docker run --name postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -d postgres:14-alpine
func openDBConnection(config util.Config) (*sql.DB, error) {
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Println("cannot open db connection:", err)
		return nil, err
	}

	var dbError error
	maxAttempts := 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		dbError = conn.Ping()
		if dbError == nil {
			break
		}
		log.Printf("cannot connect to db (%d): %s\n", attempt, dbError)
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return conn, dbError
}

func runGrpcServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterSimpleBankServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("cannot create listener:", err)
	}

	log.Printf("start gRPC server at %s", listener.Addr().String())
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("cannot start gRPC server:", err)
	}
}

func runGatewayServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = pb.RegisterSimpleBankHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal("cannot register handler server:", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal("cannot create statik fs:", err)
	}

	swaggerHandler := http.StripPrefix("/swagger/", http.FileServer(statikFS))
	mux.Handle("/swagger/", swaggerHandler)

	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot create listener:", err)
	}

	log.Printf("start HTTP gateway server at %s", listener.Addr().String())
	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal("cannot start HTTP gateway server:", err)
	}
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
