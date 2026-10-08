package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
)

type KafkaSource struct {
	Logger           *logger.ContextLogger
	Client           *kgo.Client
	mu               sync.Mutex
	uncommitted      map[int32]*kgo.Record
	assigned         map[int32]bool
	traceToPartition map[string]int32
}

func NewKafkaSource(logger *logger.ContextLogger, brokers string, groupID string, topic string) (*KafkaSource, error) {
	src := &KafkaSource{
		Logger:           logger,
		uncommitted:      make(map[int32]*kgo.Record),
		assigned:         make(map[int32]bool),
		traceToPartition: make(map[string]int32),
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(strings.Split(brokers, ",")...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topic),
		// Auto commit disabled to allow manual offset commits after processing
		kgo.DisableAutoCommit(),
		// Start consuming from the earliest offset if no committed offset is found
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		// Block rebalances while records from a poll are being processed
		kgo.BlockRebalanceOnPoll(),
		kgo.OnPartitionsAssigned(func(ctx context.Context, cl *kgo.Client, m map[string][]int32) {
			src.mu.Lock()
			defer src.mu.Unlock()
			for _, partitions := range m {
				for _, p := range partitions {
					src.assigned[p] = true
				}
			}
		}),
		kgo.OnPartitionsRevoked(func(ctx context.Context, cl *kgo.Client, m map[string][]int32) {
			src.mu.Lock()
			defer src.mu.Unlock()
			for _, partitions := range m {
				for _, p := range partitions {
					delete(src.assigned, p)
					delete(src.uncommitted, p)
				}
			}
		}),
		kgo.OnPartitionsLost(func(ctx context.Context, cl *kgo.Client, m map[string][]int32) {
			src.mu.Lock()
			defer src.mu.Unlock()
			for _, partitions := range m {
				for _, p := range partitions {
					delete(src.assigned, p)
					delete(src.uncommitted, p)
				}
			}
		}),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		logger.Error(context.Background(), "Failed to create Kafka client", zap.Error(err))
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}
	src.Client = client

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		logger.Error(context.Background(), "Failed to connect to Kafka", zap.Error(err))
		client.Close()
		return nil, fmt.Errorf("failed to connect to kafka: %w", err)
	}

	logger.Info(context.Background(), "Successfully connected to Kafka with franz-go")
	return src, nil
}

func (k *KafkaSource) Poll(ctx context.Context, max int) ([]schema.CovidEvent, error) {
	if max <= 0 {
		max = 100
	}
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		fetches := k.Client.PollRecords(ctx, max)
		k.Client.AllowRebalance()

		records := fetches.Records()
		if len(records) > 0 {
			events := make([]schema.CovidEvent, 0, len(records))
			for _, record := range records {
				var event schema.CovidEvent
				if err := json.Unmarshal(record.Value, &event); err != nil {
					k.Logger.Warn(ctx, "Failed to unmarshal event, skipping malformed record",
						zap.Error(err),
						zap.String("topic", record.Topic),
						zap.Int32("partition", record.Partition),
						zap.Int64("offset", record.Offset),
					)
					k.mu.Lock()
					if k.assigned[record.Partition] {
						k.uncommitted[record.Partition] = record
					}
					k.mu.Unlock()
					continue
				}

				k.mu.Lock()
				if !k.assigned[record.Partition] {
					k.mu.Unlock()
					k.Logger.Warn(ctx, "Record received for unassigned partition, skipping",
						zap.String("topic", record.Topic),
						zap.Int32("partition", record.Partition),
						zap.Int64("offset", record.Offset),
					)
					continue
				}
				k.uncommitted[record.Partition] = record
				k.traceToPartition[event.TraceID] = record.Partition
				k.mu.Unlock()

				events = append(events, event)
			}
			return events, nil
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fe := range errs {
				if errors.Is(fe.Err, context.DeadlineExceeded) || errors.Is(fe.Err, context.Canceled) {
					k.Client.AllowRebalance()
					return nil, fe.Err
				}
			}
			k.Client.AllowRebalance()
			return nil, errs[0].Err
		}

		if ctx.Err() != nil {
			k.Client.AllowRebalance()
			return nil, ctx.Err()
		}

		// When PollRecords returns with 0 records and no error (e.g. empty broker fetch),
		// allow rebalance so consumer group operations are not blocked, then continue polling until ctx expires.
		k.Client.AllowRebalance()
	}
}

func (k *KafkaSource) GetEvent(ctx context.Context) (*schema.CovidEvent, error) {
	events, err := k.Poll(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, context.DeadlineExceeded
	}
	return &events[0], nil
}

func (k *KafkaSource) FilterAssigned(events []schema.CovidEvent) []schema.CovidEvent {
	k.mu.Lock()
	defer k.mu.Unlock()

	out := make([]schema.CovidEvent, 0, len(events))
	for _, e := range events {
		p, ok := k.traceToPartition[e.TraceID]
		if ok && !k.assigned[p] {
			k.Logger.Warn(context.Background(), "Discarding buffered event for revoked partition",
				zap.String("trace_id", e.TraceID),
				zap.Int32("partition", p),
			)
			delete(k.traceToPartition, e.TraceID)
			continue
		}
		out = append(out, e)
	}
	return out
}

func (k *KafkaSource) Commit(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	defer k.Client.AllowRebalance()

	k.traceToPartition = make(map[string]int32)

	if len(k.uncommitted) == 0 {
		return nil
	}

	recordsToCommit := make([]*kgo.Record, 0, len(k.uncommitted))
	for partition, record := range k.uncommitted {
		if k.assigned[partition] {
			recordsToCommit = append(recordsToCommit, record)
		}
	}

	if len(recordsToCommit) == 0 {
		k.uncommitted = make(map[int32]*kgo.Record)
		return nil
	}

	if err := k.Client.CommitRecords(ctx, recordsToCommit...); err != nil {
		k.Logger.Error(ctx, "Failed to commit records", zap.Error(err))
		return err
	}

	k.uncommitted = make(map[int32]*kgo.Record)
	return nil
}

func (k *KafkaSource) Close(ctx context.Context) error {
	k.Logger.Info(ctx, "Closing Kafka client")
	if k.Client != nil {
		k.Client.CloseAllowingRebalance()
	}
	return nil
}
