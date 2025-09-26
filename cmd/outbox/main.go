package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"im-server/pkg/broker"
	"im-server/pkg/config"
	"im-server/pkg/dao"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	ctx := context.Background()

	// DB
	db, err := sql.Open("mysql", config.Config.Database.MySQL.DSN)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()
	q := dao.New(db)

	// Kafka producer
	producer := broker.NewKafkaProducer(config.Config.Broker)
	defer producer.Close()

	log.Println("Outbox dispatcher started")
	for {
		rows, err := q.GetPendingOutboxEvents(ctx, 100)
		if err != nil {
			log.Printf("load pending: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		if len(rows) == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, r := range rows {
			// 可选：从 payload 里抽取 conv_id 作为 key
			var key json.RawMessage

			pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := producer.Publish(pubCtx, producer.Topic(r.Topic), key, r.Payload)
			cancel()
			if err != nil {
				log.Printf("publish failed id=%d: %v", r.ID, err)
				if markErr := q.MarkOutboxEventFailed(ctx, r.ID); markErr != nil {
					log.Printf("mark failed error id=%d: %v", r.ID, markErr)
				}
				continue
			}

			if err := q.MarkOutboxEventSent(ctx, r.ID); err != nil {
				log.Printf("mark sent failed id=%d: %v", r.ID, err)
			}
		}
	}
}
