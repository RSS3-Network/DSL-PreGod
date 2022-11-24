package model

type WrappedResult struct {
	Social      SocialResult `json:"social"`
	Search      SearchResult `json:"search"`
	Gas         GasResult    `json:"gas"`
	Transaction TxResult     `json:"transaction"`
}

type SocialResult struct {
	Post         int64  `json:"post"`
	Comment      int64  `json:"comment"`
	Follow       int64  `json:"follow"`
	LongestHash  string `json:"longest_hash"`
	ShortestHash string `json:"shortest_hash"`
}

type SearchResult struct {
	Count int64 `json:"count"`
}

type GasResult struct {
	Total       string `json:"total"`
	Highest     string `json:"highest"`
	HighestHash string `json:"highest_hash"`
}

type TxResult struct {
	Initiate []NetworkCount `json:"initiated"`
	Receive  []NetworkCount `json:"received"`
}

type NetworkCount struct {
	Network string `json:"network"`
	Count   int64  `json:"count"`
}
