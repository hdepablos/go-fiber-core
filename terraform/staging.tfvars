environment = "staging"
aws_region  = "us-east-1"

# Nuevo mapa de variables para tus Lambdas
lambda_env_vars = {
	APP_ENV=local
	PROJECT_SLUG=go-fiber-core
	APP_NAME="${PROJECT_SLUG}"

	PORT=9009

	#########################################################
	### JWT Settings
	JWT_ACCESS_SECRET="tu_super_secreto_para_access_tokens"
	JWT_REFRESH_SECRET="tu_otro_super_secreto_para_refresh_tokens"
	JWT_ACCESS_TTL_MINUTES=15m
	JWT_REFRESH_TTL_DAYS=2160h
	#########################################################

	#########################################################
	### REDIS
	REDIS_HOST=redis
	REDIS_PORT=6379
	REDIS_PASSWORD=eYVX7EwVmmxKPCDmwMtyKVge8oLd2t81
	REDIS_DATABASE=0
	REDIS_EXPIRES_IN_SECONDS=230
	POOL_SIZE=50
	#########################################################

	#########################################################
	# GORM DATABASE CONNECTIONS WRITE => (PRIMARY)
	#########################################################
	GORM_WRITE_DRIVER=postgres
	GORM_WRITE_HOST=postgres
	GORM_WRITE_PORT=5432
	GORM_WRITE_USER=gorm_writer
	GORM_WRITE_PASSWORD=gorm_write_secret
	GORM_WRITE_DBNAME=go_fiber_core
	GORM_WRITE_SCHEMA=public
	GORM_WRITE_MAX_OPEN_CONNS=15
	GORM_WRITE_MAX_IDLE_CONNS=10
	GORM_WRITE_CONN_MAX_LIFETIME_SECONDS=300

	#########################################################
	# GORM DATABASE CONNECTIONS READ => (PRIMARY)
	#########################################################
	GORM_READ_DRIVER=postgres
	GORM_READ_HOST=postgres
	GORM_READ_PORT=5432
	GORM_READ_USER=gorm_reader
	GORM_READ_PASSWORD=gorm_read_secret
	GORM_READ_DBNAME=go_fiber_core
	GORM_READ_SCHEMA=public
	GORM_READ_MAX_OPEN_CONNS=15
	GORM_READ_MAX_IDLE_CONNS=10
	GORM_READ_CONN_MAX_LIFETIME_SECONDS=300


	#########################################################
	# PGX DATABASE CONNECTIONS WRITE => (PRIMARY)
	#########################################################
	PGX_WRITE_HOST=postgres
	PGX_WRITE_PORT=5432
	PGX_WRITE_USER=gorm_writer
	PGX_WRITE_PASSWORD=gorm_write_secret
	PGX_WRITE_DBNAME=go_fiber_core
	PGX_WRITE_MAX_CONNS=15

	#########################################################
	# PGX DATABASE CONNECTIONS READ => (REPLICA)
	#########################################################
	PGX_READ_HOST=postgres
	PGX_READ_PORT=5432
	PGX_READ_USER=gorm_reader
	PGX_READ_PASSWORD=gorm_read_secret
	PGX_READ_DBNAME=go_fiber_core
	PGX_READ_MAX_CONNS=15


	#########################################################
	# DATABASE POSTGRES TEST
	#########################################################
	TEST_DATABASE_URL="host=postgres user=postgres password=postgres dbname=go_fiber_core_test port=5432 sslmode=disable search_path=public"

	#########################################################
	# DATABASE DRIVER
	#########################################################
	DB_ACTIVE_DRIVER=postgres
	#########################################################

	#########################################################
	### DATABASE MYSQL
	MYSQL_DRIVER=
	MYSQL_HOST=
	MYSQL_PORT=
	MYSQL_USERNAME=
	MYSQL_PASSWORD=
	MYSQL_DATABASE=
	MYSQL_MAX_OPEN_CONNS=1
	MYSQL_MAX_IDLE_CONNS=2
	MYSQL_MAX_CONN_LIFE_TIME_IN_SECONDS=3
	#########################################################

	#########################################################
	### AWS
	# FUNCTIONS=api,sqs-consumer,dlq-consumer,every-1min-cron,daily-24-cron
	FUNCTIONS=api,sqs-consumer,dlq-consumer,every-1min-cron,daily-24-cron
	AWS_PROFILE_NAME=localstack
	STACK_NAME=${PROJECT_SLUG}-stack
	AWS_DEFAULT_REGION=us-east-1
	LOCALSTACK_ENDPOINT_BASE=http://localhost:4566
	S3_PREFIX=sam
	S3_BUCKET_NAME="${PROJECT_SLUG}-bucket"
	# API_URL=http://localhost


	API_PORT=9009
	URL_BASE=http://localhost:4566/restapis/ip1umux4pz/Prod/_user_request_/
	AWS_ENDPOINT_ARG=

	AWS_ENDPOINT_URL=http://localhost:4566
	# actualizar esto
	# URL_BASE=http://localhost:4566/
	# URL_lambda=${URL_BASE}restapis/l8ygy1nmj9/Prod/_user_request_


	AWS_ACCESS_KEY_ID=test
	AWS_SECRET_ACCESS_KEY=test
	#########################################################

}
