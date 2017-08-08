// Package server DigitalRebar Provision Server
//
// An RestFUL API-driven Provisioner and DHCP server
//
// Terms Of Service:
//
// There are no TOS at this moment, use at your own risk we take no responsibility
//
//     Schemes: https
//     BasePath: /api/v3
//     Version: 0.1.0
//     License: APL https://raw.githubusercontent.com/digitalrebar/digitalrebar/master/LICENSE.md
//     Contact: Greg Althaus<greg@rackn.com> http://rackn.com
//
//     Security:
//       - basicAuth: []
//       - Bearer: []
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
// swagger:meta
package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/digitalrebar/provision"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/frontend"
	"github.com/digitalrebar/provision/midlayer"
	"github.com/digitalrebar/provision/plugin"
	"github.com/digitalrebar/store"
)

type ProgOpts struct {
	VersionFlag         bool   `long:"version" description:"Print Version and exit"`
	DisableTftpServer   bool   `long:"disable-tftp" description:"Disable TFTP server"`
	DisableProvisioner  bool   `long:"disable-provisioner" description:"Disable provisioner"`
	DisableDHCP         bool   `long:"disable-dhcp" description:"Disable DHCP server"`
	StaticPort          int    `long:"static-port" description:"Port the static HTTP file server should listen on" default:"8091"`
	TftpPort            int    `long:"tftp-port" description:"Port for the TFTP server to listen on" default:"69"`
	ApiPort             int    `long:"api-port" description:"Port for the API server to listen on" default:"8092"`
	DhcpPort            int    `long:"dhcp-port" description:"Port for the DHCP server to listen on" default:"67"`
	UnknownTokenTimeout int    `long:"unknown-token-timeout" description:"The default timeout in seconds for the machine create authorization token" default:"600"`
	KnownTokenTimeout   int    `long:"known-token-timeout" description:"The default timeout in seconds for the machine update authorization token" default:"3600"`
	OurAddress          string `long:"static-ip" description:"IP address to advertise for the static HTTP file server" default:"192.168.124.11"`

	BackEndType string `long:"backend" description:"Storage to use for persistent data. Can be either 'consul', 'directory', or a store URI" default:"directory"`

	BaseRoot   string `long:"base-root" description:"Base directory for other root dirs." default:"/var/lib/dr-provision"`
	DataRoot   string `long:"data-root" description:"Location we should store runtime information in" default:"digitalrebar"`
	PluginRoot string `long:"plugin-root" description:"Directory for plugins" default:"plugins"`
	LogRoot    string `long:"log-root" description:"Directory for job logs" default:"job-logs"`
	FileRoot   string `long:"file-root" description:"Root of filesystem we should manage" default:"tftpboot"`

	DevUI          string `long:"dev-ui" description:"Root of UI Pages for Development"`
	DhcpInterfaces string `long:"dhcp-ifs" description:"Comma-seperated list of interfaces to listen for DHCP packets" default:""`
	DefaultBootEnv string `long:"default-boot-env" description:"The default bootenv for the nodes" default:"sledgehammer"`
	UnknownBootEnv string `long:"unknown-boot-env" description:"The unknown bootenv for the system.  Should be \"ignore\" or \"discovery\"" default:"ignore"`

	DebugBootEnv  int    `long:"debug-bootenv" description:"Debug level for the BootEnv System - 0 = off, 1 = info, 2 = debug" default:"0"`
	DebugDhcp     int    `long:"debug-dhcp" description:"Debug level for the DHCP Server - 0 = off, 1 = info, 2 = debug" default:"0"`
	DebugRenderer int    `long:"debug-renderer" description:"Debug level for the Template Renderer - 0 = off, 1 = info, 2 = debug" default:"0"`
	TlsKeyFile    string `long:"tls-key" description:"The TLS Key File" default:"server.key"`
	TlsCertFile   string `long:"tls-cert" description:"The TLS Cert File" default:"server.crt"`
	DrpId         string `long:"drp-id" description:"The id of this Digital Rebar Provision instance" default:""`
}

func mkdir(d string, logger *log.Logger) {
	err := os.MkdirAll(d, 0755)
	if err != nil {
		logger.Fatalf("Error creating required directory %s: %v", d, err)
	}
}

func Server(c_opts *ProgOpts) {
	var err error

	logger := log.New(os.Stderr, "dr-provision", log.LstdFlags|log.Lmicroseconds|log.LUTC)

	if c_opts.VersionFlag {
		logger.Fatalf("Version: %s", provision.RS_VERSION)
	}
	logger.Printf("Version: %s\n", provision.RS_VERSION)

	// Make base root dir
	mkdir(c_opts.BaseRoot, logger)

	// Make other dirs as needed - adjust the dirs as well.
	if strings.IndexRune(c_opts.FileRoot, filepath.Separator) != 0 {
		c_opts.FileRoot = filepath.Join(c_opts.BaseRoot, c_opts.FileRoot)
	}
	if strings.IndexRune(c_opts.PluginRoot, filepath.Separator) != 0 {
		c_opts.PluginRoot = filepath.Join(c_opts.BaseRoot, c_opts.PluginRoot)
	}
	if strings.IndexRune(c_opts.DataRoot, filepath.Separator) != 0 {
		c_opts.DataRoot = filepath.Join(c_opts.BaseRoot, c_opts.DataRoot)
	}
	if strings.IndexRune(c_opts.LogRoot, filepath.Separator) != 0 {
		c_opts.LogRoot = filepath.Join(c_opts.BaseRoot, c_opts.LogRoot)
	}
	mkdir(c_opts.FileRoot, logger)
	mkdir(c_opts.PluginRoot, logger)
	mkdir(c_opts.DataRoot, logger)
	mkdir(c_opts.LogRoot, logger)
	var backendStore store.Store
	if u, err := url.Parse(c_opts.BackEndType); err == nil && u.Scheme != "" {
		backendStore, err = store.Open(c_opts.BackEndType)
	} else {
		storeURI := fmt.Sprintf("%s://%s", c_opts.BackEndType, c_opts.DataRoot)
		backendStore, err = store.Open(storeURI)
	}
	if err != nil {
		logger.Fatalf("Error using backing store %s: %v", c_opts.BackEndType, err)
	}
	// We have a backend, now get default assets
	logger.Printf("Extracting Default Assets\n")
	if err := ExtractAssets(c_opts.FileRoot); err != nil {
		logger.Fatalf("Unable to extract assets: %v", err)
	}

	services := make([]midlayer.Service, 0, 0)
	publishers := backend.NewPublishers(logger)

	dt := backend.NewDataTracker(backendStore,
		c_opts.FileRoot,
		c_opts.LogRoot,
		c_opts.OurAddress,
		c_opts.StaticPort,
		c_opts.ApiPort,
		logger,
		map[string]string{
			"debugBootEnv":        fmt.Sprintf("%d", c_opts.DebugBootEnv),
			"debugDhcp":           fmt.Sprintf("%d", c_opts.DebugDhcp),
			"debugRenderer":       fmt.Sprintf("%d", c_opts.DebugRenderer),
			"defaultBootEnv":      c_opts.DefaultBootEnv,
			"unknownBootEnv":      c_opts.UnknownBootEnv,
			"knownTokenTimeout":   fmt.Sprintf("%d", c_opts.KnownTokenTimeout),
			"unknownTokenTimeout": fmt.Sprintf("%d", c_opts.UnknownTokenTimeout),
		},
		publishers)

	// No DrpId - get a mac address
	if c_opts.DrpId == "" {
		intfs, err := net.Interfaces()
		if err != nil {
			logger.Fatalf("Error getting interfaces for DrpId: %v", err)
		}

		for _, intf := range intfs {
			if (intf.Flags & net.FlagLoopback) == net.FlagLoopback {
				continue
			}
			if (intf.Flags & net.FlagUp) != net.FlagUp {
				continue
			}
			if strings.HasPrefix(intf.Name, "veth") {
				continue
			}
			c_opts.DrpId = intf.HardwareAddr.String()
			break
		}
	}

	pc, err := plugin.InitPluginController(c_opts.PluginRoot, dt, logger, publishers, c_opts.ApiPort)
	if err != nil {
		logger.Fatalf("Error starting plugin service: %v", err)
	} else {
		services = append(services, pc)
	}

	fe := frontend.NewFrontend(dt, logger,
		c_opts.OurAddress, c_opts.ApiPort, c_opts.StaticPort, c_opts.FileRoot,
		c_opts.DevUI, nil, publishers, c_opts.DrpId, pc,
		c_opts.DisableDHCP, c_opts.DisableTftpServer, c_opts.DisableProvisioner)
	publishers.Add(fe)

	if _, err := os.Stat(c_opts.TlsCertFile); os.IsNotExist(err) {
		buildKeys(c_opts.TlsCertFile, c_opts.TlsKeyFile)
	}

	if !c_opts.DisableTftpServer {
		logger.Printf("Starting TFTP server")
		if svc, err := midlayer.ServeTftp(fmt.Sprintf(":%d", c_opts.TftpPort), dt.FS.TftpResponder(), logger, publishers); err != nil {
			logger.Fatalf("Error starting TFTP server: %v", err)
		} else {
			services = append(services, svc)
		}
	}

	if !c_opts.DisableProvisioner {
		logger.Printf("Starting static file server")
		if svc, err := midlayer.ServeStatic(fmt.Sprintf(":%d", c_opts.StaticPort), dt.FS, logger, publishers); err != nil {
			logger.Fatalf("Error starting static file server: %v", err)
		} else {
			services = append(services, svc)
		}
	}

	if !c_opts.DisableDHCP {
		logger.Printf("Starting DHCP server")
		if svc, err := midlayer.StartDhcpHandler(dt, c_opts.DhcpInterfaces, c_opts.DhcpPort, publishers); err != nil {
			logger.Fatalf("Error starting DHCP server: %v", err)
		} else {
			services = append(services, svc)
		}
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%d", c_opts.ApiPort), Handler: fe.MgmtApi}
	services = append(services, srv)

	go func() {
		// Handle SIGINT and SIGTERM.
		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		log.Println(<-ch)

		// Stop the service gracefully.
		for _, svc := range services {
			logger.Println("Shutting down server...")
			if err := svc.Shutdown(context.Background()); err != nil {
				logger.Printf("could not shutdown: %v", err)
			}
		}
	}()

	logger.Printf("Starting API server")
	if err = srv.ListenAndServeTLS(c_opts.TlsCertFile, c_opts.TlsKeyFile); err != http.ErrServerClosed {
		// Stop the service gracefully.
		for _, svc := range services {
			logger.Println("Shutting down server...")
			if err := svc.Shutdown(context.Background()); err != http.ErrServerClosed {
				logger.Printf("could not shutdown: %v", err)
			}
		}
		logger.Fatalf("Error running API service: %v\n", err)
	}
}
