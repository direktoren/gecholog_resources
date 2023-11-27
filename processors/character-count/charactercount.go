package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
)

// INPUT FIELD: ingress_payload (json)
// OUTPUT FIELD: character_count (json)

const (
	natsServer        = "nats://localhost:4222"
	natsSubject       = "coburn.gl.charactercount"
	thisProcessorName = "character_count"
)

var myLog *log.Logger

// ------------------------------- DO --------------------------------

// Connect to nats, do basic checks and call the process function
func do(ctx context.Context) {
	// Connect to NATS
	nc, err := nats.Connect(natsServer)
	if err != nil {
		myLog.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()

	// Subscribe to the nats subject. This is where we get requests to proces
	sub, err := nc.QueueSubscribe(
		natsSubject,
		"anything",
		func(msg *nats.Msg) {
			myLog.Printf("msg.Data: %s\n", string(msg.Data)) // For verbosity

			// Unmarshal Json to a readable var
			var inputData map[string]json.RawMessage
			err := json.Unmarshal(msg.Data, &inputData)
			if err != nil {
				myLog.Fatalf("Error: %v\n", err)
			}

			// Process the data
			ingressPayload, ok := inputData["ingress_payload"]
			if !ok {
				myLog.Fatalf("Error: %v\n", "ingress_payload not found")
			}

			var outputData = make(map[string]json.RawMessage)
			outputData["character_count"] = []byte(strconv.Itoa(len(ingressPayload)))

			// Prepare response
			response, err := json.Marshal(&outputData)
			if err != nil {
				myLog.Fatalf("Error: %v\n", err)
			}

			// respond
			msg.Respond(response)
			myLog.Printf("msg.Respond: %s\n", string(response)) // For verbosity
		},
	)
	if err != nil {
		myLog.Fatalf("Error subscribing to subject: %v\n", err)
	}
	defer sub.Unsubscribe()

	// Wait for messages
	myLog.Println("Listening for messages...")
	<-ctx.Done()
}

// ------------------------------- MAIN --------------------------------

// Set up possible configs, logger, context & cancel, capture ctrl-C and call do()
func main() {
	// Custom logger
	myLog = log.New(os.Stdout, thisProcessorName+": ", log.Ldate|log.Ltime|log.Lshortfile)

	// Create context & sync
	ctx, cancelFunction := context.WithCancel(context.Background())
	defer cancelFunction()

	go do(ctx)

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
