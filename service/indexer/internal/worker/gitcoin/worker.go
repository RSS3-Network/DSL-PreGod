package gitcoin

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/opentelemetry"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/protocol/filter"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker/gitcoin/job"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

const (
	Name = protocol.PlatfromGitcoin
)

var _ worker.Worker = (*service)(nil)

type service struct {
	databaseClient         *gorm.DB
	redisClient            *redis.Client
	gitcoinProjectCacheKey string
}

func New(databaseClient *gorm.DB, redisClient *redis.Client) worker.Worker {
	return &service{
		databaseClient:         databaseClient,
		redisClient:            redisClient,
		gitcoinProjectCacheKey: "gitcoin_project",
	}
}

func (s *service) Name() string {
	return Name
}

func (s *service) Networks() []string {
	return []string{
		protocol.NetworkEthereum,
		protocol.NetworkPolygon,
		protocol.NetworkZkSync,
	}
}

func (s *service) Initialize(ctx context.Context) error {
	return nil
}

func (s *service) Handle(ctx context.Context, message *protocol.Message, transactions []model.Transaction) (internalTransactions []model.Transaction, err error) {
	tracer := otel.Tracer("gitcoin_worker")
	_, trace := tracer.Start(ctx, "gitcoin_worker:Handle")

	defer func() { opentelemetry.Log(trace, transactions, internalTransactions, err) }()

	internalTransactionMap := make(map[string]model.Transaction)

	for _, transaction := range transactions {
		for _, transfer := range transaction.Transfers {
			if transfer.AddressTo == "" ||
				transfer.AddressTo == "0x0" ||
				transfer.AddressTo == "0x0000000000000000000000000000000000000000" {
				continue
			}

			projectStr, err := s.redisClient.HGet(ctx, s.gitcoinProjectCacheKey, transfer.AddressTo).Result()
			if err != nil || len(projectStr) == 0 {
				continue
			}

			project := &model.GitcoinProject{}
			if err := json.Unmarshal([]byte(projectStr), &project); err != nil {
				logrus.Errorf("[gitcoin handle] json unmarshal project error: %v", err)
				continue
			}

			transfer.RelatedUrls = append(transfer.RelatedUrls, "https://gitcoin.co/grants"+strconv.Itoa(project.ID)+"/"+project.Slug)

			// format metadata
			var metadataModel metadata.Metadata

			if err := json.Unmarshal(transfer.Metadata, &metadataModel); err != nil {
				logrus.Errorf("[gitcoin handle] json unmarshal transfer metadata error: %v", err)
				continue
			}

			if metadataModel.Token != nil &&
				metadataModel.Token.TokenStandard != protocol.TokenStandardERC20 &&
				metadataModel.Token.TokenStandard != protocol.TokenStandardNative {
				continue
			}

			metadataModel.Gitcoin = &metadata.Gitcoin{
				ID:          project.ID,
				Slug:        project.Slug,
				Title:       project.Title,
				Description: project.Description,
				Logo:        project.Logo,
			}

			metadata, err := json.Marshal(metadataModel)
			if err != nil {
				continue
			}

			transfer.Platform = Name
			transfer.Metadata = metadata
			transfer.Tag = filter.UpdateTag(filter.TagDonation, transfer.Tag)

			if transfer.Tag == filter.TagDonation {
				transfer.Type = filter.DonationDonate
			}

			// Copy the transaction to map
			value, exist := internalTransactionMap[transaction.Hash]
			if !exist {
				value = transaction

				// Ignore transfers data that will not be updated
				value.Transfers = make([]model.Transfer, 0)
			}

			value.Tag = filter.UpdateTag(transfer.Tag, value.Tag)
			value.Transfers = append(value.Transfers, transfer)

			internalTransactionMap[transaction.Hash] = value

			// transaction tag
			transaction.Tag = filter.UpdateTag(transfer.Tag, transaction.Tag)
		}
	}

	internalTransactions = make([]model.Transaction, 0)

	for _, internalTransaction := range internalTransactionMap {
		internalTransaction.Platform = Name

		internalTransactions = append(internalTransactions, internalTransaction)
	}

	return internalTransactions, nil
}

func (s *service) Jobs() []worker.Job {
	return []worker.Job{
		&job.GitcoinProjectJob{
			DatabaseClient:         s.databaseClient,
			RedisClient:            s.redisClient,
			GitcoinProjectCacheKey: s.gitcoinProjectCacheKey,
		},
	}
}
