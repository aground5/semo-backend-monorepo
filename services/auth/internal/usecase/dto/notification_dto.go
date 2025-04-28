package dto

// EmailContent 이메일 발송 관련 정보
type EmailContent struct {
	To       string                 // 수신자 이메일
	Cc       []string               // 참조
	Subject  string                 // 제목
	Body     string                 // 내용 (HTML 지원)
	Template string                 // 템플릿 이름
	Data     map[string]interface{} // 템플릿에 주입할 데이터
}

// VerificationEmailData 인증 이메일 데이터
type VerificationEmailData struct {
	Name             string // 수신자 이름
	VerificationLink string // 인증 링크
	Token            string // 인증 토큰
	ExpireHours      int    // 만료 시간 (시간)
}

// MagicLinkEmailData 매직 링크 이메일 데이터
type MagicLinkEmailData struct {
	Name      string // 수신자 이름
	LoginLink string // 로그인 링크
	Code      string // 로그인 코드
	ExpireMin int    // 만료 시간 (분)
}
