package core

// ProjectView contains permissions already resolved by the registry for this
// identity. It never contains private paths, credentials or bearer token IDs.
type ProjectRepository struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Branch  string `json:"branch"`
	Version int64  `json:"version"`
}
type ProjectView struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	State        string              `json:"state"`
	Version      int64               `json:"version"`
	Role         string              `json:"role"`
	CanWrite     bool                `json:"can_write"`
	CanArchive   bool                `json:"can_archive"`
	CanManage    bool                `json:"can_manage"`
	Repositories []ProjectRepository `json:"repositories"`
}
