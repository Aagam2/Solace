package portable

import "solace/internal/diary"

const (
	ExportVersion  = 2
	ArchiveVersion = 1
)

type ExportRequest struct {
	Format             string   `json:"format"`
	Scope              string   `json:"scope"`
	StartDate          string   `json:"startDate"`
	EndDate            string   `json:"endDate"`
	EntryIDs           []string `json:"entryIDs"`
	IncludeAttachments bool     `json:"includeAttachments"`
	Destination        string   `json:"destination"`
}

type OperationResult struct {
	OutputPath       string   `json:"outputPath"`
	EntryCount       int      `json:"entryCount"`
	AttachmentCount  int      `json:"attachmentCount"`
	Warnings         []string `json:"warnings"`
	RecoveryBackup   string   `json:"recoveryBackup"`
	OperationMessage string   `json:"operationMessage"`
}

type BackupSettings struct {
	Schedule           string `json:"schedule"`
	Destination        string `json:"destination"`
	Retention          int    `json:"retention"`
	IncludeAttachments bool   `json:"includeAttachments"`
	LastBackupAt       string `json:"lastBackupAt"`
}

type JournalExport struct {
	ExportVersion  int                `json:"exportVersion"`
	AppVersion     string             `json:"appVersion"`
	ExportedAt     string             `json:"exportedAt"`
	SchemaVersions []string           `json:"schemaVersions"`
	Entries        []diary.Entry      `json:"entries"`
	Tags           []TagRecord        `json:"tags"`
	EntryTags      []EntryTagRecord   `json:"entryTags"`
	Attachments    []AttachmentRecord `json:"attachments"`
	Revisions      []diary.Revision   `json:"revisions"`
	Settings       map[string]string  `json:"settings"`
}

type TagRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type EntryTagRecord struct {
	EntryID string `json:"entryID"`
	TagID   string `json:"tagID"`
}

type AttachmentRecord struct {
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
}

type ArchiveManifest struct {
	ArchiveVersion  int          `json:"archiveVersion"`
	ExportVersion   int          `json:"exportVersion"`
	AppVersion      string       `json:"appVersion"`
	CreatedAt       string       `json:"createdAt"`
	EntryCount      int          `json:"entryCount"`
	AttachmentCount int          `json:"attachmentCount"`
	Files           []FileRecord `json:"files"`
}

type FileRecord struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type ArchiveSummary struct {
	ArchiveVersion     int      `json:"archiveVersion"`
	CreatedAt          string   `json:"createdAt"`
	EntryCount         int      `json:"entryCount"`
	AttachmentCount    int      `json:"attachmentCount"`
	MissingAttachments []string `json:"missingAttachments"`
	Valid              bool     `json:"valid"`
}
