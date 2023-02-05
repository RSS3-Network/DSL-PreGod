package token

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/naturalselectionlabs/pregod/common/cache"
	"github.com/naturalselectionlabs/pregod/common/database"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/erc20"
	"github.com/naturalselectionlabs/pregod/common/ethclientx"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/utils/loggerx"
	"github.com/samber/lo"

	"go.uber.org/zap"

	"gorm.io/gorm"
)

func (c *Client) ERC20(ctx context.Context, network string, contractAddress string) (*ERC20, error) {
	var token model.Token

	tokenID := fmt.Sprintf("token_%s-%s", network, contractAddress)

	if database.Global() != nil {
		// Get token from cache
		exists, err := cache.GetMsgPack(ctx, tokenID, &token)
		if err != nil {
			return nil, fmt.Errorf("get token from cache: %w", err)
		}

		if exists {
			return lo.ToPtr(tokenToERC20(token)), nil
		}

		// Get token from database
		err = database.Global().
			Model((*model.Token)(nil)).
			Where(map[string]interface{}{
				"network":          network,
				"contract_address": strings.ToLower(contractAddress),
			}).
			First(&token).
			Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return lo.ToPtr(tokenToERC20(token)), nil
		}
	}

	result := &ERC20{
		ContractAddress: contractAddress,
	}

	ethclient, err := ethclientx.Global(network)
	if err != nil {
		return nil, err
	}

	erc20Contract, err := erc20.NewERC20(common.HexToAddress(result.ContractAddress), ethclient)
	if err != nil {
		return nil, err
	}

	if result.Name, err = erc20Contract.Name(&bind.CallOpts{}); err != nil {
		return nil, err
	}

	if result.Symbol, err = erc20Contract.Symbol(&bind.CallOpts{}); err != nil {
		return nil, err
	}

	if result.Decimals, err = erc20Contract.Decimals(&bind.CallOpts{}); err != nil {
		return nil, err
	}

	// Set token to cache
	if err := cache.SetMsgPack(ctx, tokenID, result, 24*time.Hour); err != nil {
		return nil, fmt.Errorf("set token to cache: %w", err)
	}

	return result, nil
}

func (c *Client) ERC20ToMetadata(context context.Context, network, contractAddress string) (*metadata.Token, error) {
	erc20, err := c.ERC20(context, network, contractAddress)
	if err != nil {
		loggerx.Global().Error("get erc20 token error", zap.Error(err))

		return nil, err
	}

	tokenMetadata := &metadata.Token{
		Name:            erc20.Name,
		Symbol:          erc20.Symbol,
		Decimals:        erc20.Decimals,
		Standard:        protocol.TokenStandardERC20,
		ContractAddress: erc20.ContractAddress,
		Image:           erc20.Logo,
	}

	return tokenMetadata, nil
}

func tokenToERC20(token model.Token) ERC20 {
	return ERC20{
		Name:            token.Name,
		Symbol:          token.Symbol,
		Decimals:        token.Decimal,
		ContractAddress: token.ContractAddress,
		Logo:            token.Logo,
	}
}
