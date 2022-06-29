package crossbell

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/logger"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker/crossbell/contract"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker/crossbell/handler"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

//go:generate abigen --abi ./contract/erc721.abi --pkg contract --type ERC721 --out ./contract/erc721.go

const (
	Name = "crossbell"

	Endpoint = "https://rpc.crossbell.io"
)

var _ worker.Worker = (*service)(nil)

type service struct {
	ethereumClient *ethclient.Client
	handler        handler.Interface
}

func (s *service) Name() string {
	return Name
}

func (s *service) Networks() []string {
	return []string{
		protocol.NetworkCrossbell,
	}
}

func (s *service) Initialize(ctx context.Context) (err error) {
	if s.ethereumClient, err = ethclient.Dial(Endpoint); err != nil {
		return err
	}

	s.handler = handler.New(s.ethereumClient)

	return nil
}

func (s *service) Handle(ctx context.Context, message *protocol.Message, transactions []model.Transaction) ([]model.Transaction, error) {
	tracer := otel.Tracer("worker_crossbell")

	_, snap := tracer.Start(ctx, "worker_crossbell:Handle")

	defer snap.End()

	internalTransactions := make([]model.Transaction, 0)

	for _, transaction := range transactions {
		addressTo := common.HexToAddress(transaction.AddressTo)

		// Processing of transactions for contracts Profile and LinkList only
		if addressTo != contract.AddressCharacter && addressTo != contract.AddressLinkList {
			continue
		}

		transaction.Platform = s.Name()

		// Retain the action model of the transfer type
		transferMap := make(map[int64]model.Transfer)

		for _, transfer := range transaction.Transfers {
			transferMap[transfer.Index] = transfer
		}

		// Empty transfer data to avoid data duplication
		transaction.Transfers = make([]model.Transfer, 0)

		// Get the raw data directly via Ethereum RPC node
		receipt, err := s.ethereumClient.TransactionReceipt(ctx, common.HexToHash(transaction.Hash))
		if err != nil {
			return nil, err
		}

		if transaction.Transfers, err = s.handleReceipt(ctx, message, transaction, receipt, transferMap); err != nil {
			return nil, err
		}

		internalTransactions = append(internalTransactions, transaction)
	}

	return internalTransactions, nil
}

func (s *service) handleReceipt(ctx context.Context, message *protocol.Message, transaction model.Transaction, receipt *types.Receipt, transferMap map[int64]model.Transfer) ([]model.Transfer, error) {
	tracer := otel.Tracer("worker_crossbell")

	_, snap := tracer.Start(ctx, "worker_crossbell:handleReceipt")

	defer snap.End()

	internalTransfers := make([]model.Transfer, 0)

	for _, log := range receipt.Logs {
		logIndex := int64(log.Index)

		transfer, exist := transferMap[logIndex]
		if !exist {
			sourceData, err := json.Marshal(log)
			if err != nil {
				return nil, err
			}

			// Virtual transfer
			transfer = model.Transfer{
				TransactionHash: transaction.Hash,
				Timestamp:       transaction.Timestamp,
				Index:           logIndex,
				AddressFrom:     transaction.AddressFrom,
				Metadata:        metadata.Default,
				Network:         message.Network,
				Source:          protocol.SourceOrigin,
				SourceData:      sourceData,
			}
		}

		internalTransfer, err := s.handler.Handle(ctx, transaction, transfer)
		if err != nil {
			if !errors.Is(err, contract.ErrorUnknownUnknownEvent) {
				logger.Global().Error("handle crossbell transfer failed", zap.Error(err))
			}

			continue
		}

		internalTransfer.Platform = s.Name()
		internalTransfers = append(internalTransfers, *internalTransfer)
	}

	return internalTransfers, nil
}

func (s *service) Jobs() []worker.Job {
	return nil
}

func New() worker.Worker {
	return &service{}
}
