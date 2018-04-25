package resputil

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

const (
	E_EMPTY_TRACE = "no stack trace available"
)

var config struct {
	Pretty bool // formats JSON output with indentation
	Trace  bool // prints a stack backtrace if exists (pkg/errors)
}

func Pretty(pretty bool) {
	config.Pretty = pretty
}

func Trace(trace bool) {
	config.Trace = trace
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

type successMessage struct {
	Success struct {
		Message string `json:"message"`
	} `json:"success"`
}

type errorMessage struct {
	Error struct {
		Message string `json:"message"`
		Trace   string `json:"trace,omitempty"`
	} `json:"error"`
}

// getTrace prints the first available stack trace if any
func getTrace(errs ...error) string {
	for _, err := range errs {
		if err != nil {
			terr, ok := err.(stackTracer)
			if ok {
				return fmt.Sprintf("%+v", terr.StackTrace())
			}
		}
	}
	return E_EMPTY_TRACE
}

// Error returns a structured error for API responses
func Error(err ...error) errorMessage {
	response := errorMessage{}
	// add stack trace to the response if available and enabled
	response.Error.Message = "Unknown error"
	if len(err) > 0 {
		if config.Trace {
			response.Error.Trace = getTrace(errors.Cause(err[0]), err[0])
		}
		response.Error.Message = err[0].Error()
	}
	return response
}

// Success returns a structured sucess message for API responses
func Success(success ...string) successMessage {
	response := successMessage{}
	response.Success.Message = "OK"
	if len(success) > 0 {
		response.Success.Message = success[0]
	}
	return response
}

// JSON responds with the first non-nil payload, formats error messages
func JSON(w http.ResponseWriter, responses ...interface{}) {
	respond := func(payload interface{}) {
		var result []byte
		var err error
		encode := func(payload interface{}) ([]byte, error) {
			if config.Pretty {
				return json.MarshalIndent(payload, "", "\t")
			}
			return json.Marshal(payload)
		}
		switch value := payload.(type) {
		case errorMessage:
			// main key is "error"
			result, err = encode(value)
		case successMessage:
			// main key is "success"
			result, err = encode(value)
		default:
			// main key is "response"
			result, err = encode(struct {
				Response interface{} `json:"response"`
			}{value})
		}
		if err != nil {
			result, _ = encode(Error(errors.WithStack(err)))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)
	}

	for _, response := range responses {
		switch value := response.(type) {
		case nil:
			continue
		case func() (interface{}, error):
			result, err := value()
			JSON(w, err, result)
		case func() error:
			err := value()
			if err == nil {
				continue
			}
			respond(Error(err))
		case error:
			respond(Error(value))
		case string:
			if value == "" {
				continue
			}
			respond(value)
		case bool:
			if !value {
				continue
			}
			respond(value)
		case errorMessage:
			respond(value)
		case successMessage:
			respond(value)
		default:
			respond(value)
		}
		// Exit on the first output...
		return
	}
	respond(false)
}
