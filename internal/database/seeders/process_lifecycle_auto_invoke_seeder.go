package seeders

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

func ProcessLifecycleAutoInvokeSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "process_lifecycle_auto_invoke")
	logger.Info("Starting Process Lifecycle Auto Invoke Seeder")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		invalidateResolutionCache := func(processTypeID int64) {
			redisHost := os.Getenv("REDIS_HOST")
			redisPort := os.Getenv("REDIS_PORT")
			if redisHost == "" || redisPort == "" {
				return
			}

			dbStr := os.Getenv("REDIS_DATABASE")
			if dbStr == "" {
				dbStr = "0"
			}
			dbNum, err := strconv.Atoi(dbStr)
			if err != nil {
				return
			}

			password := os.Getenv("REDIS_PASSWORD")
			addr := fmt.Sprintf("%s:%s", redisHost, redisPort)
			client := redis.NewClient(&redis.Options{
				Addr:     addr,
				Password: password,
				DB:       dbNum,
			})
			defer func() {
				_ = client.Close()
			}()

			projectPrefix := os.Getenv("APP_NAME")
			if projectPrefix == "" {
				projectPrefix = "go-fiber-core"
			}

			redisCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
			defer cancel()

			pattern := fmt.Sprintf("%s:lifecycle:resolution:%d:*", projectPrefix, processTypeID)
			var cursor uint64
			for {
				keys, nextCursor, err := client.Scan(redisCtx, cursor, pattern, 100).Result()
				if err != nil {
					return
				}
				if len(keys) > 0 {
					_ = client.Del(redisCtx, keys...).Err()
				}
				cursor = nextCursor
				if cursor == 0 {
					break
				}
			}

			blockerKey := fmt.Sprintf("%s:lifecycle:block:%d", projectPrefix, processTypeID)
			_ = client.Del(redisCtx, blockerKey).Err()
		}

		ensureProcess := func(processTypeName, description, historyComment string) (int64, int64, error) {
			var processTypeID int64
			err := tx.QueryRow(ctx, "SELECT id FROM process_types WHERE name = $1", processTypeName).Scan(&processTypeID)
			if err != nil {
				if err == pgx.ErrNoRows {
					err = tx.QueryRow(ctx,
						`INSERT INTO process_types (name, description, is_visible)
						VALUES ($1, $2, $3)
						RETURNING id`,
						processTypeName,
						description,
						true,
					).Scan(&processTypeID)
					if err != nil {
						return 0, 0, fmt.Errorf("insert process_types '%s': %w", processTypeName, err)
					}
					logger.Info("Process Type Created", "name", processTypeName, "id", processTypeID)
				} else {
					return 0, 0, fmt.Errorf("select process_types '%s': %w", processTypeName, err)
				}
			} else {
				logger.Info("Process Type Existing", "name", processTypeName, "id", processTypeID)
			}

			var versionID int64
			err = tx.QueryRow(ctx,
				"SELECT id FROM process_versions WHERE process_type_id = $1 AND version_number = 1 AND sede_id IS NULL",
				processTypeID,
			).Scan(&versionID)
			if err != nil {
				if err == pgx.ErrNoRows {
					err = tx.QueryRow(ctx,
						`INSERT INTO process_versions (process_type_id, version_number, status, operator_id, sede_id)
						VALUES ($1, $2, $3, $4, NULL)
						RETURNING id`,
						processTypeID,
						1,
						"PROD",
						1,
					).Scan(&versionID)
					if err != nil {
						return 0, 0, fmt.Errorf("insert process_versions '%s': %w", processTypeName, err)
					}
					logger.Info("Process Version Created", "name", processTypeName, "id", versionID)
				} else {
					return 0, 0, fmt.Errorf("select process_versions '%s': %w", processTypeName, err)
				}
			} else {
				logger.Info("Process Version Existing", "name", processTypeName, "id", versionID)
			}

			if err := EnsureHistory(ctx, tx, versionID, processTypeID, historyComment, logger); err != nil {
				return 0, 0, err
			}

			invalidateResolutionCache(processTypeID)
			return processTypeID, versionID, nil
		}

		seedOneStep := func(versionID int64, config, stepName string) error {
			executionKey := "test/test_auto_invoke"
			if err := UpsertStep(ctx, tx, versionID, 1, stepName, executionKey, config); err != nil {
				return fmt.Errorf("upsert step failed (%s): %w", stepName, err)
			}
			return nil
		}

		{
			const name = "Test Auto Invoke Process"
			typeID, versionID, err := ensureProcess(
				name,
				"Process to demonstrate auto-invoke/recursion capabilities",
				"Initial auto-invoke process seed",
			)
			if err != nil {
				return err
			}

			_ = typeID
			if err := seedOneStep(versionID, `{"autoInvoke": true}`, "Auto Invoke Step"); err != nil {
				return err
			}
		}

		{
			const name = "Test Auto Invoke Process + async"
			typeID, versionID, err := ensureProcess(
				name,
				"Process to demonstrate auto-invoke using ASYNC dispatch",
				"Initial auto-invoke async process seed",
			)
			if err != nil {
				return err
			}

			_ = typeID
			config := `{
				"autoInvoke": true,
				"execution_policy": {
					"mode": "ASYNC",
					"label": "Test Auto Invoke Process + async",
					"auto_invoke": {
						"enabled": true,
						"cursor_field": "last_id_processed",
						"stop_condition": "is_last_batch"
					}
				}
			}`
			if err := seedOneStep(versionID, config, "Auto Invoke Step (ASYNC)"); err != nil {
				return err
			}
		}

		{
			const name = "Test Auto Invoke Process + async + finalized"
			typeID, versionID, err := ensureProcess(
				name,
				"Process to demonstrate auto-invoke using ASYNC dispatch + finalization step",
				"Initial auto-invoke async finalized process seed",
			)
			if err != nil {
				return err
			}

			_ = typeID
			config := `{
				"autoInvoke": true,
				"execution_policy": {
					"mode": "ASYNC",
					"label": "Test Auto Invoke Process + async + finalized",
					"auto_invoke": {
						"enabled": true,
						"cursor_field": "last_id_processed",
						"stop_condition": "is_last_batch"
					},
					"next_step": "test/test_auto_invoke_finalize"
				}
			}`
			if err := seedOneStep(versionID, config, "Auto Invoke Step (ASYNC + FINALIZED)"); err != nil {
				return err
			}
		}

		logger.Info("Seeder Completed Successfully")
		return nil
	})
}
