package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// INPUT FIELD: egress_payload (json)
// OUTPUT FIELD: egress_payload.regex (json)

const (
	thisProcessorName = "regex"
)

type configuration struct {
	verbose bool

	matchJSON bool

	natsServer  string
	natsToken   string
	natsSubject string
	patterns    map[string]field

	log *log.Logger
}

type field struct {
	gjsonField string
	regex      string
}

var config configuration = configuration{
	verbose:     true,
	natsSubject: "coburn.gl.regex",
	matchJSON:   false,
	patterns: map[string]field{
		"default": field{
			// https://github.com/tidwall/gjson/blob/master/SYNTAX.md
			gjsonField: "egress_payload.choices.0.message.content",
			regex:      "```(?:md|markdown)\n(.*?)\n```",
		},
		"/markdown/": field{
			// https://github.com/tidwall/gjson/blob/master/SYNTAX.md
			gjsonField: "egress_payload.choices.0.message.content",
			regex:      "```(?:md|markdown)\n(.*?)\n```",
		},
		"/json/": field{
			// https://github.com/tidwall/gjson/blob/master/SYNTAX.md
			gjsonField: "egress_payload.choices.0.message.content",
			regex:      "```json\\n([\\s\\S]*?)\\n```",
		},
	},
}

// Response message structure from the processor
type section struct {
	Text   string          `json:"text"`
	Object json.RawMessage `json:"object"`
}
type regexpResponse struct {
	Match    bool      `json:"match"`
	Sections []section `json:"sections"`
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

			_, exists := config.patterns[string(glPath)]
			if !exists {
				// Use default if not defined
				glPath = "default"
			}

			// Extract the message
			extractMessage := gjson.Get(string(msg.Data), config.patterns[string(glPath)].gjsonField)
			message := extractMessage.String()

			re := regexp.MustCompile(config.patterns[string(glPath)].regex)
			matches := re.FindAllStringSubmatch(message, -1)

			processorResponse := regexpResponse{Sections: []section{}}
			for _, match := range matches {
				processorResponse.Match = true
				text := match[1]
				newSection := section{Text: text}

				// If matchJSON is true, try to add it as a json object
				if config.matchJSON && json.Valid([]byte(text)) {
					newSection.Object = json.RawMessage(text)
				}
				processorResponse.Sections = append(processorResponse.Sections, newSection)
				vLog(true, "newSection: %v\n", newSection)
			}

			/*processorResponseBytes, err := json.Marshal(&processorResponse)
			if err != nil {
				vLog(false, "Error: %v\n", err)
				return
			}*/

			// Use sjson to update the egress_payload by adding the regex response
			egressPayloadExtract := gjson.Get(string(msg.Data), "egress_payload")
			newEgressPayload, err := sjson.Set(string(egressPayloadExtract.Raw), "regex", &processorResponse)
			if err != nil {
				vLog(false, "Error: %v\n", err)
				return
			}

			var gechologData = make(map[string]json.RawMessage)
			gechologData["egress_payload"] = json.RawMessage(newEgressPayload)

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

	if os.Getenv("MATCH_JSON") != "" {
		config.matchJSON = true
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
