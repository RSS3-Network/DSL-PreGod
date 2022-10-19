package farcaster

import (
	"context"
	"encoding/json"
	"github.com/naturalselectionlabs/pregod/common/database/model"
	"github.com/naturalselectionlabs/pregod/common/database/model/metadata"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/protocol/filter"
	"github.com/naturalselectionlabs/pregod/common/utils/loggerx"
	"github.com/naturalselectionlabs/pregod/common/utils/opentelemetry"
	"github.com/naturalselectionlabs/pregod/common/worker/farcaster"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker"
	"github.com/naturalselectionlabs/pregod/service/indexer/internal/worker/social/crossbell/handler"
	lop "github.com/samber/lo/parallel"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"time"
)

var _ worker.Worker = (*service)(nil)

type service struct {
	client  *farcaster.Client
	handler handler.Interface
}

func (s *service) Name() string {
	return protocol.PlatformFarcaster
}

func (s *service) Networks() []string {
	return []string{
		protocol.NetworkFarcaster,
	}
}

func (s *service) Initialize(ctx context.Context) (err error) {
	return nil
}

func (s *service) Handle(ctx context.Context, message *protocol.Message, transactions []model.Transaction) ([]model.Transaction, error) {
	tracer := otel.Tracer("worker_farcaster")

	_, snap := tracer.Start(ctx, "worker_farcaster:Handle")

	defer snap.End()

	internalTransactions := make([]model.Transaction, 0)

	lop.ForEach(transactions, func(transaction model.Transaction, i int) {
		transaction.Platform = protocol.PlatformFarcaster

		// Retain the action model of the transfer type
		transferMap := make(map[int64]model.Transfer)

		for _, transfer := range transaction.Transfers {
			if err := s.HandlePost(ctx, &transfer); err != nil {
				loggerx.Global().Warn("worker_farcaster: failed to HandlePost", zap.Error(err), zap.String("network", message.Network), zap.String("transaction_hash", transaction.Hash), zap.String("address", message.Address))

				continue
			}

			transferMap[transfer.Index] = transfer
		}

		// Empty transfer data to avoid data duplication
		transaction.Transfers = make([]model.Transfer, 0)
		transaction.Transfers = append(transaction.Transfers, transferMap[protocol.IndexVirtual])

		for _, transfer := range transaction.Transfers {
			transaction.Tag, transaction.Type = filter.UpdateTagAndType(transfer.Tag, transaction.Tag, transfer.Type, transaction.Type)
		}

		transaction.Owner = transaction.AddressFrom

		internalTransactions = append(internalTransactions, transaction)
	})

	return internalTransactions, nil
}

func (s *service) HandlePost(ctx context.Context, transfer *model.Transfer) (err error) {
	tracer := otel.Tracer("worker_farcaster")
	_, trace := tracer.Start(ctx, "worker_farcaster:HandlePost")

	defer func() { opentelemetry.Log(trace, transfer.AddressFrom, transfer.TransactionHash, err) }()

	var cast farcaster.Cast
	err = json.Unmarshal(transfer.SourceData, &cast)
	if err != nil {
		loggerx.Global().Warn("unable to unmarshal cast", zap.Error(err))
		return err
	}

	post := &metadata.Post{
		CreatedAt: time.UnixMilli(cast.Body.PublishedAt).Format(time.RFC3339),
		Title:     cast.Body.Data.Text,
		Summary:   "",
		Body:      cast.Body.Data.Text,
	}
	transfer.Metadata, _ = json.Marshal(post)
	transfer.Tag = filter.TagSocial
	transfer.Type = filter.SocialPost
	// TODO: farcaster does not have an API for individual posts, so we can't get the post URL
	return nil
}

func (s *service) Jobs() []worker.Job {
	return nil
}

func New() worker.Worker {
	return &service{
		client: farcaster.NewClient(),
	}
}
