package token_test

import (
	"context"
	"log"
	"math/big"
	"testing"

	configx "github.com/naturalselectionlabs/pregod/common/config"
	"github.com/naturalselectionlabs/pregod/common/ethclientx"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/internal/token"
	"github.com/stretchr/testify/assert"
)

var tokenClient *token.Client

func init() {
	rpcConfig := configx.RPC{
		General: configx.RPCNetwork{
			Ethereum: &configx.RPCEndpoint{
				WebSocket: "https://eth.llamarpc.com",
			},
			Polygon: &configx.RPCEndpoint{
				WebSocket: "https://polygon.llamarpc.com",
			},
		},
	}

	globalEthereumClientMap, err := ethclientx.Dial(&rpcConfig, []string{
		protocol.NetworkEthereum, protocol.NetworkPolygon,
	})
	if err != nil {
		log.Fatal(err)
	}

	ethclientx.ReplaceGlobal(protocol.NetworkEthereum, globalEthereumClientMap[protocol.NetworkEthereum])
	ethclientx.ReplaceGlobal(protocol.NetworkPolygon, globalEthereumClientMap[protocol.NetworkPolygon])
	tokenClient = token.New()
}

func TestClient_Native(t *testing.T) {
	native, err := tokenClient.Native(context.Background(), protocol.NetworkEthereum)
	assert.NoError(t, err)

	t.Log(native)
}

func TestClient_ERC20(t *testing.T) {
	erc20, err := tokenClient.ERC20(context.Background(), protocol.NetworkEthereum, "0xc98d64da73a6616c42117b582e832812e7b8d57f")
	assert.NoError(t, err)

	t.Log(erc20)
}

// 0x712b9720b37bd206ed938a5fff4ca48cb89643ba 231
// 0x4A0d5a851BB8e8c3d580C41Ffb2f880e0A4b2AA6 164
func TestClient_ERC721(t *testing.T) {
	erc712, err := tokenClient.ERC721(context.Background(), protocol.NetworkEthereum, "0x4A0d5a851BB8e8c3d580C41Ffb2f880e0A4b2AA6", big.NewInt(164))
	assert.NoError(t, err)

	t.Log(erc712)
	t.Log(string(erc712.Metadata))
}

func TestClient_ERC1155(t *testing.T) {
	tokenID := big.NewInt(0)
	tokenID.SetString("0", 0)

	erc1155, err := tokenClient.ERC1155(context.Background(), protocol.NetworkEthereum, "0x8442864d6ab62a9193be2f16580c08e0d7bcda2f", tokenID)
	assert.NoError(t, err)

	t.Logf("%+v", erc1155)
	t.Log(string(erc1155.Metadata))
}

func TestClient_ERC721_ZORA(t *testing.T) {
	erc712, err := tokenClient.ERC721(context.Background(), protocol.NetworkEthereum, "0xabEFBc9fD2F806065b4f3C237d4b59D9A97Bcac7", big.NewInt(13737))
	assert.NoError(t, err)

	t.Log(erc712)
	t.Log(string(erc712.Metadata))
}

func TestClient_ERC721_403(t *testing.T) {
	erc712, err := tokenClient.ERC721(context.Background(), protocol.NetworkEthereum, "0xb932a70a57673d89f4acffbe830e8ed7f75fb9e0", big.NewInt(35536))
	assert.NoError(t, err)

	t.Log(erc712)
	t.Log(string(erc712.Metadata))
}

func TestClient_NFT(t *testing.T) {
	tokenID := big.NewInt(0)
	//tokenID.SetString("13477934942365134845532687062228082192258615338355725056261642040781742014465", 0)
	tokenID.SetString("41072", 0)

	nft, err := tokenClient.NFT(context.Background(), protocol.NetworkEthereum, "0x06012C8CF97BEAD5DEAE237070F9587F8E7A266D", tokenID)
	assert.NoError(t, err)

	t.Log(nft)
	t.Log(nft.Description)
}
