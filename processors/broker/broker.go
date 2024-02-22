package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"
)

// INPUT FIELD: request.gl_path (string)
// OUTPUT FIELD: request.gl_path (string)

const (
	thisProcessorName = "broker"
)

type router struct {
	glPath    string
	disabled  bool
	errorTime time.Time
}

const (
	// 10 minute
	disableTime = 10
)

type configuration struct {
	verbose bool

	natsServer  string
	natsToken   string
	natsSubject string

	ingressRouter   string
	outboundRouters []router

	log *log.Logger
	m   *sync.Mutex
}

var config configuration = configuration{
	verbose:       true,
	natsSubject:   "coburn.gl.broker",
	ingressRouter: "/azure/",
	outboundRouters: []router{
		{
			glPath:   "/azure/gpt35turbo/",
			disabled: false,
		},
		{
			glPath:   "/azure/gpt4/",
			disabled: false,
		},
		{
			glPath:   "/azure/dud/", // This one will not work to illustrate the disable feature
			disabled: false,
		},
	},
	m: &sync.Mutex{},
}

// logger with verbose flag
func vLog(verbose bool, msg string, items ...any) {
	if verbose && !config.verbose {
		return
	}
	// Get caller info
	_, file, line, _ := runtime.Caller(1) // 1 means one level up in the call stack

	// Format the message with the file and line
	newMsg := file + ":" + strconv.Itoa(line) + ": " + msg

	// Log the message
	config.log.Printf(newMsg, items...)
}

// ------------------------------- DO --------------------------------

// Connect to nats, do basic checks and call the process function
func do(ctx context.Context, cancel context.CancelFunc) {

	// Connect to NATS
	opts := nats.GetDefaultOptions()
	opts.Url = config.natsServer
	opts.ReconnectWait = 3 * time.Second
	opts.MaxReconnect = -1 // Keep trying to reconnect
	if config.natsToken != "" {
		opts.Token = config.natsToken
	}
	opts.ReconnectedCB = func(nc *nats.Conn) {
		vLog(false, "Reconnected to NATS server!")
	}
	opts.DisconnectedErrCB = func(nc *nats.Conn, err error) {
		vLog(false, "Disconnected from NATS server: %v", err)
	}
	nc, err := opts.Connect()

	if err != nil {
		vLog(false, "Error connecting to NATS: %v", err)
		cancel()
		return
	}
	defer nc.Close()

	// Subscribe to the nats subject. This is where we get requests to process
	sub, err := nc.QueueSubscribe(
		config.natsSubject,
		"anything",
		func(msg *nats.Msg) {

			vLog(true, "Received msg.Data: %s\n", string(msg.Data))
			responseBytes := []byte{} // default response
			defer func() {
				msg.Respond(responseBytes)
				vLog(true, "Sending back: msg.Respond: %s\n", string(responseBytes)) // For verbosity
			}()

			// Figure out what router (gl_path) we are using
			glPathExtract := gjson.Get(string(msg.Data), "gl_path")
			glPath := glPathExtract.String()
			if glPath == "" {
				vLog(false, "Error: %v\n", "gl_path not found")
				return
			}

			// Check if the message is a response mesg. If error_code => Context Response
			errorCodeExtract := gjson.Get(string(msg.Data), "egress_status_code")
			errorCode := errorCodeExtract.Int()

			if errorCode > 0 {
				// It's in context response. Let's process the error_code
				if errorCode != 200 {
					for i, r := range config.outboundRouters {
						if r.glPath == glPath {
							config.m.Lock()
							config.outboundRouters[i].disabled = true
							config.outboundRouters[i].errorTime = time.Now()
							config.m.Unlock()
							vLog(true, "Disabling router '%s' for %d minutes.", glPath, disableTime)
						}
					}
				}
				return // Response context completed
			}

			// It's Request context
			if glPath != config.ingressRouter {
				vLog(false, "Router: %v\n", "gl_path not routed")
				return
			}

			// Load balance
			enabledRouters := []router{}
			for _, r := range config.outboundRouters {
				if r.disabled {
					if time.Since(r.errorTime).Minutes() > disableTime {
						config.m.Lock()
						r.disabled = false
						config.m.Unlock()
						vLog(true, "Enabling router '%s'", r.glPath)
					}
				}
				if !r.disabled {
					enabledRouters = append(enabledRouters, r)
				}
			}

			if len(enabledRouters) == 0 {
				vLog(false, "Error: No routers available\n")
				return
			}

			// Pick a router at random
			glPath = enabledRouters[rand.Intn(len(enabledRouters))].glPath

			var gechologData = make(map[string]json.RawMessage)
			bytes, _ := json.Marshal(glPath)
			gechologData["gl_path"] = json.RawMessage(bytes)

			// Prepare response
			responseBytes, err = json.Marshal(&gechologData)
			if err != nil {
				vLog(false, "Error: %v\n", err)
				return
			}
		},
	)
	if err != nil {
		vLog(false, "Error subscribing to subject: %v\n", err)
		cancel()
		return
	}
	defer sub.Unsubscribe()

	// Wait for messages
	vLog(false, "Connected to NATS server!")
	<-ctx.Done()
}

// ------------------------------- MAIN --------------------------------

// Set up possible configs, logger, context & cancel, capture ctrl-C and call do()
func main() {

	// Custom logger
	config.log = log.New(os.Stdout, thisProcessorName+": ", log.Ldate|log.Ltime)
	glHost := os.Getenv("GECHOLOG_HOST")
	if glHost == "" {
		glHost = "localhost"
	}
	config.natsServer = "nats://" + glHost + ":4222"
	config.natsToken = os.Getenv("NATS_TOKEN")

	// Create context & sync
	ctx, cancelFunction := context.WithCancel(context.Background())
	defer cancelFunction()

	go do(ctx, cancelFunction)

	// wait for ctrl-C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	select {

	case <-c:
		cancelFunction()
		time.Sleep(1 * time.Second) // Allow one second for everyone to cleanup
	case <-ctx.Done():
	}
}
