package utils

import (
	"fmt"
	"semo-server/configs-legacy"

	"gopkg.in/gomail.v2"
)

// EmailService provides email delivery functionality
type EmailService struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
}

// NewEmailService creates a new EmailService
func NewEmailService(smtpHost string, smtpPort int, smtpUsername, smtpPassword string) *EmailService {
	return &EmailService{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUsername: smtpUsername,
		smtpPassword: smtpPassword,
	}
}

// SendEmail sends an email with TLS and both HTML and plain text versions
func (s *EmailService) SendEmail(from, to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(from, "SEMO"))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	//m.SetBody("text/plain", textBody) // Plain text version for compatibility
	m.SetBody("text/html", htmlBody) // HTML version for modern clients

	d := gomail.NewDialer(s.smtpHost, s.smtpPort, s.smtpUsername, s.smtpPassword)

	return d.DialAndSend(m)
}

// GenerateWelcomeEmailHTML creates a welcome email HTML template with inline styles
func (s *EmailService) GenerateWelcomeEmailHTML(username, code string) string {
	// Return empty string if code isn't 6 characters
	if len(code) != 6 {
		return ""
	}

	// Format code as XXX-XXX
	formattedCode := fmt.Sprintf("%s-%s", code[:3], code[3:])

	// Generate HTML template with inline styles for maximum compatibility
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
										<p style="margin-top: 20px; margin-bottom: 20px;">이 코드는 24시간 동안 유효합니다.</p>
										<p style="margin-top: 0; margin-bottom: 30px;">본인이 요청하지 않은 경우, 이 이메일을 무시하시면 됩니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px;">감사합니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px; font-weight: bold; color: #5271ff;">서비스 팀 드림</p>
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
							<p style="margin: 0; margin-bottom: 10px;">© 2025 서비스명. All rights reserved.</p>
							<p style="margin: 0; margin-bottom: 10px;">문의사항은 <a href="mailto:support@service.com" style="color: #5271ff; text-decoration: none;">support@service.com</a>으로 연락주세요.</p>
							<p style="margin: 0;">이 이메일은 발신 전용입니다. 회신하지 마세요.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`, username, username, formattedCode)

	return emailHTML
}

// GenerateWelcomeEmailText creates a plain text version of the welcome email
func (s *EmailService) GenerateWelcomeEmailText(username, code string) string {
	if len(code) != 6 {
		return ""
	}

	formattedCode := fmt.Sprintf("%s-%s", code[:3], code[3:])

	emailText := fmt.Sprintf(`안녕하세요, %s님!

저희 서비스에 가입해 주셔서 진심으로 감사합니다.
고객님의 선택에 보답하기 위해 최선을 다하겠습니다.

이메일 인증을 위해 아래 코드를 입력해 주세요:

%s

이 코드는 24시간 동안 유효합니다.
본인이 요청하지 않은 경우, 이 이메일을 무시하시면 됩니다.

감사합니다.
서비스 팀 드림

© 2025 서비스명. All rights reserved.
문의사항은 support@service.com으로 연락주세요.
이 이메일은 발신 전용입니다. 회신하지 마세요.`, username, formattedCode)

	return emailText
}

// GenerateVerificationEmailHTML creates an email verification HTML template with inline styles
func (s *EmailService) GenerateVerificationEmailHTML(name, verificationLink, token string) string {
	// Generate HTML template with inline styles for maximum compatibility
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
										<p style="margin-top: 0; margin-bottom: 5px; font-weight: bold; color: #5271ff;">서비스 팀 드림</p>
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
							<p style="margin: 0; margin-bottom: 10px;">© 2025 서비스명. All rights reserved.</p>
							<p style="margin: 0; margin-bottom: 10px;">문의사항은 <a href="mailto:support@service.com" style="color: #5271ff; text-decoration: none;">support@service.com</a>으로 연락주세요.</p>
							<p style="margin: 0;">이 이메일은 발신 전용입니다. 회신하지 마세요.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`, name, verificationLink, token)

	return emailHTML
}

// GenerateVerificationEmailText creates a plain text version of the verification email
func (s *EmailService) GenerateVerificationEmailText(name, verificationLink, token string) string {
	emailText := fmt.Sprintf(`안녕하세요, %s님!

저희 서비스에 가입해 주셔서 진심으로 감사드립니다.
아래 링크를 클릭하여 이메일 주소를 인증해 주세요:

%s

링크가 작동하지 않는 경우, 아래 인증 코드를 복사하여 서비스에서 입력해 주세요:

%s

인증 코드는 24시간 동안 유효합니다.
본인이 요청하지 않은 경우, 이 이메일을 무시하시면 됩니다.

감사합니다.
서비스 팀 드림

© 2025 서비스명. All rights reserved.
문의사항은 support@service.com으로 연락주세요.
이 이메일은 발신 전용입니다. 회신하지 마세요.`, name, verificationLink, token)

	return emailText
}

// 새로운 이메일 전송 함수 (HTML과 텍스트 버전 모두 보내기)
func (s *EmailService) SendWelcomeEmail(from, to, username, code string) error {
	subject := fmt.Sprintf("%s님, 환영합니다!", username)
	htmlBody := s.GenerateWelcomeEmailHTML(username, code)
	_ = s.GenerateWelcomeEmailText(username, code)

	return s.SendEmail(from, to, subject, htmlBody)
}

// 새로운 이메일 인증 메일 전송 함수
func (s *EmailService) SendVerificationEmail(from, to, name, verificationLink, token string) error {
	subject := "이메일 주소 인증"
	htmlBody := s.GenerateVerificationEmailHTML(name, verificationLink, token)
	_ = s.GenerateVerificationEmailText(name, verificationLink, token)

	return s.SendEmail(from, to, subject, htmlBody)
}

// GenerateTeamInvitationEmailHTML 팀 초대 이메일 HTML 템플릿 생성
func (s *EmailService) GenerateTeamInvitationEmailHTML(userName, teamName, inviterName, inviteURL string) string {
	emailHTML := fmt.Sprintf(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="ko">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>팀 초대: %s</title>
</head>
<body style="margin: 0; padding: 0; font-family: 'Apple SD Gothic Neo', 'Malgun Gothic', sans-serif; background-color: #f7f9fc; -webkit-text-size-adjust: 100%%; -ms-text-size-adjust: 100%%;">
	<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="border-collapse: collapse;">
		<tr>
			<td style="padding: 40px 0;">
				<!-- 헤더 -->
				<table align="center" border="0" cellpadding="0" cellspacing="0" width="600" style="border-collapse: collapse; background-color: #5271ff; border-radius: 8px 8px 0 0;">
					<tr>
						<td align="center" style="padding: 30px 0; color: #ffffff;">
							<h1 style="margin: 0; font-size: 28px; font-weight: 700;">팀 초대</h1>
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
										<p style="margin-top: 0; margin-bottom: 20px;"><strong style="color: #5271ff;">%s</strong>님이 <strong style="color: #5271ff;">%s</strong> 팀에 초대하셨습니다.</p>
										<p style="margin-top: 0; margin-bottom: 20px;">아래 버튼을 클릭하여 초대를 수락하거나 거절할 수 있습니다.</p>
									</td>
								</tr>
								<tr>
									<td align="center" style="padding: 25px 0;">
										<!-- 버튼 -->
										<table border="0" cellpadding="0" cellspacing="0" style="border-collapse: separate; border-radius: 4px;">
											<tr>
												<td align="center" style="background-color: #5271ff; border-radius: 4px; padding: 0;">
													<a href="%s" target="_blank" style="display: inline-block; color: #ffffff; font-size: 16px; font-weight: bold; text-decoration: none; padding: 12px 30px; border: 1px solid #4a66e6;">초대 확인하기</a>
												</td>
											</tr>
										</table>
									</td>
								</tr>
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 20px; margin-bottom: 20px;">버튼이 작동하지 않는 경우, 아래 링크를 복사하여 브라우저에 붙여넣기 해주세요:</p>
									</td>
								</tr>
								<tr>
									<td align="center" style="padding: 10px 0 20px 0;">
										<table border="0" cellpadding="0" cellspacing="0" width="80%%" style="border-collapse: collapse;">
											<tr>
												<td align="center" style="background-color: #f3f5ff; border: 1px solid #e1e5ff; border-radius: 8px; padding: 15px;">
													<span style="color: #5271ff; font-size: 14px; word-break: break-all;">%s</span>
												</td>
											</tr>
										</table>
									</td>
								</tr>
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 20px; margin-bottom: 20px;">초대는 7일 동안 유효합니다.</p>
										<p style="margin-top: 0; margin-bottom: 30px;">본인이 요청하지 않은 경우, 이 이메일을 무시하시면 됩니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px;">감사합니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px; font-weight: bold; color: #5271ff;">SEMO 팀 드림</p>
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
							<p style="margin: 0; margin-bottom: 10px;">© 2025 SEMO. All rights reserved.</p>
							<p style="margin: 0; margin-bottom: 10px;">문의사항은 <a href="mailto:support@semo.com" style="color: #5271ff; text-decoration: none;">support@semo.com</a>으로 연락주세요.</p>
							<p style="margin: 0;">이 이메일은 발신 전용입니다. 회신하지 마세요.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`, teamName, userName, inviterName, teamName, inviteURL, inviteURL)

	return emailHTML
}

// GenerateTeamInvitationEmailText 팀 초대 이메일 텍스트 버전 생성
func (s *EmailService) GenerateTeamInvitationEmailText(userName, teamName, inviterName, inviteURL string) string {
	emailText := fmt.Sprintf(`안녕하세요, %s님!

%s님이 %s 팀에 초대하셨습니다.

아래 링크를 클릭하여 초대를 수락하거나 거절할 수 있습니다:

%s

초대는 7일 동안 유효합니다.
본인이 요청하지 않은 경우, 이 이메일을 무시하시면 됩니다.

감사합니다.
SEMO 팀 드림

© 2025 SEMO. All rights reserved.
문의사항은 support@semo.com으로 연락주세요.
이 이메일은 발신 전용입니다. 회신하지 마세요.`, userName, inviterName, teamName, inviteURL)

	return emailText
}

// SendTeamInvitationEmail 팀 초대 이메일 발송
func (s *EmailService) SendTeamInvitationEmail(from, to, userName, teamName, inviterName, inviteURL string) error {
	subject := fmt.Sprintf("%s 팀 초대", teamName)
	htmlBody := s.GenerateTeamInvitationEmailHTML(userName, teamName, inviterName, inviteURL)
	_ = s.GenerateTeamInvitationEmailText(userName, teamName, inviterName, inviteURL)

	return s.SendEmail(from, to, subject, htmlBody)
}

// GenerateRegistrationInvitationEmailHTML creates a registration invitation email HTML template
func (s *EmailService) GenerateRegistrationInvitationEmailHTML(userName, registrationURL string) string {
	emailHTML := fmt.Sprintf(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="ko">
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>회원가입 초대</title>
</head>
<body style="margin: 0; padding: 0; font-family: 'Apple SD Gothic Neo', 'Malgun Gothic', sans-serif; background-color: #f7f9fc; -webkit-text-size-adjust: 100%%; -ms-text-size-adjust: 100%%;">
	<table border="0" cellpadding="0" cellspacing="0" width="100%%" style="border-collapse: collapse;">
		<tr>
			<td style="padding: 40px 0;">
				<!-- 헤더 -->
				<table align="center" border="0" cellpadding="0" cellspacing="0" width="600" style="border-collapse: collapse; background-color: #5271ff; border-radius: 8px 8px 0 0;">
					<tr>
						<td align="center" style="padding: 30px 0; color: #ffffff;">
							<h1 style="margin: 0; font-size: 28px; font-weight: 700;">회원가입 초대</h1>
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
										<p style="margin-top: 0; margin-bottom: 20px;">SEMO 서비스에 초대되셨습니다. 아래 버튼을 클릭하여 회원가입을 완료해 주세요.</p>
									</td>
								</tr>
								<tr>
									<td align="center" style="padding: 25px 0;">
										<!-- 버튼 -->
										<table border="0" cellpadding="0" cellspacing="0" style="border-collapse: separate; border-radius: 4px;">
											<tr>
												<td align="center" style="background-color: #5271ff; border-radius: 4px; padding: 0;">
													<a href="%s" target="_blank" style="display: inline-block; color: #ffffff; font-size: 16px; font-weight: bold; text-decoration: none; padding: 12px 30px; border: 1px solid #4a66e6;">회원가입하기</a>
												</td>
											</tr>
										</table>
									</td>
								</tr>
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 20px; margin-bottom: 20px;">버튼이 작동하지 않는 경우, 아래 링크를 복사하여 브라우저에 붙여넣기 해주세요:</p>
									</td>
								</tr>
								<tr>
									<td align="center" style="padding: 10px 0 20px 0;">
										<table border="0" cellpadding="0" cellspacing="0" width="80%%" style="border-collapse: collapse;">
											<tr>
												<td align="center" style="background-color: #f3f5ff; border: 1px solid #e1e5ff; border-radius: 8px; padding: 15px;">
													<span style="color: #5271ff; font-size: 14px; word-break: break-all;">%s</span>
												</td>
											</tr>
										</table>
									</td>
								</tr>
								<tr>
									<td style="color: #333333; font-size: 16px; line-height: 1.6;">
										<p style="margin-top: 20px; margin-bottom: 20px;">초대는 7일 동안 유효합니다.</p>
										<p style="margin-top: 0; margin-bottom: 30px;">본인이 요청하지 않은 경우, 이 이메일을 무시하시면 됩니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px;">감사합니다.</p>
										<p style="margin-top: 0; margin-bottom: 5px; font-weight: bold; color: #5271ff;">SEMO 팀 드림</p>
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
							<p style="margin: 0; margin-bottom: 10px;">© 2025 SEMO. All rights reserved.</p>
							<p style="margin: 0; margin-bottom: 10px;">문의사항은 <a href="mailto:support@semo.com" style="color: #5271ff; text-decoration: none;">support@semo.com</a>으로 연락주세요.</p>
							<p style="margin: 0;">이 이메일은 발신 전용입니다. 회신하지 마세요.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>`, userName, registrationURL, registrationURL)

	return emailHTML
}

// SendRegistrationInvitationEmail sends a registration invitation email
func (s *EmailService) SendRegistrationInvitationEmail(from, to, userName, registrationURL string) error {
	subject := "SEMO 서비스 회원가입 초대"
	htmlBody := s.GenerateRegistrationInvitationEmailHTML(userName, registrationURL)

	return s.SendEmail(from, to, subject, htmlBody)
}

// Global instance of EmailService
var EmailSvc = NewEmailService(configs.Configs.Email.SMTPHost, configs.Configs.Email.SMTPPort, configs.Configs.Email.Username, configs.Configs.Email.Password)

// Compatibility function for existing code - delegates to EmailSvc
func SendEmail(from, to, subject, htmlBody string) error {
	return EmailSvc.SendEmail(from, to, subject, htmlBody)
}
