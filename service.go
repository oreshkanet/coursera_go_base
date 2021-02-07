package main

// protoc --go_out=plugins=grpc:. *.proto

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// StartMyMicroservice - Старт микросервиса
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	// Инициализируем экземпляр сервера
	var srv = &serverGRPC{
		listenAddr:     listenAddr + ":8082",
		ACL:            make(map[string][]string, 0),
		logChans:       make([]chan *Event, 0, 0),
		statByMethod:   make(map[string]uint64),
		statByConsumer: make(map[string]uint64),
	}

	// Парсим ACL для проверки авторизации
	err := json.Unmarshal([]byte(ACLData), &srv.ACL)
	if err != nil {
		return err
	}

	// Запускаем прослушивание порта
	lis, err := net.Listen("tcp", ":8082")
	if err != nil {
		return err
	}

	// В отдельной горутине запускаем чтение из контекста признака закрытия сервера
	go func() {
		<-ctx.Done()
		// fmt.Println("stop server at :8082")

		// Закрываем все каналы, чтобы не было протечек
		for _, logChan := range srv.logChans {
			close(logChan)
		}

		// Закрываем порт
		lis.Close()
	}()

	// Создаем новый gRPC сервер
	server := grpc.NewServer(
		grpc.UnaryInterceptor(srv.unaryInterceptor),
		grpc.StreamInterceptor(srv.streamInterceptor),
	)
	RegisterBizServer(server, newBizManager(srv))
	RegisterAdminServer(server, newAdminManager(srv))

	// Запускаем прослушивание порта
	go server.Serve(lis)
	// fmt.Println("starting server at :8082")

	return nil
}

type serverGRPC struct {
	mu             sync.RWMutex
	listenAddr     string
	ACL            map[string][]string
	logChans       []chan *Event
	statByMethod   map[string]uint64
	statByConsumer map[string]uint64
}

func (s *serverGRPC) addLoggerChannel(newlogChan chan *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logChans = append(s.logChans, newlogChan)
}

func (s *serverGRPC) addLoggerEvent(event *Event) {
	for _, logChan := range s.logChans {
		logChan <- event
	}
}

func (s *serverGRPC) addStatByMethod(method string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, isExists := s.statByMethod[method]; !isExists {
		s.statByMethod[method] = 1
	} else {
		s.statByMethod[method]++
	}
}

func (s *serverGRPC) addStatByConsumer(consumer string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, isExists := s.statByConsumer[consumer]; !isExists {
		s.statByConsumer[consumer] = 1
	} else {
		s.statByConsumer[consumer]++
	}
}

func (s *serverGRPC) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {

	err := s.authLogStatistics(ctx, info.FullMethod)
	if err != nil {
		return nil, err
	}

	reply, err := handler(ctx, req)

	return reply, err
}

func (s *serverGRPC) streamInterceptor(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {

	ctx := stream.Context()

	err := s.authLogStatistics(ctx, info.FullMethod)
	if err != nil {
		return err
	}

	err = handler(srv, stream)

	return err
}

func (s *serverGRPC) authLogStatistics(
	ctx context.Context,
	method string,
) error {
	// Получаем метадату из контекста
	md, _ := metadata.FromIncomingContext(ctx)

	// Парсим консьмера
	var consumer string = "unknown"
	consumers := md["consumer"]
	if len(consumers) > 0 {
		consumer = consumers[0]
	}

	// Проверяем права доступа консьмера на метод
	methodEnabled := false
	if cons, consExist := s.ACL[consumer]; consExist {
		for _, pattern := range cons {
			if v, _ := regexp.MatchString(pattern, method); v {
				methodEnabled = true
				break
			}
		}
	}
	if !methodEnabled {
		return status.Errorf(codes.Unauthenticated, "Unauthenticated error")
	}

	// Добавляем в лог событие
	s.addLoggerEvent(&Event{
		Timestamp: time.Now().UnixNano(),
		Host:      s.listenAddr,
		Consumer:  consumer,
		Method:    method,
	})

	// Добавляем статистику
	s.addStatByConsumer(consumer)
	s.addStatByMethod(method)

	return nil
}

/***************************************************************************
* Admin Manager
***************************************************************************/

type adminManager struct {
	srv *serverGRPC
}

func (adm *adminManager) Logging(inStream *Nothing, srv Admin_LoggingServer) error {
	// Создаем новый канал, в который будут логироваться события
	logChannel := make(chan *Event, 0)
	// Добавляем канал в общий слайс каналов текущего сервера
	adm.srv.addLoggerChannel(logChannel)

	// В цикле слушаем канал, а заодно и проверяем контекст на закрытие
	for {
		select {
		case <-srv.Context().Done():
			// fmt.Println(srv.Context().Err().Error())
			return nil
		case s := <-logChannel:
			err := srv.SendMsg(s)
			if err != nil {
				fmt.Println(err.Error())
				return nil
			}
		}
	}
}

func (adm *adminManager) Statistics(statInterval *StatInterval, srv Admin_StatisticsServer) error {
	statChannel := time.NewTicker(time.Duration(statInterval.IntervalSeconds) * time.Second)

	for {
		select {
		case <-statChannel.C:
			err := srv.Send(&Stat{
				ByMethod:   adm.srv.statByMethod,
				ByConsumer: adm.srv.statByConsumer,
			})
			if err != nil {
				return nil
			}

			return nil
		}
	}

	//}()
	return nil
}

func newAdminManager(srv *serverGRPC) *adminManager {
	return &adminManager{
		srv: srv,
	}
}

/***************************************************************************
* Admin Manager
***************************************************************************/

type bizManager struct {
	srv *serverGRPC
}

func (*bizManager) Check(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return nothing, nil
}

func (*bizManager) Add(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return nothing, nil
}

func (*bizManager) Test(ctx context.Context, nothing *Nothing) (*Nothing, error) {
	return nothing, nil
}

func newBizManager(srv *serverGRPC) *bizManager {
	return &bizManager{
		srv: srv,
	}
}
