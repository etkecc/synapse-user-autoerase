package models

// DeletedMediaResponse is a struct that holds the response from the Synapse server,
// note that the included fields are only the ones that are needed for this application.
type DeletedMediaResponse struct {
	Total int `json:"total"`
}
