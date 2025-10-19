package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	generated "github.com/silencefog/telemetry/sensor/generated"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/joho/godotenv"
)

type TemperatureGenerator struct {
	currentTemp float64
}

// Создаёт новый генератор с начальным значением
func NewTemperatureGenerator(initialTemp float64) *TemperatureGenerator {
	return &TemperatureGenerator{currentTemp: initialTemp}
}

// Генерирует следующее значение температуры в пределах ±0.1 от предыдушего
func (tg *TemperatureGenerator) Next() float64 {
	change := (rand.Float64()*0.2 - 0.1)
	tg.currentTemp += change
	return tg.currentTemp
}

func OnStartLoad() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf(".env файл не найден")
	}
}

func ConnectToServer() *grpc.ClientConn {
	serverAddress := os.Getenv("SERVER_ADDRESS")
	// В продакшене можно заменить на шифрованное соединение
	conn, err := grpc.NewClient(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Не удалось подключиться: %v", err)
	}
	return conn
}

func main() {
	OnStartLoad()

	conn := ConnectToServer()
	defer conn.Close()

	// Создание клиента (виртуального сенсора)
	sensor := generated.NewTemperatureServiceClient(conn)

	rand.New(rand.NewSource((time.Now().UnixNano())))

	// Создаём генератор температуры с начальным значением
	tempGen := NewTemperatureGenerator(20.0)

	// Создание контекста с токеном
	authToken := os.Getenv("AUTH_TOKEN")
	md := metadata.Pairs("authorization", "Bearer "+authToken)
	baseCtx := metadata.NewOutgoingContext(context.Background(), md)

	// Цикл для генерации температур каждую секунду
	for {
		temperature := tempGen.Next()

		// Формирование пакета данных для отправки
		req := &generated.TemperatureRequest{
			Value:     temperature,
			Timestamp: timestamppb.New(time.Now()),
		}

		// Отправка данных с контекстом с таймаутом
		ctx, cancel := context.WithTimeout(baseCtx, time.Second)
		_, err := sensor.PushTemperature(ctx, req)
		cancel()

		if err != nil {
			log.Printf("Ошибка отправки: %v", err)
		} else {
			log.Printf("Температура отправлена: %2f C", temperature)
		}

		time.Sleep(1 + time.Second)
	}
}
