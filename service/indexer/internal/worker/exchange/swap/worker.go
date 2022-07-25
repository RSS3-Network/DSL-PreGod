package swap

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	configx "github.com/naturalselectionlabs/pregod/common/config"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/uniswap"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/protocol/filter"
	"github.com/naturalselectionlabs/pregod/common/utils/logger"
	"github.com/naturalselectionlabs/pregod/common/utils/opentelemetry"
	"github.com/naturalselectionlabs/pregod/common/utils/shedlock"
	"github.com/naturalselectionlabs/pregod/internal/token"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker"
	lop "github.com/samber/lo/parallel"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	Name = "swap"
)

var _ worker.Worker = &service{}

type service struct {
	employer          *shedlock.Employer
	databaseClient    *gorm.DB
	tokenClient       *token.Client
	ethereumClientMap map[string]*ethclient.Client
}

func (s *service) Name() string {
	return Name
}

func (s *service) Networks() []string {
	return []string{
		protocol.NetworkEthereum, protocol.NetworkPolygon, protocol.NetworkBinanceSmartChain,
	}
}

func (s *service) Initialize(ctx context.Context) error {
	return nil
}

func (s *service) Handle(ctx context.Context, message *protocol.Message, transactions []model.Transaction) (data []model.Transaction, err error) {
	tracer := otel.Tracer("worker_swap")
	_, trace := tracer.Start(ctx, "worker_swap:Handle")

	defer opentelemetry.Log(trace, transactions, data, err)

	switch message.Network {
	case protocol.NetworkZkSync:
		return s.handleZkSync(ctx, message, transactions)
	default:
		return s.handleEthereum(ctx, message, transactions)
	}
}

func (s *service) handleEthereum(ctx context.Context, message *protocol.Message, transactions []model.Transaction) (data []model.Transaction, err error) {
	tracer := otel.Tracer("worker_swap")
	_, trace := tracer.Start(ctx, "worker_swap:handleEthereum")

	defer opentelemetry.Log(trace, transactions, data, err)

	internalTransactionMap := make(map[string]model.Transaction)
	opt := lop.NewOption().WithConcurrency(ethereum.RPCMaxConcurrency)

	var mu sync.Mutex

	lop.ForEach(transactions, func(transaction model.Transaction, i int) {
		address := message.Address

		// Exclude transfers to self
		if transaction.AddressTo == address {
			return
		}

		// Handle swap entry
		mu.Lock()
		router, exist := routerMap[transaction.AddressTo]
		mu.Unlock()
		if !exist {
			return
		}

		if transaction.Transfers, err = s.handleEthereumTransaction(ctx, message, &transaction, router); err != nil {
			logger.Global().Warn("failed to handle ethereum transaction", zap.Error(err), zap.String("network", message.Network), zap.String("transaction_hash", transaction.Hash), zap.String("address", message.Address))

			return
		}

		transaction.Tag, transaction.Type = filter.UpdateTagAndType(filter.TagExchange, transaction.Tag, filter.ExchangeSwap, transaction.Type)

		mu.Lock()
		internalTransactionMap[transaction.Hash] = transaction
		mu.Unlock()
	}, opt)

	// Lay the map flat
	internalTransactions := make([]model.Transaction, 0)

	for _, transaction := range internalTransactionMap {
		internalTransactions = append(internalTransactions, transaction)
	}

	return internalTransactions, nil
}

func (s *service) handleEthereumTransaction(ctx context.Context, message *protocol.Message, transaction *model.Transaction, router Router) (internalTransfers []model.Transfer, err error) {
	tracer := otel.Tracer("worker_swap")
	_, trace := tracer.Start(ctx, "worker_swap:handleEthereumTransaction")

	defer opentelemetry.Log(trace, transaction, internalTransfers, err)

	ethereumClient, exist := s.ethereumClientMap[message.Network]
	if !exist {
		return nil, errors.New("not supported network")
	}

	tokenMap := map[common.Address]*big.Int{}

	for _, transfer := range transaction.Transfers {
		if transfer.Index == protocol.IndexVirtual {
			internalTransfers = append(internalTransfers, transfer)

			break
		}
	}

	receipt, err := ethereumClient.TransactionReceipt(ctx, common.HexToHash(transaction.Hash))
	if err != nil {
		return nil, err
	}

	for _, log := range receipt.Logs {
		for _, topic := range log.Topics {
			var internalTokenMap map[common.Address]*big.Int

			switch topic {
			case uniswap.EventHashSwapV2:
				internalTokenMap, err = s.handleUniswapV2(ctx, message, *log, tokenMap, ethereumClient)
			case uniswap.EventHashSwapV3:
				internalTokenMap, err = s.handleUniswapV3(ctx, message, *log, tokenMap, ethereumClient)
			default:
				continue
			}

			if err != nil {
				return nil, err
			}

			tokenMap = internalTokenMap
		}
	}

	swapMetadataModel := metadata.Swap{
		Protocol: router.Protocol,
	}

	for token, value := range tokenMap {
		erc20, err := s.tokenClient.ERC20(ctx, message.Network, token.String())
		if err != nil {
			return nil, err
		}

		tokenValueTo := decimal.NewFromBigInt(value, 0)

		tokenValueFrom := tokenValueTo.Abs()

		switch value.Cmp(big.NewInt(0)) {
		case -1:
			swapMetadataModel.TokenFrom = metadata.Token{
				Name:            erc20.Name,
				Symbol:          erc20.Symbol,
				Decimals:        erc20.Decimals,
				Value:           &tokenValueFrom,
				ContractAddress: token.String(),
				Standard:        protocol.TokenStandardERC20,
				Image:           erc20.Logo,
			}
		case 0:
			continue
		case 1:
			swapMetadataModel.TokenTo = metadata.Token{
				Name:            erc20.Name,
				Symbol:          erc20.Symbol,
				Decimals:        erc20.Decimals,
				Value:           &tokenValueTo,
				ContractAddress: token.String(),
				Standard:        protocol.TokenStandardERC20,
				Image:           erc20.Logo,
			}
		}
	}

	swapMetadata, err := json.Marshal(swapMetadataModel)
	if err != nil {
		return nil, err
	}

	transaction.Platform = router.Name

	internalTransfers = append(internalTransfers, model.Transfer{
		TransactionHash: transaction.Hash,
		Timestamp:       transaction.Timestamp,
		BlockNumber:     big.NewInt(transaction.BlockNumber),
		Tag:             filter.TagExchange,
		Type:            filter.ExchangeSwap,
		Index:           0, // TODO
		AddressFrom:     transaction.AddressFrom,
		AddressTo:       transaction.AddressFrom,
		Metadata:        swapMetadata,
		Network:         message.Network,
		Platform:        router.Name,
		Source:          protocol.SourceOrigin,
		// SourceData:      sourceDa,
		RelatedUrls: []string{
			ethereum.BuildScanURL(message.Network, transaction.Hash),
		},
	})

	return internalTransfers, nil
}

func (s *service) handleZkSync(ctx context.Context, message *protocol.Message, transactions []model.Transaction) ([]model.Transaction, error) {
	// TODO Not yet supported

	return nil, nil
}

func (s *service) Jobs() []worker.Job {
	return []worker.Job{
		&Job{
			databaseClient: s.databaseClient,
		},
	}
}

func New(config *configx.RPC, employer *shedlock.Employer, databaseClient *gorm.DB) (worker.Worker, error) {
	var err error

	svc := service{
		ethereumClientMap: make(map[string]*ethclient.Client),
		employer:          employer,
		databaseClient:    databaseClient,
	}

	if svc.ethereumClientMap[protocol.NetworkEthereum], err = ethclient.Dial(config.General.Ethereum.HTTP); err != nil {
		return nil, err
	}

	if svc.ethereumClientMap[protocol.NetworkPolygon], err = ethclient.Dial(config.General.Polygon.HTTP); err != nil {
		return nil, err
	}

	if svc.ethereumClientMap[protocol.NetworkBinanceSmartChain], err = ethclient.Dial(config.General.BinanceSmartChain.HTTP); err != nil {
		return nil, err
	}

	if svc.ethereumClientMap[protocol.NetworkXDAI], err = ethclient.Dial(config.General.XDAI.HTTP); err != nil {
		return nil, err
	}

	svc.tokenClient = token.New(svc.databaseClient, svc.ethereumClientMap)

	return &svc, nil
}
