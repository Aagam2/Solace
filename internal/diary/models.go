package diary

type Entry struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	ContentJSON  string   `json:"contentJSON"`
	ContentText  string   `json:"contentText"`
	StartDate    string   `json:"startDate"`
	EndDate      string   `json:"endDate"`
	CreatedAt    string   `json:"createdAt"`
	UpdatedAt    string   `json:"updatedAt"`
	Mood         string   `json:"mood"`
	Favorite     bool     `json:"favorite"`
	Pinned       bool     `json:"pinned"`
	Draft        bool     `json:"draft"`
	LocationText string   `json:"locationText"`
	DisplayOrder *int64   `json:"displayOrder"`
	DeletedAt    string   `json:"deletedAt"`
	Version      int64    `json:"version"`
	Tags         []string `json:"tags"`
}

type EntryInput struct {
	Title        string   `json:"title"`
	ContentJSON  string   `json:"contentJSON"`
	ContentText  string   `json:"contentText"`
	StartDate    string   `json:"startDate"`
	EndDate      string   `json:"endDate"`
	Mood         string   `json:"mood"`
	Favorite     bool     `json:"favorite"`
	Pinned       bool     `json:"pinned"`
	Draft        bool     `json:"draft"`
	LocationText string   `json:"locationText"`
	DisplayOrder *int64   `json:"displayOrder"`
	Tags         []string `json:"tags"`
}

type Revision struct {
	ID            string   `json:"id"`
	EntryID       string   `json:"entryID"`
	Title         string   `json:"title"`
	ContentJSON   string   `json:"contentJSON"`
	ContentText   string   `json:"contentText"`
	StartDate     string   `json:"startDate"`
	EndDate       string   `json:"endDate"`
	Mood          string   `json:"mood"`
	Favorite      bool     `json:"favorite"`
	Pinned        bool     `json:"pinned"`
	Draft         bool     `json:"draft"`
	LocationText  string   `json:"locationText"`
	DisplayOrder  *int64   `json:"displayOrder"`
	Tags          []string `json:"tags"`
	SourceVersion int64    `json:"sourceVersion"`
	SavedAt       string   `json:"savedAt"`
}

type EntrySummary struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Preview         string `json:"preview"`
	StartDate       string `json:"startDate"`
	EndDate         string `json:"endDate"`
	UpdatedAt       string `json:"updatedAt"`
	Mood            string `json:"mood"`
	Favorite        bool   `json:"favorite"`
	Pinned          bool   `json:"pinned"`
	Draft           bool   `json:"draft"`
	AttachmentCount int    `json:"attachmentCount"`
}

type CalendarDay struct {
	Date           string `json:"date"`
	EntryCount     int    `json:"entryCount"`
	Mood           string `json:"mood"`
	HasAttachments bool   `json:"hasAttachments"`
}

type TimelineOptions struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
}

type TimelineOccurrence struct {
	Date  string       `json:"date"`
	Entry EntrySummary `json:"entry"`
}

type TimelinePage struct {
	Occurrences []TimelineOccurrence `json:"occurrences"`
	Total       int                  `json:"total"`
	HasMore     bool                 `json:"hasMore"`
}

type TodayDashboard struct {
	Today          string         `json:"today"`
	TodayEntries   []EntrySummary `json:"todayEntries"`
	RecentlyEdited []EntrySummary `json:"recentlyEdited"`
	RecentMemories []EntrySummary `json:"recentMemories"`
	OnThisDay      []EntrySummary `json:"onThisDay"`
}

type ListOptions struct {
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	Sort          string `json:"sort"`
	FavoritesOnly bool   `json:"favoritesOnly"`
	DraftsOnly    bool   `json:"draftsOnly"`
	IncludeTrash  bool   `json:"includeTrash"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
}

type SearchOptions struct {
	Query         string   `json:"query"`
	StartDate     string   `json:"startDate"`
	EndDate       string   `json:"endDate"`
	Tags          []string `json:"tags"`
	Mood          string   `json:"mood"`
	FavoritesOnly bool     `json:"favoritesOnly"`
	HasImages     bool     `json:"hasImages"`
	HasVideos     bool     `json:"hasVideos"`
	DraftStatus   string   `json:"draftStatus"`
	Sort          string   `json:"sort"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
}

type SearchResult struct {
	Entry            EntrySummary `json:"entry"`
	HighlightedTitle string       `json:"highlightedTitle"`
	Snippet          string       `json:"snippet"`
	Tags             []string     `json:"tags"`
}

type SearchPage struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
	HasMore bool           `json:"hasMore"`
}

type Tag struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	EntryCount int    `json:"entryCount"`
}
