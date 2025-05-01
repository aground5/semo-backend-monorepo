package entity

import (
	"errors"
	"time"
)

// ItemType은 아이템 유형을 정의합니다
type ItemType string

const (
	ItemTypeTask    ItemType = "task"
	ItemTypeProject ItemType = "project"
)

// Item은 태스크, 프로젝트와 같은 작업 항목을 표현하는 도메인 엔티티입니다
type Item struct {
	ID          string
	ParentID    *string
	Type        ItemType
	Name        string
	Contents    string
	Objective   string
	Deliverable string
	Role        string
	Color       string
	Position    float64
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewItem은 새로운 아이템을 생성하는 팩토리 함수입니다
func NewItem(name, contents string, itemType ItemType, createdBy string, parentID *string) (*Item, error) {
	if name == "" {
		return nil, errors.New("이름은 필수입니다")
	}

	if itemType == "" {
		return nil, errors.New("아이템 유형은 필수입니다")
	}

	if createdBy == "" {
		return nil, errors.New("생성자는 필수입니다")
	}

	now := time.Now()

	return &Item{
		Name:      name,
		Contents:  contents,
		Type:      itemType,
		ParentID:  parentID,
		CreatedBy: createdBy,
		Color:     "000000", // 기본 색상
		Position:  0,        // 기본 위치
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// SetObjective는 아이템의 목표를 설정합니다
func (i *Item) SetObjective(objective string) {
	i.Objective = objective
	i.UpdatedAt = time.Now()
}

// SetDeliverable은 아이템의 예상 결과물을 설정합니다
func (i *Item) SetDeliverable(deliverable string) {
	i.Deliverable = deliverable
	i.UpdatedAt = time.Now()
}

// SetRole은 아이템의 역할을 설정합니다
func (i *Item) SetRole(role string) {
	i.Role = role
	i.UpdatedAt = time.Now()
}

// SetColor는 아이템의 색상을 설정합니다
func (i *Item) SetColor(color string) {
	i.Color = color
	i.UpdatedAt = time.Now()
}

// SetPosition은 아이템의 위치를 설정합니다
func (i *Item) SetPosition(position float64) {
	i.Position = position
	i.UpdatedAt = time.Now()
}

// Update는 아이템 정보를 업데이트합니다
func (i *Item) Update(name, contents string) {
	if name != "" {
		i.Name = name
	}

	if contents != "" {
		i.Contents = contents
	}

	i.UpdatedAt = time.Now()
}

// MoveToParent는 아이템을 다른 부모 아래로 이동합니다
func (i *Item) MoveToParent(parentID *string) {
	i.ParentID = parentID
	i.UpdatedAt = time.Now()
}
