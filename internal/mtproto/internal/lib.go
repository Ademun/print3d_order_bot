package internal

import (
	"bytes"
	"encoding/binary"
)

func ParseFileID(fileID string) (*FileInfo, error) {
	data, err := parseBase64RLE(fileID)
	if err != nil {
		return nil, err
	}

	result := &FileInfo{}

	version := int(data[len(data)-1])
	subVersion := 0
	if version == 4 {
		subVersion = int(data[len(data)-2])
	}

	result.Version = version
	result.SubVersion = subVersion

	buf := bytes.NewBuffer(data)

	var typeIDInt int32
	if err = binary.Read(buf, binary.LittleEndian, &typeIDInt); err != nil {
		return nil, err
	}
	var hasWebLocation bool
	if (typeIDInt & WebLocationFlag) > 0 {
		hasWebLocation = true
	}
	var hasFileReference bool
	if (typeIDInt & FileReferenceFlag) > 0 {
		hasFileReference = true
	}
	typeIDInt &= ^WebLocationFlag
	typeIDInt &= ^FileReferenceFlag
	typeID := FileIDType(typeIDInt)
	result.Type = typeID

	var dcID int32
	if err = binary.Read(buf, binary.LittleEndian, &dcID); err != nil {
		return nil, err
	}
	result.DatacenterID = int(dcID)

	if hasFileReference {
		fileReference, consumed, err := readTLBytes(buf.Bytes())
		if err != nil {
			return nil, err
		}
		result.FileReference = fileReference
		buf.Next(consumed)
	}

	if hasWebLocation {
		url, consumed, err := readTLBytes(buf.Bytes())
		if err != nil {
			return nil, err
		}
		result.URL = qptr(string(url))
		buf.Next(consumed)

		var accessHash int64
		if err := binary.Read(buf, binary.LittleEndian, &accessHash); err != nil {
			return nil, err
		}
		result.AccessHash = int(accessHash)
		return result, nil
	} else {
		var id int64
		if err := binary.Read(buf, binary.LittleEndian, &id); err != nil {
			return nil, err
		}
		result.ID = qptr(int(id))
		var accessHash int64
		if err := binary.Read(buf, binary.LittleEndian, &accessHash); err != nil {
			return nil, err
		}
		result.AccessHash = int(accessHash)
	}

	if typeID <= IDPhoto {
		photoInfo, err := readPhotoInfo(buf, subVersion)
		if err != nil {
			return nil, err
		}
		result.PhotoInfo = photoInfo
	}

	return result, nil
}

func readPhotoInfo(buf *bytes.Buffer, subVersion int) (*PhotoInfo, error) {
	photoInfo := &PhotoInfo{}

	if subVersion < 32 {
		var volumeID int64
		if err := binary.Read(buf, binary.LittleEndian, &volumeID); err != nil {
			return nil, err
		}
		photoInfo.VolumeID = qptr(int(volumeID))
	}

	var argInt uint32
	if subVersion >= 22 {
		if err := binary.Read(buf, binary.LittleEndian, &argInt); err != nil {
			return nil, err
		}
	}

	argType := PhotoSizeSourceType(argInt)
	switch argType {
	case SourceLegacy, SourceFullLegacy:
		var secret int64
		if err := binary.Read(buf, binary.LittleEndian, &secret); err != nil {
			return nil, err
		}
		photoInfo.PhotoSizeSource = &PhotoSizeSourceLegacy{Secret: int(secret)}
	case SourceThumbnail:
		var typeIDInt int32
		if err := binary.Read(buf, binary.LittleEndian, &typeIDInt); err != nil {
			return nil, err
		}
		typeID := FileIDType(typeIDInt)
		thumbnailType := string([]rune(string(buf.Next(4)))[0])
		photoInfo.PhotoSizeSource = &PhotoSizeSourceThumbnail{
			FileType:      typeID,
			ThumbnailSize: thumbnailType,
		}
	case SourceDialogPhotoSmall, SourceDialogPhotoSmallLegacy, SourceDialogPhotoBig, SourceDialogPhotoBigLegacy:
		var dialogId int64
		if err := binary.Read(buf, binary.LittleEndian, &dialogId); err != nil {
			return nil, err
		}
		var dialogAccessHash int64
		if err := binary.Read(buf, binary.LittleEndian, &dialogAccessHash); err != nil {
			return nil, err
		}
		baseSource := BasePhotoSizeDialogPhoto{
			DialogID:         int(dialogId),
			DialogAccessHash: int(dialogAccessHash),
		}
		if argType == SourceDialogPhotoSmall || argType == SourceDialogPhotoSmallLegacy {
			photoInfo.PhotoSizeSource = &PhotoSizeDialogPhotoSmall{
				BasePhotoSizeDialogPhoto: baseSource,
			}
		} else {
			photoInfo.PhotoSizeSource = &PhotoSizeDialogPhotoBig{
				BasePhotoSizeDialogPhoto: baseSource,
			}
		}
	case SourceStickerSetThumbnail, SourceStickerSetThumbnailLegacy, SourceStickerSetThumbnailVersion:
		var stickerSetId int64
		if err := binary.Read(buf, binary.LittleEndian, &stickerSetId); err != nil {
			return nil, err
		}
		var stickerSetAccessHash int64
		if err := binary.Read(buf, binary.LittleEndian, &stickerSetAccessHash); err != nil {
			return nil, err
		}
		base := PhotoSizeSourceStickerSetThumbnail{
			ID:         int(stickerSetId),
			AccessHash: int(stickerSetAccessHash),
		}
		if argType == SourceStickerSetThumbnailVersion {
			var version int32
			if err := binary.Read(buf, binary.LittleEndian, &version); err != nil {
				return nil, err
			}
			photoInfo.PhotoSizeSource = &PhotoSizeSourceStickerSetThumbnailVersion{
				PhotoSizeSourceStickerSetThumbnail: base,
				Version:                            int(version),
			}
		} else {
			photoInfo.PhotoSizeSource = &base
		}
	}

	if argType == SourceFullLegacy || (subVersion >= 22 && subVersion < 32) {
		var localID int32
		if err := binary.Read(buf, binary.LittleEndian, &localID); err != nil {
			return nil, err
		}
		photoInfo.LocalID = qptr(int(localID))
	}
	return photoInfo, nil
}
