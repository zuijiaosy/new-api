package model

// OssImage 记录图片 OSS 转存结果，定时任务按 CreatedAt 清理。
type OssImage struct {
	Id        int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	FileKey   string `json:"file_key" gorm:"type:varchar(512);uniqueIndex;not null"`
	PublicUrl string `json:"public_url" gorm:"type:varchar(1024);not null"`
	MimeType  string `json:"mime_type" gorm:"type:varchar(64)"`
	SizeBytes int64  `json:"size_bytes"`

	UserId    int    `json:"user_id" gorm:"index"`
	ChannelId int    `json:"channel_id" gorm:"index"`
	TokenId   int    `json:"token_id"`
	ModelName string `json:"model_name" gorm:"type:varchar(128)"`

	UpstreamUrl string `json:"upstream_url" gorm:"type:varchar(2048)"`

	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime;index"`
}

func (OssImage) TableName() string { return "oss_images" }

// BatchCreateOssImages 批量插入。空切片直接返回 nil。
func BatchCreateOssImages(imgs []OssImage) error {
	if len(imgs) == 0 {
		return nil
	}
	return DB.Create(&imgs).Error
}

// CreateOssImage 单条插入（测试/辅助路径用）。
func CreateOssImage(img *OssImage) error {
	return DB.Create(img).Error
}

// GetOssImageById 按 id 查询。
func GetOssImageById(id int64) (*OssImage, error) {
	var img OssImage
	if err := DB.First(&img, id).Error; err != nil {
		return nil, err
	}
	return &img, nil
}

// ListExpiredOssImages 返回 CreatedAt < beforeUnix 的记录，最多 limit 条。
// 语义：严格小于（等于 threshold 不返回），按 created_at 升序先清老的。
func ListExpiredOssImages(beforeUnix int64, limit int) ([]OssImage, error) {
	if limit <= 0 {
		return nil, nil
	}
	var out []OssImage
	err := DB.Where("created_at < ?", beforeUnix).
		Order("created_at ASC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

// DeleteOssImagesByIds 批量删除，返回实际影响行数。
func DeleteOssImagesByIds(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := DB.Where("id IN ?", ids).Delete(&OssImage{})
	return res.RowsAffected, res.Error
}
