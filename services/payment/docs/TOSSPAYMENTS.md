# 토스페이먼츠 통합 가이드

## 개요

이 문서는 SEMO 백엔드 결제 서비스의 토스페이먼츠 통합 현황을 설명합니다.

### 현재 구현 상태

| 기능 | 상태 | 비고 |
|------|------|------|
| 일반 결제 (일회성) | 구현 완료 | 카드, 간편결제 지원 |
| 결제 승인 | 구현 완료 | `/v1/payments/confirm` |
| 웹훅 처리 | 구현 완료 | 결제 상태 변경 자동 처리 |
| 자동결제 (빌링) | 미구현 | 향후 추가 예정 |

### SDK vs API 버전

토스페이먼츠는 **SDK 버전**과 **API 버전**이 별개로 관리됩니다.

| 구분 | 버전 체계 | 현재 사용 |
|------|----------|----------|
| SDK (프론트엔드) | v1, v2 | v2 (`@tosspayments/tosspayments-sdk`) |
| API (백엔드) | 날짜 기반 (YYYY-MM-DD) | `/v1/payments/confirm` |

**중요**: 프론트엔드 SDK v2를 사용해도 백엔드 API 경로는 동일하게 `/v1/`을 사용합니다. `/v2/payments/confirm`은 존재하지 않습니다.

---

## 현재 구현된 API

### 결제 승인 (Confirm Payment)

**엔드포인트**: `POST https://api.tosspayments.com/v1/payments/confirm`

**인증**: Basic Auth
```
Authorization: Basic {base64(secretKey:)}
```

**요청 본문**:
```json
{
  "paymentKey": "5zJ4xY7m0kODnyRpQWGrN2xqGlNvLrKwv1M9ENjbeoPaZdL6",
  "orderId": "a4CWyWY5m89PNh7xJwhk1",
  "amount": 15000
}
```

**구현 위치**: `internal/infrastructure/provider/toss/toss.go:102-228`

### 웹훅 처리

**엔드포인트**: `POST /webhook/toss`

**지원 이벤트**:

| 토스 상태 | 내부 상태 | 처리 |
|----------|----------|------|
| `DONE` | `completed` | 결제 완료, 크레딧 할당 |
| `CANCELED` | `cancelled` | 결제 취소 |
| `PARTIAL_CANCELED` | `refunded` | 부분 환불 |
| `EXPIRED` | `failed` | 결제 만료 |
| `ABORTED` | `failed` | 결제 중단 |

**구현 위치**: `internal/adapter/handler/http/toss_webhook_handler.go`

---

## 상태 매핑

토스페이먼츠 상태를 내부 결제 상태로 매핑합니다.

```go
// internal/infrastructure/provider/toss/toss.go:331-346
func mapTossStatus(tossStatus string) provider.PaymentStatus {
    switch tossStatus {
    case "READY", "IN_PROGRESS":
        return provider.PaymentStatusPending
    case "DONE":
        return provider.PaymentStatusCompleted
    case "CANCELED":
        return provider.PaymentStatusCancelled
    case "PARTIAL_CANCELED":
        return provider.PaymentStatusRefunded
    case "ABORTED", "EXPIRED":
        return provider.PaymentStatusFailed
    default:
        return provider.PaymentStatusPending
    }
}
```

---

## 에러 핸들링

토스페이먼츠 API 에러는 `ProviderError` 구조체로 래핑됩니다.

| 에러 코드 | 발생 상황 |
|----------|----------|
| `MARSHAL_ERROR` | 요청 본문 직렬화 실패 |
| `REQUEST_ERROR` | HTTP 요청 생성 실패 |
| `API_ERROR` | 토스페이먼츠 API 통신 실패 |
| `PARSE_ERROR` | 응답 파싱 실패 |
| (토스 에러 코드) | 토스페이먼츠에서 반환한 에러 |

---

## 버전 관리

### API 버전 정책

토스페이먼츠는 2022년 6월부터 **날짜 기반 버전 관리(Calendar Versioning)**를 사용합니다.

- 버전 형식: `YYYY-MM-DD` (예: `2022-11-16`)
- 설정 위치: [토스페이먼츠 개발자센터](https://developers.tosspayments.com/my/api-keys)

### 하위 호환성

- 새 버전 릴리즈 시에도 기존 상점의 API 버전은 자동 변경되지 않음
- 응답에 새 필드 추가는 버전 변경 없이 반영될 수 있음

---

## 설정

### 환경 변수

```yaml
toss:
  secret_key: ${PAYMENT_TOSS_SECRET_KEY}    # API 시크릿 키
  client_key: ${PAYMENT_TOSS_CLIENT_KEY}    # 클라이언트 키 (프론트엔드용)
  webhook_secret: ${PAYMENT_TOSS_WEBHOOK_SECRET}  # 웹훅 검증용 (선택)
  plans_file: configs/toss_plans.yaml       # KRW 플랜 설정
  usd_plans_file: configs/toss_plans_usd.yaml  # USD 플랜 설정
```

### 설정 파일 위치

- `internal/config/service.go:39-45`
- `configs/toss_plans.yaml`
- `configs/toss_plans_usd.yaml`

---

## 향후 자동결제 추가 시

### 필요 API

| API | 엔드포인트 | 용도 |
|-----|-----------|------|
| 빌링키 발급 | `POST /v1/billing/authorizations/card` | 카드 정보로 빌링키 생성 |
| 빌링키 결제 | `POST /v1/billing/{billingKey}` | 빌링키로 결제 승인 |

### 사전 준비 사항

1. **추가 계약 필요**: 토스페이먼츠 고객센터 (1544-7772, support@tosspayments.com)
2. **customerKey 형식**: UUID와 같이 무작위적인 고유 값 사용 권장
3. **타임아웃 설정**: 자동결제 승인은 최대 60초 소요 가능
4. **스케줄링 구현**: 토스페이먼츠는 자체 스케줄링 미제공, 직접 구현 필요

### 지원 결제수단

- 국내 카드만 지원
- 간편결제(토스페이, 카카오페이, 네이버페이) 미지원
- 해외 간편결제(PayPal) 미지원

---

## 미구현 항목 (TODO)

1. **웹훅 서명 검증** (`toss.go:232`)
   - 현재 `X-Toss-Signature` 헤더를 수신하지만 검증하지 않음

2. **환불 시 크레딧 차감** (`toss_webhook_handler.go:489`)
   - 현재 환불 웹훅 수신 시 결제 상태만 업데이트, 크레딧 차감 미구현

---

## 파일 구조

```
services/payment/
├── internal/
│   ├── adapter/
│   │   ├── handler/http/
│   │   │   ├── product_handler.go      # 결제 생성/승인 핸들러
│   │   │   └── toss_webhook_handler.go # 웹훅 핸들러
│   │   └── repository/
│   │       └── toss_webhook_repository.go
│   ├── domain/
│   │   ├── model/
│   │   │   └── toss_webhook_event.go   # 웹훅 이벤트 모델
│   │   └── provider/
│   │       └── provider.go             # 결제 프로바이더 인터페이스
│   └── infrastructure/
│       └── provider/
│           ├── factory.go              # 프로바이더 팩토리
│           └── toss/
│               └── toss.go             # 토스 프로바이더 구현
├── configs/
│   ├── toss_plans.yaml                 # KRW 결제 플랜
│   └── toss_plans_usd.yaml             # USD 결제 플랜
└── migrations/
    └── 007_create_toss_webhook_events.sql
```

---

## 참고 자료

- [토스페이먼츠 개발자센터](https://docs.tosspayments.com/)
- [코어 API 레퍼런스](https://docs.tosspayments.com/reference)
- [SDK v2 마이그레이션 가이드](https://docs.tosspayments.com/guides/v2/get-started/migration-guide)
- [API 버전 정책](https://docs.tosspayments.com/reference/versioning)
- [자동결제(빌링) 가이드](https://docs.tosspayments.com/guides/v2/billing)
