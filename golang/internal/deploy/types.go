package deploy

type DeployRequest struct {
	ProjectName string `json:"projectName"`
	RepoURL     string `json:"repoURL"`
}

type DeployResponse struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}
