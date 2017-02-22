// Package Rocket Skates Server
//
// the purpose of this application is to provide an application
// that is using plain go code to define an API
//
// Terms Of Service:
//
// there are no TOS at this moment, use at your own risk we take no responsibility
//
//     Schemes: https
//     Host: localhost
//     BasePath: /api/v3
//     Version: 0.1.0
//     License: APL http://opensource.org/licenses/MIT
//     Contact: Greg Althaus<greg@rackn.com> http://rackn.com
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
package main

//go:generate swagger generate spec -o ../embedded/assets/swagger.json
//go:generate go-bindata -prefix ../embedded/assets -pkg embedded -o ../embedded/embed.go ../embedded/assets/...

import (
	"fmt"
	"log"
	"os"

	"github.com/digitalrebar/digitalrebar/go/common/cert"
	"github.com/digitalrebar/digitalrebar/go/common/client"
	"github.com/digitalrebar/digitalrebar/go/common/service"
	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/digitalrebar/go/common/version"
	consul "github.com/hashicorp/consul/api"
	"github.com/jessevdk/go-flags"
	"github.com/rackn/rocket-skates/backend"
	"github.com/rackn/rocket-skates/frontend"
)

var c_opts struct {
	VersionFlag    bool   `long:"version" description:"Print Version and exit"`
	BackEndType    string `long:"backend" description:"Storage backend to use. Can be either 'consul' or 'directory'" default:"consul"`
	DataRoot       string `long:"data-root" description:"Location we should store runtime information in" default:"digitalrebar/provisioner/boot-info"`
	StaticPort     int    `long:"static-port" description:"Port the static HTTP file server should listen on" default:"8091"`
	TftpPort       int    `long:"tftp-port" description:"Port for the TFTP server to listen on" default:"69"`
	ApiPort        int    `long:"api-port" description:"Port for the API server to listen on" default:"8092"`
	FileRoot       string `long:"file-root" description:"Root of filesystem we should manage" default:"/tftpboot"`
	OurAddress     string `long:"static-ip" description:"IP address to advertise for the static HTTP file server" default:"192.168.124.11"`
	CommandURL     string `long:"endpoint" description:"DigitalRebar Endpoint" env:"EXTERNAL_REBAR_ENDPOINT"`
	RegisterConsul bool   `long:"register-consul" description:"Register services with Consul"`
}

var logger *log.Logger

func main() {
	var err error

	logger = log.New(os.Stderr, "provisioner-mgmt", log.LstdFlags|log.Lmicroseconds|log.LUTC)

	parser := flags.NewParser(&c_opts, flags.Default)
	if _, err = parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	if c_opts.VersionFlag {
		logger.Fatalf("Version: %s", version.REBAR_VERSION)
	}
	logger.Printf("Version: %s\n", version.REBAR_VERSION)

	var consulClient *consul.Client
	if c_opts.RegisterConsul {
		consulClient, err = client.Consul(true)
		if err != nil {
			logger.Fatalf("Error talking to Consul: %v", err)
		}

		// Register service with Consul before continuing
		if err = service.Register(consulClient,
			&consul.AgentServiceRegistration{
				Name: "provisioner-service",
				Tags: []string{"deployment:system"},
				Port: c_opts.StaticPort,
				Check: &consul.AgentServiceCheck{
					HTTP:     fmt.Sprintf("http://[::]:%d/", c_opts.StaticPort),
					Interval: "10s",
				},
			},
			true); err != nil {
			log.Fatalf("Failed to register provisioner-service with Consul: %v", err)
		}

		if err = service.Register(consulClient,
			&consul.AgentServiceRegistration{
				Name: "provisioner-mgmt-service",
				Tags: []string{"revproxy"}, // We want to be exposed through the revproxy
				Port: c_opts.ApiPort,
				Check: &consul.AgentServiceCheck{
					HTTP:     fmt.Sprintf("http://[::]:%d/", c_opts.StaticPort),
					Interval: "10s",
				},
			},
			false); err != nil {
			log.Fatalf("Failed to register provisioner-mgmt-service with Consul: %v", err)
		}
		if err = service.Register(consulClient,
			&consul.AgentServiceRegistration{
				Name: "provisioner-tftp-service",
				Port: c_opts.TftpPort,
				Check: &consul.AgentServiceCheck{
					HTTP:     fmt.Sprintf("http://[::]:%d/", c_opts.StaticPort),
					Interval: "10s",
				},
			},
			true); err != nil {
			log.Fatalf("Failed to register provisioner-tftp-service with Consul: %v", err)
		}
	}

	var backendStore store.SimpleStore
	switch c_opts.BackEndType {
	case "consul":
		if consulClient == nil {
			consulClient, err = client.Consul(true)
			if err != nil {
				logger.Fatalf("Error talking to Consul: %v", err)
			}
		}
		backendStore, err = store.NewSimpleConsulStore(consulClient, c_opts.DataRoot)
	case "directory":
		backendStore, err = store.NewFileBackend(c_opts.DataRoot)
	case "memory":
		backendStore = store.NewSimpleMemoryStore()
		err = nil
	case "bolt", "local":
		backendStore, err = store.NewSimpleLocalStore(c_opts.DataRoot)
	default:
		logger.Fatalf("Unknown storage backend type %v\n", c_opts.BackEndType)
	}
	if err != nil {
		logger.Fatalf("Error using backing store %s: %v", c_opts.BackEndType, err)
	}

	dt := backend.NewDataTracker(backendStore, true, true)

	fe, err := frontend.NewFrontend(dt, "/api/v3", c_opts.FileRoot)
	if err != nil {
		logger.Fatal(err)
	}

	s, err := cert.Server("internal", "provisioner-mgmt-service")
	if err != nil {
		log.Fatalf("Error creating trusted server: %v", err)
	}
	s.Addr = fmt.Sprintf(":%d", c_opts.ApiPort)
	s.Handler = fe.MgmtApi

	go func() {
		if err = s.ListenAndServeTLS("", ""); err != nil {
			log.Fatalf("Error running API service: %v", err)
		}
	}()
	if err = frontend.ServeTftp(fmt.Sprintf(":%d", c_opts.TftpPort), c_opts.FileRoot); err != nil {
		log.Fatalf("Error starting TFTP server: %v", err)
	}
	// Static file server must always be last, as all our health checks key off of it.
	if err = frontend.ServeStatic(fmt.Sprintf(":%d", c_opts.StaticPort), c_opts.FileRoot); err != nil {
		log.Fatalf("Error starting static file server: %v", err)
	}
}
