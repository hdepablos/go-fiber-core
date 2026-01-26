package catalog

import (
	"context"
	"encoding/json"
	"time"

	"go-fiber-core/internal/models"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type CatalogRepository interface {
	GetAll(ctx context.Context, db *gorm.DB) (models.AllCatalogsResponse, error)
	InvalidateCache(ctx context.Context) error
}

type catalogRepository struct {
	redis      *redis.Client
	cacheKey   string
	expiration time.Duration
}

func NewCatalogRepository(redisClient *redis.Client) CatalogRepository {
	return &catalogRepository{
		redis:      redisClient,
		cacheKey:   "catalogs:all",
		expiration: 24 * time.Hour, // TTL largo porque invalidaremos manualmente
	}
}

func (r *catalogRepository) GetAll(ctx context.Context, db *gorm.DB) (models.AllCatalogsResponse, error) {
	var response models.AllCatalogsResponse

	// 1. Intentar obtener de Redis
	cachedData, err := r.redis.Get(ctx, r.cacheKey).Result()
	if err == nil {
		// Si existe, deserializamos y retornamos
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			return response, nil
		}
	}

	// 2. Si no hay cache (Cache Miss), consultamos DB concurrentemente
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return db.WithContext(ctx).Table("banks").
			Select("id, name, enabled as is_active").
			Where("deleted_at IS NULL").Scan(&response.Banks).Error
	})

	g.Go(func() error {
		return db.WithContext(ctx).Table("roles").
			Select("id, name, is_active").
			Where("deleted_at IS NULL").Scan(&response.Roles).Error
	})

	if err := g.Wait(); err != nil {
		return models.AllCatalogsResponse{}, err
	}

	// 3. Guardar en Redis para la próxima vez
	data, _ := json.Marshal(response)
	r.redis.Set(ctx, r.cacheKey, data, r.expiration)

	return response, nil
}

func (r *catalogRepository) InvalidateCache(ctx context.Context) error {
	return r.redis.Del(ctx, r.cacheKey).Err()
}
