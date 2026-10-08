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
	Logger      *logger.ContextLogger
	Client      *kgo.Client
	mu          sync.Mutex
	uncommitted map[int32]*kgo.Record
	assigned    map[int32]bool
}

func NewKafkaSource(logger *logger.ContextLogger, brokers string, groupID string, topic string) (*KafkaSource, error) {
	src := &KafkaSource{
		Logger:      logger,
		uncommitted: make(map[int32]*kgo.Record),
		assigned:    make(map[int32]bool),
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

func (k *KafkaSource) GetEvent(ctx context.Context) (*schema.CovidEvent, error) {
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		fetches := k.Client.PollRecords(ctx, 1)

		records := fetches.Records()
		if len(records) > 0 {
			record := records[0]

			k.mu.Lock()
			assigned := k.assigned[record.Partition]
			k.mu.Unlock()
			if !assigned {
				continue
			}

			var event schema.CovidEvent
			if err := json.Unmarshal(record.Value, &event); err != nil {
				k.Logger.Warn(ctx, "Failed to unmarshal event, skipping malformed record",
					zap.Error(err),
					zap.String("topic", record.Topic),
					zap.Int32("partition", record.Partition),
					zap.Int64("offset", record.Offset),
				)
				k.trackRecord(record)
				continue
			}

			k.trackRecord(record)
			k.Logger.Debug(ctx, "Kafka event received", zap.String("trace_id", event.TraceID))
			return &event, nil
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fe := range errs {
				if errors.Is(fe.Err, context.DeadlineExceeded) || errors.Is(fe.Err, context.Canceled) {
					return nil, fe.Err
				}
			}
			return nil, errs[0].Err
		}

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		return nil, context.DeadlineExceeded
	}
}

func (k *KafkaSource) trackRecord(record *kgo.Record) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.uncommitted[record.Partition] = record
}

func (k *KafkaSource) Commit(ctx context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	defer k.Client.AllowRebalance()

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
