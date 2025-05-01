package repository

// Repositories 모든 레포지토리 인터페이스의 컬렉션
type Repositories struct {
	Item     ItemRepository
	File     FileRepository
	Cache    CacheRepository
	Team     TeamRepository
	Profile  ProfileRepository
	Comment  CommentRepository
	Share    ShareRepository
	Activity ActivityRepository
}

// NewRepositories 모든 레포지토리를 포함하는 컬렉션 생성
func NewRepositories(
	itemRepo ItemRepository,
	fileRepo FileRepository,
	cacheRepo CacheRepository,
	teamRepo TeamRepository,
	profileRepo ProfileRepository,
	commentRepo CommentRepository,
	shareRepo ShareRepository,
	activityRepo ActivityRepository,
) *Repositories {
	return &Repositories{
		Item:     itemRepo,
		File:     fileRepo,
		Cache:    cacheRepo,
		Team:     teamRepo,
		Profile:  profileRepo,
		Comment:  commentRepo,
		Share:    shareRepo,
		Activity: activityRepo,
	}
}
