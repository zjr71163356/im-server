package mongo

import (
	"context"
	"time"

	"im-server/pkg/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Client 封装 Mongo 客户端与常用集合句柄
type Client struct {
	DB       *mongo.Database
	Messages *mongo.Collection
}

// New 创建 Mongo 客户端
func New(ctx context.Context, cfg config.MongoConfig) (*Client, error) {
	cli, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.URI).SetAuth(options.Credential{AuthSource: cfg.AuthSource}))
	if err != nil {
		return nil, err
	}
	ctxPing, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := cli.Ping(ctxPing, nil); err != nil {
		return nil, err
	}
	db := cli.Database(cfg.Database)
	c := &Client{DB: db, Messages: db.Collection("messages")}
	return c, nil
}
