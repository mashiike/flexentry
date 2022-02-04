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

	mu           sync.Mutex
	commandQuery *gojq.Query
	environQuery *gojq.Query
}

func (e *Entrypoint) Run(ctx context.Context, args ...string) error {
	if e.isLambda() {
		log.Println("[debug] start lambda handler")
		lambda.Start(e.getHandler(args...))
		return nil
	}
	return e.Execute(ctx, &ExecuteOption{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}, args...)
}

func (e *Entrypoint) isLambda() bool {
	return strings.HasPrefix(os.Getenv("AWS_EXECUTION_ENV"), "AWS_Lambda") ||
		os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""
}

type Event interface{}

func (e *Entrypoint) Execute(ctx context.Context, opt *ExecuteOption, commands ...string) error {
	if e.Executer == nil {
		return nil
	}
	return e.Executer.Execute(ctx, opt, commands...)
}

func (e *Entrypoint) getHandler(args ...string) func(ctx context.Context, event Event) (interface{}, error) {
	return func(ctx context.Context, event Event) (interface{}, error) {
		commands, err := e.DetectCommand(ctx, event)
		if err != nil {
			log.Println("[error] ", err)
			return nil, err
		}
		opt := &ExecuteOption{
			Stderr: os.Stderr,
			Stdout: os.Stdout,
		}
		var bufInput, bufOutput bytes.Buffer
		if err := json.NewEncoder(&bufInput).Encode(event); err != nil {
			log.Println("[warn] failed event encode", err)
		} else {
			opt.Stdin = &bufInput
		}
		if os.Getenv("FLEXENTRY_FUNCTION_OUTPUT") == "enable" {
			opt.Stdout = io.MultiWriter(os.Stdout, &bufOutput)
		}
		executeCommand := make([]string, 0, len(args)+len(commands))
		executeCommand = append(executeCommand, args...)
		executeCommand = append(executeCommand, commands...)

		err = e.Execute(ctx, opt, executeCommand...)
		if err != nil {
			log.Println("[error] ", err)
			return nil, err
		}
		if bufOutput.Len() > 0 {
			var functionOutput interface{}
			if err := json.NewDecoder(&bufOutput).Decode(&functionOutput); err != nil {
				log.Println("[warn] failed output encode", err)
				return nil, nil
			}
			return functionOutput, nil
		}
		return nil, nil
	}
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
		query, err := e.getCommandQuery()
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

func (e *Entrypoint) DetectEnviron(ctx context.Context, event Event) ([]string, error) {
	if data, ok := event.(map[string]interface{}); ok {
		query, err := e.getEnvironQuery()
		if err != nil {
			return nil, err
		}
		environ := make([]string, 0, 1)
		iter := query.RunWithContext(ctx, data)
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				return nil, fmt.Errorf("environ parse failed: %w", err)
			}
			if e, ok := v.(string); ok {
				environ = MergeEnv(environ, []string{e})
				continue
			}
			if es, ok := v.([]string); ok {
				environ = MergeEnv(environ, es)
				continue
			}
			if es, ok := v.(map[string]string); ok {
				environ = MergeEnvWithMap(environ, es)
				continue
			}
		}
		return environ, nil
	}
	return []string{}, nil
}

func (e *Entrypoint) getCommandQuery() (*gojq.Query, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.commandQuery != nil {
		return e.commandQuery, nil
	}
	jqExpr := ".cmd"
	if j := os.Getenv("FLEXENTRY_COMMAND_JQ_EXPR"); j != "" {
		jqExpr = j
	}
	var err error
	e.commandQuery, err = gojq.Parse(jqExpr)
	if err != nil {
		return nil, err
	}
	return e.commandQuery, nil
}

func (e *Entrypoint) getEnvironQuery() (*gojq.Query, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.environQuery != nil {
		return e.environQuery, nil
	}
	jqExpr := ".env"
	if j := os.Getenv("FLEXENTRY_ENVIRON_JQ_EXPR"); j != "" {
		jqExpr = j
	}
	var err error
	e.environQuery, err = gojq.Parse(jqExpr)
	if err != nil {
		return nil, err
	}
	return e.environQuery, nil
}
