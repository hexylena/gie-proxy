package main

import (
	"os"
	"time"

	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "gie-proxy"
	app.Usage = "proxy for Galaxy GIEs"
	app.Version = "0.3.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listenAddr",
			Value: "0.0.0.0:8800",
			Usage: "address to listen on",
		},
		cli.StringFlag{
			Name:  "listenPath",
			Value: "/galaxy/gie_proxy",
			Usage: "path to listen on (for cookies)",
		},
		cli.StringFlag{
			Name:  "cookieName",
			Usage: "cookie name",
			Value: "galaxysession",
		},
		cli.StringFlag{
			Name:  "storage",
			Value: "./sessionMap.xml",
			Usage: "Session map file. Used to (re)store route lists across restarts",
		},
		cli.StringFlag{
			Name:  "apiKey",
			Value: "THE_DEFAULT_IS_NOT_SECURE",
			Usage: "Key to access to the API",
		},
		cli.IntFlag{
			Name:  "noAccess",
			Value: 60,
			Usage: "Length of time a proxy route must be unused before automatically being removed",
		},
		cli.StringFlag{
			Name:  "dockerAddr",
			Value: "unix:///var/run/docker.sock",
			Usage: "Endpoint at which we can access docker. No TLS Support yet",
		},
	}

	app.Action = func(c *cli.Context) {
		setupLogging()
		startServer(
			c.String("sessionMap"),
			c.String("dockerAddr"),
			c.String("cookieName"),
			c.String("listenAddr"),
			c.String("listenPath"),
			c.String("apiKey"),
			c.Int("noAccess"),
		)
	}
	app.Run(os.Args)
}

func startServer(sessionMap, dockerEndpoint, cookieName, listenAddr, listenPath, apiKey string, noAccessThreshold int) {
	// Logging

	log.Debug("Starting up")
	// Load up route mapping
	rm := &RouteMapping{
		Storage:           sessionMap,
		AuthCookieName:    cookieName,
		NoAccessThreshold: time.Second * time.Duration(noAccessThreshold),
		DockerEndpoint:    dockerEndpoint,
	}
	InitializeRouteMapper(rm)
	rm.Save()

	// Build the frontend
	f := &frontend{
		Addr:       listenAddr,
		Path:       listenPath,
		APIKey:     apiKey,
		CookieName: cookieName,
	}
	// Start our proxy
	log.Info("Starting frontend ...")
	f.Start(rm)
}
