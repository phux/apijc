package app

type Results struct {
	Findings []Finding
}

type Finding struct {
	URL   string `json:"url"`
	Error string `json:"error"`
	Diff  string `json:"diff"`
}
