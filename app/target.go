package app

type Target struct {
	RelativePath       string            `json:"relativePath"`
	ExpectedStatusCode int               `json:"expectedStatusCode"`
	RequestBody        *string           `json:"requestBody"`
	RequestHeaders     map[string]string `json:"requestHeaders"`
	PatternPrefix      *string           `json:"patternPrefix,omitempty"`
	PatternSuffix      *string           `json:"patternSuffix,omitempty"`
}
