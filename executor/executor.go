package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dop251/goja"
	"go.uber.org/zap"
)

var errExecutionTimeout = errors.New("execution timeout exceeded (200ms)")

const (
	maxExecutionTimeout = 200 * time.Millisecond
	maxCodeSize         = 10 * 1024       // 10KB
	maxCallStackSize    = 1024            // 1024 calls
	maxOutputSize       = 1 * 1024 * 1024 // 1MB
)

// Validation errors
var (
	ErrCodeEmpty    = errors.New("code parameter is required and cannot be empty")
	ErrCodeTooLarge = errors.New("code size exceeds maximum allowed size")
)

// ExecutionResponse represents a successful execution result
type ExecutionResponse struct {
	Success       bool    `json:"success"`
	Result        string  `json:"result"`
	ResultType    string  `json:"resultType"`
	ExecutionTime float64 `json:"executionTime"`
}

// ErrorResponse represents an error result
type ErrorResponse struct {
	Success    bool         `json:"success"`
	Error      *ErrorDetail `json:"error"`
	Suggestion string       `json:"suggestion,omitempty"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Type    string   `json:"type"`
	Message string   `json:"message"`
	Line    int      `json:"line,omitempty"`
	Column  int      `json:"column,omitempty"`
	Stack   []string `json:"stack,omitempty"`
}

// Executor executes JavaScript code in a sandboxed environment
type Executor struct{}

// NewExecutor creates a new Executor instance
func NewExecutor() *Executor {
	return &Executor{}
}

// Execute runs JavaScript code and returns JSON result string
func (e *Executor) Execute(ctx context.Context, code string) (string, error) {
	zap.S().Debugw("executing javascript", "code_size", len(code))

	if err := validateCode(code); err != nil {
		return marshalErrorResponse(&ErrorResponse{
			Success: false,
			Error: &ErrorDetail{
				Type:    "ValidationError",
				Message: err.Error(),
			},
		})
	}

	startTime := time.Now()
	value, err := executeJavaScript(ctx, code)
	executionTime := time.Since(startTime).Seconds() * 1000 // milliseconds

	if err != nil {
		resp := handleExecutionError(err)
		return marshalErrorResponse(resp)
	}

	resp := processResult(value, executionTime)
	jsonBytes, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		return "", fmt.Errorf("failed to marshal execution response: %w", marshalErr)
	}

	if len(jsonBytes) > maxOutputSize {
		return marshalErrorResponse(&ErrorResponse{
			Success: false,
			Error: &ErrorDetail{
				Type:    "OutputTooLargeError",
				Message: fmt.Sprintf("output size %d bytes exceeds maximum %d bytes", len(jsonBytes), maxOutputSize),
			},
			Suggestion: "Reduce the size of the returned value.",
		})
	}

	zap.S().Debugw("javascript execution completed",
		"execution_time_ms", executionTime,
		"result_type", resp.ResultType)

	return string(jsonBytes), nil
}

func validateCode(code string) error {
	if len(code) == 0 {
		return ErrCodeEmpty
	}
	if len(code) > maxCodeSize {
		return fmt.Errorf("%w: %d bytes (max %d bytes)", ErrCodeTooLarge, len(code), maxCodeSize)
	}
	return nil
}

type execResult struct {
	value goja.Value
	err   error
}

func executeJavaScript(ctx context.Context, code string) (goja.Value, error) {
	vm := goja.New()
	vm.SetMaxCallStackSize(maxCallStackSize)

	timeoutCtx, cancel := context.WithTimeout(ctx, maxExecutionTimeout)
	defer cancel()

	resultChan := make(chan execResult, 1) // buffered to prevent goroutine leak

	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- execResult{err: fmt.Errorf("panic: %v", r)}
			}
		}()
		value, err := vm.RunString(code)
		resultChan <- execResult{value: value, err: err}
	}()

	select {
	case result := <-resultChan:
		return result.value, result.err
	case <-timeoutCtx.Done():
		vm.Interrupt("execution timeout exceeded")
		<-resultChan // wait for goroutine to finish (prevent goroutine leak)
		return nil, errExecutionTimeout
	}
}

func processResult(value goja.Value, executionTime float64) *ExecutionResponse {
	if value == nil || goja.IsNull(value) || goja.IsUndefined(value) {
		return &ExecutionResponse{
			Success:       true,
			Result:        "null",
			ResultType:    "null",
			ExecutionTime: executionTime,
		}
	}

	exported := value.Export()
	var resultType string
	var resultString string

	switch v := exported.(type) {
	case int64, float64:
		resultType = "number"
		if fv, ok := v.(float64); ok {
			switch {
			case math.IsNaN(fv):
				resultString = "NaN"
			case math.IsInf(fv, 1):
				resultString = "Infinity"
			case math.IsInf(fv, -1):
				resultString = "-Infinity"
			case fv == float64(int64(fv)):
				resultString = fmt.Sprintf("%v", int64(fv))
			default:
				resultString = fmt.Sprintf("%v", fv)
			}
		} else {
			resultString = fmt.Sprintf("%v", v)
		}
	case string:
		resultType = "string"
		resultString = v
	case bool:
		resultType = "boolean"
		resultString = fmt.Sprintf("%v", v)
	case []interface{}:
		resultType = "array"
		jsonBytes, marshalErr := json.Marshal(v)
		if marshalErr != nil {
			zap.S().Warnw("failed to marshal array result", "error", marshalErr)
			resultString = fmt.Sprintf("[array: marshal error - %v]", marshalErr)
		} else {
			resultString = string(jsonBytes)
		}
	case map[string]interface{}:
		resultType = "object"
		jsonBytes, marshalErr := json.Marshal(v)
		if marshalErr != nil {
			zap.S().Warnw("failed to marshal object result", "error", marshalErr)
			resultString = fmt.Sprintf("[object: marshal error - %v]", marshalErr)
		} else {
			resultString = string(jsonBytes)
		}
	default:
		resultType = "unknown"
		resultString = fmt.Sprintf("%v", v)
	}

	return &ExecutionResponse{
		Success:       true,
		Result:        resultString,
		ResultType:    resultType,
		ExecutionTime: executionTime,
	}
}

func handleExecutionError(err error) *ErrorResponse {
	errMsg := err.Error()

	if errors.Is(err, errExecutionTimeout) {
		return &ErrorResponse{
			Success: false,
			Error: &ErrorDetail{
				Type:    "TimeoutError",
				Message: "Script execution timed out after 200ms",
			},
			Suggestion: "Execution exceeded 200ms. Check for infinite loops or overly complex calculations.",
		}
	}

	var stackOverflowErr *goja.StackOverflowError
	if errors.As(err, &stackOverflowErr) {
		return &ErrorResponse{
			Success: false,
			Error: &ErrorDetail{
				Type:    "StackOverflowError",
				Message: "Maximum call stack size exceeded (1024 calls)",
			},
			Suggestion: "Stack overflow occurred. Reduce recursion depth or use iterative approach.",
		}
	}

	var syntaxErr *goja.CompilerSyntaxError
	if errors.As(err, &syntaxErr) {
		return &ErrorResponse{
			Success: false,
			Error: &ErrorDetail{
				Type:    "SyntaxError",
				Message: err.Error(),
			},
			Suggestion: "JavaScript syntax error. Ensure code is ECMAScript 5.1 compatible (no let/const, arrow functions).",
		}
	}

	var exception *goja.Exception
	if errors.As(err, &exception) {
		stack := []string{}
		for _, frame := range exception.Stack() {
			pos := frame.Position()
			stack = append(stack, fmt.Sprintf(
				"%s at line %d, column %d",
				frame.FuncName(),
				pos.Line,
				pos.Column,
			))
		}

		errorType := "RuntimeError"
		if strings.Contains(exception.Error(), "SyntaxError") {
			errorType = "SyntaxError"
		}

		return &ErrorResponse{
			Success: false,
			Error: &ErrorDetail{
				Type:    errorType,
				Message: exception.Error(),
				Stack:   stack,
			},
			Suggestion: "Runtime error occurred. Check variable definitions and function calls.",
		}
	}

	return &ErrorResponse{
		Success: false,
		Error: &ErrorDetail{
			Type:    "UnknownError",
			Message: errMsg,
		},
		Suggestion: "An unexpected error occurred.",
	}
}

func marshalErrorResponse(resp *ErrorResponse) (string, error) {
	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		return fmt.Sprintf(`{"success":false,"error":{"type":"UnknownError","message":"%s"}}`, err.Error()), nil
	}
	return string(jsonBytes), nil
}
