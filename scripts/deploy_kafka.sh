#!/bin/bash
# 0Chain Kafka Deployment with SASL Authentication
#
# CRITICAL: Both the sharder and 0box HARDCODE SASL.Enable=true in their Kafka
# producer/consumer code. Therefore Kafka MUST be configured with SASL_PLAINTEXT.
#
# Source references:
#   Sharder: 0chain.net/smartcontract/dbs/queueProvider/kafka.go:40
#   0box:    0box/code/zboxcore/entity/kafka.go:239
#
# Config keys that MUST be set in the respective services:
#   Sharder (0chain.yaml):
#     server_chain.kafka.username: admin
#     server_chain.kafka.password: admin-secret
#   0box (0box.yaml):
#     kafka.username: admin
#     kafka.password: admin-secret
#
# Usage:
#   ./deploy_kafka.sh                    # Deploy with defaults
#   ./deploy_kafka.sh --base-dir ~/Code  # Custom base directory
#   ./deploy_kafka.sh --verify           # Just verify Kafka is running
#
# Prerequisites:
#   - Docker and docker compose
#   - testnet0 Docker network exists
#   - Sharder and 0box configs have matching SASL credentials

set -e

BASE_DIR="${HOME}/Code"
KAFKA_USERNAME="admin"
KAFKA_PASSWORD="admin-secret"
KAFKA_IP="198.19.0.99"
KAFKA_PORT="9092"
VERIFY_ONLY=false

# Parse args
while [[ $# -gt 0 ]]; do
    case "$1" in
        --base-dir) BASE_DIR="$2"; shift 2 ;;
        --username) KAFKA_USERNAME="$2"; shift 2 ;;
        --password) KAFKA_PASSWORD="$2"; shift 2 ;;
        --verify)   VERIFY_ONLY=true; shift ;;
        *)          echo "Unknown arg: $1"; exit 1 ;;
    esac
done

KAFKA_DIR="${BASE_DIR}/0chain/docker.local/build.kafka"
KAFKA_CONFIG_DIR="${KAFKA_DIR}/config"
KAFKA_DATA_DIR="${BASE_DIR}/0chain/docker.local/kafka"

# Verify-only mode
if [ "$VERIFY_ONLY" = "true" ]; then
    echo "Checking Kafka status..."
    if docker ps --format '{{.Names}}' | grep -q "kafka"; then
        echo "Kafka container: RUNNING"
        # Try to list topics
        CONTAINER=$(docker ps --format '{{.Names}}' | grep kafka | head -1)
        docker exec "$CONTAINER" kafka-topics.sh --bootstrap-server localhost:${KAFKA_PORT} --list 2>/dev/null && {
            echo "Kafka broker: RESPONSIVE"
        } || echo "Kafka broker: NOT RESPONDING (may still be starting)"
    else
        echo "Kafka container: NOT RUNNING"
        exit 1
    fi
    exit 0
fi

echo "========================================="
echo "  Deploying Kafka with SASL Authentication"
echo "========================================="
echo ""
echo "Config:"
echo "  Base dir:  ${BASE_DIR}"
echo "  Kafka IP:  ${KAFKA_IP}:${KAFKA_PORT}"
echo "  Username:  ${KAFKA_USERNAME}"
echo "  Network:   testnet0"
echo ""

# Verify testnet0 network exists
if ! docker network inspect testnet0 &>/dev/null; then
    echo "ERROR: Docker network 'testnet0' does not exist."
    echo "Start the chain first (miners/sharders create this network)."
    exit 1
fi

# Create directories
mkdir -p "${KAFKA_DIR}" "${KAFKA_CONFIG_DIR}" "${KAFKA_DATA_DIR}"

# Step 1: Create JAAS configuration
echo "[1/4] Creating JAAS configuration..."
cat > "${KAFKA_CONFIG_DIR}/kafka_server_jaas.conf" << EOF
KafkaServer {
    org.apache.kafka.common.security.plain.PlainLoginModule required
    username="${KAFKA_USERNAME}"
    password="${KAFKA_PASSWORD}"
    user_${KAFKA_USERNAME}="${KAFKA_PASSWORD}";
};
EOF
echo "  Written: ${KAFKA_CONFIG_DIR}/kafka_server_jaas.conf"

# Step 2: Create docker-compose.yml
echo "[2/4] Creating docker-compose.yml..."
cat > "${KAFKA_DIR}/docker-compose.yml" << EOF
version: '3'
services:
  kafka:
    image: 'bitnami/kafka:latest'
    networks:
      default:
      testnet0:
        ipv4_address: ${KAFKA_IP}
    volumes:
      - ../kafka:/bitnami/kafka
      - ./config/kafka_server_jaas.conf:/opt/bitnami/kafka/config/kafka_server_jaas.conf
    environment:
      - KAFKA_CFG_NODE_ID=0
      - KAFKA_CFG_PROCESS_ROLES=controller,broker
      - KAFKA_CFG_LISTENERS=SASL_PLAINTEXT://:${KAFKA_PORT},CONTROLLER://:9093
      - KAFKA_CFG_ADVERTISED_LISTENERS=SASL_PLAINTEXT://${KAFKA_IP}:${KAFKA_PORT}
      - KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,SASL_PLAINTEXT:SASL_PLAINTEXT
      - KAFKA_CFG_CONTROLLER_QUORUM_VOTERS=0@kafka:9093
      - KAFKA_CFG_CONTROLLER_LISTENER_NAMES=CONTROLLER
      - KAFKA_CFG_INTER_BROKER_LISTENER_NAME=SASL_PLAINTEXT
      - KAFKA_CFG_SASL_MECHANISM_INTER_BROKER_PROTOCOL=PLAIN
      - KAFKA_CFG_SASL_ENABLED_MECHANISMS=PLAIN
      - KAFKA_CFG_AUTO_CREATE_TOPICS_ENABLE=true
      - KAFKA_OPTS=-Djava.security.auth.login.config=/opt/bitnami/kafka/config/kafka_server_jaas.conf
      - ALLOW_PLAINTEXT_LISTENER=yes
    restart: unless-stopped
networks:
  default:
    driver: bridge
  testnet0:
    external: true
EOF
echo "  Written: ${KAFKA_DIR}/docker-compose.yml"

# Step 3: Start Kafka
echo "[3/4] Starting Kafka..."
cd "${KAFKA_DIR}"
docker compose up -d 2>/dev/null || docker-compose up -d 2>/dev/null || {
    echo "ERROR: Failed to start Kafka"
    exit 1
}

# Step 4: Wait and verify
echo "[4/4] Waiting for Kafka to start..."
MAX_WAIT=60
WAITED=0
while [ $WAITED -lt $MAX_WAIT ]; do
    CONTAINER=$(docker compose ps -q kafka 2>/dev/null || docker ps --filter "name=kafka" --format '{{.ID}}' | head -1)
    if [ -n "$CONTAINER" ]; then
        if docker exec "$CONTAINER" kafka-topics.sh --bootstrap-server localhost:${KAFKA_PORT} --list 2>/dev/null; then
            echo ""
            echo "Kafka is ready!"
            echo "  Broker:      ${KAFKA_IP}:${KAFKA_PORT}"
            echo "  Auth:        SASL_PLAINTEXT"
            echo "  Credentials: ${KAFKA_USERNAME} / ****"
            echo ""
            echo "Sharder config (0chain.yaml):"
            echo "  server_chain.kafka.username: ${KAFKA_USERNAME}"
            echo "  server_chain.kafka.password: (set from --password arg or default)"
            echo ""
            echo "0box config (0box.yaml):"
            echo "  kafka.username: ${KAFKA_USERNAME}"
            echo "  kafka.password: (set from --password arg or default)"
            exit 0
        fi
    fi
    sleep 2
    WAITED=$((WAITED + 2))
    echo -n "."
done

echo ""
if docker ps --format '{{.Names}}' | grep -q "kafka"; then
    echo "Kafka container is running but not yet responsive."
    echo "It may need more time to initialize. Check logs:"
    echo "  docker compose -f ${KAFKA_DIR}/docker-compose.yml logs -f kafka"
else
    echo "ERROR: Kafka container failed to start."
    echo "Check logs: docker compose -f ${KAFKA_DIR}/docker-compose.yml logs kafka"
    exit 1
fi
