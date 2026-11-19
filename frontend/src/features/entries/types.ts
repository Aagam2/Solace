export interface Entry {
  id: string
  title: string
  contentJSON: string
  contentText: string
  startDate: string
  endDate: string
  createdAt: string
  updatedAt: string
  mood: string
  favorite: boolean
  pinned: boolean
  draft: boolean
  locationText: string
  displayOrder: number | null
  deletedAt: string
  version: number
  tags: string[]
}

export type EntryInput = Pick<Entry, 'title' | 'contentJSON' | 'contentText' | 'startDate' | 'endDate' | 'mood' | 'favorite' | 'pinned' | 'draft' | 'locationText' | 'displayOrder' | 'tags'>

export interface Revision {
  id:string; entryID:string; title:string; contentJSON:string; contentText:string; startDate:string; endDate:string;
  mood:string; favorite:boolean; pinned:boolean; draft:boolean; locationText:string; displayOrder:number|null;
  tags:string[]; sourceVersion:number; savedAt:string;
}

export interface Attachment {
  id: string
  entryID: string
  originalFilename: string
  storedFilename: string
  relativePath: string
  mimeType: string
  fileSize: number
  createdAt: string
  caption: string
  displayOrder: number
  checksum: string
  metadataJSON: string
  available: boolean
  url: string
  thumbnailURL: string
}

export interface ListOptions {
  startDate?: string
  endDate?: string
  sort?: 'newest' | 'oldest' | 'updated' | 'favorites'
  favoritesOnly?: boolean
  draftsOnly?: boolean
  includeTrash?: boolean
  limit?: number
  offset?: number
}

export interface EntrySummary {
  id: string
  title: string
  preview: string
  startDate: string
  endDate: string
  updatedAt: string
  mood: string
  favorite: boolean
  pinned: boolean
  draft: boolean
  attachmentCount: number
}

export interface CalendarDay { date:string; entryCount:number; mood:string; hasAttachments:boolean }
export interface TimelineOccurrence { date:string; entry:EntrySummary }
export interface TimelinePage { occurrences:TimelineOccurrence[]; total:number; hasMore:boolean }
export interface TodayDashboard { today:string; todayEntries:EntrySummary[]; recentlyEdited:EntrySummary[]; recentMemories:EntrySummary[]; onThisDay:EntrySummary[] }

export interface SearchOptions {
  query:string; startDate:string; endDate:string; tags:string[]; mood:string; favoritesOnly:boolean;
  hasImages:boolean; hasVideos:boolean; draftStatus:'any'|'draft'|'published'; sort:'relevance'|'newest'|'oldest'; limit:number; offset:number;
}
export interface SearchResult { entry:EntrySummary; highlightedTitle:string; snippet:string; tags:string[] }
export interface SearchPage { results:SearchResult[]; total:number; hasMore:boolean }
export interface TagSummary { id:string; name:string; entryCount:number }
