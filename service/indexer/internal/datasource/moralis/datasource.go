package moralis

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/moralis"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/datasource"
	"github.com/shopspring/decimal"
)

const (
	Source = "moralis"
)

var _ datasource.Datasource = &Datasource{}

type Datasource struct {
	moralisClient *moralis.Client
}

func (d *Datasource) Name() string {
	return Source
}

func (d *Datasource) Networks() []string {
	return []string{
		protocol.NetworkEthereum, protocol.NetworkPolygon, protocol.NetworkBinanceSmartChain,
	}
}

func (d *Datasource) Handle(ctx context.Context, message *protocol.Message) ([]model.Transfer, error) {
	transferModels := make([]model.Transfer, 0)

	nonNativeTransferMap := make(map[string]struct{})

	switch message.Network {
	case protocol.NetworkEthereum, protocol.NetworkPolygon, protocol.NetworkBinanceSmartChain:
		tokenTransferMap := make(map[string][]moralis.TokenTransfer)

		tokenTransfers, err := d.getTokenTransfers(ctx, message)
		if err != nil {
			return nil, err
		}

		for _, tokenTransfer := range tokenTransfers {
			tokenTransferMap[tokenTransfer.TransactionHash] = append(tokenTransferMap[tokenTransfer.TransactionHash], tokenTransfer)
			nonNativeTransferMap[tokenTransfer.TransactionHash] = struct{}{}
		}

		for _, tokenTransfers := range tokenTransferMap {
			for i, tokenTransfer := range tokenTransfers {
				timestamp, err := time.Parse(time.RFC3339, tokenTransfer.BlockTimestamp)
				if err != nil {
					return nil, err
				}

				sourceData, err := json.Marshal(tokenTransfer)
				if err != nil {
					return nil, err
				}

				transferModels = append(transferModels, model.Transfer{
					TransactionHash:     tokenTransfer.TransactionHash,
					Timestamp:           timestamp,
					TransactionLogIndex: decimal.NewFromInt(int64(i)),
					AddressFrom:         tokenTransfer.FromAddress,
					AddressTo:           tokenTransfer.ToAddress,
					Network:             message.Network,
					Metadata:            metadata.Default,
					Source:              d.Name(),
					SourceData:          sourceData,
				})
			}
		}

		nftTransfers, err := d.getNFTTransfers(ctx, message)
		if err != nil {
			return nil, err
		}

		for _, nftTransfer := range nftTransfers {
			timestamp, err := time.Parse(time.RFC3339, nftTransfer.BlockTimestamp)
			if err != nil {
				return nil, err
			}

			sourceData, err := json.Marshal(nftTransfer)
			if err != nil {
				return nil, err
			}

			transferModels = append(transferModels, model.Transfer{
				TransactionHash:     nftTransfer.TransactionHash,
				Timestamp:           timestamp,
				TransactionLogIndex: nftTransfer.LogIndex,
				AddressFrom:         nftTransfer.FromAddress,
				AddressTo:           nftTransfer.ToAddress,
				Network:             message.Network,
				Metadata:            metadata.Default,
				Source:              d.Name(),
				SourceData:          sourceData,
			})

			nonNativeTransferMap[nftTransfer.TransactionHash] = struct{}{}
		}

		transactions, err := d.getTransactions(ctx, message)
		if err != nil {
			return nil, err
		}

		for _, transaction := range transactions {
			if _, exist := nonNativeTransferMap[transaction.Hash]; exist {
				continue
			}

			transactionLogIndex, err := strconv.Atoi(transaction.TransactionIndex)
			if err != nil {
				return nil, err
			}

			timestamp, err := time.Parse(time.RFC3339, transaction.BlockTimestamp)
			if err != nil {
				return nil, err
			}

			sourceData, err := json.Marshal(transaction)
			if err != nil {
				return nil, err
			}

			internalTransfer := model.Transfer{
				TransactionHash:     transaction.Hash,
				Timestamp:           timestamp,
				TransactionLogIndex: decimal.NewFromInt(int64(transactionLogIndex)),
				AddressFrom:         transaction.FromAddress,
				AddressTo:           transaction.ToAddress,
				Network:             message.Network,
				Source:              d.Name(),
				SourceData:          sourceData,
			}

			transferModels = append(transferModels, internalTransfer)
		}
	}

	return transferModels, nil
}

func (d *Datasource) getTransactions(ctx context.Context, message *protocol.Message) ([]moralis.Transaction, error) {
	address := common.HexToAddress(message.Address)

	transactions, response, err := d.moralisClient.GetTransactions(ctx, address, &moralis.GetTransactionsOption{
		Chain: protocol.NetworkToID(message.Network),
	})
	if err != nil {
		return nil, err
	}

	for int64(len(transactions)) < response.Total && len(transactions) <= moralis.MaxOffset {
		var internalTransactions []moralis.Transaction

		internalTransactions, response, err = d.moralisClient.GetTransactions(ctx, address, &moralis.GetTransactionsOption{
			Chain:  protocol.NetworkToID(message.Network),
			Cursor: response.Cursor,
		})

		if err != nil {
			return nil, err
		}

		transactions = append(transactions, internalTransactions...)
	}

	return transactions, nil
}

func (d *Datasource) getTokenTransfers(ctx context.Context, message *protocol.Message) ([]moralis.TokenTransfer, error) {
	address := common.HexToAddress(message.Address)

	tokenTransfers, response, err := d.moralisClient.GetTokenTransfers(ctx, address, &moralis.GetTokenTransfersOption{
		Chain: protocol.NetworkToID(message.Network),
	})
	if err != nil {
		return nil, err
	}

	for int64(len(tokenTransfers)) < response.Total && len(tokenTransfers) <= moralis.MaxOffset {
		var internalTokenTransfers []moralis.TokenTransfer

		internalTokenTransfers, response, err = d.moralisClient.GetTokenTransfers(ctx, address, &moralis.GetTokenTransfersOption{
			Chain:  protocol.NetworkToID(message.Network),
			Cursor: response.Cursor,
		})

		if err != nil {
			return nil, err
		}

		tokenTransfers = append(tokenTransfers, internalTokenTransfers...)
	}

	return tokenTransfers, nil
}

func (d *Datasource) getNFTTransfers(ctx context.Context, message *protocol.Message) ([]moralis.NFTTransfer, error) {
	address := common.HexToAddress(message.Address)

	nftTransfers, response, err := d.moralisClient.GetNFTTransfers(ctx, address, &moralis.GetNFTTransfersOption{
		Chain: protocol.NetworkToID(message.Network),
	})
	if err != nil {
		return nil, err
	}

	for int64(len(nftTransfers)) < response.Total && len(nftTransfers) <= moralis.MaxOffset {
		var internalNFTTransfers []moralis.NFTTransfer

		internalNFTTransfers, response, err = d.moralisClient.GetNFTTransfers(ctx, address, &moralis.GetNFTTransfersOption{
			Chain:  protocol.NetworkToID(message.Network),
			Cursor: response.Cursor,
		})

		if err != nil {
			return nil, err
		}

		nftTransfers = append(nftTransfers, internalNFTTransfers...)
	}

	return nftTransfers, nil
}

func New(key string) datasource.Datasource {
	moralisClient := moralis.NewClient(key)

	return &Datasource{
		moralisClient: moralisClient,
	}
}
