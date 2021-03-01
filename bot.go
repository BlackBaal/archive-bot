package main

import (
	"database/sql"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/lib/pq"
	"log"
	"strings"
)
type callbackVals struct {
	send 		string
	recieve 	string
	adm			string
	random		string
	approveYes	string
	approveNo	string
	next		string
	back		string
}

func initCallbackVals() callbackVals{
	var callbacks callbackVals
	callbacks.send = "send"
	callbacks.recieve = "want"
	callbacks.adm = "adm"
	callbacks.random = "rand"
	callbacks.approveYes = "yes"
	callbacks.approveNo = "no"
	callbacks.next = "next"
	callbacks.back = "back"
	return callbacks
}

//goland:noinspection SpellCheckingInspection
func dbConnect() (info string) {
	var host =
	var port =
	var user =
	var password =
	var dbname =
	var sslmode = ""
	var dbInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, password, dbname, sslmode)
	return dbInfo
}

func savePicture(bot *tgbotapi.BotAPI, db *sql.DB, updates tgbotapi.UpdatesChannel, keyboards [][][]tgbotapi.InlineKeyboardButton) {
	for update := range updates {
		if update.CallbackQuery != nil && update.CallbackQuery.Data == "back" {
			edit := tgbotapi.EditMessageTextConfig{
				BaseEdit:		tgbotapi.BaseEdit{ChatID: update.CallbackQuery.Message.Chat.ID, MessageID: update.CallbackQuery.Message.MessageID,
					ReplyMarkup:	&tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboards[1]}},
				Text:			"Выбери действие"}
			bot.Send(edit)
			return
		}
		if update.Message.Photo == nil {
			return
		}
		query := `INSERT INTO tempStorage (logins, file_id) VALUES ($1, $2)`
		logins := strings.Fields(update.Message.Caption)
		_, err := db.Exec(query, pq.Array(logins), (*update.Message.Photo)[0].FileID)
		if err != nil {
			log.Panic(err)
		}
		edit := tgbotapi.EditMessageReplyMarkupConfig{BaseEdit: tgbotapi.BaseEdit{ChatID: update.Message.Chat.ID, MessageID: update.Message.MessageID}}
		bot.Send(edit)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выбери действие")
		msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{keyboards[1]}
		bot.Send(msg)
		return
	}
}

func sendPicture(bot *tgbotapi.BotAPI,db *sql.DB, updates tgbotapi.UpdatesChannel)  {
	var fileID string
	for update := range updates {
		query := `SELECT file_id FROm archive WHERE login = $1 ORDER BY RANDOM() LIMIT 1`
		res, _ := db.Query(query, update.Message.Text)
		for res.Next() {
			err := res.Scan(&fileID)
			if err != nil {
				log.Fatal(err)
			}
		}
		msg := tgbotapi.NewPhotoShare(update.Message.Chat.ID, fileID)
		bot.Send(msg)
	}
}

func sendRandPicture(bot *tgbotapi.BotAPI,db *sql.DB, chatID int64, messageID int,keyboards [][][]tgbotapi.InlineKeyboardButton)  {
	var fileID string
		query := `SELECT file_id FROM screenshots ORDER BY RANDOM() LIMIT 1`
		rows, _ := db.Query(query)
		for rows.Next() {
			err := rows.Scan(&fileID)
			if err != nil {
				log.Fatal(err)
			}
		}
		edit := tgbotapi.EditMessageReplyMarkupConfig{BaseEdit: 	tgbotapi.BaseEdit{ChatID: chatID, MessageID: messageID}}
		bot.Send(edit)
		msg := tgbotapi.NewPhotoShare(chatID, fileID)
		msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{keyboards[1]}
		bot.Send(msg)
}

func iterateLogins(bot *tgbotapi.BotAPI, db *sql.DB, updates tgbotapi.UpdatesChannel, update tgbotapi.Update, logins []string, keyboards [][][]tgbotapi.InlineKeyboardButton, i int, fileID string, len int)  {
	caption := fmt.Sprintf("На скриншоте есть %s?", logins[i])
	edit := tgbotapi.EditMessageCaptionConfig{
		BaseEdit: 	tgbotapi.BaseEdit{ChatID: update.CallbackQuery.Message.Chat.ID, MessageID: update.CallbackQuery.Message.MessageID, ReplyMarkup: &tgbotapi.InlineKeyboardMarkup{keyboards[2]}},
		Caption: caption,
	}
	bot.Send(edit)
	L:
	for update := range updates{
		switch command := update.CallbackQuery.Data; command {
		case "yes":
			query := `INSERT INTO screenshots (file_id) VALUES ($1)`
			_, err := db.Exec(query, fileID)
			if err != nil {
				log.Panic(err)
			}
			query = `INSERT INTO users (login) VALUES ($1) ON CONFLICT (login) DO NOTHING`
			_, err = db.Exec(query, logins[i])
			if err != nil {
				log.Panic(err)
			}
			query = `INSERT INTO archive (user_id, screen_id) SELECT users.id, screenshots.id FROM 
										users CROSS JOIN screenshots WHERE users.login = $2 AND screenshots.file_id = $1`
			_, err = db.Exec(query, fileID, logins[i])
			if err != nil {
				log.Panic(err)
			}
			i++
			if i < len {
				iterateLogins(bot, db, updates, update, logins, keyboards, i, fileID, len)
			}
			break L
		case "no":
			continue
		}
	}
}

func sendPictureForApproval(bot *tgbotapi.BotAPI, db *sql.DB, updates tgbotapi.UpdatesChannel, chatID int64,keyboards [][][]tgbotapi.InlineKeyboardButton)  {
	var fileID	string
	var logins	[]string
	var id		int
	i := 0
	query := `SELECT * FROM tempstorage`
	rows, _ := db.Query(query)
	for rows.Next() {
		err := rows.Scan(&id, pq.Array(&logins), &fileID)
		if err != nil {
			log.Fatal(err)
		}
		msg := tgbotapi.NewPhotoShare(chatID, fileID)
		msg.Caption = "Это скриншот?"
		msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{keyboards[2]}
		bot.Send(msg)
		L:
		for update := range updates {
			switch command := update.CallbackQuery.Data; command {
			case "yes":
				if i < len(logins) {
					iterateLogins(bot, db, updates, update, logins, keyboards, i, fileID, len(logins))
				}
				break L
			case "no":
				}
			}
		query := `DELETE FROM tempstorage WHERE ID = $1`
		_, err = db.Exec(query, id)
		if err != nil {
			log.Panic(err)
		}
	}
}

func makeKeyboard(callbacks callbackVals) [][][]tgbotapi.InlineKeyboardButton {
	var keyboards [][][]tgbotapi.InlineKeyboardButton

	//admin keyboard (index = 0)
	keyboards = append(keyboards, [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.InlineKeyboardButton{Text: "Хочу кек", CallbackData: &callbacks.recieve}, tgbotapi.InlineKeyboardButton{Text: "Рандомный кек", CallbackData: &callbacks.random}},
		{tgbotapi.InlineKeyboardButton{Text: "Отправить кек", CallbackData: &callbacks.send}, tgbotapi.InlineKeyboardButton{Text: "Админ панель", CallbackData: &callbacks.adm}}})

	//regular user keyboard (index = 1)
	keyboards = append(keyboards, [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.InlineKeyboardButton{Text: "Хочу кек", CallbackData: &callbacks.recieve}}, {tgbotapi.InlineKeyboardButton{Text: "Рандомный кек", CallbackData: &callbacks.random}},
		{tgbotapi.InlineKeyboardButton{Text: "Отправить кек", CallbackData: &callbacks.send}}})

	//keyboard for pic approval (index = 2)
	keyboards = append(keyboards, [][]tgbotapi.InlineKeyboardButton{{tgbotapi.InlineKeyboardButton{Text: "Да", CallbackData: &callbacks.approveYes}, tgbotapi.InlineKeyboardButton{Text: "Нет", CallbackData: &callbacks.approveNo}}})

	//keyboard when submitting pictures (index = 3)
	keyboards = append(keyboards, [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.InlineKeyboardButton{Text: "Отменить", CallbackData: &callbacks.back}}})
	return keyboards
}

func getModList (db *sql.DB, modList []string) []string {
	var username string
	query := `SELECT username FROM modList`
	rows, err := db.Query(query)
	if err != nil {
		log.Panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&username)
		if err != nil {
			log.Panic(err)
		}
		modList = append(modList, username)
	}
	return modList
}

func checkMod(modList []string, username string)  bool {
	for _, name := range modList {
		if name == username {
			return true
		}
	}
	return false
}

func responseHandler(bot *tgbotapi.BotAPI, db *sql.DB, updates tgbotapi.UpdatesChannel, update tgbotapi.Update, keyboards [][][]tgbotapi.InlineKeyboardButton, modFlag bool)  {
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		var respKeyboard tgbotapi.InlineKeyboardMarkup
		if modFlag {
			respKeyboard = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboards[0]}
		} else {
			respKeyboard = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboards[1]}
		}
		switch command := update.Message.Text; command {
		case "/start":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "vibirai")
			msg.ReplyMarkup = respKeyboard
			bot.Send(msg)
		}
}

func callbackHandler(bot *tgbotapi.BotAPI, db *sql.DB, updates tgbotapi.UpdatesChannel, update tgbotapi.Update, keyboards [][][]tgbotapi.InlineKeyboardButton, modFlag bool)  {
	var respKeyboard tgbotapi.InlineKeyboardMarkup
	if modFlag {
		respKeyboard = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboards[0]}
	} else {
		respKeyboard = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboards[1]}
	}
	switch command := update.CallbackQuery.Data; command {
	case "send":
		msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Пришли фото с логинами через пробел в подписи")
		msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{keyboards[3]}
		bot.Send(msg)
		savePicture(bot, db, updates, keyboards)

	case "want":
		msg := tgbotapi.EditMessageTextConfig{
			BaseEdit:              tgbotapi.BaseEdit{ChatID: update.CallbackQuery.Message.Chat.ID, MessageID: update.CallbackQuery.Message.MessageID,
				ReplyMarkup: &respKeyboard},
			Text:                  "Чей?",
		}
		bot.Send(msg)
		//sendPicture(bot, db, updates)
	case "rand":
		sendRandPicture(bot, db, update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, keyboards)
	case "adm":
		sendPictureForApproval(bot, db, updates, update.CallbackQuery.Message.Chat.ID, keyboards)
	}
}

func main() {
	modFlag := false
	var modList []string
	bot, err := tgbotapi.NewBotAPI("")
	if err != nil {
		log.Panic(err)
	}

	db, err := sql.Open("postgres", dbConnect())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	bot.Debug = true


	modList = getModList(db, modList)
	keyboards := makeKeyboard(initCallbackVals())
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Panic(err)
	}

	for update := range updates {
		if update.Message != nil {
			modFlag = checkMod(modList, update.Message.From.UserName)
			responseHandler(bot, db, updates, update, keyboards, modFlag)
		} else if update.CallbackQuery != nil {
			modFlag = checkMod(modList, update.CallbackQuery.From.UserName)
			callbackHandler(bot, db, updates, update, keyboards, modFlag)
		} else {
			continue
		}
	}
}
