package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"
)

type router struct {
	glPath             string
	responsePayload    json.RawMessage
	responseHeaders    json.RawMessage
	responseStatusCode json.RawMessage
}

type configuration struct {
	natsServer  string
	natsToken   string
	natsSubject string

	mockRouter      string
	recordedRouters map[string]router
	lambda          float64

	m *sync.Mutex
}

var config configuration = configuration{
	natsSubject:     "coburn.gl.mock",
	mockRouter:      "/mock/",
	recordedRouters: make(map[string]router, 10), // Best practice to allocate memory for the map
	lambda:          0,                           // default value
	m:               &sync.Mutex{},
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
		slog.Info("Reconnected to NATS server!")
	}
	opts.DisconnectedErrCB = func(nc *nats.Conn, err error) {
		slog.Warn("Disconnected from NATS server", slog.Any("error", err))
	}
	nc, err := opts.Connect()

	if err != nil {
		slog.Error("Error connecting to NATS", slog.Any("error", err))
		cancel()
		return
	}
	defer nc.Close()

	// Subscribe to the nats subject. This is where we get requests to process
	sub, err := nc.QueueSubscribe(
		config.natsSubject,
		"anything",
		func(msg *nats.Msg) {

			slog.Debug("received", slog.String("data", string(msg.Data)))
			responseBytes := []byte{} // default response

			defer func() {
				// Always end by sending back a response
				msg.Respond(responseBytes)
				slog.Debug("sending back", slog.String("response", string(responseBytes)))
			}()

			// Figure out what router (gl_path) we are using
			glPathExtract := gjson.Get(string(msg.Data), "gl_path")
			glPath := glPathExtract.String()
			if glPath == "" {
				slog.Error("gl_path not found")
				return
			}

			// Check if the message is in a response/request context. If ingress_subpath exists => Context Request
			ingressSubpathExtract := gjson.Get(string(msg.Data), "ingress_subpath")

			// ------------------ REQUEST CONTEXT ------------------

			if ingressSubpathExtract.Exists() {
				// It's a request context
				ingressSubpath := ingressSubpathExtract.String()

				// Check if its a request to the mock router
				if glPath != config.mockRouter {
					// It's not a request to the mock router
					// We ignore it
					return
				}

				// It's a request to the mock router
				// But we need a subpath to proceed
				if ingressSubpath == "" {
					slog.Warn("no subpath")
					return
				}

				// We store the ingress subpath in the control field
				// control field means request will not be forwarded
				// gecholog will write from control to inbound_payload and egress_payload
				var gechologData = make(map[string]string, 1)
				gechologData["control"] = "/" + ingressSubpath // Add leading slash

				// Prepare response
				responseBytes, err = json.Marshal(&gechologData)
				if err != nil {
					slog.Error("error marshalling response", slog.Any("error", err))
				}

				return
			}

			// ------------------ RESPONSE CONTEXT ------------------

			// It's a response context, this is where we record responses
			// Check if it's a response from the mock router
			if glPath == config.mockRouter {
				// It's a response from the mock router
				// Let's add the recorded payload & headers and send back

				// get the path we are mocking
				// This is what we wrote to the control field in the request context
				egressPayloadExtract := gjson.Get(string(msg.Data), "egress_payload")
				egressPayload := egressPayloadExtract.String()
				if egressPayload == "" {
					slog.Error("egress_payload not found")
					return

				}

				var recordedRouter *router = nil
				for paths, r := range config.recordedRouters {
					slog.Debug("checking path", slog.String("path", paths), slog.String("subpath", egressPayload))
					if strings.HasPrefix(egressPayload, paths) {
						// Found a match
						recordedRouter = &r
						break
					}
				}
				if recordedRouter == nil {
					slog.Warn("no mock router found")
					return
				}

				// Prepare response
				var gechologData = make(map[string]json.RawMessage, 3)
				gechologData["egress_payload"] = recordedRouter.responsePayload
				gechologData["egress_headers"] = recordedRouter.responseHeaders
				gechologData["egress_status_code"] = recordedRouter.responseStatusCode

				responseBytes, err = json.Marshal(&gechologData)
				if err != nil {
					slog.Error("error marshalling response", slog.Any("error", err))
				}

				// We simulate latency
				if config.lambda <= 0 {
					// No latency simulation
					return
				}
				sleepTime := int(rand.ExpFloat64()/config.lambda) * 100 // Exponential distribution in milliseconds
				slog.Debug("sleeping", slog.Int("sleepTime", sleepTime))
				time.Sleep(time.Duration(sleepTime) * time.Millisecond)

				return
			}

			// It's not to the mock router, let's record the payload & headers
			egressPayloadExtract := gjson.Get(string(msg.Data), "egress_payload")
			egressPayload := egressPayloadExtract.String()
			if egressPayload == "" {
				slog.Error("egress_payload not found")
				return

			}

			egressHeadersExtract := gjson.Get(string(msg.Data), "egress_headers")
			egressHeaders := egressHeadersExtract.String()
			if egressHeaders == "" {
				slog.Error("egress_headers not found")
				return
			}

			egressStatusCodeExtract := gjson.Get(string(msg.Data), "egress_status_code")
			egressStatusCode := egressStatusCodeExtract.String()
			if egressStatusCode == "" {
				slog.Error("egress_status_code not found")
				return
			}

			// Store the response
			slog.Debug("storing response", slog.String("gl_path", glPath))
			config.m.Lock() // mutex lock since maps are not thread safe for writing
			config.recordedRouters[glPath] = router{
				glPath:             glPath,
				responsePayload:    json.RawMessage(egressPayload),
				responseHeaders:    json.RawMessage(egressHeaders),
				responseStatusCode: json.RawMessage(egressStatusCode),
			}
			config.m.Unlock()

		},
	)
	if err != nil {
		slog.Error("error subscribing to subject", slog.Any("error", err))
		cancel()
		return
	}
	defer sub.Unsubscribe()

	// Wait for messages
	slog.Info("Connected to NATS server!")
	<-ctx.Done()
}

// ------------------------------- MAIN --------------------------------

// Set up possible configs, logger, context & cancel, capture ctrl-C and call do()
func main() {

	// Set logger level
	slog.SetLogLoggerLevel(slog.LevelDebug)

	glHost := os.Getenv("GECHOLOG_HOST")
	if glHost == "" {
		glHost = "localhost"
	}
	config.natsServer = "nats://" + glHost + ":4222"
	config.natsToken = os.Getenv("NATS_TOKEN")

	lambda := os.Getenv("LAMBDA") // Used for latency simulation
	if lambda != "" {
		config.lambda, _ = strconv.ParseFloat(lambda, 64)
	}

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
