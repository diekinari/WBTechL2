package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	var (
		portFlag = flag.String("port", "8080", "Порт для прослушивания (по умолчанию 8080)")
		modeFlag = flag.String("mode", "echo", "Режим работы: echo (эхо), smtp (SMTP), custom (кастомный)")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Использование: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Простой TCP сервер для тестирования telnet-клиента\n\n")
		fmt.Fprintf(os.Stderr, "Опции:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nРежимы работы:\n")
		fmt.Fprintf(os.Stderr, "  echo - простой echo-сервер (отправляет обратно все полученные данные)\n")
		fmt.Fprintf(os.Stderr, "  smtp - имитация SMTP сервера с базовыми командами\n")
		fmt.Fprintf(os.Stderr, "  custom - кастомный режим с приветствием и командами\n\n")
		fmt.Fprintf(os.Stderr, "Примеры:\n")
		fmt.Fprintf(os.Stderr, "  %s -port 8080 -mode echo\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -port 25 -mode smtp\n", os.Args[0])
	}
	flag.Parse()

	address := ":" + *portFlag
	fmt.Fprintf(os.Stderr, "Запуск TCP сервера на %s в режиме '%s'\n", address, *modeFlag)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка запуска сервера: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Fprintf(os.Stderr, "Сервер запущен и ожидает подключений на %s\n", address)
	fmt.Fprintf(os.Stderr, "Для остановки нажмите Ctrl+C\n\n")

	// Обработка сигналов для корректного завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(os.Stderr, "\nПолучен сигнал завершения, останавливаю сервер...\n")
		listener.Close()
	}()

	// Принимаем подключения
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Проверяем, не закрыт ли listener
			if _, ok := err.(net.Error); ok {
				continue
			}
			fmt.Fprintf(os.Stderr, "Ошибка принятия подключения: %v\n", err)
			break
		}

		// Обрабатываем каждое подключение в отдельной горутине
		go handleConnection(conn, *modeFlag)
	}
}

func handleConnection(conn net.Conn, mode string) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	fmt.Fprintf(os.Stderr, "[%s] Новое подключение от %s\n", time.Now().Format("15:04:05"), remoteAddr)

	switch mode {
	case "echo":
		handleEcho(conn)
	case "smtp":
		handleSMTP(conn)
	case "custom":
		handleCustom(conn)
	default:
		handleEcho(conn)
	}

	fmt.Fprintf(os.Stderr, "[%s] Соединение с %s закрыто\n", time.Now().Format("15:04:05"), remoteAddr)
}

// handleEcho - простой echo-сервер, отправляет обратно все полученные данные
func handleEcho(conn net.Conn) {
	// Отправляем приветствие
	fmt.Fprintf(conn, "Echo server ready. Type 'quit' to disconnect.\r\n")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Проверяем команду выхода
		if strings.ToLower(line) == "quit" || strings.ToLower(line) == "exit" {
			fmt.Fprintf(conn, "Goodbye!\r\n")
			break
		}

		// Отправляем обратно полученную строку
		fmt.Fprintf(conn, "Echo: %s\r\n", line)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Ошибка чтения: %v\n", err)
	}
}

// handleSMTP - имитация SMTP сервера
func handleSMTP(conn net.Conn) {
	// Отправляем приветствие SMTP
	fmt.Fprintf(conn, "220 localhost ESMTP Server ready\r\n")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		command := strings.ToUpper(line)

		// Обработка SMTP команд
		switch {
		case strings.HasPrefix(command, "HELO") || strings.HasPrefix(command, "EHLO"):
			fmt.Fprintf(conn, "250 localhost Hello\r\n")
		case command == "MAIL FROM:" || strings.HasPrefix(command, "MAIL FROM:"):
			fmt.Fprintf(conn, "250 OK\r\n")
		case command == "RCPT TO:" || strings.HasPrefix(command, "RCPT TO:"):
			fmt.Fprintf(conn, "250 OK\r\n")
		case command == "DATA":
			fmt.Fprintf(conn, "354 End data with <CR><LF>.<CR><LF>\r\n")
			// Читаем данные до точки на отдельной строке
			for scanner.Scan() {
				dataLine := scanner.Text()
				if strings.TrimSpace(dataLine) == "." {
					break
				}
			}
			fmt.Fprintf(conn, "250 OK: queued\r\n")
		case command == "QUIT":
			fmt.Fprintf(conn, "221 Bye\r\n")

		case command == "NOOP":
			fmt.Fprintf(conn, "250 OK\r\n")
		case command == "RSET":
			fmt.Fprintf(conn, "250 OK\r\n")
		case command == "VRFY":
			fmt.Fprintf(conn, "252 Cannot verify user\r\n")
		case command == "EXPN":
			fmt.Fprintf(conn, "550 Access denied\r\n")
		default:
			fmt.Fprintf(conn, "500 Command not recognized\r\n")
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Ошибка чтения: %v\n", err)
	}
}

// handleCustom - кастомный режим с командами
func handleCustom(conn net.Conn) {
	// Отправляем приветствие
	fmt.Fprintf(conn, "Welcome to Custom TCP Server!\r\n")
	fmt.Fprintf(conn, "Available commands: help, time, echo <text>, quit\r\n")
	fmt.Fprintf(conn, "> ")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)

		if len(parts) == 0 {
			fmt.Fprintf(conn, "> ")
			continue
		}

		command := strings.ToLower(parts[0])

		switch command {
		case "help":
			fmt.Fprintf(conn, "Available commands:\r\n")
			fmt.Fprintf(conn, "  help - show this help message\r\n")
			fmt.Fprintf(conn, "  time - show current server time\r\n")
			fmt.Fprintf(conn, "  echo <text> - echo back the text\r\n")
			fmt.Fprintf(conn, "  quit - disconnect\r\n")
			fmt.Fprintf(conn, "> ")
		case "time":
			fmt.Fprintf(conn, "Server time: %s\r\n", time.Now().Format("2006-01-02 15:04:05"))
			fmt.Fprintf(conn, "> ")
		case "echo":
			if len(parts) > 1 {
				echoText := strings.Join(parts[1:], " ")
				fmt.Fprintf(conn, "Echo: %s\r\n", echoText)
			} else {
				fmt.Fprintf(conn, "Usage: echo <text>\r\n")
			}
			fmt.Fprintf(conn, "> ")
		case "quit", "exit":
			fmt.Fprintf(conn, "Goodbye!\r\n")
			return
		default:
			fmt.Fprintf(conn, "Unknown command: %s. Type 'help' for available commands.\r\n", parts[0])
			fmt.Fprintf(conn, "> ")
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Ошибка чтения: %v\n", err)
	}
}
