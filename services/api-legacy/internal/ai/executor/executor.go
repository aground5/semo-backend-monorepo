package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"semo-server/configs"
)

// channelWriter implements io.Writer interface for writing directly to a channel
type channelWriter struct {
	ch  chan<- string
	ctx context.Context
}

// Write sends the data to the channel as a string
func (w channelWriter) Write(p []byte) (n int, err error) {
	select {
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	default:
		w.ch <- string(p)
		return len(p), nil
	}
}

// AIExecutorRequest represents the input to be sent to the AI executor
type AIExecutorRequest struct {
	PromptName    string                 `json:"promptName"`
	Variables     map[string]interface{} `json:"variables"`
	Temperature   float64                `json:"temperature,omitempty"`
	Model         string                 `json:"model,omitempty"`
	TraceId       string                 `json:"traceId,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
	UserId        string                 `json:"userId,omitempty"`
	SessionId     string                 `json:"sessionId,omitempty"`
	LineByLine    bool                   `json:"lineByLine,omitempty"` // 한 줄씩 출력 여부를 결정하는 옵션
}

// AIExecutorResponse represents the AI output and any errors
type AIExecutorResponse struct {
	Output     string
	Errors     []string
	ExecError  error
	Terminated bool
}

// AIExecutor handles the execution of the AI tool
type AIExecutor struct {
	ExecutablePath string
	Logger         *zap.Logger
	Options        *Options
}

// NewAIExecutor creates a new AIExecutor instance
func NewAIExecutor() *AIExecutor {
	return &AIExecutor{
		ExecutablePath: configs.Configs.AiExecutor.Path,
		Logger:         configs.Logger,
	}
}

// Execute runs the AI executor with a default 5-minute timeout
func (e *AIExecutor) Execute(request AIExecutorRequest) (<-chan string, <-chan string, <-chan error) {
	// Use ExecuteWithTimeout with a default 5-minute timeout
	outputCh, errorCh, execErrCh, _ := e.ExecuteWithTimeout(request, 5*time.Minute)

	// ExecuteWithTimeout 내에서 이미 context가 취소되면 알아서 자원이 정리됨
	return outputCh, errorCh, execErrCh
}

// ExecuteAndCollect runs the AI executor and collects all output and errors
func (e *AIExecutor) ExecuteAndCollect(request AIExecutorRequest) (*AIExecutorResponse, error) {
	// Use ExecuteAndCollectWithTimeout with a default 5-minute timeout
	return e.ExecuteAndCollectWithTimeout(request, 5*time.Minute)
}

// ExecutionContext holds the context and cancelFunc for a running process
type ExecutionContext struct {
	cmd        *exec.Cmd
	cancelFunc context.CancelFunc
	terminated bool
	mutex      sync.Mutex
}

// ExecuteWithTimeout adds a timeout for the execution with improved error handling
func (e *AIExecutor) ExecuteWithTimeout(request AIExecutorRequest, timeout time.Duration) (<-chan string, <-chan string, <-chan error, *ExecutionContext) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// Create execution context to track this process
	execCtx := &ExecutionContext{
		cancelFunc: cancel,
		terminated: false,
		mutex:      sync.Mutex{},
	}

	outputCh := make(chan string)
	errorCh := make(chan string)
	execErrCh := make(chan error, 1) // Buffered channel to avoid blocking on error

	go func() {
		defer close(outputCh)
		defer close(errorCh)
		defer close(execErrCh)
		defer cancel() // Ensure context is cancelled when goroutine exits

		// Create the command with context
		cmd := exec.CommandContext(ctx, e.ExecutablePath)
		execCtx.cmd = cmd

		// Set environment variables
		cmd.Env = e.getEnvironmentVariables()

		// Get stdin pipe to send JSON input
		stdin, err := cmd.StdinPipe()
		if err != nil {
			e.Logger.Error("Failed to get stdin pipe", zap.Error(err))
			execErrCh <- fmt.Errorf("failed to get stdin pipe: %w", err)
			return
		}

		// Get stdout pipe to receive AI output
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			e.Logger.Error("Failed to get stdout pipe", zap.Error(err))
			execErrCh <- fmt.Errorf("failed to get stdout pipe: %w", err)
			return
		}

		// Get stderr pipe to receive logs and errors
		stderr, err := cmd.StderrPipe()
		if err != nil {
			e.Logger.Error("Failed to get stderr pipe", zap.Error(err))
			execErrCh <- fmt.Errorf("failed to get stderr pipe: %w", err)
			return
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			e.Logger.Error("Failed to start AI executor", zap.Error(err))
			execErrCh <- fmt.Errorf("failed to start AI executor: %w", err)
			return
		}

		// Send JSON input to stdin
		go func() {
			defer func() {
				// Recover from potential panic when writing to stdin
				if r := recover(); r != nil {
					e.Logger.Error("Panic recovered when writing to stdin", zap.Any("panic", r))
					select {
					case execErrCh <- fmt.Errorf("panic when writing to stdin: %v", r):
					default:
						// Channel might be closed, just log
						e.Logger.Error("Cannot send error to channel, it might be closed", zap.Any("error", r))
					}
				}
				stdin.Close()
			}()

			encoder := json.NewEncoder(stdin)
			if err := encoder.Encode(request); err != nil {
				e.Logger.Error("Failed to encode request", zap.Error(err))
				select {
				case execErrCh <- fmt.Errorf("failed to encode request: %w", err):
				default:
					// Channel might be closed or full, just log
					e.Logger.Error("Cannot send error to channel", zap.Error(err))
				}
			}
		}()

		// Set up a WaitGroup to wait for both stdout and stderr to be processed
		var wg sync.WaitGroup
		wg.Add(2)

		// Read from stdout (AI output)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					e.Logger.Error("Panic recovered when reading stdout", zap.Any("panic", r))
				}
				wg.Done()
			}()

			if request.LineByLine {
				// 한 줄씩 처리하는 모드
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					select {
					case <-ctx.Done():
						// Context timeout or cancellation
						return
					default:
						outputCh <- scanner.Text()
					}
				}
				if err := scanner.Err(); err != nil && err != io.EOF {
					e.Logger.Error("Error reading stdout", zap.Error(err))
					select {
					case execErrCh <- fmt.Errorf("error reading stdout: %w", err):
					default:
						// Channel might be closed or full, just log
						e.Logger.Error("Cannot send error to channel", zap.Error(err))
					}
				}
			} else {
				// 직접 stdout을 outputCh에 연결
				writer := channelWriter{ch: outputCh, ctx: ctx}
				_, err := io.Copy(writer, stdout)
				if err != nil && err != io.EOF {
					select {
					case <-ctx.Done():
						// Context was canceled, no need to report error
						return
					default:
						e.Logger.Error("Error copying from stdout", zap.Error(err))
						select {
						case execErrCh <- fmt.Errorf("error copying from stdout: %w", err):
						default:
							// Channel might be closed or full, just log
							e.Logger.Error("Cannot send error to channel", zap.Error(err))
						}
					}
				}
			}
		}()

		// Read from stderr (logs and errors)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					e.Logger.Error("Panic recovered when reading stderr", zap.Any("panic", r))
				}
				wg.Done()
			}()

			if request.LineByLine {
				// 한 줄씩 처리하는 모드
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					select {
					case <-ctx.Done():
						// Context timeout or cancellation
						return
					default:
						errorCh <- scanner.Text()
					}
				}
				if err := scanner.Err(); err != nil && err != io.EOF {
					e.Logger.Error("Error reading stderr", zap.Error(err))
					select {
					case execErrCh <- fmt.Errorf("error reading stderr: %w", err):
					default:
						// Channel might be closed or full, just log
						e.Logger.Error("Cannot send error to channel", zap.Error(err))
					}
				}
			} else {
				// 직접 stderr을 errorCh에 연결 (수정: stderr을 errorCh로 연결해야 함)
				writer := channelWriter{ch: errorCh, ctx: ctx}
				_, err := io.Copy(writer, stderr)
				if err != nil && err != io.EOF {
					select {
					case <-ctx.Done():
						// Context was canceled, no need to report error
						return
					default:
						e.Logger.Error("Error copying from stderr", zap.Error(err))
						select {
						case execErrCh <- fmt.Errorf("error copying from stderr: %w", err):
						default:
							// Channel might be closed or full, just log
							e.Logger.Error("Cannot send error to channel", zap.Error(err))
						}
					}
				}
			}
		}()

		// Wait for both stdout and stderr to be processed or context to be done
		doneCh := make(chan struct{})
		go func() {
			wg.Wait()
			close(doneCh)
		}()

		select {
		case <-ctx.Done():
			// Timeout or cancellation occurred
			execCtx.mutex.Lock()
			if !execCtx.terminated {
				// Only report timeout if not manually terminated
				e.Logger.Warn("AI executor execution timed out or was cancelled")
				select {
				case execErrCh <- ctx.Err():
				default:
					// Channel might be closed or full, just log
					e.Logger.Error("Cannot send timeout error to channel", zap.Error(ctx.Err()))
				}

				// 강제 종료 처리 추가
				e.forceTerminateProcess(execCtx)
			}
			execCtx.mutex.Unlock()
		case <-doneCh:
			// Normal completion, wait for the command to finish
			err := cmd.Wait()
			execCtx.mutex.Lock()
			if err != nil && !execCtx.terminated {
				// Only report error if not manually terminated
				e.Logger.Error("AI executor exited with error", zap.Error(err))
				select {
				case execErrCh <- fmt.Errorf("AI executor exited with error: %w", err):
				default:
					// Channel might be closed or full, just log
					e.Logger.Error("Cannot send execution error to channel", zap.Error(err))
				}
			}
			execCtx.mutex.Unlock()
		}
	}()

	return outputCh, errorCh, execErrCh, execCtx
}

// forceTerminateProcess ensures the process is forcibly terminated
func (e *AIExecutor) forceTerminateProcess(execCtx *ExecutionContext) {
	if execCtx == nil || execCtx.cmd == nil || execCtx.cmd.Process == nil {
		return
	}

	// 먼저 SIGTERM으로 종료 시도
	err := execCtx.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		e.Logger.Warn("Failed to terminate process with SIGTERM, trying SIGKILL", zap.Error(err))

		// SIGTERM 실패시 SIGKILL로 강제 종료
		killErr := execCtx.cmd.Process.Kill()
		if killErr != nil {
			e.Logger.Error("Failed to kill process with SIGKILL", zap.Error(killErr))
		} else {
			e.Logger.Info("Process forcibly terminated with SIGKILL")
		}
	} else {
		e.Logger.Info("Process terminated with SIGTERM")

		// SIGTERM 성공했지만 프로세스가 즉시 종료되지 않을 수 있으므로
		// 짧은 대기 후 프로세스 상태 확인 및 필요시 SIGKILL 사용
		go func() {
			time.Sleep(500 * time.Millisecond)

			// 프로세스가 여전히 실행 중인지 확인
			if p, err := os.FindProcess(execCtx.cmd.Process.Pid); err == nil {
				// Unix/Linux에서는 FindProcess가 항상 성공하므로 시그널을 보내 확인
				err = p.Signal(syscall.Signal(0))
				if err == nil {
					// 프로세스가 여전히 살아있음, SIGKILL 사용
					e.Logger.Warn("Process still running after SIGTERM, using SIGKILL")
					killErr := p.Kill()
					if killErr != nil {
						e.Logger.Error("Failed to kill process with SIGKILL", zap.Error(killErr))
					} else {
						e.Logger.Info("Process forcibly terminated with SIGKILL after SIGTERM timeout")
					}
				}
			}
		}()
	}
}

// Terminate forcefully terminates a running process
func (e *AIExecutor) Terminate(execCtx *ExecutionContext) error {
	if execCtx == nil {
		return fmt.Errorf("no execution context provided")
	}

	execCtx.mutex.Lock()
	defer execCtx.mutex.Unlock()

	// If already terminated, return success
	if execCtx.terminated {
		return nil
	}

	// Mark as terminated so we don't report timeout errors
	execCtx.terminated = true

	// Cancel the context
	if execCtx.cancelFunc != nil {
		execCtx.cancelFunc()
	}

	// If command exists, try to kill it
	if execCtx.cmd != nil && execCtx.cmd.Process != nil {
		// 프로세스 강제 종료 로직을 forceTerminateProcess 메서드로 위임
		e.forceTerminateProcess(execCtx)
	}

	return nil
}

// ExecuteAndCollectWithTimeout runs the AI executor with timeout and collects all output and errors
func (e *AIExecutor) ExecuteAndCollectWithTimeout(request AIExecutorRequest, timeout time.Duration) (*AIExecutorResponse, error) {
	outputCh, errorCh, execErrCh, execCtx := e.ExecuteWithTimeout(request, timeout)

	// Use defer with a recovery to prevent panics from propagating
	defer func() {
		if r := recover(); r != nil {
			e.Logger.Error("Panic recovered in ExecuteAndCollectWithTimeout", zap.Any("panic", r))
		}
		e.Terminate(execCtx) // Ensure resources are cleaned up
	}()

	response := &AIExecutorResponse{
		Output: "",
		Errors: []string{},
	}

	var outputBuffer string
	errors := []string{}
	var currentError string

	// Collect all output and errors with timeout safety
	collectDone := make(chan struct{})
	go func() {
		defer close(collectDone)
		defer func() {
			if r := recover(); r != nil {
				e.Logger.Error("Panic recovered while collecting output", zap.Any("panic", r))
			}
		}()

		for {
			select {
			case output, ok := <-outputCh:
				if !ok {
					outputCh = nil
				} else {
					outputBuffer += output
				}
			case errLine, ok := <-errorCh:
				if !ok {
					errorCh = nil
					// 마지막 에러 라인이 있으면 추가
					if currentError != "" {
						errors = append(errors, currentError)
					}
				} else {
					// LineByLine 옵션에 관계없이 전체 에러 라인을 수집
					// 개행 문자를 기준으로 에러 메시지를 분리
					if errLine == "\n" {
						if currentError != "" {
							errors = append(errors, currentError)
							currentError = ""
						}
					} else {
						currentError += errLine
					}
				}
			case err, ok := <-execErrCh:
				if !ok {
					execErrCh = nil
				} else {
					response.ExecError = err
				}
			}

			// Exit when all channels are closed
			if outputCh == nil && errorCh == nil && execErrCh == nil {
				break
			}
		}
	}()

	// Wait for collection to complete with timeout protection
	select {
	case <-collectDone:
		// Normal completion
	case <-time.After(timeout + 5*time.Second): // Give extra time after the original timeout
		e.Logger.Warn("Collection of output timed out, forcibly terminating")
		e.Terminate(execCtx)
	}

	response.Output = outputBuffer
	response.Errors = errors
	response.Terminated = execCtx.terminated

	return response, response.ExecError
}

// getEnvironmentVariables returns environment variables for the AI executor
func (e *AIExecutor) getEnvironmentVariables() []string {
	config := configs.Configs.AiExecutor

	// Get the current environment
	env := os.Environ()

	// Add our specific environment variables
	if config.OpenAIAPIKey != "" {
		env = append(env, fmt.Sprintf("OPENAI_API_KEY=%s", config.OpenAIAPIKey))
	}
	if config.OpenAIModel != "" {
		env = append(env, fmt.Sprintf("OPENAI_MODEL=%s", config.OpenAIModel))
	}
	if config.ContextSize != "" {
		env = append(env, fmt.Sprintf("CONTEXT_SIZE=%s", config.ContextSize))
	}
	if config.OpenAIEndpoint != "" {
		env = append(env, fmt.Sprintf("OPENAI_ENDPOINT=%s", config.OpenAIEndpoint))
	}

	// Add Anthropic variables
	if config.AnthropicAPIKey != "" {
		env = append(env, fmt.Sprintf("ANTHROPIC_API_KEY=%s", config.AnthropicAPIKey))
	}
	if config.AnthropicModel != "" {
		env = append(env, fmt.Sprintf("ANTHROPIC_MODEL=%s", config.AnthropicModel))
	}

	// Add Google variables
	if config.GoogleAPIKey != "" {
		env = append(env, fmt.Sprintf("GOOGLE_GENERATIVE_AI_API_KEY=%s", config.GoogleAPIKey))
	}
	if config.GoogleModel != "" {
		env = append(env, fmt.Sprintf("GOOGLE_MODEL=%s", config.GoogleModel))
	}

	// Add Grok variables
	if config.GrokAPIKey != "" {
		env = append(env, fmt.Sprintf("GROK_API_KEY=%s", config.GrokAPIKey))
	}
	if config.GrokModel != "" {
		env = append(env, fmt.Sprintf("GROK_MODEL=%s", config.GrokModel))
	}
	if config.GrokEndpoint != "" {
		env = append(env, fmt.Sprintf("GROK_ENDPOINT=%s", config.GrokEndpoint))
	}

	// Add Langfuse variables
	if config.LangfusePublicKey != "" {
		env = append(env, fmt.Sprintf("LANGFUSE_PUBLIC_KEY=%s", config.LangfusePublicKey))
	}
	if config.LangfuseSecretKey != "" {
		env = append(env, fmt.Sprintf("LANGFUSE_SECRET_KEY=%s", config.LangfuseSecretKey))
	}
	if config.LangfuseBaseURL != "" {
		env = append(env, fmt.Sprintf("LANGFUSE_BASEURL=%s", config.LangfuseBaseURL))
	}
	if config.EnableTelemetry != "" {
		env = append(env, fmt.Sprintf("ENABLE_TELEMETRY=%s", config.EnableTelemetry))
	}

	// Add Logging variables
	if config.LogLevel != "" {
		env = append(env, fmt.Sprintf("LOG_LEVEL=%s", config.LogLevel))
	}
	if config.EnableFileLogging != "" {
		env = append(env, fmt.Sprintf("ENABLE_FILE_LOGGING=%s", config.EnableFileLogging))
	}
	if config.LogDir != "" {
		env = append(env, fmt.Sprintf("LOG_DIR=%s", config.LogDir))
	}
	if config.LogMaxSize != "" {
		env = append(env, fmt.Sprintf("LOG_MAX_SIZE=%s", config.LogMaxSize))
	}
	if config.LogMaxFiles != "" {
		env = append(env, fmt.Sprintf("LOG_MAX_FILES=%s", config.LogMaxFiles))
	}

	return env
}
