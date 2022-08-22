package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/naturalselectionlabs/pregod/common/utils/opentelemetry"
	"github.com/naturalselectionlabs/pregod/common/worker/snapshot"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel"
)

type (
	PullInfoStatus int32
)

const (
	PullInfoStatusNotLatest PullInfoStatus = 1
	PullInfoStatusLatest    PullInfoStatus = 2
)

type StatusStroge struct {
	Pos    int32          `json:"pos"`
	Status PullInfoStatus `json:"status"`
}

type SnapshotJobBase struct {
	Name           string
	RedisClient    *redis.Client
	SnapshotClient *snapshot.Client

	Limit          int32
	HighUpdateTime time.Duration
	LowUpdateTime  time.Duration
}

func (job *SnapshotJobBase) Check() error {
	if job.Name == "" {
		return fmt.Errorf("job name is empty")
	}

	if job.SnapshotClient == nil {
		return fmt.Errorf("snapshot worker is nil")
	}

	return nil
}

func (job *SnapshotJobBase) GetLastStatusFromCache(ctx context.Context) (statusStroge StatusStroge, err error) {
	tracer := otel.Tracer("snapshot_job")
	_, trace := tracer.Start(ctx, "snapshot_job:GetLastStatusFromCache")

	defer func() { opentelemetry.Log(trace, nil, statusStroge, err) }()

	if job.Name == "" {
		return StatusStroge{}, fmt.Errorf("job name is empty")
	}

	if job.RedisClient == nil {
		return StatusStroge{}, fmt.Errorf("redis worker is nil")
	}

	statusKey := job.Name + "_status"
	statusStroge = StatusStroge{
		Pos:    0,
		Status: PullInfoStatusNotLatest,
	}

	data, err := job.RedisClient.Get(ctx, statusKey).Result()
	if err != nil {
		return StatusStroge{}, err
	}

	if err = json.Unmarshal([]byte(data), &statusStroge); err != nil {
		return StatusStroge{}, fmt.Errorf("unmarshal %s from cache error:%+v", statusKey, err)
	}

	return statusStroge, nil
}

func (job *SnapshotJobBase) SetCurrentStatus(ctx context.Context, stroge StatusStroge) error {
	if job.Name == "" {
		return fmt.Errorf("job name is empty")
	}

	if job.RedisClient == nil {
		return fmt.Errorf("redis worker is nil")
	}

	if stroge.Pos <= 0 {
		return fmt.Errorf("pos is less than 0")
	}

	data, err := json.Marshal(stroge)
	if err != nil {
		return fmt.Errorf("marshal %+v to json error:%+v", stroge, err)
	}

	job.RedisClient.Set(ctx, job.Name+"_status", data, 0)

	return nil
}
