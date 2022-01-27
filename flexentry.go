package flexentry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/itchyny/gojq"
)

type Entrypoint struct {
	Executer

	mu    sync.Mutex
	query *gojq.Query
}

func (e *Entrypoint) Run(ctx context.Context) error {
	if e.isLambda() {
		log.Println("[debug] start lambda handler")
		lambda.Start(e.handleRequest)
		return nil
	}
	return e.Execute(ctx, os.Stdin, os.Args[1:]...)
}

func (e *Entrypoint) isLambda() bool {
	return strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") ||
		os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""
}

type Event interface{}

func (e *Entrypoint) Execute(ctx context.Context, stdin io.Reader, commands ...string) error {
	if e.Executer == nil {
		return nil
	}
	return e.Executer.Execute(ctx, stdin, commands...)
}

func (e *Entrypoint) handleRequest(ctx context.Context, event Event) error {
	commands, err := e.DetectCommand(ctx, event)
	if err != nil {
		log.Println("[error] ", err)
		return err
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(event); err != nil {
		log.Println("[error] failed event encode", err)
		return err
	}
	err = e.Execute(ctx, &buf, commands...)
	if err != nil {
		log.Println("[error] ", err)
		return err
	}
	return err
}

func (e *Entrypoint) DetectCommand(ctx context.Context, event Event) ([]string, error) {
	if command := os.Getenv("FLEXENTRY_COMMAND"); command != "" {
		return []string{command}, nil
	}
	if command, ok := event.(string); ok {
		return []string{command}, nil
	}
	if commands, ok := event.([]string); ok {
		return commands, nil
	}
	if data, ok := event.(map[string]interface{}); ok {
		query, err := e.getQuery()
		if err != nil {
			return nil, err
		}
		commands := make([]string, 0, 1)
		iter := query.RunWithContext(ctx, data)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				return nil, fmt.Errorf("command parse failed: %w", err)
			}
			if command, ok := v.(string); ok {
				commands = append(commands, command)
				continue
			}
			if cs, ok := v.([]string); ok {
				commands = append(commands, cs...)
				continue
			}
			if num, ok := v.(int); ok {
				commands = append(commands, strconv.Itoa(num))
			}
		}
		return commands, nil
	}

	return nil, errors.New("FLEXENTRY_COMMAND is required")
}

func (e *Entrypoint) getQuery() (*gojq.Query, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.query != nil {
		return e.query, nil
	}
	jqExpr := ".cmd"
	if j := os.Getenv("FLEXENTRY_COMMAND_JQ_EXPR"); j != "" {
		jqExpr = j
	}
	var err error
	e.query, err = gojq.Parse(jqExpr)
	if err != nil {
		return nil, err
	}
	return e.query, nil
}
