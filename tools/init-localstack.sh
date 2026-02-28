#!/bin/bash
set -e

# Configuración
BUCKET_NAME="terraform-state"
TABLE_NAME="terraform-lock"
ENDPOINT="http://localhost:4566"
REGION="us-east-1"
PROFILE="default" # Usa perfil por defecto o configura uno dummy

echo "🔍 Verificando recursos de Backend en LocalStack..."

# 1. Crear Bucket S3 si no existe
if aws --endpoint-url=$ENDPOINT s3 ls "s3://$BUCKET_NAME" 2>&1 | grep -q 'NoSuchBucket'; then
    echo "📦 Creando bucket S3: $BUCKET_NAME..."
    aws --endpoint-url=$ENDPOINT s3 mb "s3://$BUCKET_NAME" --region $REGION
else
    echo "✅ Bucket S3 ya existe: $BUCKET_NAME"
fi

# 2. Crear Tabla DynamoDB si no existe
if aws --endpoint-url=$ENDPOINT dynamodb describe-table --table-name $TABLE_NAME --region $REGION 2>&1 | grep -q 'ResourceNotFoundException'; then
    echo "🔒 Creando tabla DynamoDB: $TABLE_NAME..."
    aws --endpoint-url=$ENDPOINT dynamodb create-table \
        --table-name $TABLE_NAME \
        --attribute-definitions AttributeName=LockID,AttributeType=S \
        --key-schema AttributeName=LockID,KeyType=HASH \
        --provisioned-throughput ReadCapacityUnits=1,WriteCapacityUnits=1 \
        --region $REGION
else
    echo "✅ Tabla DynamoDB ya existe: $TABLE_NAME"
fi

echo "🚀 Backend LocalStack listo."
