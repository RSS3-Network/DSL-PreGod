package protocol

const (
	NetworkEthereum          = "ethereum"
	NetworkEthereumClassic   = "ethereum_classic"
	NetworkBinanceSmartChain = "binance_smart_chain"
	NetworkPolygon           = "polygon"
	NetworkZkSync            = "zksync"
	NetworkXDAI              = "xdai"
	NetworkArweave           = "arweave"
	NetworkArbitrum          = "arbitrum"
	NetworkOptimism          = "optimism"
	NetworkFantom            = "fantom"
	NetworkAvalanche         = "avalanche"
	NetworkCrossbell         = "crossbell"
)

func NetworkToID(networkName string) string {
	switch networkName {
	case NetworkEthereum:
		return "0x1"
	case NetworkBinanceSmartChain:
		return "0x38"
	case NetworkPolygon:
		return "0x89"
	case NetworkZkSync:
		return "0x118"
	case NetworkXDAI:
		return "0x64"
	case NetworkArbitrum:
		return "0xA4B1"
	case NetworkOptimism:
		return "0xA"
	case NetworkFantom:
		return "0xFA"
	case NetworkAvalanche:
		return "0xA86A"
	default:
		return "0x0"
	}
}
