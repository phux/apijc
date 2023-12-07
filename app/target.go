package app

type Target struct {
	RelativePath       string            `json:"relativePath"`
	HTTPMethod         string            `json:"httpMethod"`
	ExpectedStatusCode int               `json:"expectedStatusCode"`
	RequestBody        *string           `json:"requestBody"`
	RequestBodyFile    *string           `json:"requestBodyFile"`
	RequestHeaders     map[string]string `json:"requestHeaders"`
	PatternPrefix      *string           `json:"patternPrefix,omitempty"`
	PatternSuffix      *string           `json:"patternSuffix,omitempty"`
}
