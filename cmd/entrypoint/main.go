package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"
	"voda_parser/pkg/logger"
	"voda_parser/pkg/parser"
)

func main() {
	logger.New()

	chLog := make(chan string)

	go parser.StartParse(chLog)

	for log := range chLog {
		if log == "stop" {
			close(chLog)
			wait := time.Second * 15
			_, cancel := context.WithTimeout(context.Background(), wait)
			cancel()
			bufio.NewReader(os.Stdin).ReadString('\n') // Ожидание Enter
			os.Exit(0)
		}
		fmt.Println(log)
	}

	//Для правильного завершения приложения
	{
		wait := time.Second * 15
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		//Блокируем горутину до вызова сигнала Interrupt
		<-c

		_, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel()
		os.Exit(0)
	}
}
