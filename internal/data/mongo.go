package data

import (
	"context"
	"time"

	"kratos-demo/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoGreeter MongoDB中的Greeter文档结构
type MongoGreeter struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Hello     string             `bson:"hello"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// mongoGreeterRepo MongoDB实现的GreeterRepo
type mongoGreeterRepo struct {
	data   *Data
	logger log.Logger
}

// NewMongoGreeterRepo 创建MongoDB的GreeterRepo实现
// 注意：这是一个MongoDB的扩展实现，不完全符合原始接口
// 如果需要完全兼容，建议创建新的接口
func NewMongoGreeterRepo(data *Data, logger log.Logger) *mongoGreeterRepo {
	return &mongoGreeterRepo{
		data:   data,
		logger: logger,
	}
}

// Save 保存Greeter到MongoDB
func (r *mongoGreeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	helper := log.NewHelper(r.logger)

	collection := r.data.mongodb.Collection("greeters")

	doc := &MongoGreeter{
		Hello:     g.Hello,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	result, err := collection.InsertOne(ctx, doc)
	if err != nil {
		helper.Errorf("failed to insert greeter: %v", err)
		return nil, err
	}

	helper.Infof("greeter saved with ID: %s", result.InsertedID)

	// 返回保存后的对象
	return g, nil
}

// Update 更新Greeter
func (r *mongoGreeterRepo) Update(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	helper := log.NewHelper(r.logger)

	collection := r.data.mongodb.Collection("greeters")

	// 这里简化处理，实际应用中可能需要其他方式来标识文档
	update := bson.M{
		"$set": bson.M{
			"hello":      g.Hello,
			"updated_at": time.Now(),
		},
	}

	// 根据hello字段更新（示例）
	result, err := collection.UpdateOne(ctx, bson.M{"hello": g.Hello}, update)
	if err != nil {
		helper.Errorf("failed to update greeter: %v", err)
		return nil, err
	}

	if result.MatchedCount == 0 {
		helper.Errorf("greeter not found with hello: %s", g.Hello)
		return nil, mongo.ErrNoDocuments
	}

	helper.Infof("greeter updated with hello: %s", g.Hello)
	return g, nil
}

// FindByID 根据ID查找Greeter
func (r *mongoGreeterRepo) FindByID(ctx context.Context, id int64) (*biz.Greeter, error) {
	helper := log.NewHelper(r.logger)

	collection := r.data.mongodb.Collection("greeters")

	// 这里简化处理，使用数字ID作为查询条件
	var doc MongoGreeter
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			helper.Warnf("greeter not found with ID: %d", id)
			return nil, nil
		}
		helper.Errorf("failed to find greeter: %v", err)
		return nil, err
	}

	return &biz.Greeter{
		Hello: doc.Hello,
	}, nil
}

// ListByHello 根据Hello字段查找Greeter列表
func (r *mongoGreeterRepo) ListByHello(ctx context.Context, hello string) ([]*biz.Greeter, error) {
	helper := log.NewHelper(r.logger)

	collection := r.data.mongodb.Collection("greeters")

	// 按创建时间倒序排列
	opts := options.Find().SetSort(bson.D{{"created_at", -1}})
	cursor, err := collection.Find(ctx, bson.M{"hello": hello}, opts)
	if err != nil {
		helper.Errorf("failed to find greeters by hello: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var greeters []*biz.Greeter
	for cursor.Next(ctx) {
		var doc MongoGreeter
		if err := cursor.Decode(&doc); err != nil {
			helper.Errorf("failed to decode greeter: %v", err)
			continue
		}

		greeters = append(greeters, &biz.Greeter{
			Hello: doc.Hello,
		})
	}

	if err := cursor.Err(); err != nil {
		helper.Errorf("cursor error: %v", err)
		return nil, err
	}

	helper.Infof("found %d greeters with hello: %s", len(greeters), hello)
	return greeters, nil
}

// ListAll 列出所有Greeter
func (r *mongoGreeterRepo) ListAll(ctx context.Context) ([]*biz.Greeter, error) {
	helper := log.NewHelper(r.logger)

	collection := r.data.mongodb.Collection("greeters")

	// 按创建时间倒序排列
	opts := options.Find().SetSort(bson.D{{"created_at", -1}})
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		helper.Errorf("failed to find greeters: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var greeters []*biz.Greeter
	for cursor.Next(ctx) {
		var doc MongoGreeter
		if err := cursor.Decode(&doc); err != nil {
			helper.Errorf("failed to decode greeter: %v", err)
			continue
		}

		greeters = append(greeters, &biz.Greeter{
			Hello: doc.Hello,
		})
	}

	if err := cursor.Err(); err != nil {
		helper.Errorf("cursor error: %v", err)
		return nil, err
	}

	helper.Infof("found %d greeters", len(greeters))
	return greeters, nil
}

// Delete 删除Greeter
func (r *mongoGreeterRepo) Delete(ctx context.Context, id int64) error {
	helper := log.NewHelper(r.logger)

	collection := r.data.mongodb.Collection("greeters")

	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		helper.Errorf("failed to delete greeter: %v", err)
		return err
	}

	if result.DeletedCount == 0 {
		helper.Warnf("greeter not found with ID: %d", id)
		return mongo.ErrNoDocuments
	}

	helper.Infof("greeter deleted with ID: %d", id)
	return nil
}

// Count 统计Greeter数量
func (r *mongoGreeterRepo) Count(ctx context.Context) (int64, error) {
	helper := log.NewHelper(r.logger)

	collection := r.data.mongodb.Collection("greeters")

	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		helper.Errorf("failed to count greeters: %v", err)
		return 0, err
	}

	helper.Infof("total greeters count: %d", count)
	return count, nil
}