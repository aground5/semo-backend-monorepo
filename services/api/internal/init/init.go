package init

import (
	"github.com/wekeepgrowing/semo-backend-monorepo/services/api/internal/config"
)

// Init 애플리케이션 초기화
func Init() error {
	// 설정 로드
	_, err := config.Load()
	if err != nil {
		return err
	}

	return nil
}
