package mail

import (
	"fmt"
	"time"

	"github.com/wekeepgrowing/semo-backend-monorepo/services/auth/internal/config"
	"go.uber.org/zap"
)

// EmailTemplateService 이메일 템플릿 생성 서비스
type EmailTemplateService struct {
	appURL       string // 앱 기본 URL (인증 링크 생성에 사용)
	supportEmail string // 지원 이메일
	companyName  string // 회사/서비스 이름
}

// NewEmailTemplateService 이메일 템플릿 서비스 생성
func NewEmailTemplateService(appURL, supportEmail, companyName string) *EmailTemplateService {
	return &EmailTemplateService{
		appURL:       appURL,
		supportEmail: supportEmail,
		companyName:  companyName,
	}
}

// GenerateWelcomeEmailHTML 환영 이메일 HTML 템플릿 생성
func (s *EmailTemplateService) GenerateWelcomeEmailHTML(username, code string) string {
	// 코드 형식 확인
	if len(code) != 6 {
		config.AppConfig.Logger.Error("잘못된 형식의 매직 코드",
			zap.String("code", code),
			zap.Int("length", len(code)),
		)
		return ""
	}

	// 코드 형식화 (XXX-XXX)
	formattedCode := fmt.Sprintf("%s-%s", code[:3], code[3:])

	// HTML 템플릿 생성
	emailHTML := fmt.Sprintf(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="ko">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>%s님, 환영합니다!</title>
</head>
<body style="margin: 0; padding: 0; font-family: 'Apple SD Gothic Neo', 'Malgun Gothic', sans-serif; background-color: #f7f9fc; -webkit-text-size-adjust: 100%%; -ms-text-size-adjust: 100%%;">
	<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="border-collapse: collapse;">
		<tr>
			<td style="padding: 40px 0;">
				<!-- 헤더 -->
				<table align="center" border="0" cellpadding="0" cellspacing="0" width="600" style="border-collapse: collapse; background-color: #5271ff; border-radius: 8px 8px 0 0;">
					<tr>
						<td align="center" style="padding: 30px 0; color: #ffffff;">
							<h1 style="margin: 0; font-size: 28px; font-weight: 700;">환영합니다!</h1>
						</td>
					</tr>
				</table>
				
				<!-- 본문 -->
				<table align="center" border="0" cellpadding="0" cellspacing="0" width="600" style="border-collapse: collapse; background-color: #ffffff; box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);">
					<tr>
						<td style="padding: 40px 30px;">
							<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="border-collapse: collapse;">
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 0; margin-bottom: 20px;">안녕하세요, <strong style="color: #5271ff;">%s</strong>님!</p>
										<p style="margin-top: 0; margin-bottom: 20px;">저희 서비스에 가입해 주셔서 진심으로 감사합니다. 고객님의 선택에 보답하기 위해 최선을 다하겠습니다.</p>
										<p style="margin-top: 0; margin-bottom: 20px;">이메일 인증을 위해 아래 코드를 입력해 주세요:</p>
									</td>
								</tr>
								<tr>
									<td align="center" style="padding: 20px 0;">
										<table border="0" cellpadding="0" cellspacing="0" style="border-collapse: collapse;">
											<tr>
												<td align="center" style="background-color: #f3f5ff; border: 1px solid #e1e5ff; border-radius: 8px; padding: 15px 40px;">
													<span style="color: #5271ff; font-size: 24px; font-weight: bold; letter-spacing: 2px;">%s</span>
												</td>
											</tr>
										</table>
									</td>
								</tr>
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 20px; margin-bottom: 20px;">이 코드는 5분 동안 유효합니다.</p>
										<p style="margin-top: 0; margin-bottom: 30px;">본인이 요청하지 않은 경우, 이 이메일을 무시하시면 됩니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px;">감사합니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px; font-weight: bold; color: #5271ff;">%s 팀 드림</p>
									</td>
								</tr>
							</table>
						</td>
					</tr>
				</table>
				
				<!-- 푸터 -->
				<table align="center" border="0" cellpadding="0" cellspacing="0" width="600" style="border-collapse: collapse; background-color: #f0f2fa; border-radius: 0 0 8px 8px;">
					<tr>
						<td align="center" style="padding: 20px; color: #666666; font-size: 12px; line-height: 1.5;">
							<p style="margin: 0; margin-bottom: 10px;">© %d %s. All rights reserved.</p>
							<p style="margin: 0; margin-bottom: 10px;">문의사항은 <a href="mailto:%s" style="color: #5271ff; text-decoration: none;">%s</a>으로 연락주세요.</p>
							<p style="margin: 0;">이 이메일은 발신 전용입니다. 회신하지 마세요.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`, username, username, formattedCode, s.companyName, time.Now().Year(), s.companyName, s.supportEmail, s.supportEmail)

	return emailHTML
}

// GenerateVerificationEmailHTML 이메일 인증 HTML 템플릿 생성
func (s *EmailTemplateService) GenerateVerificationEmailHTML(name, token string) string {
	// 인증 링크 생성
	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, token)

	// HTML 템플릿 생성
	emailHTML := fmt.Sprintf(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="ko">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>이메일 주소 인증</title>
</head>
<body style="margin: 0; padding: 0; font-family: 'Apple SD Gothic Neo', 'Malgun Gothic', sans-serif; background-color: #f7f9fc; -webkit-text-size-adjust: 100%%; -ms-text-size-adjust: 100%%;">
	<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="border-collapse: collapse;">
		<tr>
			<td style="padding: 40px 0;">
				<!-- 헤더 -->
				<table align="center" border="0" cellpadding="0" cellspacing="0" width="600" style="border-collapse: collapse; background-color: #5271ff; border-radius: 8px 8px 0 0;">
					<tr>
						<td align="center" style="padding: 30px 0; color: #ffffff;">
							<h1 style="margin: 0; font-size: 28px; font-weight: 700;">이메일 인증</h1>
						</td>
					</tr>
				</table>
				
				<!-- 본문 -->
				<table align="center" border="0" cellpadding="0" cellspacing="0" width="600" style="border-collapse: collapse; background-color: #ffffff; box-shadow: 0 4px 15px rgba(0, 0, 0, 0.08);">
					<tr>
						<td style="padding: 40px 30px;">
							<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="border-collapse: collapse;">
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 0; margin-bottom: 20px;">안녕하세요, <strong style="color: #5271ff;">%s</strong>님!</p>
										<p style="margin-top: 0; margin-bottom: 20px;">저희 서비스에 가입해 주셔서 진심으로 감사드립니다. 아래 버튼을 클릭하여 이메일 주소를 인증해 주세요.</p>
									</td>
								</tr>
								<tr>
									<td align="center" style="padding: 25px 0;">
										<!-- 버튼 -->
										<table border="0" cellpadding="0" cellspacing="0" style="border-collapse: separate; border-radius: 4px;">
											<tr>
												<td align="center" style="background-color: #5271ff; border-radius: 4px; padding: 0;">
													<a href="%s" target="_blank" style="display: inline-block; color: #ffffff; font-size: 16px; font-weight: bold; text-decoration: none; padding: 12px 30px; border: 1px solid #4a66e6;">이메일 인증하기</a>
												</td>
											</tr>
										</table>
									</td>
								</tr>
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 20px; margin-bottom: 20px;">버튼이 작동하지 않는 경우, 아래 인증 코드를 복사하여 서비스에서 입력해 주세요:</p>
									</td>
								</tr>
								<tr>
									<td align="center" style="padding: 10px 0 20px 0;">
										<table border="0" cellpadding="0" cellspacing="0" width="80%%" style="border-collapse: collapse;">
											<tr>
												<td align="center" style="background-color: #f3f5ff; border: 1px solid #e1e5ff; border-radius: 8px; padding: 15px;">
													<span style="color: #5271ff; font-size: 18px; font-weight: bold; letter-spacing: 1px; word-break: break-all;">%s</span>
												</td>
											</tr>
										</table>
									</td>
								</tr>
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 20px; margin-bottom: 20px;">인증 코드는 24시간 동안 유효합니다.</p>
										<p style="margin-top: 0; margin-bottom: 30px;">본인이 요청하지 않은 경우, 이 이메일을 무시하시면 됩니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px;">감사합니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px; font-weight: bold; color: #5271ff;">%s 팀 드림</p>
									</td>
								</tr>
							</table>
						</td>
					</tr>
				</table>
				
				<!-- 푸터 -->
				<table align="center" border="0" cellpadding="0" cellspacing="0" width="600" style="border-collapse: collapse; background-color: #f0f2fa; border-radius: 0 0 8px 8px;">
					<tr>
						<td align="center" style="padding: 20px; color: #666666; font-size: 12px; line-height: 1.5;">
							<p style="margin: 0; margin-bottom: 10px;">© %d %s. All rights reserved.</p>
							<p style="margin: 0; margin-bottom: 10px;">문의사항은 <a href="mailto:%s" style="color: #5271ff; text-decoration: none;">%s</a>으로 연락주세요.</p>
							<p style="margin: 0;">이 이메일은 발신 전용입니다. 회신하지 마세요.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`, name, verificationLink, token, s.companyName, time.Now().Year(), s.companyName, s.supportEmail, s.supportEmail)

	return emailHTML
}
