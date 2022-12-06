package matters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	kurora "github.com/naturalselectionlabs/kurora/client"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/protocol/filter"
	"github.com/naturalselectionlabs/pregod/common/utils/loggerx"
	"github.com/naturalselectionlabs/pregod/common/utils/opentelemetry"
	"github.com/naturalselectionlabs/pregod/internal/token"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var (
	_               worker.Worker = (*service)(nil)
	AddressCuration               = strings.ToLower("0x5edebbdae7b5c79a69aacf7873796bb1ec664db8")
)

const Source = "kurora"

type service struct {
	kuroraClient *kurora.Client
	tokenClient  *token.Client
}

func (s *service) Name() string {
	return protocol.PlatformMatters
}

func (s *service) Networks() []string {
	return []string{
		protocol.NetworkPolygon,
	}
}

func (s *service) Initialize(ctx context.Context) (err error) {
	return nil
}

func (s *service) Handle(ctx context.Context, message *protocol.Message, transactions []model.Transaction) (internalTransactions []model.Transaction, err error) {
	tracer := otel.Tracer("worker_matters")
	_, trace := tracer.Start(ctx, "worker_matters:Handle")

	defer func() { opentelemetry.Log(trace, transactions, internalTransactions, err) }()

	// datasource from kurora
	mattersQueryFrom := kurora.DatasetMattersEntryQuery{
		From:            lo.ToPtr(common.HexToAddress(message.Address)),
		BlockNumberFrom: lo.ToPtr(decimal.NewFromInt(message.BlockNumber)),
	}

	txs, err := s.fetchAndHandleTransfer(ctx, message, mattersQueryFrom)
	if err != nil {
		return nil, fmt.Errorf("matters query address from error: %w", err)
	}

	internalTransactions = append(internalTransactions, txs...)

	mattersQueryTo := kurora.DatasetMattersEntryQuery{
		To:              lo.ToPtr(common.HexToAddress(message.Address)),
		BlockNumberFrom: lo.ToPtr(decimal.NewFromInt(message.BlockNumber)),
	}

	txs, err = s.fetchAndHandleTransfer(ctx, message, mattersQueryTo)
	if err != nil {
		return nil, fmt.Errorf("matters query address to error: %w", err)
	}

	internalTransactions = append(internalTransactions, txs...)

	return internalTransactions, nil
}

func (s *service) fetchAndHandleTransfer(ctx context.Context, message *protocol.Message, mattersQuery kurora.DatasetMattersEntryQuery) (internalTransactions []model.Transaction, err error) {
	success := true

	for first := true; mattersQuery.Cursor != nil || first; first = false {
		entries, err := s.kuroraClient.FetchDatasetMattersEntries(ctx, mattersQuery)
		if err != nil {
			return nil, fmt.Errorf("matters entries: %w", err)
		}

		for _, entry := range entries {

			transaction := model.Transaction{
				BlockNumber: int64(entry.BlockNumber),
				Hash:        strings.ToLower(entry.TransactionHash.String()),
				Timestamp:   entry.Timestamp,
				Owner:       message.Address,
				Network:     message.Network,
				Tag:         filter.TagSocial,
				Type:        filter.SocialReward,
				AddressFrom: strings.ToLower(entry.From.String()),
				AddressTo:   AddressCuration,
				Success:     &success,
				Platform:    protocol.PlatformMatters,
				Source:      Source,
			}

			// handle transfer
			tokenMetadata, err := s.tokenClient.ERC20ToMetadata(ctx, message.Network, strings.ToLower(entry.Token.String()))
			if err != nil {
				loggerx.Global().Warn("failed to handle token metadata", zap.Error(err), zap.String("network", message.Network), zap.String("transaction_hash", transaction.Hash), zap.String("address", message.Address), zap.String("token", entry.Token.String()))

				continue
			}

			tokenValueDisplayTo := entry.Amount.Shift(-int32(tokenMetadata.Decimals))

			tokenMetadata.Value = &entry.Amount
			tokenMetadata.ValueDisplay = &tokenValueDisplayTo

			postOuter := &metadata.Post{
				Reward:         tokenMetadata,
				TypeOnPlatform: []string{"Curation"},
			}

			if len(entry.Author) > 0 && len(entry.Summary) > 0 {
				postOuter.Target = &metadata.Post{
					Title:          entry.Title,
					Summary:        entry.Summary,
					Body:           entry.ContentMarkdown,
					Author:         []string{fmt.Sprintf("%s%s", "https://matters.news/", entry.Author), entry.Author},
					TypeOnPlatform: []string{"Post"},
				}
			}

			metadataRaw, err := json.Marshal(postOuter)
			if err != nil {
				loggerx.Global().Warn("failed to handle marshall", zap.Error(err), zap.String("network", message.Network), zap.String("transaction_hash", transaction.Hash), zap.String("address", message.Address))

				continue
			}

			internalTransfer := model.Transfer{
				TransactionHash: strings.ToLower(entry.TransactionHash.String()),
				Timestamp:       entry.Timestamp,
				Tag:             filter.TagSocial,
				Type:            filter.SocialReward,
				Index:           int64(entry.LogIndex),
				Network:         message.Network,
				AddressFrom:     strings.ToLower(entry.From.String()),
				AddressTo:       strings.ToLower(entry.To.String()),
				Platform:        protocol.PlatformMatters,
				Source:          Source,
				Metadata:        metadataRaw,
				RelatedUrls: []string{
					ethereum.BuildScanURL(message.Network, strings.ToLower(entry.TransactionHash.String())),
					strings.ReplaceAll(entry.URI, "ipfs://", "https://ipfs.io/ipfs/"),
				},
			}

			transaction.Transfers = append(transaction.Transfers, internalTransfer)

			internalTransactions = append(internalTransactions, transaction)

		}

		if len(entries) == 0 {
			break
		}

		lastEntry, _ := lo.Last(entries)

		// query cursor
		mattersQuery.Cursor = lo.ToPtr(fmt.Sprintf("%s-%d", lastEntry.TransactionHash.String(), lastEntry.LogIndex))
	}

	return internalTransactions, nil
}

func (s *service) Jobs() []worker.Job {
	return []worker.Job{}
}

func New(ctx context.Context, endpoint string) (worker.Worker, error) {
	kuroraClient, err := kurora.Dial(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("dial kurora: %w", err)
	}

	return &service{
		kuroraClient: kuroraClient,
		tokenClient:  token.New(),
	}, nil
}
