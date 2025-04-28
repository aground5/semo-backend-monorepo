package entity

import (
	"errors"
	"time"
)

// User 비즈니스 도메인 엔티티
type User struct {
	ID                string
	Username          string
	Name              string
	Email             string
	Password          string
	Salt              string
	EmailVerified     bool
	AccountStatus     string
	LastLoginAt       *time.Time
	LastLoginIP       string
	FailedLoginCount  int
	PasswordChangedAt *time.Time
}

// NewUser 사용자 생성 팩토리 함수
func NewUser(username, name, email, password, salt string) (*User, error) {
	if username == "" {
		return nil, errors.New("사용자 이름은 필수입니다")
	}

	if email == "" {
		return nil, errors.New("이메일은 필수입니다")
	}

	if password == "" || salt == "" {
		return nil, errors.New("비밀번호와 솔트는 필수입니다")
	}

	return &User{
		Username:         username,
		Name:             name,
		Email:            email,
		Password:         password,
		Salt:             salt,
		EmailVerified:    false,
		AccountStatus:    "active",
		FailedLoginCount: 0,
	}, nil
}

// IsActive 계정이 활성 상태인지 확인
func (u *User) IsActive() bool {
	return u.AccountStatus == "active"
}

// VerifyEmail 이메일 인증 처리
func (u *User) VerifyEmail() {
	u.EmailVerified = true
}

// RecordLogin 로그인 성공 기록
func (u *User) RecordLogin(ip string) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ip
	u.FailedLoginCount = 0
}

// IncrementFailedLogin 로그인 실패 기록
func (u *User) IncrementFailedLogin() {
	u.FailedLoginCount++
}

// ChangePassword 비밀번호 변경
func (u *User) ChangePassword(newPassword, newSalt string) error {
	if newPassword == "" || newSalt == "" {
		return errors.New("새 비밀번호와 솔트는 필수입니다")
	}

	now := time.Now()
	u.Password = newPassword
	u.Salt = newSalt
	u.PasswordChangedAt = &now

	return nil
}
