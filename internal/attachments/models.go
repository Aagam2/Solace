package attachments

type Attachment struct {
	ID               string `json:"id"`
	EntryID          string `json:"entryID"`
	OriginalFilename string `json:"originalFilename"`
	StoredFilename   string `json:"storedFilename"`
	RelativePath     string `json:"relativePath"`
	MIMEType         string `json:"mimeType"`
	FileSize         int64  `json:"fileSize"`
	CreatedAt        string `json:"createdAt"`
	Caption          string `json:"caption"`
	DisplayOrder     int    `json:"displayOrder"`
	Checksum         string `json:"checksum"`
	MetadataJSON     string `json:"metadataJSON"`
	Available        bool   `json:"available"`
	URL              string `json:"url"`
	ThumbnailURL     string `json:"thumbnailURL"`
}
