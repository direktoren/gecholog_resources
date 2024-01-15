package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
)

// INPUT FIELD: ingress_payload (json)
// OUTPUT FIELD: character_count (json)

const (
	thisProcessorName = "charactercount"
)

type configuration struct {
	verbose bool

	natsServer  string
	natsToken   string
	natsSubject string

	log *log.Logger
}

var config configuration = configuration{
	verbose:     true,
	natsSubject: "coburn.gl.charactercount",
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
	nc, err := func() (*nats.Conn, error) {
		if config.natsToken != "" {
			return nats.Connect(config.natsServer, nats.Token(config.natsToken))
		}
		return nats.Connect(config.natsServer)
	}()

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
			response := []byte{}
			defer func() {
				msg.Respond(response)
				vLog(true, "Sending back: msg.Respond: %s\n", string(response)) // For verbosity

			}()

			// Unmarshal Json to a readable var
			var inputData map[string]json.RawMessage
			err := json.Unmarshal(msg.Data, &inputData)
			if err != nil {
				vLog(false, "Error: %v\n", err)
				return
			}

			// Process the data
			ingressPayload, ok := inputData["ingress_payload"]
			if !ok {
				vLog(false, "Error: %v\n", "ingress_payload not found")
				return
			}

			var outputData = make(map[string]json.RawMessage)
			outputData["character_count"] = []byte(strconv.Itoa(len(ingressPayload)))

			// Prepare response
			response, err = json.Marshal(&outputData)
			if err != nil {
				vLog(false, "Error: %v\n", err)
				return
			}

			// respond
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
