package responses

type GenericErrorResponse struct {
	Code     int    `json:"code"`
	Status   bool   `json:"status"`
	Error    string `json:"error"`
	Database string `json:"database"`
	Table    string `json:"table"`
	Query    any    `json:"query"`
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
	Query      any                             `json:"query"`
	Results    []map[string]any                `json:"results"`
}

type MongoOperationsResultResponse struct {
	Status    bool   `json:"status"`
	Code      int    `json:"code"`
	Database  string `json:"database"`
	Table     string `json:"table"`
	Operation string `json:"operation"`
	Message   string `json:"message"`
	Query     any    `json:"query"`
}

type SelectSingleResultResponse struct {
	Status   bool   `json:"status"`
	Code     int    `json:"code"`
	Database string `json:"database"`
	Table    string `json:"table"`
	Result   any    `json:"result"`
}

// inner pagination for each group
type InnerPaginationResponse struct {
	TotalPages  int `json:"total_pages"`
	CurrentPage int `json:"current_page"`
	NextPage    int `json:"next_page"`
	PrevPage    int `json:"prev_page"`
	LastPage    int `json:"last_page"`
	PerPage     int `json:"per_page"`
}

// one bucket of grouped results
type GroupBucketResponse struct {
	ID              any                     `json:"_id"`           // the group key
	TotalRecords    int64                   `json:"total_records"` // how many items in that group
	Records         []map[string]any        `json:"records"`       // the paged slice
	InnerPagination InnerPaginationResponse `json:"inner_pagination"`
}

// the top-level response when grouping is used
type SelectGroupedResultsResponse struct {
	Status     bool                    `json:"status"`
	Code       int                     `json:"code"`
	Database   string                  `json:"database"`
	Table      string                  `json:"table"`
	Count      int                     `json:"count"`      // number of group buckets (usually 1)
	Pagination InnerPaginationResponse `json:"pagination"` // show inner pagination here
	Query      any                     `json:"query"`
	Results    []GroupBucketResponse   `json:"results"`
}
