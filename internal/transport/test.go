import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	// Імпортуйте свій згенерований пакет, який ви вказали в option go_package
	pb "github.com/Noxie-Arbitrage/internal/transport" 
)

// 1. Створюємо структуру сервера
type server struct {
	pb.UnimplementedDataServiceServer // Обов'язкове вбудовування [4]
}

// 2. Реалізуємо метод SendUser [6, 7]
func (s *server) SendUser(ctx context.Context, req *pb.User) (*pb.Ack, error) {
	// Тут ви отримуєте доступ до ваших даних через req.UserData
	log.Printf("Отримано дані розміром: %d байт", len(req.UserData))

	// Повертаємо підтвердження Ack [8]
	return &pb.Ack{
		Status:  true,
		Details: "Дані успішно отримані сервером",
	}, nil
}

func main() {
	// 3. Відкриваємо TCP-порт для прослуховування [9]
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Не вдалося запустити слухача: %v", err)
	}

	// 4. Створюємо новий gRPC сервер [10]
	grpcServer := grpc.NewServer()

	// 5. Реєструємо наш сервіс на gRPC сервері [11, 12]
	pb.RegisterDataServiceServer(grpcServer, &server{})

	log.Println("Сервер запущено на порту :50051...")
	
	// 6. Запускаємо сервер [10, 13]
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Помилка сервера: %v", err)
	}
}
