package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

type Exporter struct {
	redisAddress string
	redisDb      int
	interval     int
	storage      *prometheus.GaugeVec
	messages     *prometheus.GaugeVec
}

// Collect implements prometheus.Collector.
func (e Exporter) Collect(ch chan<- prometheus.Metric) {
	e.storage.Collect(ch)
	e.messages.Collect(ch)
}

// Describe implements prometheus.Collector.
func (e Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.storage.Describe(ch)
	e.messages.Describe(ch)
}

func (e Exporter) Gather(ctx context.Context) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     e.redisAddress,
		Password: "",
		DB:       e.redisDb,
	})
	defer rdb.Close()

	var cursor uint64
	pattern := "*/quota/*"
	iter := rdb.Scan(ctx, cursor, pattern, 0).Iterator()
	for iter.Next(ctx) {
		// key is "jedrzej@qweqwe.pl/quota/storage" or "jedrzej@qweqwe.pl/quota/messages"
		key := iter.Val()
		keyParts := strings.SplitN(key, "/", 3)
		if len(keyParts) != 3 {
			log.Printf("error splitting key %s", key)

			continue
		}

		email := keyParts[0]

		parts := strings.SplitN(email, "@", 2)
		if len(parts) != 2 {
			log.Printf("error splitting key %s", email)

			continue
		}
		domain := parts[1]

		v, err := rdb.Get(ctx, key).Float64()
		if err != nil {
			log.Printf("error retrieving key [%v]: %v", key, err)

			continue
		}

		switch keyParts[2] {
		case "storage":
			e.storage.WithLabelValues(email, domain).Set(v)
		case "messages":
			e.messages.WithLabelValues(email, domain).Set(v)
		default:
			log.Printf("unknown component in the key: %s", key)
		}

	}
}

func (e Exporter) Start(ctx context.Context) {
	go func() {
		for {
			e.Gather(ctx)
			select {
			case <-ctx.Done():
				log.Print("Exporter shutdown")
				return
			case <-time.After(time.Second * time.Duration(e.interval)):
			}
		}
	}()
}

func New(redisAddress string, redisDb int) Exporter {
	exporter := Exporter{
		redisAddress: redisAddress,
		redisDb:      redisDb,
		interval:     60,
		storage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "dovecot_mbox_storage",
				Help: "Mailbox storage usage",
			}, []string{"email", "domain"}),
		messages: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dovecot_mbox_messages",
			Help: "Mailbox messages count",
		}, []string{"email", "domain"}),
	}

	return exporter
}
