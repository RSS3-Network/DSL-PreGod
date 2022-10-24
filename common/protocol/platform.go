package protocol

import "github.com/naturalselectionlabs/pregod/common/protocol/filter"

const (
	// social
	PlatformMirror         = "Mirror"
	PlatformLens           = "Lens"
	PlatformLenster        = "Lenster" // sub platform
	PlatformCrossbell      = "Crossbell"
	PlatformCrossbellXLog  = "xLog"
	PlatformCrossbellXCast = "xCast"
	PlatformFarcaster      = "Farcaster"
	PlatformEveripedia     = "Everipedia"

	// collectible
	PlatformPOAP   = "POAP"
	PlatformGalaxy = "Galaxy"
	PlatformEns    = "ENS Registrar"

	// donation
	PlatformGitcoin = "Gitcoin"

	// governance
	PlatformSnapshot = "Snapshot"

	// exchange
	// dex
	PlatformUniswap     = "Uniswap"
	PlatformSushiswap   = "SushiSwap"
	PlatformPancakeswap = "PancakeSwap"
	Platform1inch       = "1inch"
	PlatformMetamask    = "MetaMask"
	Platform0x          = "0x"
	PlatformAAVE        = "AAVE"
	PlatformCurve       = "Curve"
)

var PlatformList = map[string][]string{
	filter.TagSocial: {
		PlatformMirror,
		PlatformLens,
		PlatformLenster,
		PlatformCrossbell,
		PlatformCrossbellXLog,
		PlatformFarcaster,
		NetworkEIP1577,
		PlatformEveripedia,
	},
	filter.TagCollectible: {
		PlatformPOAP,
		PlatformGalaxy,
		PlatformEns,
	},
	filter.TagDonation: {
		PlatformGitcoin,
	},
	filter.TagGovernance: {
		PlatformSnapshot,
	},
	filter.TagExchange: {
		PlatformUniswap,
		PlatformSushiswap,
		PlatformPancakeswap,
		Platform1inch,
		PlatformMetamask,
		Platform0x,
		PlatformAAVE,
		PlatformCurve,
	},
}
