package model

type Category struct {
	BaseModel
	Name     string         `gorm:"column:name;type:varchar(64);not null;default:''" json:"name"`
	ParentID uint64         `gorm:"column:parent_id;not null;default:0" json:"parent_id"`
	Sort     uint           `gorm:"column:sort;not null;default:0" json:"sort"`
	Status   CategoryStatus `gorm:"column:status;not null;default:1" json:"status"`
}

func (Category) TableName() string { return "categories" }
