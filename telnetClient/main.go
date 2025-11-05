package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	var (
		hostFlag    = flag.String("host", "", "Адрес хоста для подключения (обязательный)")
		portFlag    = flag.String("port", "", "Порт для подключения (обязательный)")
		timeoutFlag = flag.Duration("timeout", 10*time.Second, "Таймаут подключения (по умолчанию 10s)")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Простой telnet-клиент для подключения к TCP-серверу\n\n")
		fmt.Fprintf(os.Stderr, "Опции:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nПримеры:\n")
		fmt.Fprintf(os.Stderr, "  %s -host localhost -port 8080\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -host example.com -port 25 -timeout 5s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -host echo.tcpbin.com -port 4242\n", os.Args[0])
	}
	flag.Parse()

	if *hostFlag == "" || *portFlag == "" {
		fmt.Fprintf(os.Stderr, "Ошибка: необходимо указать host и port\n\n")
		flag.Usage()
		os.Exit(1)
	}

	address := net.JoinHostPort(*hostFlag, *portFlag)
	fmt.Fprintf(os.Stderr, "Подключение к %s (таймаут: %v)...\n", address, *timeoutFlag)

	// Устанавливаем TCP соединение с таймаутом
	conn, err := net.DialTimeout("tcp", address, *timeoutFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка подключения: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Fprintf(os.Stderr, "Подключено к %s\n", address)
	fmt.Fprintf(os.Stderr, "Для завершения нажмите Ctrl+D\n\n")

	// Канал для сигнала завершения
	done := make(chan struct{})
	var once sync.Once
	closeDone := func() {
		once.Do(func() {
			close(done)
		})
	}

	var wg sync.WaitGroup

	// Обработка сигналов (Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(os.Stderr, "\nПолучен сигнал завершения, закрываю соединение...\n")
		conn.Close()
		closeDone()
	}()

	// Горутина для чтения из сокета и вывода в STDOUT
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(os.Stdout, conn)
		if err != nil && err != io.EOF {
			// Выводим ошибку только если это не EOF
			fmt.Fprintf(os.Stderr, "\nОшибка чтения из сокета: %v\n", err)
		}
		// Если чтение завершилось, значит сервер закрыл соединение
		fmt.Fprintf(os.Stderr, "\nСервер закрыл соединение\n")
		closeDone()
	}()

	// Горутина для чтения из STDIN и отправки в сокет
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Используем io.Copy для передачи данных из STDIN в сокет
		// io.Copy завершится при EOF (Ctrl+D) или при ошибке
		_, err := io.Copy(conn, os.Stdin)
		if err != nil {
			// Если это не EOF, выводим ошибку
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Ошибка отправки данных: %v\n", err)
			}
		}
		// Закрываем соединение на запись (shutdown write) для корректного завершения
		// Это позволяет серверу узнать, что мы больше не будем отправлять данные
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		}
		// Если io.Copy завершился без ошибки или с EOF, значит получили Ctrl+D
		if err == nil || err == io.EOF {
			fmt.Fprintf(os.Stderr, "\nПолучен EOF (Ctrl+D), закрываю соединение...\n")
		}
		closeDone()
	}()

	// Ждем завершения одной из горутин
	<-done

	// Принудительно закрываем соединение
	conn.Close()

	// Ждем завершения всех горутин
	wg.Wait()

	fmt.Fprintf(os.Stderr, "Соединение закрыто\n")
}
