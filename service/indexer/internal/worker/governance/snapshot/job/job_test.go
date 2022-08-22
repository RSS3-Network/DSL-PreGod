package job_test

import (
	"github.com/go-redis/redis/v8"
	"github.com/naturalselectionlabs/pregod/common/cache"
	configx "github.com/naturalselectionlabs/pregod/common/config"
	"github.com/naturalselectionlabs/pregod/common/worker/snapshot"
	job2 "github.com/naturalselectionlabs/pregod/service/indexer/internal/worker/governance/snapshot/job"
	"testing"
)

var (
	redisClient    *redis.Client
	snapshotClient *snapshot.Client
)

func init() {
	var err error

	if err != nil {
		panic(err)
	}

	redisClient, err = cache.Dial(&configx.Redis{
		Addr:       "127.0.0.1:6379",
		Password:   "",
		DB:         0,
		TLSEnabled: false,
	})
	if err != nil {
		panic(err)
	}

	snapshotClient = snapshot.NewClient()
}

func TestSnapshotSpaceInnerJob(t *testing.T) {
	count := 0

	for {
		if count > 2 {
			break
		}

		spaceJob := job2.SnapshotSpaceJob{
			SnapshotJobBase: job2.SnapshotJobBase{
				Name:           "snapshot_space_job",
				SnapshotClient: snapshotClient,
				Limit:          100,
			},
		}

		if _, err := spaceJob.InnerJobRun(); err != nil {
			panic(err)
		}

		count = count + 1
	}
}

func TestSnapshotProposalInnerJob(t *testing.T) {
	count := 0

	for {
		if count > 2 {
			break
		}

		proposalJob := job2.SnapshotProposalJob{
			SnapshotJobBase: job2.SnapshotJobBase{
				Name:           "snapshot_proposal_job",
				SnapshotClient: snapshotClient,
				Limit:          100,
			},
		}

		if _, err := proposalJob.InnerJobRun(); err != nil {
			panic(err)
		}

		count = count + 1
	}
}

func TestSnapshotVoteInnerJob(t *testing.T) {
	count := 0

	for {
		if count > 2 {
			break
		}
		voteJob := job2.SnapshotVoteJob{
			SnapshotJobBase: job2.SnapshotJobBase{
				Name:           "snapshot_vote_job",
				SnapshotClient: snapshotClient,
				Limit:          100,
			},
		}

		if _, err := voteJob.InnerJobRun(); err != nil {
			t.Error(err)
		}

		count = count + 1
	}
}
