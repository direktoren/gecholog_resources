package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"
)

type router struct {
	glPath    string
	disabled  bool
	errorTime time.Time
}

type configuration struct {
	natsServer  string
	natsToken   string
	natsSubject string

	ingressRouter   string
	outboundRouters []router

	disabledTime float64
	m            *sync.Mutex
}

var config configuration = configuration{
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
	disabledTime: 10, // 10 minutes default
	m:            &sync.Mutex{},
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

	// Preallocate `enabledIndex` with a capacity equal to the total number of routers.
	var enabledIndex = make([]int, 0, len(config.outboundRouters))

	// Subscribe to the nats subject. This is where we get requests to process
	sub, err := nc.QueueSubscribe(
		config.natsSubject,
		"anything",
		func(msg *nats.Msg) {

			slog.Debug("received", slog.String("data", string(msg.Data)))
			responseBytes := []byte{} // default response
			defer func() {
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

			// Check if the message is a response mesg. If error_code => Context Response
			errorCodeExtract := gjson.Get(string(msg.Data), "egress_status_code")
			errorCode := errorCodeExtract.Int()

			if errorCode > 0 {
				// It's in context response. Let's process the error_code
				if errorCode != 200 {
					for i, _ := range config.outboundRouters {
						if config.outboundRouters[i].glPath == glPath {
							config.m.Lock()
							config.outboundRouters[i].disabled = true
							config.outboundRouters[i].errorTime = time.Now()
							config.m.Unlock()
							slog.Warn("disabling router", slog.String("router", glPath), slog.Float64("minutes", config.disabledTime))
						}
					}
				}
				return // Response context completed
			}

			// It's Request context
			if glPath != config.ingressRouter {
				slog.Debug("ignoring request", slog.String("router", glPath))
				return
			}

			// Load balance

			// Reset the length of `enabledIndex` to 0, without affecting its capacity.
			enabledIndex = enabledIndex[:0]

			for i, _ := range config.outboundRouters {
				if config.outboundRouters[i].disabled {
					if time.Since(config.outboundRouters[i].errorTime).Minutes() > config.disabledTime {
						config.m.Lock()
						config.outboundRouters[i].disabled = false
						config.m.Unlock()
						slog.Warn("enabling router", slog.String("router", config.outboundRouters[i].glPath))
					}
				}
				if !config.outboundRouters[i].disabled {
					// Append to `enabledIndex` within its existing capacity.
					enabledIndex = append(enabledIndex, i)
				}
			}

			if len(enabledIndex) == 0 {
				slog.Error("no routers available")
				return
			}

			// Pick a router at random
			selectedRouterIndex := enabledIndex[rand.IntN(len(enabledIndex))]
			if selectedRouterIndex < 0 || selectedRouterIndex >= len(config.outboundRouters) {
				slog.Error("invalid router index", slog.Int("index", selectedRouterIndex))
				return
			}
			glPath = config.outboundRouters[selectedRouterIndex].glPath

			var gechologData = make(map[string]json.RawMessage)
			bytes, _ := json.Marshal(glPath)
			gechologData["gl_path"] = json.RawMessage(bytes)

			// Prepare response
			responseBytes, err = json.Marshal(&gechologData)
			if err != nil {
				slog.Error("error marshalling response", slog.Any("error", err))
				return
			}
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

	disabledTime := os.Getenv("DISABLED_TIME") // In minutes. Used to disable a router for a certain amount of time
	if disabledTime != "" {
		config.disabledTime, _ = strconv.ParseFloat(disabledTime, 64)
	}
	slog.Debug("disabledTime", slog.Float64("minutes", config.disabledTime))

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
