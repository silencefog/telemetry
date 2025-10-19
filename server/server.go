package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/joho/godotenv"

	generated "github.com/silencefog/telemetry/server/generated"
)

type server struct {
	generated.UnimplementedTemperatureServiceServer

	subscribers []chan *generated.TemperatureResponse
	mu          sync.RWMutex
}

func (s *server) PushTemperature(ctx context.Context, req *generated.TemperatureRequest) (*emptypb.Empty, error) {

	if authenticated, err := Authenticate(ctx); !authenticated {
		return nil, err
	}

	log.Printf("Получены данные: %.2f C (время: %s)", req.Value, req.Timestamp.AsTime().Format(time.ANSIC))

	// Создание ответа для подписчиков
	response := &generated.TemperatureResponse{
		Value:     req.Value,
		Timestamp: req.Timestamp,
	}

	// Блокирование для записи
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Отправка данных всем подписчикам
	for _, sub := range s.subscribers {
		select {
		case sub <- response:
		default:
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *server) StreamTemperature(req *generated.StreamRequest, stream generated.TemperatureService_StreamTemperatureServer) error {
	// Создаётся канал для подпичсика
	ch := make(chan *generated.TemperatureResponse, 100)

	// Блокирование для записи и чтения
	s.mu.Lock()
	// Добавление канала в список подписчиков
	s.subscribers = append(s.subscribers, ch)
	s.mu.Unlock()

	// Гарантия удаления подписчика при отключении
	defer func() {
		s.mu.Lock()

		for i, sub := range s.subscribers {
			if sub == ch {
				s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
		close(ch)
	}()

	// Бесконечное чтение из канала и отправка в gRPC поток
	for response := range ch {
		if err := stream.Send(response); err != nil {
			return err
		}
	}

	return nil
}

func Authenticate(ctx context.Context) (bool, error) {
	authToken := os.Getenv("AUTH_TOKEN")

	// достаём заголовки из запроса
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false, status.Error(codes.Unauthenticated, "нет токена")
	}

	// ищем токен в заголовках
	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return false, status.Error(codes.Unauthenticated, "токен не указан")
	}

	// проверяем токен
	if tokens[0] != "Bearer "+authToken {
		return false, status.Error(codes.Unauthenticated, "неверный токен")
	}

	return true, nil
}

func OnStartLoad() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf(".env файл не найден")
	}
}

func main() {
	OnStartLoad()

	grpcServer := grpc.NewServer()

	generated.RegisterTemperatureServiceServer(grpcServer, &server{})

	listener, err := net.Listen("tcp", os.Getenv("SERVER_ADDRESS"))
	if err != nil {
		log.Fatalf("Не удалось слушать порт: %v", err)
	}

	log.Printf("Сервер запущен на %s", os.Getenv("SERVER_ADDRESS"))

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
	}
}
