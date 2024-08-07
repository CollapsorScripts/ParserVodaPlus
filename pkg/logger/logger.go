package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const Console = false

var mutex sync.Mutex
var fullPathToFile string

// getFuncName - получаем имя функции из которой вызван логгер
func getFuncName() string {
	pc, _, _, ok := runtime.Caller(2)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		parseDetails := strings.Split(details.Name(), "/")
		pkgFuncName := ""
		if len(parseDetails) == 2 {
			pkgFuncParse := strings.Split(parseDetails[1], ".")
			if len(pkgFuncParse) == 2 {
				pkgFuncName = fmt.Sprintf("[P]%s=>[F]%s", pkgFuncParse[0], pkgFuncParse[1])
			} else if len(pkgFuncParse) == 3 {
				pkgFuncName = fmt.Sprintf("[P]%s=>[S]%s=>[F]%s", pkgFuncParse[0], pkgFuncParse[1], pkgFuncParse[1])
			}
		} else if len(parseDetails) == 1 {
			pkgFuncParse := strings.Split(parseDetails[0], ".")
			if len(pkgFuncParse) == 2 {
				pkgFuncName = fmt.Sprintf("[P]%s=>[F]%s", pkgFuncParse[0], pkgFuncParse[1])
			} else if len(pkgFuncParse) == 3 {
				if pkgFuncParse[0] == pkgFuncParse[1] {
					pkgFuncName = fmt.Sprintf("[P]%s=>[F]%s=>[AF]: %s", pkgFuncParse[0], pkgFuncParse[1],
						pkgFuncParse[2])
				}
			}
		} else if len(parseDetails) > 2 {
			pkgFuncParse := strings.Split(parseDetails[len(parseDetails)-1], ".")
			if len(pkgFuncParse) == 2 {
				pkgFuncName = fmt.Sprintf("[P]%s=>[F]%s", pkgFuncParse[0], pkgFuncParse[1])
			} else if len(pkgFuncParse) == 3 {
				pkgFuncName = fmt.Sprintf("[P]%s=>[S]%s=>[F]%s", pkgFuncParse[0], pkgFuncParse[1],
					pkgFuncParse[2])
			}
		}

		return pkgFuncName
	}
	return "undefined"
}

func writeToLog(str string, a ...any) error {
	mutex.Lock()
	// Открываем файл с логом
	file, err := os.OpenFile(fullPathToFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}

	defer func() {
		mutex.Unlock()
		file.Close()
	}()

	// Дозаписываем данные в файл
	data := []byte(fmt.Sprintf(str+"\n", a...))
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	fmt.Printf(str+"\n", a...)

	return nil
}

// New - инициализация логгера, создание папки и файла с логом
func New() error {
	//Текущая директория
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	//Файл с логом
	fileName := "logs.log"

	//Папка куда будет складывать лог
	folderName := "log"

	//Полный путь к папке
	dirPath := filepath.Join(pwd, folderName)

	//Полный путь к файлу с логом
	fullPathToFile = filepath.Join(dirPath, fileName)

	// Создаем папку с правами доступа для текущего пользователя
	err = os.Mkdir(dirPath, 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}

	// Создаем файл с логом
	file, err := os.OpenFile(fullPathToFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	mutex = sync.Mutex{}

	return nil
}

// Info - лог с пометкой info
func Info(format string, a ...any) {
	funcName := fmt.Sprintf("%s ->", getFuncName())
	go func() {
		date := time.Now()
		dateFormat := fmt.Sprintf("[%s]", date.Format("02.01.2006 15.01"))
		str := fmt.Sprintf("%s %s [info]: %s", dateFormat, funcName, format)
		if Console {
			fmt.Printf(str, a...)
			fmt.Println()
		} else {
			err := writeToLog(str, a...)
			if err != nil {
				fmt.Printf("Ошибка при записи в лог: %v\n", err)
				return
			}
		}
	}()
}

// Error - лог с пометкой error
func Error(format string, a ...any) {
	funcName := fmt.Sprintf("%s ->", getFuncName())
	go func() {
		date := time.Now()
		dateFormat := fmt.Sprintf("[%s]", date.Format("02.01.2006 15.01"))
		str := fmt.Sprintf("%s %s [error]: %s", dateFormat, funcName, format)
		if Console {
			fmt.Printf(str, a...)
			fmt.Println()
		} else {
			err := writeToLog(str, a...)
			if err != nil {
				fmt.Printf("Ошибка при записи в лог: %v\n", err)
				return
			}
		}
	}()
}

// Warn - лог с пометкой warn
func Warn(format string, a ...any) {
	funcName := fmt.Sprintf("%s ->", getFuncName())
	go func() {
		date := time.Now()
		dateFormat := fmt.Sprintf("[%s]", date.Format("02.01.2006 15.01"))
		str := fmt.Sprintf("%s %s [warn]: %s", dateFormat, funcName, format)
		if Console {
			fmt.Printf(str, a...)
			fmt.Println()
		} else {
			err := writeToLog(str, a...)
			if err != nil {
				fmt.Printf("Ошибка при записи в лог: %v\n", err)
				return
			}
		}
	}()
}
