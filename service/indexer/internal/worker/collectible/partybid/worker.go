package partybid

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/party"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/party/partybid"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/party/partybidfac"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/party/partybuy"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/party/partybuyfac"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/party/partyco"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/party/partycofac"
	"github.com/naturalselectionlabs/pregod/common/ethclientx"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/protocol/filter"
	"github.com/naturalselectionlabs/pregod/common/utils/opentelemetry"
	"github.com/naturalselectionlabs/pregod/internal/token"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var _ worker.Worker = (*internal)(nil)

const (
	SourceName = "partybid"
)

type internal struct {
	tokenClient *token.Client
}

func (i *internal) Name() string {
	return SourceName
}

func (i *internal) Networks() []string {
	return []string{
		protocol.NetworkEthereum,
	}
}

func (i *internal) Initialize(ctx context.Context) error {
	return nil
}

func (i *internal) Handle(ctx context.Context, message *protocol.Message, transactions []model.Transaction) ([]model.Transaction, error) {
	internalTransactions := make([]model.Transaction, 0)

	ethclient, err := ethclientx.Global(message.Network)
	if err != nil {
		return nil, err
	}

	for _, transaction := range transactions {
		addressTo := common.HexToAddress(transaction.AddressTo)

		partyProxy, err := party.NewPartyProxy(addressTo, ethclient)
		if err != nil {
			zap.L().Error("failed to create party proxy", zap.Error(err), zap.String("transaction_hash", transaction.Hash), zap.String("network", transaction.Network))

			continue
		}
		partyLogicAddr, err := partyProxy.Logic(&bind.CallOpts{})
		if err != nil {
			partyLogicAddr = common.Address{}
		}

		switch addressTo {
		case party.AddressPartyBidDeployed:
			internalTransaction, err := i.handlePartyBidDeployed(ctx, message, transaction)
			if err != nil {
				zap.L().Error("failed to handle party bid deployed transaction", zap.Error(err), zap.String("transaction_hash", transaction.Hash), zap.String("network", transaction.Network))

				continue
			}

			internalTransactions = append(internalTransactions, *internalTransaction)
		case party.AddressPartyBuyDeployed:
			internalTransaction, err := i.handlePartyBuyDeployed(ctx, message, transaction)
			if err != nil {
				zap.L().Error("failed to handle party buy deployed transaction", zap.Error(err), zap.String("transaction_hash", transaction.Hash), zap.String("network", transaction.Network))

				continue
			}

			internalTransactions = append(internalTransactions, *internalTransaction)
		case party.AddressCollectionPartyDeployed:
			internalTransaction, err := i.handlePartyCollectionDeployed(ctx, message, transaction)
			if err != nil {
				zap.L().Error("failed to handle party collection deployed transaction", zap.Error(err), zap.String("transaction_hash", transaction.Hash), zap.String("network", transaction.Network))

				continue
			}

			internalTransactions = append(internalTransactions, *internalTransaction)
		default:
			switch partyLogicAddr {
			case party.AddressPartyBid:
				internalTransaction, err := i.handlePartyBidEvent(ctx, message, transaction)
				if err != nil {
					zap.L().Error("failed to handle party bid event transaction", zap.Error(err), zap.String("transaction_hash", transaction.Hash), zap.String("network", transaction.Network))

					continue
				}

				internalTransactions = append(internalTransactions, *internalTransaction)
			case party.AddressPartyBuy:
				internalTransaction, err := i.handlePartyBuyEvent(ctx, message, transaction)
				if err != nil {
					zap.L().Error("failed to handle party buy event transaction", zap.Error(err), zap.String("transaction_hash", transaction.Hash), zap.String("network", transaction.Network))

					continue
				}

				internalTransactions = append(internalTransactions, *internalTransaction)
			case party.AddressPartyCollection:
				internalTransaction, err := i.handlePartyCollectionEvent(ctx, message, transaction)
				if err != nil {
					zap.L().Error("failed to handle party collection event transaction", zap.Error(err), zap.String("transaction_hash", transaction.Hash), zap.String("network", transaction.Network))

					continue
				}

				internalTransactions = append(internalTransactions, *internalTransaction)
			default:
				continue
			}
		}
	}

	return internalTransactions, nil
}

func (i *internal) handlePartyBidDeployed(ctx context.Context, message *protocol.Message, transaction model.Transaction) (*model.Transaction, error) {
	tracer := otel.Tracer("worker_partybid")
	_, span := tracer.Start(ctx, "worker_partybid:handle_party_bid_deployed")

	defer opentelemetry.Log(span, transaction, nil, nil)

	ethclient, err := ethclientx.Global(message.Network)
	if err != nil {
		return nil, err
	}
	receipt, err := ethclient.TransactionReceipt(context.Background(), common.HexToHash(transaction.Hash))
	if err != nil {
		return nil, err
	}

	internalTransaction := transaction
	internalTransaction.Transfers = make([]model.Transfer, 0)

	partybidFacContract, err := partybidfac.NewPartyBidFactoryFilterer(common.HexToAddress(transaction.AddressTo), ethclient)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		if log.Topics[0] == party.EventHashPartyBidDeployed {
			event, err := partybidFacContract.ParsePartyBidDeployed(*log)
			if err != nil {
				return nil, err
			}

			partyMetadata, err := json.Marshal(metadata.Party{
				PartyAddress:  strings.ToLower(event.PartyBidProxy.String()),
				Name:          event.Name,
				Symbol:        event.Symbol,
				PartyType:     filter.PartyBid,
				Creator:       strings.ToLower(event.Creator.String()),
				NftContract:   strings.ToLower(event.NftContract.String()),
				TokenId:       event.TokenId,
				MarketWrapper: party.AddressMapToMarket[event.MarketWrapper.String()],
				AuctionId:     event.AuctionId,
				Action:        filter.PartyBidStart,
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(event.Creator.String()),
				AddressTo:       strings.ToLower(party.AddressPartyBidDeployed.String()),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBid, strings.ToLower(event.PartyBidProxy.String())), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		}
	}

	if len(internalTransaction.Transfers) == 0 {
		return nil, errors.New("not found nft trade")
	}

	internalTransaction.Tag, internalTransaction.Type = filter.UpdateTagAndType(filter.TagCollectible, internalTransaction.Tag, filter.CollectibleTrade, internalTransaction.Type)
	internalTransaction.Platform = party.PlatformPartyBid

	return &internalTransaction, nil
}

func (i *internal) handlePartyBuyDeployed(ctx context.Context, message *protocol.Message, transaction model.Transaction) (*model.Transaction, error) {
	tracer := otel.Tracer("worker_partybid")
	_, span := tracer.Start(ctx, "worker_partybid:handle_party_buy_deployed")

	defer opentelemetry.Log(span, transaction, nil, nil)

	ethclient, err := ethclientx.Global(message.Network)
	if err != nil {
		return nil, err
	}
	receipt, err := ethclient.TransactionReceipt(context.Background(), common.HexToHash(transaction.Hash))
	if err != nil {
		return nil, err
	}

	internalTransaction := transaction
	internalTransaction.Transfers = make([]model.Transfer, 0)

	partybuyFacContract, err := partybuyfac.NewPartyBuyFactoryFilterer(common.HexToAddress(transaction.AddressTo), ethclient)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		if log.Topics[0] == party.EventHashPartyBuyDeployed {
			event, err := partybuyFacContract.ParsePartyBuyDeployed(*log)
			if err != nil {
				return nil, err
			}

			partyMetadata, err := json.Marshal(metadata.Party{
				PartyAddress:  strings.ToLower(event.PartyProxy.String()),
				Name:          event.Name,
				Symbol:        event.Symbol,
				PartyType:     filter.PartyBuy,
				Creator:       strings.ToLower(event.Creator.String()),
				NftContract:   strings.ToLower(event.NftContract.String()),
				TokenId:       event.TokenId,
				MarketWrapper: party.AddressMapToMarket["opensea"],
				MaxPrice:      event.MaxPrice,
				ExpiredTime:   event.SecondsToTimeout,
				Action:        filter.PartyBidStart,
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(event.Creator.String()),
				AddressTo:       strings.ToLower(party.AddressPartyBuyDeployed.String()),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBuy, strings.ToLower(event.PartyProxy.String())), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		}
	}

	if len(internalTransaction.Transfers) == 0 {
		return nil, errors.New("not found nft trade")
	}

	internalTransaction.Tag, internalTransaction.Type = filter.UpdateTagAndType(filter.TagCollectible, internalTransaction.Tag, filter.CollectibleTrade, internalTransaction.Type)
	internalTransaction.Platform = party.PlatformPartyBid

	return &internalTransaction, nil
}

func (i *internal) handlePartyCollectionDeployed(ctx context.Context, message *protocol.Message, transaction model.Transaction) (*model.Transaction, error) {
	tracer := otel.Tracer("worker_partybid")
	_, span := tracer.Start(ctx, "worker_partybid:handle_party_collection_deployed")

	defer opentelemetry.Log(span, transaction, nil, nil)

	ethclient, err := ethclientx.Global(message.Network)
	if err != nil {
		return nil, err
	}
	receipt, err := ethclient.TransactionReceipt(context.Background(), common.HexToHash(transaction.Hash))
	if err != nil {
		return nil, err
	}

	internalTransaction := transaction
	internalTransaction.Transfers = make([]model.Transfer, 0)

	partycolFacContract, err := partycofac.NewCollectionPartyFactoryFilterer(common.HexToAddress(transaction.AddressTo), ethclient)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		if log.Topics[0] == party.EventHashCollectionPartyDeployed {
			event, err := partycolFacContract.ParseCollectionPartyDeployed(*log)
			if err != nil {
				return nil, err
			}

			partyMetadata, err := json.Marshal(metadata.Party{
				PartyAddress:  strings.ToLower(event.PartyProxy.String()),
				Name:          event.Name,
				Symbol:        event.Symbol,
				PartyType:     filter.PartyCollection,
				Creator:       strings.ToLower(event.Creator.String()),
				NftContract:   strings.ToLower(event.NftContract.String()),
				MarketWrapper: party.AddressMapToMarket["opensea"],
				MaxPrice:      event.MaxPrice,
				ExpiredTime:   event.SecondsToTimeout,
				Deciders:      event.Deciders,
				Action:        filter.PartyBidStart,
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(event.Creator.String()),
				AddressTo:       strings.ToLower(party.AddressCollectionPartyDeployed.String()),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyCollection, strings.ToLower(event.PartyProxy.String())), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		}
	}

	if len(internalTransaction.Transfers) == 0 {
		return nil, errors.New("not found nft trade")
	}

	internalTransaction.Tag, internalTransaction.Type = filter.UpdateTagAndType(filter.TagCollectible, internalTransaction.Tag, filter.CollectibleTrade, internalTransaction.Type)
	internalTransaction.Platform = party.PlatformPartyBid

	return &internalTransaction, nil
}

func (i *internal) handlePartyBidEvent(ctx context.Context, message *protocol.Message, transaction model.Transaction) (*model.Transaction, error) {
	tracer := otel.Tracer("worker_partybid")
	_, span := tracer.Start(ctx, "worker_partybid:handle_party_bid_event")

	defer opentelemetry.Log(span, transaction, nil, nil)

	ethclient, err := ethclientx.Global(message.Network)
	if err != nil {
		return nil, err
	}
	receipt, err := ethclient.TransactionReceipt(context.Background(), common.HexToHash(transaction.Hash))
	if err != nil {
		return nil, err
	}

	internalTransaction := transaction
	internalTransaction.Transfers = make([]model.Transfer, 0)

	partybidContract, err := partybid.NewPartyBid(common.HexToAddress(transaction.AddressTo), ethclient)
	if err != nil {
		return nil, err
	}

	partyInfo := metadata.Party{}

	partyInfo.PartyAddress = transaction.AddressTo

	if partyInfo.Symbol, err = partybidContract.Symbol(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	if partyInfo.Name, err = partybidContract.Name(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	nftContract, err := partybidContract.NftContract(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	partyInfo.NftContract = strings.ToLower(nftContract.String())
	if partyInfo.PartyStatus, err = partybidContract.PartyStatus(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	var market common.Address
	if market, err = partybidContract.MarketWrapper(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	partyInfo.MarketWrapper = party.AddressMapToMarket[market.String()]
	partyInfo.PartyType = filter.PartyBid

	native, err := i.tokenClient.Native(ctx, message.Network)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		switch log.Topics[0] {
		case party.EventHashContributed:
			event, err := partybidContract.ParseContributed(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyContribute{
				Action:                          filter.PartyBidContribute,
				PartyInfo:                       partyInfo,
				Contributor:                     strings.ToLower(event.Contributor.String()),
				Amount:                          decimal.NewFromBigInt(event.Amount, 0).Shift(-int32(native.Decimals)),
				PreviousTotalContributedToParty: decimal.NewFromBigInt(event.PreviousTotalContributedToParty, 0).Shift(-int32(native.Decimals)),
				TotalFromContributor:            decimal.NewFromBigInt(event.TotalFromContributor, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(event.Contributor.String()),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBid, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashBid:
			event, err := partybidContract.ParseBid(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyBid{
				Action:    filter.PartyBidBid,
				PartyInfo: partyInfo,
				BidAmount: decimal.NewFromBigInt(event.Amount, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBid, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashFinalized:
			event, err := partybidContract.ParseFinalized(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyFinalize{
				Action:           filter.PartyBidFinalize,
				PartyInfo:        partyInfo,
				Result:           event.Result,
				TotalSpent:       decimal.NewFromBigInt(event.TotalSpent, 0).Shift(-int32(native.Decimals)),
				Fee:              decimal.NewFromBigInt(event.Fee, 0).Shift(-int32(native.Decimals)),
				TotalContributed: decimal.NewFromBigInt(event.TotalContributed, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBid, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashClaimed:
			event, err := partybidContract.ParseClaimed(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyClaim{
				Action:             filter.PartyBidClaim,
				PartyInfo:          partyInfo,
				Contributor:        strings.ToLower(event.Contributor.String()),
				TotalContributed:   decimal.NewFromBigInt(event.TotalContributed, 0).Shift(-int32(native.Decimals)),
				ExcessContribution: decimal.NewFromBigInt(event.ExcessContribution, 0).Shift(-int32(native.Decimals)),
				TokenAmount:        decimal.NewFromBigInt(event.TokenAmount, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBid, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		default:
			continue
		}
	}

	if len(internalTransaction.Transfers) == 0 {
		return nil, errors.New("not found nft trade")
	}

	internalTransaction.Tag, internalTransaction.Type = filter.UpdateTagAndType(filter.TagCollectible, internalTransaction.Tag, filter.CollectibleTrade, internalTransaction.Type)
	internalTransaction.Platform = party.PlatformPartyBid

	return &internalTransaction, nil
}

func (i *internal) handlePartyBuyEvent(ctx context.Context, message *protocol.Message, transaction model.Transaction) (*model.Transaction, error) {
	tracer := otel.Tracer("worker_partybid")
	_, span := tracer.Start(ctx, "worker_partybid:handle_party_buy_event")

	defer opentelemetry.Log(span, transaction, nil, nil)

	ethclient, err := ethclientx.Global(message.Network)
	if err != nil {
		return nil, err
	}
	receipt, err := ethclient.TransactionReceipt(context.Background(), common.HexToHash(transaction.Hash))
	if err != nil {
		return nil, err
	}

	internalTransaction := transaction
	internalTransaction.Transfers = make([]model.Transfer, 0)

	partybuyContract, err := partybuy.NewPartyBuy(common.HexToAddress(transaction.AddressTo), ethclient)
	if err != nil {
		return nil, err
	}

	partyInfo := metadata.Party{}

	partyInfo.PartyAddress = strings.ToLower(transaction.AddressTo)
	partyInfo.PartyType = filter.PartyBuy
	if partyInfo.Symbol, err = partybuyContract.Symbol(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	if partyInfo.Name, err = partybuyContract.Name(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	nftContract, err := partybuyContract.NftContract(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	partyInfo.NftContract = strings.ToLower(nftContract.String())
	if partyInfo.PartyStatus, err = partybuyContract.PartyStatus(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	partyInfo.MarketWrapper = party.AddressMapToMarket["opensea"]

	native, err := i.tokenClient.Native(ctx, message.Network)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		switch log.Topics[0] {
		case party.EventHashContributed:
			event, err := partybuyContract.ParseContributed(*log)
			if err != nil {
				return nil, err
			}
			partyInfo.Action = filter.PartyBidContribute
			partyMetadata, err := json.Marshal(metadata.PartyContribute{
				Action:                          filter.PartyBidContribute,
				PartyInfo:                       partyInfo,
				Contributor:                     strings.ToLower(event.Contributor.String()),
				Amount:                          decimal.NewFromBigInt(event.Amount, 0).Shift(-int32(native.Decimals)),
				PreviousTotalContributedToParty: decimal.NewFromBigInt(event.PreviousTotalContributedToParty, 0).Shift(-int32(native.Decimals)),
				TotalFromContributor:            decimal.NewFromBigInt(event.TotalFromContributor, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(event.Contributor.String()),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBuy, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashPbBought:
			event, err := partybuyContract.ParseBought(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyBuy{
				Action:           filter.PartyBidBuy,
				PartyInfo:        partyInfo,
				TriggeredBy:      strings.ToLower(event.TriggeredBy.String()),
				TargetAddress:    strings.ToLower(event.TargetAddress.String()),
				EthSpent:         decimal.NewFromBigInt(event.EthSpent, 0).Shift(-int32(native.Decimals)),
				EthFeePaid:       decimal.NewFromBigInt(event.EthFeePaid, 0).Shift(-int32(native.Decimals)),
				TotalContributed: decimal.NewFromBigInt(event.TotalContributed, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBuy, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashPbExpired:
			event, err := partybuyContract.ParseExpired(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyExpire{
				Action:      filter.PartyBidExpire,
				PartyInfo:   partyInfo,
				TriggeredBy: strings.ToLower(event.TriggeredBy.String()),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBuy, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashClaimed:
			event, err := partybuyContract.ParseClaimed(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyClaim{
				Action:             filter.PartyBidClaim,
				PartyInfo:          partyInfo,
				TotalContributed:   decimal.NewFromBigInt(event.TotalContributed, 0).Shift(-int32(native.Decimals)),
				ExcessContribution: decimal.NewFromBigInt(event.ExcessContribution, 0).Shift(-int32(native.Decimals)),
				TokenAmount:        decimal.NewFromBigInt(event.TokenAmount, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyBuy, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		default:
			continue
		}
	}
	if len(internalTransaction.Transfers) == 0 {
		return nil, errors.New("not found nft trade")
	}

	internalTransaction.Tag, internalTransaction.Type = filter.UpdateTagAndType(filter.TagCollectible, internalTransaction.Tag, filter.CollectibleTrade, internalTransaction.Type)
	internalTransaction.Platform = party.PlatformPartyBid

	return &internalTransaction, nil
}

func (i *internal) handlePartyCollectionEvent(ctx context.Context, message *protocol.Message, transaction model.Transaction) (*model.Transaction, error) {
	tracer := otel.Tracer("worker_partybid")
	_, span := tracer.Start(ctx, "worker_partybid:handle_party_collection_event")

	defer opentelemetry.Log(span, transaction, nil, nil)

	ethclient, err := ethclientx.Global(message.Network)
	if err != nil {
		return nil, err
	}
	receipt, err := ethclient.TransactionReceipt(context.Background(), common.HexToHash(transaction.Hash))
	if err != nil {
		return nil, err
	}

	internalTransaction := transaction
	internalTransaction.Transfers = make([]model.Transfer, 0)

	partycoContract, err := partyco.NewCollectionParty(common.HexToAddress(transaction.AddressTo), ethclient)
	if err != nil {
		return nil, err
	}

	partyInfo := metadata.Party{}

	partyInfo.PartyAddress = transaction.AddressTo

	if partyInfo.Symbol, err = partycoContract.Symbol(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	if partyInfo.Name, err = partycoContract.Name(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	nftContract, err := partycoContract.NftContract(&bind.CallOpts{})
	if err != nil {
		return nil, err
	}
	partyInfo.NftContract = strings.ToLower(nftContract.String())

	if partyInfo.PartyStatus, err = partycoContract.PartyStatus(&bind.CallOpts{}); err != nil {
		return nil, err
	}
	partyInfo.MarketWrapper = party.AddressMapToMarket["opensea"]
	partyInfo.PartyType = filter.PartyCollection

	native, err := i.tokenClient.Native(ctx, message.Network)
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		switch log.Topics[0] {
		case party.EventHashContributed:
			event, err := partycoContract.ParseContributed(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyContribute{
				Action:                          filter.PartyBidContribute,
				PartyInfo:                       partyInfo,
				Contributor:                     strings.ToLower(event.Contributor.String()),
				Amount:                          decimal.NewFromBigInt(event.Amount, 0).Shift(-int32(native.Decimals)),
				PreviousTotalContributedToParty: decimal.NewFromBigInt(event.PreviousTotalContributedToParty, 0).Shift(-int32(native.Decimals)),
				TotalFromContributor:            decimal.NewFromBigInt(event.TotalFromContributor, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(event.Contributor.String()),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyCollection, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashPcBought:
			event, err := partycoContract.ParseBought(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyBuy{
				Action:           filter.PartyBidBuy,
				PartyInfo:        partyInfo,
				TokenId:          event.TokenId,
				TriggeredBy:      strings.ToLower(event.TriggeredBy.String()),
				TargetAddress:    strings.ToLower(event.TargetAddress.String()),
				EthSpent:         decimal.NewFromBigInt(event.EthSpent, 0).Shift(-int32(native.Decimals)),
				EthFeePaid:       decimal.NewFromBigInt(event.EthFeePaid, 0).Shift(-int32(native.Decimals)),
				TotalContributed: decimal.NewFromBigInt(event.TotalContributed, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyCollection, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashPcExpired:
			event, err := partycoContract.ParseExpired(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyExpire{
				Action:      filter.PartyBidExpire,
				PartyInfo:   partyInfo,
				TriggeredBy: strings.ToLower(event.TriggeredBy.String()),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyCollection, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		case party.EventHashClaimed:
			event, err := partycoContract.ParseClaimed(*log)
			if err != nil {
				return nil, err
			}
			partyMetadata, err := json.Marshal(metadata.PartyClaim{
				Action:             filter.PartyBidClaim,
				PartyInfo:          partyInfo,
				Contributor:        strings.ToLower(event.Contributor.String()),
				TotalContributed:   decimal.NewFromBigInt(event.TotalContributed, 0).Shift(-int32(native.Decimals)),
				ExcessContribution: decimal.NewFromBigInt(event.ExcessContribution, 0).Shift(-int32(native.Decimals)),
				TokenAmount:        decimal.NewFromBigInt(event.TokenAmount, 0).Shift(-int32(native.Decimals)),
			})
			if err != nil {
				return nil, err
			}

			internalTransaction.Transfers = append(internalTransaction.Transfers, model.Transfer{
				TransactionHash: internalTransaction.Hash,
				Timestamp:       internalTransaction.Timestamp,
				BlockNumber:     big.NewInt(internalTransaction.BlockNumber),
				Tag:             filter.TagCollectible,
				Type:            filter.CollectibleTrade,
				Index:           int64(log.Index),
				AddressFrom:     strings.ToLower(transaction.AddressFrom),
				AddressTo:       strings.ToLower(transaction.AddressTo),
				Metadata:        partyMetadata,
				Network:         internalTransaction.Network,
				Platform:        party.PlatformPartyBid,
				Source:          ethereum.Source,
				RelatedUrls: party.BuildURL(
					[]string{},
					append([]string{}, party.BuildPartyURL(filter.PartyCollection, strings.ToLower(transaction.AddressTo)), ethereum.BuildScanURL(internalTransaction.Network, internalTransaction.Hash))...,
				),
			})
		default:
			continue
		}
	}
	if len(internalTransaction.Transfers) == 0 {
		return nil, errors.New("not found nft trade")
	}

	internalTransaction.Tag, internalTransaction.Type = filter.UpdateTagAndType(filter.TagCollectible, internalTransaction.Tag, filter.CollectibleTrade, internalTransaction.Type)
	internalTransaction.Platform = party.PlatformPartyBid

	return &internalTransaction, nil
}

func (i *internal) Jobs() []worker.Job {
	return nil
}

func New() worker.Worker {
	return &internal{}
}
