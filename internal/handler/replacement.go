package handler

type Replacement struct{
	ShouldReplaceBody bool `json:"shouldReplaceBody"`
	Body string `json:"body"`
	// ShouldReplaceUri bool `json:"shouldReplaceUri"`
	// Uri string `json:"uri"`
	ShouldReplaceHeader bool `json:"shouldReplaceHeader"`
	Header map[string][]string `json:"Header"`
}