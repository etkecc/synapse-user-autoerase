package models

// DeletedMediaResponse holds the Synapse response; only fields needed by this application are included.
type DeletedMediaResponse struct {
	Total int `json:"total"`
}

// MediaResponse holds the Synapse response; only fields needed by this application are included.
type MediaResponse struct {
	Total int64 `json:"total"`
}
