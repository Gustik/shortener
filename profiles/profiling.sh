#!/bin/bash
set -e

PROFILE_NAME="${1:-base}"
PROFILE_FILE="profiles/${PROFILE_NAME}.pprof"
APP_PORT=8080
PPROF_PORT=6060

echo "=== Профилирование: ${PROFILE_NAME} ==="

# Сборка
echo "Сборка бинарника..."
go build -o shortener ./cmd/shortener

# Запуск сервера с включённым pprof
echo "Запуск сервера..."
PPROF_ENABLED=true ./shortener &
SERVER_PID=$!

cleanup() {
    echo "Останавливаю сервер (PID: ${SERVER_PID})..."
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
    rm -f shortener
}
trap cleanup EXIT

# Ждём готовности
echo "Жду готовности сервера..."
for i in $(seq 1 30); do
    if curl -s "http://localhost:${APP_PORT}/ping" > /dev/null 2>&1; then
        echo "Сервер готов"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "Сервер не запустился за 30 секунд"
        exit 1
    fi
    sleep 1
done

# Нагрузка через wrk с уникальными URL
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
echo "Генерирую нагрузку (wrk, 30s, 10 потоков, 10 соединений)..."
wrk -t10 -c10 -d30s -s "${SCRIPT_DIR}/wrk_post.lua" "http://localhost:${APP_PORT}/"
echo "Нагрузка завершена"

# Снятие профиля
echo "Снимаю heap-профиль..."
curl -s -o "${PROFILE_FILE}" "http://localhost:${PPROF_PORT}/debug/pprof/heap"
echo "Профиль сохранён в ${PROFILE_FILE}"

echo "=== Готово ==="
