package model

import (
	"encoding/json"
	"time"

	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	bridge "github.com/naturalselectionlabs/pregod/common/database/model/transaction"
)

type WrappedResult struct {
	Social      SocialResult `json:"social"`
	Search      SearchResult `json:"search"`
	Gas         GasResult    `json:"gas"`
	Transaction TxResult     `json:"transaction"`
	NFT         NFTResult    `json:"nft"`
	DApp        DAppResult   `json:"dapp"`
	DeFi        DeFiResult   `json:"defi"`
}

type SocialResult struct {
	Post         int64           `json:"post"`
	Comment      int64           `json:"comment"`
	Following    int64           `json:"following"`
	Follower     int64           `json:"follower"`
	LongestHash  string          `json:"longest_hash"`
	ShortestHash string          `json:"shortest_hash"`
	List         []DApp          `json:"list" gorm:"-"`
	Heatmap      []HeatmapSingle `json:"heatmap" gorm:"-"`
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
	Initiate []NetworkCount  `json:"initiated"`
	Receive  []NetworkCount  `json:"received"`
	Heatmap  []HeatmapSingle `json:"heatmap" gorm:"-"`
}

type NetworkCount struct {
	Network string `json:"network"`
	Total   int64  `json:"total"`
}

type NFTResult struct {
	Bought []metadata.Token `json:"bought"`
	Sold   []metadata.Token `json:"sold"`
	Mint   []metadata.Token `json:"mint"`
	Total  int              `json:"total"`
	First  *NFTSingle       `json:"first"`
	Last   *NFTSingle       `json:"last"`
}

type NFT struct {
	Metadata  json.RawMessage `json:"metadata"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
}

type NFTSingle struct {
	Metadata  metadata.Token `json:"metadata"`
	Timestamp time.Time      `json:"timestamp"`
}

type DAppResult struct {
	List []DApp `json:"list"`
}

type DApp struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type DeFiResult struct {
	PlatformList []DApp          `json:"platform_list"`
	SwapPair     []SwapPair      `json:"swap_pair"`
	Bridge       []bridge.Bridge `json:"bridge"`
	Liquidity    Liquidity       `json:"liquidity"`
	Heatmap      []HeatmapSingle `json:"heatmap" gorm:"-"`
}

type SwapPair struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Liquidity struct {
	Add      []metadata.Token `json:"add"`
	Remove   []metadata.Token `json:"remove"`
	Supply   []metadata.Token `json:"supply"`
	Withdraw []metadata.Token `json:"withdraw"`
	Borrow   []metadata.Token `json:"borrow"`
	Repay    []metadata.Token `json:"repay"`
	Collect  []metadata.Token `json:"collect"`
}

type HeatmapSingle struct {
	Count int64  `json:"count"`
	Date  string `json:"date"`
}
