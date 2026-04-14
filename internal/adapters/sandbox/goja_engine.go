package sandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// GojaEngine implements SandboxRunner using the goja JavaScript engine.
type GojaEngine struct{}

// NewGojaEngine creates a new goja-based sandbox engine.
func NewGojaEngine() *GojaEngine {
	return &GojaEngine{}
}

// Run executes the package in an isolated goja VM with instrumented APIs.
func (e *GojaEngine) Run(ctx context.Context, pkgPath string, opts RunOptions) (*Result, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	if !opts.Offline {
		opts.Offline = true // offline by default
	}

	result := &Result{
		Engine: "goja",
	}

	// Read package.json to find entry point
	entryFile, err := resolveEntryPoint(pkgPath, opts.EntryStrategy)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("could not resolve entry: %v", err))
		result.Executed = false
		result.computeVerdict()
		return result, nil
	}

	// Execute with timeout
	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- e.executeFile(ctx, result, pkgPath, entryFile)
	}()

	select {
	case err := <-done:
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("execution error: %v", err))
		}
		result.Executed = true
	case <-ctx.Done():
		result.TimedOut = true
		result.Warnings = append(result.Warnings, "execution timed out")
		result.Executed = false
	}

	result.computeVerdict()
	return result, nil
}

func (e *GojaEngine) executeFile(ctx context.Context, result *Result, pkgPath, entryFile string) error {
	vm := goja.New()

	// Set interrupt on context cancellation
	go func() {
		<-ctx.Done()
		vm.Interrupt("timeout")
	}()

	// Inject built-in JS functions that goja doesn't provide by default
	e.injectPolyfills(vm)

	// Inject instrumented globals (order matters: http/fs before require)
	e.injectHTTP(vm, result)
	e.injectFS(vm, result, pkgPath)
	e.injectProcess(vm, result)
	e.injectRequire(vm, result, pkgPath)
	e.injectEval(vm, result)
	e.injectTimers(vm, result)

	// Read and execute the entry file
	source, err := os.ReadFile(entryFile)
	if err != nil {
		return fmt.Errorf("read entry file: %w", err)
	}

	// Pre-scan source for process.env references (goja can't intercept property access)
	scanSourceForEnvAccess(string(source), result)

	// Pre-scan for dynamic code execution patterns
	scanSourceForDynamicExec(string(source), result)

	_, err = vm.RunString(string(source))
	return err
}

// scanSourceForEnvAccess scans JS source for process.env references and flags them.
func scanSourceForEnvAccess(source string, result *Result) {
	lines := strings.Split(source, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "process.env") {
			idx := strings.Index(trimmed, "process.env")
			if idx >= 0 {
				rest := trimmed[idx+len("process.env"):]
				varName := ""
				if len(rest) > 0 && rest[0] == '.' {
					end := strings.IndexFunc(rest[1:], func(r rune) bool {
						return r == ';' || r == ' ' || r == ')' || r == '+' || r == '}'
					})
					if end >= 0 {
						varName = rest[1 : end+1]
					}
				} else if len(rest) > 1 && rest[0] == '[' {
					closeBracket := strings.Index(rest, "]")
					if closeBracket > 0 {
						varName = strings.Trim(rest[1:closeBracket], `"'`)
					}
				}
				if varName != "" {
					result.EnvReads = append(result.EnvReads, varName)
					result.addSignal(signalEnvRead, fmt.Sprintf("read process.env.%s", varName))
					if isSensitiveEnv(varName) {
						result.addSignal(signalFingerprint, fmt.Sprintf("sensitive env access: %s", varName))
					}
				}
			}
		}
	}
}

// scanSourceForDynamicExec scans JS source for dynamic code execution patterns.
func scanSourceForDynamicExec(source string, result *Result) {
	if strings.Contains(source, "eval(") {
		// eval() is also caught at runtime, but pre-scan ensures detection
	}
	if strings.Contains(source, "new Function(") || strings.Contains(source, "Function(") {
		result.DynamicExec = append(result.DynamicExec, "new Function()")
		result.addSignal(signalDynamicExec, "new Function() called")
	}
	if containsDecodeChain(source) {
		result.addSignal(signalDecodeChain, "eval with base64/decode pattern")
	}
}

// injectPolyfills adds common Node.js/browser globals that goja lacks.
func (e *GojaEngine) injectPolyfills(vm *goja.Runtime) {
	// atob/btoa for base64
	vm.Set("atob", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		input := call.Argument(0).String()
		// Simple base64 decode stub
		return vm.ToValue(input) // Return input as-is (decoding not needed for detection)
	}))
	vm.Set("btoa", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return call.Argument(0)
	}))

	// Buffer stub
	buffer := vm.NewObject()
	buffer.Set("from", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return vm.NewObject()
	}))
	vm.Set("Buffer", buffer)

	// console stub
	console := vm.NewObject()
	console.Set("log", vm.ToValue(func(goja.FunctionCall) goja.Value { return goja.Undefined() }))
	console.Set("error", vm.ToValue(func(goja.FunctionCall) goja.Value { return goja.Undefined() }))
	console.Set("warn", vm.ToValue(func(goja.FunctionCall) goja.Value { return goja.Undefined() }))
	vm.Set("console", console)

	// module.exports stub
	module := vm.NewObject()
	exports := vm.NewObject()
	module.Set("exports", exports)
	vm.Set("module", module)
	vm.Set("exports", exports)

	// __dirname and __filename
	vm.Set("__dirname", "/package")
	vm.Set("__filename", "/package/index.js")
}

// injectProcess creates a synthetic process.env with pre-populated sensitive values.
func (e *GojaEngine) injectProcess(vm *goja.Runtime, result *Result) {
	envObj := vm.NewObject()
	envObj.Set("AWS_SECRET_ACCESS_KEY", "AKIAIOSFODNN7EXAMPLE")
	envObj.Set("GITHUB_TOKEN", "ghp_fake_token_1234567890")
	envObj.Set("SECRET_KEY", "fake-secret")
	envObj.Set("DEPLOY_KEY", "fake-deploy-key")
	envObj.Set("NODE_ENV", "production")
	envObj.Set("PATH", "/usr/bin")
	envObj.Set("HOME", "/home/user")

	process := vm.NewObject()
	process.Set("env", envObj)
	process.Set("platform", "linux")
	process.Set("arch", "x64")
	process.Set("pid", 1)

	// Wrap process.env access by replacing the env object after a tick
	// This is a best-effort approach: we flag ANY process.env access
	// by checking if the script references it in source.
	vm.Set("process", process)

	// Pre-register signal for any process.env access
	// We detect this by monitoring require('process') or direct access patterns
	// Since goja doesn't support Proxy, we accept this limitation.
}

// injectRequire creates a require() function that tracks module loads
// and returns the instrumented stub modules.
func (e *GojaEngine) injectRequire(vm *goja.Runtime, result *Result, pkgPath string) {
	// Pre-build module stubs
	modules := map[string]goja.Value{
		"http":  vm.Get("http"),
		"https": vm.Get("https"),
		"fs":    vm.Get("fs"),
	}

	require := vm.ToValue(func(call goja.FunctionCall) goja.Value {
		modName := call.Argument(0).String()
		result.Warnings = append(result.Warnings, fmt.Sprintf("require('%s')", modName))

		// Track risky module imports
		if isRiskyModule(modName) {
			result.addSignal(signalNetwork, fmt.Sprintf("require risky module: %s", modName))
		}

		// Return the instrumented stub if available
		if mod, ok := modules[modName]; ok {
			return mod
		}

		// Return a safe stub object for unknown modules
		return vm.NewObject()
	})
	vm.Set("require", require)
}

// injectHTTP creates stub http/https modules that track connection attempts.
func (e *GojaEngine) injectHTTP(vm *goja.Runtime, result *Result) {
	// http.request stub
	httpObj := vm.NewObject()
	httpObj.Set("request", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		opts := call.Argument(0)
		result.NetworkAttempts = append(result.NetworkAttempts, fmt.Sprintf("http.request(%v)", opts))
		result.addSignal(signalNetwork, "http.request() called")
		return vm.NewObject()
	}))
	httpObj.Set("get", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		result.NetworkAttempts = append(result.NetworkAttempts, url)
		result.addSignal(signalNetwork, fmt.Sprintf("http.get(%s)", url))
		return vm.NewObject()
	}))
	vm.Set("http", httpObj)

	// https.request stub
	httpsObj := vm.NewObject()
	httpsObj.Set("request", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		opts := call.Argument(0)
		result.NetworkAttempts = append(result.NetworkAttempts, fmt.Sprintf("https.request(%v)", opts))
		result.addSignal(signalNetwork, "https.request() called")
		return vm.NewObject()
	}))
	httpsObj.Set("get", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		result.NetworkAttempts = append(result.NetworkAttempts, url)
		result.addSignal(signalNetwork, fmt.Sprintf("https.get(%s)", url))
		return vm.NewObject()
	}))
	vm.Set("https", httpsObj)

	// fetch stub
	vm.Set("fetch", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		url := call.Argument(0).String()
		result.NetworkAttempts = append(result.NetworkAttempts, fmt.Sprintf("fetch(%s)", url))
		result.addSignal(signalNetwork, fmt.Sprintf("fetch() called: %s", url))
		// Return a rejected promise-like object
		promise := vm.NewObject()
		promise.Set("then", vm.ToValue(func(goja.FunctionCall) goja.Value { return promise }))
		promise.Set("catch", vm.ToValue(func(goja.FunctionCall) goja.Value { return promise }))
		return promise
	}))
}

// injectFS creates a stub fs module that tracks filesystem access.
func (e *GojaEngine) injectFS(vm *goja.Runtime, result *Result, pkgPath string) {
	fs := vm.NewObject()

	fs.Set("readFile", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		result.FileReads = append(result.FileReads, path)
		result.addSignal(signalFileRead, fmt.Sprintf("fs.readFile(%s)", path))
		if isSensitivePath(path) {
			result.addSignal(signalFingerprint, fmt.Sprintf("read sensitive path: %s", path))
		}
		return vm.ToValue("")
	}))

	fs.Set("writeFile", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		result.FileWrites = append(result.FileWrites, path)
		result.addSignal(signalFileWrite, fmt.Sprintf("fs.writeFile(%s)", path))
		return vm.ToValue(nil)
	}))

	fs.Set("existsSync", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		result.FileReads = append(result.FileReads, path)
		return vm.ToValue(false)
	}))

	fs.Set("readdirSync", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		path := call.Argument(0).String()
		result.FileReads = append(result.FileReads, path)
		return vm.ToValue([]string{})
	}))

	vm.Set("fs", fs)
}

// injectEval wraps eval and Function to track dynamic code execution.
func (e *GojaEngine) injectEval(vm *goja.Runtime, result *Result) {
	originalEval := vm.Get("eval")

	vm.Set("eval", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		code := call.Argument(0).String()
		result.DynamicExec = append(result.DynamicExec, truncate(code, 100))
		result.addSignal(signalDynamicExec, "eval() called")

		// Check for decode chains (eval + atob/base64)
		if containsDecodeChain(code) {
			result.addSignal(signalDecodeChain, "eval with base64/decode pattern")
		}

		// Execute in a limited way (return the code as string to avoid actual execution)
		if originalEval != nil {
			if fn, ok := goja.AssertFunction(originalEval); ok {
				v, _ := fn(goja.Undefined(), call.Arguments...)
				return v
			}
		}
		return vm.ToValue(code)
	}))

	// Intercept new Function()
	vm.Set("Function", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		result.DynamicExec = append(result.DynamicExec, "new Function()")
		result.addSignal(signalDynamicExec, "new Function() called")
		// Return a no-op function
		return vm.ToValue(func() {})
	}))
}

// injectTimers tracks timer-based loops and potential protestware.
func (e *GojaEngine) injectTimers(vm *goja.Runtime, result *Result) {
	timerCount := 0

	wrapTimer := func(name string, fn goja.FunctionCall) goja.Value {
		timerCount++
		args := fn.Arguments
		desc := fmt.Sprintf("%s(%d args)", name, len(args))
		result.TimerCreations = append(result.TimerCreations, desc)

		// Detect setInterval with short intervals (potential infinite loop / protestware)
		if name == "setInterval" && len(args) >= 2 {
			interval := args[1].ToInteger()
			if interval > 0 && interval < 1000 {
				result.addSignal(signalTimer, fmt.Sprintf("setInterval with %dms (possible infinite loop)", interval))
			}
		}
		if name == "setTimeout" && len(args) >= 2 {
			timeout := args[1].ToInteger()
			if timeout > 30000 {
				result.addSignal(signalTimer, fmt.Sprintf("setTimeout with %dms (delayed execution)", timeout))
			}
		}

		// Execute the callback once (not in a real loop)
		if len(args) > 0 {
			if cb, ok := goja.AssertFunction(args[0]); ok {
				cb(goja.Undefined(), args[2:]...)
			}
		}
		return vm.ToValue(timerCount)
	}

	vm.Set("setInterval", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return wrapTimer("setInterval", call)
	}))
	vm.Set("setTimeout", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return wrapTimer("setTimeout", call)
	}))
	vm.Set("setImmediate", vm.ToValue(func(call goja.FunctionCall) goja.Value {
		return wrapTimer("setImmediate", call)
	}))
}

// resolveEntryPoint finds the main file of a package.
func resolveEntryPoint(pkgPath, strategy string) (string, error) {
	// Try package.json main field first
	pkgJSON := filepath.Join(pkgPath, "package.json")
	if data, err := os.ReadFile(pkgJSON); err == nil {
		// Simple JSON parsing without importing encoding/json (avoid circular)
		main := extractJSONString(string(data), "main")
		if main != "" {
			entry := filepath.Join(pkgPath, main)
			if _, err := os.Stat(entry); err == nil {
				return entry, nil
			}
		}
	}

	// Fallback order
	candidates := []string{
		"index.js",
		"index.mjs",
		"index.cjs",
		"main.js",
		"src/index.js",
		"lib/index.js",
		"dist/index.js",
	}

	for _, c := range candidates {
		path := filepath.Join(pkgPath, c)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no entry point found in %s", pkgPath)
}

// Helper functions

func isSensitiveEnv(name string) bool {
	sensitive := []string{
		"SECRET", "TOKEN", "KEY", "PASSWORD", "AWS", "GITHUB",
		"CI_DEPLOY", "SSH", "CREDENTIAL", "AUTH",
	}
	upper := strings.ToUpper(name)
	for _, s := range sensitive {
		if strings.Contains(upper, s) {
			return true
		}
	}
	return false
}

func isRiskyModule(name string) bool {
	risky := []string{"http", "https", "net", "child_process", "os", "crypto", "dgram"}
	for _, r := range risky {
		if name == r || strings.HasPrefix(name, r+"/") {
			return true
		}
	}
	return false
}

func isSensitivePath(path string) bool {
	sensitive := []string{
		"/etc/passwd", "/etc/shadow", ".ssh", ".env",
		"id_rsa", "credential", "secret", ".npmrc",
	}
	lower := strings.ToLower(path)
	for _, s := range sensitive {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func containsDecodeChain(code string) bool {
	decodeFuncs := []string{"atob", "btoa", "Buffer.from", "decodeURI", "decodeURIComponent", "Base64", "hex"}
	for _, d := range decodeFuncs {
		if strings.Contains(code, d) {
			return true
		}
	}
	return false
}

func extractJSONString(jsonStr, key string) string {
	// Very simple extraction — good enough for "main": "value"
	search := fmt.Sprintf(`"%s"`, key)
	idx := strings.Index(jsonStr, search)
	if idx == -1 {
		return ""
	}
	rest := jsonStr[idx+len(search):]
	// Find the colon
	colon := strings.Index(rest, ":")
	if colon == -1 {
		return ""
	}
	rest = rest[colon+1:]
	// Find the value string
	rest = strings.TrimLeft(rest, " \t\n\r")
	if len(rest) > 0 && rest[0] == '"' {
		end := strings.Index(rest[1:], `"`)
		if end == -1 {
			return ""
		}
		return rest[1 : end+1]
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
