package ethereum

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/erc1155"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/erc20"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/erc721"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/gitcoin"
	mrc202 "github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/mrc20"
	"github.com/naturalselectionlabs/pregod/common/ethclientx"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/utils/loggerx"
	"github.com/naturalselectionlabs/pregod/common/utils/opentelemetry"
	"github.com/naturalselectionlabs/pregod/internal/allowlist"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

const (
	Source = "ethereum"

	MaxConcurrency = 200
)

var (
	ErrorUnsupportedEvent            = errors.New("unsupported event")
	ErrorUnrelatedEvent              = errors.New("unrelated event")
	ErrorInvalidTransactionVRSValues = errors.New("invalid transaction v, r, s values")
)

func BuildTransactions(ctx context.Context, message *protocol.Message, transactions []*model.Transaction) ([]*model.Transaction, error) {
	tracer := otel.Tracer("ethereum")
	ctx, span := tracer.Start(ctx, "ethereum:BuildTransactions")

	var err error

	defer opentelemetry.Log(span, message, transactions, err)

	chunkSize := 5 * MaxConcurrency
	tempTransactions := make([]*model.Transaction, 0)

	for _, ts := range lo.Chunk(transactions, chunkSize) {

		blocks, err := lop.MapWithError(ts, makeBlockHandlerFunc(ctx, message), lop.NewOption().WithConcurrency(MaxConcurrency))
		if err != nil {
			loggerx.Global().Error("failed to handle blocks", zap.Error(err), zap.String("network", message.Network), zap.String("address", message.Address))

			return nil, err
		}

		blockMap := make(map[int64]*types.Block)
		for _, block := range blocks {
			blockMap[block.Number().Int64()] = block
		}

		// Error topic/field count mismatch
		ts, err = lop.MapWithError(ts, makeTransactionHandlerFunc(ctx, message, blockMap), lop.NewOption().WithConcurrency(MaxConcurrency))
		if err != nil {
			loggerx.Global().Error("failed to handle transactions", zap.Error(err), zap.String("network", message.Network), zap.String("address", message.Address))

			return nil, err
		}

		tempTransactions = append(tempTransactions, ts...)

	}

	internalTransactions := make([]*model.Transaction, 0)

	for _, transaction := range tempTransactions {
		if transaction != nil {
			internalTransactions = append(internalTransactions, transaction)
		}
	}

	return internalTransactions, nil
}

func makeBlockHandlerFunc(ctx context.Context, message *protocol.Message) func(transaction *model.Transaction, i int) (*types.Block, error) {
	return func(transaction *model.Transaction, i int) (*types.Block, error) {
		ethclient, err := ethclientx.Global(message.Network)
		if err != nil {
			return nil, err
		}

		block, err := ethclient.BlockByNumber(ctx, big.NewInt(transaction.BlockNumber))
		if err != nil {
			zap.L().Error("failed to get block", zap.Error(err), zap.String("network", message.Network), zap.String("address", message.Address), zap.Int64("block_number", transaction.BlockNumber))

			return nil, err
		}

		return block, nil
	}
}

func makeTransactionHandlerFunc(ctx context.Context, message *protocol.Message, blockMap map[int64]*types.Block) func(transaction *model.Transaction, i int) (*model.Transaction, error) {
	return func(transaction *model.Transaction, i int) (*model.Transaction, error) {
		block := blockMap[transaction.BlockNumber]

		transaction.Timestamp = time.Unix(int64(block.Time()), 0)

		var internalTransaction *types.Transaction

		for index, blockTransaction := range block.Transactions() {
			if blockTransaction.Hash().String() == transaction.Hash {
				transaction.Index = int64(index)
				internalTransaction = blockTransaction

				break
			}
		}

		if internalTransaction == nil {
			return nil, nil
		}

		// Supports EIP-2930 and EIP-2718 and EIP-1559 and EIP-155 and legacy transactions
		transactionMessage, err := internalTransaction.AsMessage(types.LatestSignerForChainID(internalTransaction.ChainId()), block.BaseFee())
		if err != nil {
			return nil, err
		}

		if err != nil {
			// Filter transactions in incompatible blocks
			if err.Error() == ErrorInvalidTransactionVRSValues.Error() {
				return nil, nil
			}

			zap.L().Error("failed to get transaction message", zap.Error(err), zap.String("network", message.Network), zap.String("address", message.Address), zap.String("hash", transaction.Hash))

			return nil, err
		}

		transaction.AddressFrom = strings.ToLower(transactionMessage.From().String())

		if transaction.AddressFrom != "" && !strings.EqualFold(transaction.AddressFrom, message.Address) && !allowlist.AllowList.Contains(transaction.AddressFrom) {
			return nil, nil
		}

		addressTo := AddressGenesis.String()

		if internalTransaction.To() != nil {
			addressTo = internalTransaction.To().String()
		}

		transaction.AddressTo = strings.ToLower(addressTo)

		ethclient, err := ethclientx.Global(message.Network)
		if err != nil {
			return nil, err
		}

		receipt, err := ethclient.TransactionReceipt(ctx, internalTransaction.Hash())
		if err != nil {
			loggerx.Global().Error("failed to get transaction receipt", zap.Error(err), zap.String("network", message.Network), zap.String("address", message.Address), zap.String("hash", transaction.Hash))

			return nil, err
		}

		switch internalTransaction.Type() {
		case types.LegacyTxType, types.AccessListTxType:
			fee := decimal.NewFromBigInt(internalTransaction.GasPrice(), 0).Mul(decimal.NewFromInt(int64(receipt.GasUsed))).Shift(-18)
			transaction.Fee = &fee
		case types.DynamicFeeTxType:
			fee := (decimal.NewFromBigInt(block.BaseFee(), 0).Add(decimal.NewFromBigInt(internalTransaction.GasTipCap(), 0))).Mul(decimal.NewFromInt(int64(receipt.GasUsed))).Shift(-18)
			transaction.Fee = &fee
		}

		transactionSuccess := receipt.Status == types.ReceiptStatusSuccessful
		transaction.Success = &transactionSuccess

		transaction.Source = Source

		if transaction.SourceData, err = json.Marshal(&SourceData{
			Transaction: internalTransaction,
			Receipt:     receipt,
		}); err != nil {
			return nil, err
		}

		if transaction.Transfers, err = handleReceipt(ctx, message, transaction, receipt); err != nil {
			loggerx.Global().Error("failed to handle receipt", zap.Error(err), zap.String("network", message.Network), zap.String("address", message.Address))

			return nil, err
		}

		transaction.Transfers = append(transaction.Transfers, model.Transfer{
			// This is a virtual transfer
			TransactionHash: transaction.Hash,
			Timestamp:       transaction.Timestamp,
			Index:           protocol.IndexVirtual,
			AddressFrom:     transaction.AddressFrom,
			AddressTo:       transaction.AddressTo,
			Metadata:        metadata.Default,
			Network:         message.Network,
			Source:          Source,
			SourceData:      transaction.SourceData,
			RelatedUrls: []string{
				BuildScanURL(message.Network, transaction.Hash),
			},
		})

		return transaction, nil
	}
}

func handleReceipt(ctx context.Context, message *protocol.Message, transaction *model.Transaction, receipt *types.Receipt) ([]model.Transfer, error) {
	tracer := otel.Tracer("ethereum")
	ctx, trace := tracer.Start(ctx, "ethereum:handleReceipt")
	transfers := make([]model.Transfer, 0)
	var err error
	defer func() { opentelemetry.Log(trace, message, transfers, err) }()

	for _, log := range receipt.Logs {
		transfer, err := handleLog(ctx, message, transaction, *log)
		if err != nil {
			if errors.Is(err, ErrorUnsupportedEvent) || errors.Is(err, ErrorUnrelatedEvent) {
				continue
			}

			return nil, err
		}

		transfers = append(transfers, *transfer)
	}

	return transfers, nil
}

func handleLog(ctx context.Context, message *protocol.Message, transaction *model.Transaction, log types.Log) (*model.Transfer, error) {
	tracer := otel.Tracer("ethereum")
	_, trace := tracer.Start(ctx, "ethereum:handleLog")

	transfer := model.Transfer{
		TransactionHash: transaction.Hash,
		Timestamp:       transaction.Timestamp,
		Index:           int64(log.Index),
		Network:         transaction.Network,
		Metadata:        metadata.Default,
		Source:          Source,
		RelatedUrls: []string{
			BuildScanURL(message.Network, transaction.Hash),
		},
	}

	var err error
	defer opentelemetry.Log(trace, message, transfer, err)

	if len(log.Topics) == 0 {
		return nil, ErrorUnsupportedEvent
	}

	switch log.Topics[0] {
	case erc20.EventHashTransfer, erc721.EventHashTransfer:
		switch len(log.Topics) {
		case 3:
			filterer, err := erc20.NewERC20Filterer(log.Address, nil)
			if err != nil {
				return nil, err
			}

			event, err := filterer.ParseTransfer(log)
			if err != nil {
				return nil, err
			}

			transfer.AddressFrom = strings.ToLower(event.From.String())
			transfer.AddressTo = strings.ToLower(event.To.String())
		case 4:
			filterer, err := erc721.NewERC721Filterer(log.Address, nil)
			if err != nil {
				return nil, err
			}

			event, err := filterer.ParseTransfer(log)
			if err != nil {
				return nil, err
			}

			transfer.AddressFrom = strings.ToLower(event.From.String())
			transfer.AddressTo = strings.ToLower(event.To.String())
		default:
			return nil, ErrorUnsupportedEvent
		}
	case erc1155.EventHashTransferSingle:
		filterer, err := erc1155.NewERC1155Filterer(log.Address, nil)
		if err != nil {
			return nil, err
		}

		event, err := filterer.ParseTransferSingle(log)
		if err != nil {
			return nil, err
		}

		transfer.AddressFrom = strings.ToLower(event.From.String())
		transfer.AddressTo = strings.ToLower(event.To.String())
	case erc1155.EventHashTransferBatch:
		filterer, err := erc1155.NewERC1155Filterer(log.Address, nil)
		if err != nil {
			return nil, err
		}

		event, err := filterer.ParseTransferBatch(log)
		if err != nil {
			return nil, err
		}

		transfer.AddressFrom = strings.ToLower(event.From.String())
		transfer.AddressTo = strings.ToLower(event.To.String())
	case mrc202.EventHashLogTransfer:
		filterer, err := mrc202.NewMRC20Filterer(log.Address, nil)
		if err != nil {
			return nil, err
		}

		event, err := filterer.ParseLogTransfer(log)
		if err != nil {
			return nil, err
		}

		transfer.AddressFrom = strings.ToLower(event.From.String())
		transfer.AddressTo = strings.ToLower(event.To.String())
	case gitcoin.EventHashDonationSent:
		flterer, err := gitcoin.NewGitcoinFilterer(log.Address, nil)
		if err != nil {
			return nil, err
		}

		event, err := flterer.ParseDonationSent(log)
		if err != nil {
			return nil, err
		}

		transfer.AddressFrom = strings.ToLower(event.Donor.String())
		transfer.AddressTo = strings.ToLower(event.Dest.String())
	default:
		return nil, ErrorUnsupportedEvent
	}

	address := strings.ToLower(message.Address)

	// Ignore transfers that are not related to message.address
	if len(address) > 0 && transaction.AddressFrom != address && transfer.AddressTo != address && transfer.AddressFrom != address {
		return nil, ErrorUnrelatedEvent
	}

	if transfer.SourceData, err = json.Marshal(log); err != nil {
		return nil, err
	}

	return &transfer, nil
}
