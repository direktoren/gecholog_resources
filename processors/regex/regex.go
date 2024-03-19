package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type configuration struct {
	matchJSON bool

	natsServer  string
	natsToken   string
	natsSubject string
	patterns    map[string]field
}

type field struct {
	gjsonField string
	regex      string
}

var config configuration = configuration{
	natsSubject: "coburn.gl.regex",
	matchJSON:   true,
	patterns: map[string]field{
		"default": field{
			// https://github.com/tidwall/gjson/blob/master/SYNTAX.md
			gjsonField: "egress_payload.choices.0.message.content",
			regex:      "```(?:md|markdown)\\n([\\s\\S]*?)\\n```",
		},
		"/markdown/": field{
			// https://github.com/tidwall/gjson/blob/master/SYNTAX.md
			gjsonField: "egress_payload.choices.0.message.content",
			regex:      "```(?:md|markdown)\\n([\\s\\S]*?)\\n```",
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

			_, exists := config.patterns[string(glPath)]
			if !exists {
				// Use default if it exists
				if _, exists := config.patterns["default"]; !exists {
					slog.Debug("noop: gl_path not found")
					return
				}
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
				slog.Debug("newSection", slog.Any("newSection", newSection))
			}

			// Use sjson to update the egress_payload by adding the regex response
			egressPayloadExtract := gjson.Get(string(msg.Data), "egress_payload")
			newEgressPayload, err := sjson.Set(string(egressPayloadExtract.Raw), "regex", &processorResponse)
			if err != nil {
				slog.Error("problem setting regex field", slog.Any("error", err))
				return
			}

			var gechologData = make(map[string]json.RawMessage)
			gechologData["egress_payload"] = json.RawMessage(newEgressPayload)

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
