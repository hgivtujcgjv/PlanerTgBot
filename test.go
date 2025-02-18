package main

import (
	//"encoding/json"
	"fmt"
	//"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Task struct {
	AuthorChatID int64
	Author       string
	TaskMaker    string
	TaskName     string
}

type MyOwnerTask struct {
	CreatedByMe map[int]struct{}
	IHaveToMake map[int]struct{}
}

type Database struct {
	Tasks  map[int]Task
	Users  map[string]MyOwnerTask
	DbSize int
}

var (
	Db Database
)

func init() {
	Db = Database{
		Tasks: make(map[int]Task),
		Users: make(map[string]MyOwnerTask),
	}
}

func (Db *Database) GetTaskCreatedByMe(Username string) []string {
	var Res []string
	Res = append(Res, "Нету задач которые вы создали")
	for key := range Db.Users[Username].CreatedByMe {
		var TempStr string
		if Db.Tasks[key].TaskMaker == "" {
			TempStr = fmt.Sprintf("%d. %s , испольнитель пока не назначен", key, Db.Tasks[key].TaskName)
		} else {
			TempStr = fmt.Sprintf("%d. %s @%s", key, Db.Tasks[key].TaskName, Db.Tasks[key].TaskMaker)
		}
		Res = append(Res, TempStr)
	}
	if len(Res) > 1 {
		Res = Res[1:]
	}
	return Res
}

func (Db *Database) GetTasksThatINeedToDo(Username string) []string {
	var Res []string
	Res = append(Res, "Нету задач которые вы должны сделать")
	for key, _ := range Db.Users[Username].IHaveToMake {
		TempStr := fmt.Sprintf("%s. %s @%s \nassign_%s", strconv.Itoa(key), Db.Tasks[key].TaskName, Db.Tasks[key].Author, strconv.Itoa(key))
		Res = append(Res, TempStr)
	}
	if len(Res) > 1 {
		Res = Res[1:]
	}
	return Res
}

func (Db *Database) TaskList(Username string) []string {
	var Result []string
	Result = append(Result, "Нет задач")
	for key, CurrentTask := range Db.Tasks {
		var TempForPush string
		TempForPush += strconv.Itoa(key) + " " + CurrentTask.TaskName + " by " + CurrentTask.Author
		if CurrentTask.TaskMaker == "" {
			TempForPush += fmt.Sprintf("\n/assign_%d", key)
		} else if CurrentTask.TaskMaker == Username {
			TempForPush += fmt.Sprintf("\n assignee: я\n/unassign_%d /resolve_%d\n", key, key)
		} else {
			TempForPush += fmt.Sprintf("\n assignee: @%s\n", CurrentTask.TaskMaker)
		}
		Result = append(Result, TempForPush)
	}
	if len(Result) > 1 {
		Result = Result[1:]
	}

	return Result
}

func (Db *Database) Assign(Username string, TaskId int) (string, string, int64) {
	CurrTask, exist := Db.Tasks[TaskId]
	if !exist {
		return "Такого задания не существует", "", 0
	}
	if _, ok := Db.Users[Username]; !ok {
		Db.Users[Username] = MyOwnerTask{
			CreatedByMe: make(map[int]struct{}),
			IHaveToMake: make(map[int]struct{}),
		}
	}
	if CurrTask.TaskMaker == "" {
		CurrTask.TaskMaker = Username
		Db.Tasks[TaskId] = CurrTask
		Db.Users[Username].IHaveToMake[TaskId] = struct{}{}

		if CurrTask.TaskMaker == CurrTask.Author {
			return fmt.Sprintf("Задача \"%s\" назначена на вас", CurrTask.TaskName), "", 0
		}
		return fmt.Sprintf("Задача \"%s\" назначена на вас", CurrTask.TaskName),
			fmt.Sprintf("Задача \"%s\" назначена на @%s", CurrTask.TaskName, Username),
			CurrTask.AuthorChatID
	} else {
		if CurrTask.TaskMaker != "" && CurrTask.TaskMaker != Username {
			delete(Db.Users[CurrTask.TaskMaker].IHaveToMake, TaskId)
			PrevTaskMaker := CurrTask.TaskMaker
			CurrTask.TaskMaker = Username
			Db.Tasks[TaskId] = CurrTask
			Db.Users[Username].IHaveToMake[TaskId] = struct{}{}

			return fmt.Sprintf("Задача \"%s\" переназначена вам", CurrTask.TaskName),
				fmt.Sprintf("Задача \"%s\" переназначена с @%s на @%s", CurrTask.TaskName, PrevTaskMaker, Username),
				CurrTask.AuthorChatID
		}
	}
	return "", "", 0
}

func (Db *Database) Unassign(TaskId int, Username string) (string, string, int64) {
	result1 := "Задача не на вас"
	result2 := ""
	var AuthorID int64 = 0
	CurrTask, exist := Db.Tasks[TaskId]
	if !exist {
		return "Такой задачи не существует", "", 0
	}
	if CurrTask.TaskMaker == Username {
		delete(Db.Users[Username].IHaveToMake, TaskId)
		CurrTask.TaskMaker = ""
		Db.Tasks[TaskId] = CurrTask

		if CurrTask.Author == Username {
			result1 = "Вы открепили свою задачу"
		} else {
			result1 = "Вы открепились от задачи \"" + CurrTask.TaskName + "\""
			result2 = "Задача \"" + CurrTask.TaskName + "\" осталась без исполнителя"
			AuthorID = CurrTask.AuthorChatID
		}
	}
	return result1, result2, AuthorID
}

// хранить все в формате map[username], а никому не привязанные задачи хранить под именем default_user и
// проверять все по юзернейму запроса и таблице slices.Contains(some_slice, some_value))
// назначить задачу можно только на себя
func (Db *Database) CreateTask(MakerOfTask string, TaskGoal string, AutorID int64) string {
	if _, exists := Db.Users[MakerOfTask]; !exists {
		Db.Users[MakerOfTask] = MyOwnerTask{
			CreatedByMe: make(map[int]struct{}),
			IHaveToMake: make(map[int]struct{}),
		}
	}
	newTask := Task{
		AuthorChatID: AutorID,
		Author:       MakerOfTask,
		TaskName:     TaskGoal,
	}
	Db.DbSize++
	Db.Tasks[Db.DbSize] = newTask
	Db.Users[MakerOfTask].CreatedByMe[Db.DbSize] = struct{}{}
	return fmt.Sprintf("Задача \"%s\" создана, id=%d", newTask.TaskName, Db.DbSize)
}

func (Db *Database) Resolve(TaskId int, Username string) (string, string, int64) {
	result1 := "Вы не можете открепить данную задачу, тк она принадлежит не вам"
	result2 := ""
	var AuthorID int64 = 0
	if _, exists := Db.Users[Username].IHaveToMake[TaskId]; exists {
		if Db.Tasks[TaskId].Author == Username {
			result1 = "Задача \"" + Db.Tasks[TaskId].TaskName + "\" выполнена"
		} else {
			result1 = "Задача \"" + Db.Tasks[TaskId].TaskName + "\" выполнена"
			result2 = "Задача \"" + Db.Tasks[TaskId].TaskName + "\" выполнена @" + Username
			AuthorID = Db.Tasks[TaskId].AuthorChatID
		}
		delete(Db.Users[Db.Tasks[TaskId].Author].CreatedByMe, TaskId)
		delete(Db.Users[Username].IHaveToMake, TaskId)
		delete(Db.Tasks, TaskId)
	} else {
		result1 = "Введенная задача не существует или у нее назначен другой исполнитель"
	}
	return result1, result2, AuthorID
}

func (Db *Database) Router(tempstr []string, bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	switch len(tempstr) {
	case 1:
		switch tempstr[0] {
		case "my":
			ResCreating := Db.GetTasksThatINeedToDo(update.Message.From.UserName)
			for _, curr := range ResCreating {
				bot.Send(tgbotapi.NewMessage(
					update.Message.Chat.ID,
					curr,
				))
			}
		case "owner":
			ResCreating := Db.GetTaskCreatedByMe(update.Message.From.UserName)
			for _, curr := range ResCreating {
				bot.Send(tgbotapi.NewMessage(
					update.Message.Chat.ID,
					curr,
				))
			}
		case "tasks":
			ResCreating := Db.TaskList(update.Message.From.UserName)
			for _, curr := range ResCreating {
				bot.Send(tgbotapi.NewMessage(
					update.Message.Chat.ID,
					curr,
				))
			}
		default:
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				"Unknown command"))
		}
	case 2:
		switch tempstr[0] {
		case "assign":
			AssignId, err := strconv.Atoi(tempstr[1])
			if err != nil {
				log.Fatal("unexpected resolve type")
			}
			UserMadeReqStr, TaskOwnerStr, OwnerID := Db.Assign(update.Message.From.UserName, AssignId)
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				UserMadeReqStr,
			))
			if OwnerID != 0 && TaskOwnerStr != "" {
				bot.Send(tgbotapi.NewMessage(
					OwnerID,
					TaskOwnerStr,
				))
			}
		case "unassign":
			UnAssignId, err := strconv.Atoi(tempstr[1])
			if err != nil {
				log.Fatal("unexpected resolve type")
			}
			UserMadeReqStr, TaskOwnerStr, OwnerID := Db.Unassign(UnAssignId, update.Message.From.UserName)
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				UserMadeReqStr,
			))
			if OwnerID != 0 && TaskOwnerStr != "" {
				bot.Send(tgbotapi.NewMessage(
					OwnerID,
					TaskOwnerStr,
				))
			}
		case "resolve":
			ResolvedId, err := strconv.Atoi(tempstr[1])
			if err != nil {
				log.Fatal("unexpected resolve type")
			}
			UserMadeReqStr, TaskOwnerStr, OwnerID := Db.Resolve(ResolvedId, update.Message.From.UserName)
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				UserMadeReqStr,
			))
			if OwnerID != 0 && TaskOwnerStr != "" {
				bot.Send(tgbotapi.NewMessage(
					OwnerID,
					TaskOwnerStr,
				))
			}
		case "new":
			ResCreating := Db.CreateTask(update.Message.From.UserName, tempstr[1], update.Message.Chat.ID)
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				ResCreating,
			))
		}
	}
}

func main() {
	bot, err1 := tgbotapi.NewBotAPI("тут токен вашего бота") // не покажу )))
	if err1 != nil {
		panic(err1)
	}
	fmt.Printf("Bot name: %s", bot.Self.UserName)
	_, err2 := bot.SetWebhook(tgbotapi.NewWebhook("https://nymr3y-176-57-76-184.ru.tuna.am"))
	if err2 != nil {
		panic(err2)
	}
	updates := bot.ListenForWebhook("/")

	go http.ListenAndServe(":8081", nil)
	fmt.Println("start listen :8080")
	for update := range updates {
		trimmedStr := strings.Trim(update.Message.Text, "/")
		fp := strings.Split(trimmedStr, "_")
		if len(fp) == 1 {
			commandParts := strings.SplitN(trimmedStr, " ", 2)
			Db.Router(commandParts, bot, update)
		} else {
			Db.Router(fp, bot, update)
		}

	}
}
