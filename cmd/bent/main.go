package main

import (
	"flag"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"

	"github.com/moolen/bent/envoy/api/v2"
	discovery "github.com/moolen/bent/envoy/service/discovery/v2"
	"github.com/moolen/bent/pkg/cache"
	"github.com/moolen/bent/pkg/provider"
	"github.com/moolen/bent/pkg/provider/fargate"
	"github.com/moolen/bent/pkg/provider/file"
	xds "github.com/moolen/bent/pkg/server"
)

var (
	providerType string
	providerImpl provider.ServiceProvider
	configFile   string
)

func main() {
	flag.StringVar(&providerType, "provider", "fargate", "set the provider, oneof [fargate,file]")
	flag.StringVar(&configFile, "config", "", "path to the configuration file")
	flag.Parse()

	var err error
	log.SetLevel(log.DebugLevel)
	config := cache.NewSnapshotCache(false)
	if providerType == "fargate" {
		providerImpl, err = fargate.NewProvider()
		if err != nil {
			panic(err)
		}
	}
	if providerType == "file" {
		providerImpl, err = file.NewProvider(configFile)
		if err != nil {
			panic(err)
		}
	}

	if providerImpl == nil {
		panic(fmt.Errorf("invalid provider: %s", providerType))
	}

	updater := provider.NewUpdater(config, providerImpl)
	server := xds.NewServer(config, nil)
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":50000")

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	go updater.Run()
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("error starting server: %s", err)
	}
}
