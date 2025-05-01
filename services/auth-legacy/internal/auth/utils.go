package auth

import (
	"authn-server/internal/logics"
	"authn-server/internal/models"
	"context"
	"crypto/rand"
	"encoding/base64"
	"math/big"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// GenerateRandomString는 지정된 길이의 무작위 문자열을 생성합니다.
func GenerateRandomString(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		randIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[randIndex.Int64()]
	}
	return string(b)
}

// GenerateRandomCode는 지정된 길이의 무작위 코드를 생성합니다.
func GenerateRandomCode(length int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	for i := range b {
		randIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		b[i] = letters[int(randIndex.Int64())]
	}
	return string(b)
}

// HashPassword는 비밀번호를 해싱하고 솔트를 반환합니다.
func HashPassword(password string) (hashedPassword string, salt string, err error) {
	// 솔트 생성
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", err
	}
	salt = base64.StdEncoding.EncodeToString(saltBytes)

	// 비밀번호 해싱
	hash, err := bcrypt.GenerateFromPassword([]byte(password+salt), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	return string(hash), salt, nil
}

// VerifyPassword는 제공된 비밀번호가 저장된 해시와 일치하는지 확인합니다.
func VerifyPassword(hashedPassword, inputPassword, salt string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword+salt))
}

// ExtractUsernameFromEmail은 이메일에서 사용자 이름 부분을 추출합니다.
func ExtractUsernameFromEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// LogUserAction은 사용자 활동에 대한 감사 로그를 기록합니다.
func LogUserAction(ctx context.Context, logType string, email, ip, userAgent string, userID *string) {
	// 로그 콘텐츠 구성
	content := map[string]interface{}{
		"email":      email,
		"ip":         ip,
		"user_agent": userAgent,
	}

	// logics.AuditLogSvc의 AddLog 메서드를 사용하여 로그 기록
	_ = logics.AuditLogSvc.AddLog(models.AuditLogType(logType), content, userID)
}
