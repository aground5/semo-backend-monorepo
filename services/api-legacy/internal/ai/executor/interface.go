package executor

import "time"

// AIExecutorInterface는 AI 실행기에 대한 인터페이스를 정의합니다.
type AIExecutorInterface interface {
	// Execute는 AI 실행기를 실행하고 결과를 채널로 반환합니다.
	Execute(request AIExecutorRequest) (<-chan string, <-chan string, <-chan error)

	// ExecuteAndCollect는 AI 실행기를 실행하고 모든 출력과 오류를 수집합니다.
	ExecuteAndCollect(request AIExecutorRequest) (*AIExecutorResponse, error)

	// ExecuteWithTimeout은 제한 시간을 설정하여 AI 실행기를 실행합니다.
	ExecuteWithTimeout(request AIExecutorRequest, timeout time.Duration) (<-chan string, <-chan string, <-chan error, *ExecutionContext)

	// ExecuteAndCollectWithTimeout은 제한 시간을 설정하여 AI 실행기를 실행하고 모든 출력과 오류를 수집합니다.
	ExecuteAndCollectWithTimeout(request AIExecutorRequest, timeout time.Duration) (*AIExecutorResponse, error)

	// Terminate는 실행 중인 프로세스를 강제로 종료합니다.
	Terminate(execCtx *ExecutionContext) error
}
