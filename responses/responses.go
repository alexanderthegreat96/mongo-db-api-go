package responses

type GenericErrorResponse struct {
	Code   int         `json:"code"`
	Status bool        `json:"status"`
	Error  string      `json:"error"`
	Query  interface{} `json:"query"`
}

type DatabaseListResponse struct {
	Status    bool     `json:"status"`
	Databases []string `json:"databases"`
}

type DeleteDatabaseSuccessResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type TablesInDatabaseResponse struct {
	Status bool     `json:"status"`
	Tables []string `json:"tables"`
}

type WipeTableInDatabaseResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

type SelectResultsPaginationResponse struct {
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	NextPage    int `json:"next_page"`
	PrevPage    int `json:"prev_page"`
	LastPage    int `json:"last_page"`
	PerPage     int `json:"per_page"`
}

// results mapping
type SelectResultsResponse struct {
	Status     bool                            `json:"status"`
	Code       int                             `json:"code"`
	Database   string                          `json:"database"`
	Table      string                          `json:"table"`
	Count      int64                           `json:"count"`
	Pagination SelectResultsPaginationResponse `json:"pagination"`
	Query      interface{}                     `json:"query"`
	Results    []map[string]interface{}        `json:"results"`
}
