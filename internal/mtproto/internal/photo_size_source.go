package internal

type PhotoSizeSource interface {
	PhotoSizeSource()
}

type PhotoSizeSourceType int

const (
	SourceLegacy PhotoSizeSourceType = iota
	SourceThumbnail
	SourceDialogPhotoSmall
	SourceDialogPhotoBig
	SourceStickerSetThumbnail
	SourceFullLegacy
	SourceDialogPhotoSmallLegacy
	SourceDialogPhotoBigLegacy
	SourceStickerSetThumbnailLegacy
	SourceStickerSetThumbnailVersion
)

type PhotoSizeSourceLegacy struct {
	Secret int
}

func (s *PhotoSizeSourceLegacy) PhotoSizeSource() {

}

type PhotoSizeSourceThumbnail struct {
	FileType      FileIDType
	ThumbnailSize string
}

func (s *PhotoSizeSourceThumbnail) PhotoSizeSource() {

}

type PhotoSizeSourceDialogPhoto interface {
	IsSmallDialogPhoto() bool
}

type BasePhotoSizeDialogPhoto struct {
	DialogID         int
	DialogAccessHash int
}

type PhotoSizeDialogPhotoSmall struct {
	BasePhotoSizeDialogPhoto
}

func (s *PhotoSizeDialogPhotoSmall) PhotoSizeSource() {}

func (s *PhotoSizeDialogPhotoSmall) IsSmallDialogPhoto() bool {
	return true
}

type PhotoSizeDialogPhotoBig struct {
	BasePhotoSizeDialogPhoto
}

func (s *PhotoSizeDialogPhotoBig) PhotoSizeSource() {}

func (s *PhotoSizeDialogPhotoBig) IsSmallDialogPhoto() bool {
	return true
}

type PhotoSizeSourceStickerSetThumbnail struct {
	ID         int
	AccessHash int
}

func (s *PhotoSizeSourceStickerSetThumbnail) PhotoSizeSource() {}

type PhotoSizeSourceStickerSetThumbnailVersion struct {
	PhotoSizeSourceStickerSetThumbnail
	Version int
}

func (s *PhotoSizeSourceStickerSetThumbnailVersion) PhotoSizeSource() {}

type PhotoInfo struct {
	PhotoSizeSource PhotoSizeSource
	VolumeID        *int
	LocalID         *int
}
