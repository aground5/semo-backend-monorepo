# Semo Backend Server

Semo Backend Server는 태스크 관리, AI 기반 분석, 팀 협업 기능을 통합적으로 제공하는 마이크로서비스 아키텍처 기반의 백엔드 시스템입니다.

## 서비스 구조 개요

Semo Backend는 다음과 같은 핵심 서비스들로 구성되어 있습니다:

```
                     ┌─────────────────┐
                     │   HTTP/gRPC     │
                     │    인터페이스    │
                     └────────┬────────┘
                              │
┌───────────────────────────────────────────────┐
│                                               │
│  ┌─────────┐   ┌─────────┐    ┌─────────┐    │
│  │ 태스크   │   │ 프로젝트 │    │  팀     │    │
│  │ 서비스   │   │ 서비스   │    │ 서비스   │    │
│  └─────────┘   └─────────┘    └─────────┘    │
│                                               │
│  ┌─────────┐   ┌─────────┐    ┌─────────┐    │
│  │ 파일     │   │ 프로필   │    │ 속성    │    │
│  │ 서비스   │   │ 서비스   │    │ 서비스   │    │
│  └─────────┘   └─────────┘    └─────────┘    │
│                                               │
│                ┌─────────┐                    │
│                │   AI    │                    │
│                │  서비스 │                    │
│                └─────────┘                    │
│                                               │
└──────────────────┬────────────────────────────┘
                   │
        ┌──────────┴───────────┐
        │                      │
┌───────┴────────┐   ┌─────────┴─────┐
│   데이터베이스   │   │  외부 서비스   │
│  (Postgres,    │   │  (AI API,     │
│   Redis, S3)   │   │   이메일 등)   │
└────────────────┘   └───────────────┘
```

## 핵심 서비스 컴포넌트

### 1. 태스크 관리 서비스 (TaskService)

계층적인 작업 관리와 구성을 담당하는 핵심 서비스입니다.

**주요 기능:**
- 계층형 태스크 구조 생성 및 관리 (부모-자식 관계)
- 태스크 위치 재배열 및 순서 관리
- 태스크 메타데이터 관리 (목표, 결과물, 상태 등)
- 루트 태스크 및 프로젝트 연결 관리

**API 예시:**
```go
// 태스크 생성
task, err := taskService.CreateTask(&models.Item{
    Name: "새 태스크",
    Contents: "태스크 내용",
    Type: "task",
    CreatedBy: userID,
}, nil)

// 자식 태스크 조회
children, err := taskService.GetChildTasks(parentID, pagination)

// 태스크 업데이트
task, err := taskService.UpdateTask(taskID, models.ItemUpdate{
    Name: &newName,
    Contents: &newContents,
}, userEmail)
```

**구현 관련 파일:**
- `/internal/logics/task_service.go`
- `/internal/logics/task_formatter_service.go`
- `/internal/controllers/task_controller.go`

### 2. 프로젝트 관리 서비스 (ProjectService)

최상위 조직 단위인 프로젝트를 관리하는 서비스입니다.

**주요 기능:**
- 프로젝트 생성, 조회, 수정, 삭제
- 프로젝트-태스크 연결 관리
- 프로젝트 순서 관리 및 위치 조정
- 프로젝트 목록 페이지네이션

**API 예시:**
```go
// 프로젝트 생성
project, err := projectService.CreateProject(&models.Item{
    Name: "새 프로젝트",
    Contents: "프로젝트 설명",
    Type: "project",
    CreatedBy: userID,
}, nil)

// 프로젝트 목록 조회 (페이지네이션)
result, err := projectService.ListProjectsPaginated(userID, pagination)
```

**구현 관련 파일:**
- `/internal/logics/project_service.go`
- `/internal/controllers/project_controller.go`

### 3. 팀 관리 서비스 (TeamService, ProjectMemberService)

프로젝트 참여자와 팀 구성을 관리하는 서비스입니다.

**주요 기능:**
- 팀 생성 및 관리
- 사용자 초대 및 권한 부여
- 초대 수락/거절 처리
- 프로젝트-팀 연결 관리
- 이메일 초대 시스템

**API 예시:**
```go
// 팀 생성
team, err := teamService.CreateTeam("개발팀", "프로젝트 개발팀", "", creatorID)

// 사용자 초대
err := teamService.InviteUserToTeam(teamID, userID, inviterID, "member")

// 프로젝트에 멤버 추가
err := projectMemberService.AddMemberToProject(projectID, userID, inviterID, "editor")
```

**구현 관련 파일:**
- `/internal/logics/team_service.go`
- `/internal/logics/project_member_service.go`
- `/internal/controllers/project_controller.go`

### 4. 권한 관리 서비스 (TaskPermissionService)

태스크 및 프로젝트 접근 권한을 관리하는 서비스입니다.

**주요 기능:**
- 태스크 접근 권한 부여/회수
- 권한 검증
- 엔트리(Entry) 기반 권한 관리
- 부모-자식 태스크 권한 상속

**API 예시:**
```go
// 권한 부여
err := taskPermissionService.GrantPermission(taskID, profileID)

// 권한 회수
err := taskPermissionService.RevokePermission(taskID, profileID)

// 권한 확인
hasPermission, err := taskPermissionService.CheckPermission(taskID, profileID)
```

**구현 관련 파일:**
- `/internal/logics/task_permission_service.go`
- `/internal/controllers/task_permission_controller.go`

### 5. 파일 관리 서비스 (FileService)

프로젝트와 태스크에 첨부되는 파일을 관리하는 서비스입니다.

**주요 기능:**
- S3 기반 파일 업로드/다운로드
- 파일 메타데이터 관리
- 임시 다운로드 URL 생성
- 파일 유형 관리

**API 예시:**
```go
// 파일 업로드
file, err := fileService.UploadFile(ctx, itemID, fileReader, fileHeader)

// 다운로드 링크 생성
url, err := fileService.GetDownloadLink(ctx, fileID, itemID)

// 아이템 관련 파일 목록 조회
files, err := fileService.ListFilesByItem(ctx, itemID)
```

**구현 관련 파일:**
- `/internal/logics/file_service.go`
- `/internal/controllers/file_controller.go`

### 6. AI 통합 서비스 (LLMService)

대규모 언어 모델(LLM)을 활용한 AI 기능을 제공하는 서비스입니다.

**주요 기능:**
- AI 기반 태스크 분해 및 서브태스크 생성
- 태스크 의존성 분석
- 태스크 맥락 기반 질문-응답
- 다양한 AI 모델 통합 (OpenAI, Anthropic, Google 등)
- 스트리밍 응답 처리

**AI 처리 아키텍처:**
```
┌──────────────────┐    ┌──────────────┐    ┌─────────────┐
│     Context      │    │   Executor   │    │   Parsers   │
│    Collectors    │--->│   (AI 실행)   │--->│  (응답 해석) │
└──────────────────┘    └──────────────┘    └─────────────┘
         │                     │                   │
         │                     V                   │
         │               ┌──────────┐              │
         └───────────-->│  Cache   │<─────────────┘
                        └──────────┘
```

**API 예시:**
```go
// 서브태스크 생성
err := llmService.GenerateSubtasks(&GenerateSubtaskRequest{
    TaskID: taskID,
    Answer: userAnswer,
}, userID, sessionID, streamChan)

// 사전 질문 생성
err := llmService.GeneratePreQuestions(taskID, userID, sessionID, streamChan)
```

**구현 관련 파일:**
- `/internal/logics/llm_service.go`
- `/internal/ai/executor/executor.go`
- `/internal/ai/services/` (다양한 AI 서비스)
- `/internal/controllers/kickoff_controller.go`

### 7. 속성 관리 서비스 (AttributeService, AttributeValueService)

태스크에 연결되는 다양한 유형의 속성을 관리하는 서비스입니다.

**주요 기능:**
- 커스텀 속성 정의 및 관리
- 속성 값 연결 및 저장
- 계층적 속성 구성
- 속성 유형별 검증 및 처리

**API 예시:**
```go
// 속성 생성
attribute, err := attributeService.CreateAttribute(models.Attribute{
    Name: "우선순위",
    Type: "select",
    RootTaskID: rootTaskID,
    Config: attributeConfig,
}, nil)

// 속성 값 설정
value, err := attributeValueService.EditAttributeValue(&models.AttributeValueUpdate{
    AttributeID: attributeID,
    TaskID: taskID,
    Value: "high",
})
```

**구현 관련 파일:**
- `/internal/logics/attribute_service.go`
- `/internal/logics/attribute_value_service.go`
- `/internal/controllers/attribute_controller.go`

### 8. 프로필 서비스 (ProfileService)

사용자 프로필 및 설정을 관리하는 서비스입니다.

**주요 기능:**
- 사용자 프로필 생성 및 관리
- 사용자 기본 설정 관리
- 타임존 및 지역 설정
- 프로필 검색

**API 예시:**
```go
// 프로필 조회 또는 생성
profile, err := profileService.GetOrCreateProfile(email)

// 프로필 정보 업데이트
profile, err := profileService.UpdateProfile(userEmail, ip, models.ProfileUpdate{
    Name: &newName,
    DisplayName: &newDisplayName,
})
```

**구현 관련 파일:**
- `/internal/logics/profile_service.go`
- `/internal/controllers/profile_controller.go`

### 9. 검색 서비스 (SearchService)

프로젝트, 태스크, 사용자 등을 검색하는 기능을 제공하는 서비스입니다.

**주요 기능:**
- 키워드 기반 통합 검색
- 필터링 및 정렬
- 권한 기반 검색 결과 필터링
- 페이지네이션 지원

**API 예시:**
```go
// 프로필 검색
results, err := searchService.SearchProfiles(keyword, pagination)

// 태스크 및 프로젝트 검색
results, err := searchService.SearchItems(userID, keyword, itemType, pagination)
```

**구현 관련 파일:**
- `/internal/logics/search_service.go`

### 10. 엔트리 서비스 (EntryService)

사용자의 태스크 접근 정보를 관리하는 서비스입니다.

**주요 기능:**
- 태스크 엔트리 생성 및 관리
- 사용자별 태스크 목록 조회
- 권한 연결 관리

**API 예시:**
```go
// 엔트리 생성
entry, err := entryService.CreateEntry(&models.Entry{
    Name: task.Name,
    TaskID: taskID,
    RootTaskID: rootTaskID,
    CreatedBy: userID,
})

// 사용자의 엔트리 목록 조회
result, err := entryService.ListEntriesPaginated(profileID, pagination)
```

**구현 관련 파일:**
- `/internal/logics/entry_service.go`
- `/internal/controllers/entry_controller.go`

## 시스템 인프라 서비스

### 1. 데이터베이스 서비스

여러 데이터 저장소를 통합하여 관리합니다.

**주요 구성요소:**
- **PostgreSQL**: 관계형 데이터 저장
- **Redis**: 캐싱 및 세션 관리
- **MongoDB**: 로깅 및 비정형 데이터 (선택적)
- **S3**: 파일 스토리지

**구현 관련 파일:**
- `/internal/repositories/init.go`

### 2. 인증 서비스

JWT 기반 사용자 인증을 담당합니다.

**주요 기능:**
- JWT 토큰 검증
- 사용자 인증 처리
- 공개키 관리

**구현 관련 파일:**
- `/internal/middlewares/jwt_middleware.go`
- `/internal/logics/public_key_service.go`

### 3. 지역 정보 서비스 (GeoLite)

사용자의 지역 정보를 처리합니다.

**주요 기능:**
- IP 기반 지역 탐지
- 타임존 정보 제공

**구현 관련 파일:**
- `/internal/logics/geolite_service.go`
- `/proto/geolite/geolite.proto`

## 설치 및 구성

### 사전 요구사항
- Go 1.24 이상
- Docker 및 Docker Compose
- PostgreSQL 15
- Redis 7
- AWS S3 또는 호환 스토리지

### 환경 설정
`configs/file/configs.yaml` 파일을 통해 다양한 서비스를 구성할 수 있습니다:

```yaml
# 데이터베이스 설정
postgres:
  address: "postgres:5432"
  username: "semo"
  password: "semo"
  database: "semo"
  schema: "public"

redis:
  addresses: ["redis:6379"]
  username: ""
  password: "REMOVED_REDIS_PASSWORD_2"
  database: 0
  tls: false

# 서비스 설정
service:
  http_port: "8080"
  grpc_port: "9090"
  service_name: "semo-server"
  base_url: "https://semo.world"

# AI 설정 
ai_executor:
  path: "/app/bin/ai-executor"
  openai_api_key: "sk-..."
  anthropic_api_key: "sk-ant-..."
  google_generative_ai_api_key: "..."
  openai_model: "gpt-4o"
  anthropic_model: "claude-3-opus-20240229"
  google_model: "gemini-2.0-flash"
  
# 외부 서비스 연결
spicedb:
  address: "spicedb:50051"
  token: "token"

# 이메일 설정
email:
  smtp_host: "smtp.example.com"
  smtp_port: 587
  username: "noreply@example.com"
  password: "password"
  sender_email: "noreply@example.com"
```

### Docker로 실행

```bash
# Docker Compose로 서비스 실행
cd docker
docker-compose up -d
```

Docker Compose는 다음 서비스들을 설정합니다:
- `semo-server`: 메인 백엔드 애플리케이션
- `postgres`: PostgreSQL 데이터베이스
- `redis`: Redis 캐시 서버

### 로컬에서 직접 실행

```bash
# 필요한 의존성 설치
go mod download

# 서버 실행
go run cmd/main.go -c configs/file/configs.yaml
```

## 서비스 확장 가이드

### 새로운 서비스 추가

1. `internal/logics` 디렉토리에 새 서비스 파일 생성:

```go
package logics

// NewService는 새로운 비즈니스 로직을 처리합니다
type NewService struct {
    db *gorm.DB
}

// NewNewService는 NewService 인스턴스를 생성합니다
func NewNewService(db *gorm.DB) *NewService {
    return &NewService{
        db: db,
    }
}

// 서비스 메서드 구현
func (s *NewService) DoSomething(param string) (string, error) {
    // 구현
}
```

2. `internal/controllers` 디렉토리에 컨트롤러 추가:

```go
package controllers

// NewController는 새 서비스 관련 API 요청을 처리합니다
type NewController struct {
    newService *logics.NewService
}

// NewNewController는 NewController 인스턴스를 생성합니다
func NewNewController(newService *logics.NewService) *NewController {
    return &NewController{
        newService: newService,
    }
}

// 컨트롤러 핸들러 구현
func (c *NewController) HandleRequest(ctx echo.Context) error {
    // 구현
}
```

3. `internal/app/http/routes.go`에 라우트 등록:

```go
// 서비스 초기화
newService := logics.NewNewService(db)

// 컨트롤러 초기화
newController := controllers.NewNewController(newService)

// 라우트 등록
apiV1.GET("/new-endpoint", newController.HandleRequest)
```

### AI 서비스 확장

1. `internal/ai/services` 디렉토리에 새 AI 서비스 추가:

```go
package services

// NewAIService는 새로운 AI 기능을 처리합니다
type NewAIService struct {
    Executor *executor.AIExecutor
    Logger   *zap.Logger
}

// NewNewAIService는 NewAIService 인스턴스를 생성합니다
func NewNewAIService(exec *executor.AIExecutor, logger *zap.Logger) *NewAIService {
    return &NewAIService{
        Executor: exec,
        Logger:   logger,
    }
}

// AI 처리 메서드 구현
func (s *NewAIService) ProcessRequest(ctx context.Context, data map[string]any, streamChan chan<- string) (string, error) {
    // 구현
}
```

2. `internal/logics/llm_service.go`를 확장하여 새 서비스 연동:

```go
func NewLLMService(...) *LLMService {
    // 기존 코드...
    
    // 새 AI 서비스 추가
    newAIService := services.NewNewAIService(aiExecutor, configs.Logger)
    
    return &LLMService{
        // 기존 필드...
        newAIService: newAIService,
    }
}

// 새 AI 기능을 위한 메서드 추가
func (cs *LLMService) ProcessNewAIFunction(...) error {
    // 구현
}
```

## 개발 팁

### 효율적인 코드 패턴

1. **서비스 계층화**: 비즈니스 로직(logics)과 API 처리(controllers)를 명확히 분리

2. **의존성 주입**: 서비스 간 의존성은 생성자를 통해 주입

3. **트랜잭션 처리**:
```go
tx := s.db.Begin()
if tx.Error != nil {
    return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
}

// 작업 수행
if err := tx.Create(&entity).Error; err != nil {
    tx.Rollback()
    return nil, fmt.Errorf("failed to create entity: %w", err)
}

return tx.Commit().Error
```

4. **페이지네이션 처리**:
```go
// CursorPagination 구조체 활용
pagination := utils.CursorPagination{
    Cursor: cursor,
    Limit: limit,
}

// 서비스 호출
result, err := service.ListItemsPaginated(userID, pagination)
```

5. **에러 처리 및 래핑**:
```go
if err != nil {
    return nil, fmt.Errorf("서비스 작업 실패: %w", err)
}
```

## API 문서

주요 API 엔드포인트는 다음과 같습니다:

### 태스크 관리
- `GET /api/v1/tasks/:id` - 태스크 조회
- `POST /api/v1/tasks` - 태스크 생성
- `PUT /api/v1/tasks/:id` - 태스크 수정
- `DELETE /api/v1/tasks/:id` - 태스크 삭제
- `GET /api/v1/tasks/:id/children` - 자식 태스크 목록

### 프로젝트 관리
- `GET /api/v1/projects` - 프로젝트 목록
- `GET /api/v1/projects/:id` - 프로젝트 상세 조회
- `POST /api/v1/projects` - 프로젝트 생성
- `PUT /api/v1/projects/:id` - 프로젝트 수정
- `DELETE /api/v1/projects/:id` - 프로젝트 삭제

### 팀 및 멤버 관리
- `GET /api/v1/projects/:id/members` - 프로젝트 멤버 목록
- `POST /api/v1/projects/:id/members` - 프로젝트 멤버 초대
- `DELETE /api/v1/projects/:id/members/:user_id` - 멤버 제거
- `GET /api/v1/projects/invitations` - 초대 목록 조회

### 태스크 권한 관리
- `GET /api/v1/tasks/:id/permissions` - 권한 목록 조회
- `POST /api/v1/tasks/:id/permissions` - 권한 부여
- `DELETE /api/v1/tasks/:id/permissions/:profile_id` - 권한 제거

### 파일 관리
- `POST /api/v1/items/:item_id/files` - 파일 업로드
- `GET /api/v1/items/:item_id/files` - 파일 목록 조회
- `GET /api/v1/files/:id` - 파일 다운로드
- `DELETE /api/v1/files/:id` - 파일 삭제

### AI 기능
- `POST /api/v1/kickoff/preview` - AI 분석 미리보기
- `POST /api/v1/kickoff/pre-questions` - AI 사전 질문 생성

### 프로필 관리
- `GET /api/v1/profile` - 사용자 프로필 조회
- `PUT /api/v1/profile` - 프로필 업데이트
- `GET /api/v1/profile/search` - 프로필 검색

## 라이센스

이 프로젝트는 내부용으로 개발되었으며, 모든 권리는 해당 기업에 있습니다.

## 연락처

문의사항은 support@example.com으로 연락해주세요.