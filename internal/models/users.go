package models

// Account is a struct that holds the information about a user account,
// note that the included fields are only the ones that are needed for this application.
type Account struct {
	Name          string `json:"name"`
	IsGuest       bool   `json:"is_guest"`
	Admin         bool   `json:"admin"`
	Deactivated   bool   `json:"deactivated"`
	Locked        bool   `json:"locked"`
	CreationTs    int64  `json:"creation_ts"`
	UploadedMedia int64  `json:"-"` // This field is not included in the JSON response
}

// AccountsResponse is a struct that holds the response from the Synapse server
type AccountsResponse struct {
	Accounts  []*Account `json:"users"`
	NextToken string     `json:"next_token"`
	Total     int        `json:"total"`
}
