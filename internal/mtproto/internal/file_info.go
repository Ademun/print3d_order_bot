package internal

var (
	WebLocationFlag   = int32(1 << 24)
	FileReferenceFlag = int32(1 << 25)
)

type FileIDType int

const (
	IDThumbnail FileIDType = iota
	IDProfilePhoto
	IDPhoto
	IDVoice
	IDVideo
	IDDocument
	IDEncrypted
	IDTemp
	IDSticker
	IDAudio
	IDAnimation
	IDEncryptedThumbnail
	IDWallpaper
	IDVideoNote
	IDSecureRaw
	IDSecure
	IDBackground
	IDSize
)

type FileInfo struct {
	DatacenterID  int
	Type          FileIDType
	ID            *int
	AccessHash    int
	PhotoInfo     *PhotoInfo
	FileReference []byte
	URL           *string
	Version       int
	SubVersion    int
}
