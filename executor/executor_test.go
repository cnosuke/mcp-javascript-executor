package executor

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func setupTestLogger(t *testing.T) {
	t.Helper()
	logger := zaptest.NewLogger(t)
	zap.ReplaceGlobals(logger)
}

func parseResult(t *testing.T, jsonStr string) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	require.NoError(t, err, "failed to parse result JSON: %s", jsonStr)
	return result
}

func TestExecutor_BasicCalculation(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, "100 + 200")
	require.NoError(t, err)

	m := parseResult(t, result)
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "300", m["result"])
	assert.Equal(t, "number", m["resultType"])
	assert.NotNil(t, m["executionTime"])
}

func TestExecutor_MathFunctions(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	tests := []struct {
		code           string
		expectedResult string
		expectedType   string
	}{
		{"Math.sqrt(144)", "12", "number"},
		{"Math.pow(2, 10)", "1024", "number"},
		{"Math.floor(3.7)", "3", "number"},
		{"Math.ceil(3.2)", "4", "number"},
		{"Math.abs(-5)", "5", "number"},
		{"Math.max(1, 2, 3)", "3", "number"},
		{"Math.min(1, 2, 3)", "1", "number"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result, err := e.Execute(ctx, tt.code)
			require.NoError(t, err)
			m := parseResult(t, result)
			assert.Equal(t, true, m["success"])
			assert.Equal(t, tt.expectedResult, m["result"])
			assert.Equal(t, tt.expectedType, m["resultType"])
		})
	}
}

func TestExecutor_ArrayOperations(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, "[1,2,3].reduce(function(a,b){return a+b},0)")
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "6", m["result"])
	assert.Equal(t, "number", m["resultType"])
}

func TestExecutor_StringResult(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, `"hello " + "world"`)
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "hello world", m["result"])
	assert.Equal(t, "string", m["resultType"])
}

func TestExecutor_BooleanResult(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, "1 < 2")
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "true", m["result"])
	assert.Equal(t, "boolean", m["resultType"])
}

func TestExecutor_ObjectResult(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, `JSON.stringify({a:1, b:2})`)
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "string", m["resultType"])
}

func TestExecutor_NullResult(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, "null")
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "null", m["result"])
	assert.Equal(t, "null", m["resultType"])
}

func TestExecutor_SyntaxError(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, "if (")
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	assert.Equal(t, "SyntaxError", errObj["type"])
}

func TestExecutor_RuntimeError(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, "undefinedFunction()")
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	assert.Equal(t, "RuntimeError", errObj["type"])
}

func TestExecutor_Timeout(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	start := time.Now()
	result, err := e.Execute(ctx, "while(true){}")
	elapsed := time.Since(start)

	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	assert.Equal(t, "TimeoutError", errObj["type"])

	// Should complete within ~500ms (200ms timeout + overhead)
	assert.Less(t, elapsed, 500*time.Millisecond, "execution should time out quickly")
}

func TestExecutor_StackOverflow(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, "function f(){return f()} f()")
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	// Either StackOverflowError or TimeoutError is acceptable
	errType := errObj["type"].(string)
	assert.True(t, errType == "StackOverflowError" || errType == "TimeoutError",
		"expected StackOverflowError or TimeoutError, got %s", errType)
}

func TestExecutor_EmptyCode(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	result, err := e.Execute(ctx, "")
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	assert.Equal(t, "ValidationError", errObj["type"])
	assert.Contains(t, errObj["message"].(string), "required")
}

func TestExecutor_CodeTooLarge(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	// 11KB code
	largeCode := strings.Repeat("a", 11*1024)
	result, err := e.Execute(ctx, largeCode)
	require.NoError(t, err)
	m := parseResult(t, result)
	assert.Equal(t, false, m["success"])
	errObj := m["error"].(map[string]interface{})
	assert.Equal(t, "ValidationError", errObj["type"])
	assert.Contains(t, errObj["message"].(string), "exceeds")
}

func TestExecutor_NoGoroutineLeak(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	// Get baseline goroutine count
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Execute timeout-triggering code multiple times
	for i := 0; i < 5; i++ {
		result, err := e.Execute(ctx, "while(true){}")
		require.NoError(t, err)
		m := parseResult(t, result)
		assert.Equal(t, false, m["success"])
	}

	// Allow goroutines to settle
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	// Allow some slack for test infrastructure goroutines
	assert.LessOrEqual(t, finalGoroutines, baselineGoroutines+3,
		"goroutines should not leak: baseline=%d, final=%d", baselineGoroutines, finalGoroutines)
}

func TestExecutor_SpecialFloatValues(t *testing.T) {
	setupTestLogger(t)
	e := NewExecutor()
	ctx := context.Background()

	tests := []struct {
		code     string
		expected string
	}{
		{"1/0", "Infinity"},
		{"-1/0", "-Infinity"},
		{"0/0", "NaN"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result, err := e.Execute(ctx, tt.code)
			require.NoError(t, err)
			m := parseResult(t, result)
			assert.Equal(t, true, m["success"])
			assert.Equal(t, tt.expected, m["result"])
		})
	}
}
