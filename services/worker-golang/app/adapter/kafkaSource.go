package adapter

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
	"safezone.service.worker-golang/app/pkg/logger"
	"safezone.service.worker-golang/app/schema"
)

type KafkaSource struct {
	Logger *logger.ContextLogger
	Client *kgo.Client
	Reader *kgo.FetchesRecordIter
}

func NewKafkaSource(logger *logger.ContextLogger, brokers string, groupID string, topic string) *KafkaSource {

	opts := []kgo.Opt{
		kgo.SeedBrokers(strings.Split(brokers, ",")...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topic),
		// Auto commit disabled to allow manual offset commits after processing
		kgo.DisableAutoCommit(),
		// Start consuming from the earliest offset if no committed offset is found
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		logger.Error(context.Background(), "Failed to create Kafka client", zap.Error(err))
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		logger.Error(context.Background(), "Failed to connect to Kafka", zap.Error(err))
		return nil
	}

	logger.Info(context.Background(), "Successfully connected to Kafka with franz-go")

	return &KafkaSource{
		Logger: logger,
		Client: client,
		Reader: nil,
	}
}

func (k *KafkaSource) GetEvent(ctx context.Context) (*schema.CovidEvent, error) {
	if k.Reader == nil || k.Reader.Done() {
		fetches := k.Client.PollFetches(ctx)

		// handle fetch errors
		if errs := fetches.Errors(); len(errs) > 0 {
			return nil, errs[0].Err
		}

		k.Reader = fetches.RecordIter()
		// check reader is done again， in case no records were fetched
		if k.Reader.Done() {
			return nil, context.DeadlineExceeded
		}
	}

	record := k.Reader.Next()

	var event schema.CovidEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		k.Logger.Warn(ctx, "json unmarshal failed", zap.Error(err))
		return nil, err
	}
	// Manual offset: Commit the record after processing
	if err := k.Client.CommitRecords(ctx, record); err != nil {
		k.Logger.Error(ctx, "failed to commit offset", zap.Error(err))
	}

	k.Logger.Debug(ctx, "Kafka event received", zap.String("trace_id", event.TraceID))

	return &event, nil
}

func (k *KafkaSource) Close(ctx context.Context) error {
	k.Logger.Info(ctx, "Closing Kafka client")
	k.Client.Close()
	return nil
}
