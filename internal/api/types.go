package api

// Album is a row from the library catalog. Fields mirror the backend's
// /library response; only the subset the CLI renders is modeled.
type Album struct {
	ID               int    `json:"id"`
	CodeLetters      string `json:"code_letters"`
	CodeArtistNumber int    `json:"code_artist_number"`
	CodeNumber       int    `json:"code_number"`
	ArtistName       string `json:"artist_name"`
	AlphabeticalName string `json:"alphabetical_name"`
	AlbumTitle       string `json:"album_title"`
	Format           string `json:"format"`
	Label            string `json:"label"`
}

// FlowsheetEntry is one row of the on-air log. A row is either a played track
// (entry_type "track") or a marker (show_start, show_end, dj_join, …). Marker
// rows leave the track fields empty and carry dj_name.
type FlowsheetEntry struct {
	ID          int    `json:"id"`
	ShowID      int    `json:"show_id"`
	PlayOrder   int    `json:"play_order"`
	EntryType   string `json:"entry_type"`
	ArtistName  string `json:"artist_name"`
	AlbumTitle  string `json:"album_title"`
	TrackTitle  string `json:"track_title"`
	RecordLabel string `json:"record_label"`
	DJName      string `json:"dj_name"`
	RequestFlag bool   `json:"request_flag"`
	Segue       bool   `json:"segue"`
	RotationID  *int   `json:"rotation_id"`
}

// flowsheetResponse is the envelope the /flowsheet endpoint wraps entries in.
type flowsheetResponse struct {
	Entries []FlowsheetEntry `json:"entries"`
}

// BinItem is an album a DJ has saved to their bin (mailbox).
type BinItem struct {
	AlbumID     int    `json:"album_id"`
	AlbumTitle  string `json:"album_title"`
	ArtistName  string `json:"artist_name"`
	Label       string `json:"label"`
	CodeLetters string `json:"code_letters"`
}

// Genre is a row from the genres catalog.
type Genre struct {
	ID          int     `json:"id"`
	GenreName   string  `json:"genre_name"`
	Description *string `json:"description"`
	Plays       int     `json:"plays"`
	AddDate     string  `json:"add_date"`
}

// Format is a physical/media format (vinyl, cd, …).
type Format struct {
	ID         int    `json:"id"`
	FormatName string `json:"format_name"`
	DateAdded  string `json:"date_added"`
}

// Label is a record label.
type Label struct {
	ID            int    `json:"id"`
	LabelName     string `json:"label_name"`
	ParentLabelID *int   `json:"parent_label_id"`
}

// ScheduleEntry is one recurring show slot. Day is 0-6.
type ScheduleEntry struct {
	ID            int     `json:"id"`
	Day           int     `json:"day"`
	StartTime     string  `json:"start_time"`
	ShowDuration  int     `json:"show_duration"`
	SpecialtyID   *int    `json:"specialty_id"`
	AssignedDJID  *string `json:"assigned_dj_id"`
	AssignedDJID2 *string `json:"assigned_dj_id2"`
}
