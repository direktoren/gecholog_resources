package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
)

type configuration struct {
	natsServer  string
	natsToken   string
	natsSubject string
}

var config configuration = configuration{
	natsSubject: "coburn.gl.charactercount",
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
			responseBytes := []byte{}
			defer func() {
				msg.Respond(responseBytes)
				slog.Debug("sending back", slog.String("response", string(responseBytes)))

			}()

			// Unmarshal Json to a readable var
			var inputData map[string]json.RawMessage
			err := json.Unmarshal(msg.Data, &inputData)
			if err != nil {
				slog.Error("unmarshal error", slog.Any("error", err))
				return
			}

			// Process the data
			ingressPayload, ok := inputData["ingress_payload"]
			if !ok {
				slog.Error("ingress_payload not found")
				return
			}

			var outputData = make(map[string]json.RawMessage)
			outputData["character_count"] = []byte(strconv.Itoa(len(ingressPayload)))

			// Prepare response
			responseBytes, err = json.Marshal(&outputData)
			if err != nil {
				slog.Error("error marshalling response", slog.Any("error", err))
				return
			}

			// respond
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
