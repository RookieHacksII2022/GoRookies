package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var optionKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Yes"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("No"),
	),
)

// func sendMsg() {

// }

func main() {

	// Load the telegram bot key
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	var botState string = "inactive"
	// var botState string = "idle"

	// inactive -> current user has not be logged by bot. User is otherwise logged by the bot
	// idle -> user has been logged by bot and waiting command

	var currentUser string = ""

	for update := range updates {
		// ignore non-Message updates
		if update.Message == nil {
			continue
		}

		if botState == "inactive" {
			if !update.Message.IsCommand() { // ignore any non-command Messages
				continue
			}

			if update.Message.Command() == "start" {
				// Check if the focus user id is already in the USERS collection, else create new user
				currentUser = update.Message.From.UserName

				// Return message: "Hello <username>!"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
				msg.Text = "Hello " + currentUser + "!"

				if _, err := bot.Send(msg); err != nil {
					log.Panic(err)
				}

			}
		} else if botState == "idle" {
			if !update.Message.IsCommand() { // ignore any non-command Messages
				continue
			}

			switch update.Message.Command() {
			case "help":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
				msg.Text = "I understand the following commands: \n" +
					"<strong>/help</strong>  - get list of commands\n" +
					"<strong>/addQuiz <i>quiz_name</i></strong> - add a new quiz\n" +
					"<strong>/addQns <i>quiz_name</i></strong> - add questions to a selected quiz\n" +
					"<strong>/tryQuiz <i>quiz_name</i></strong> - try a selected quiz\n" +
					"<strong>/deleteQuiz <i>quiz_name</i></strong> - delete a selected quiz\n" +
					"<strong>/listQuizzes</strong> - list all ofyour quizzes"
				msg.ParseMode = "HTML"

				if _, err := bot.Send(msg); err != nil {
					log.Panic(err)
				}

			case "start":
				// log new focus user

			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
				msg.Text = "Sorry I don't understand you! Type <strong>/help</strong> for a list of commands!"
				msg.ParseMode = "HTML"

				if _, err := bot.Send(msg); err != nil {
					log.Panic(err)
				}
			}
		}

		// // create new message text
		// msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		// msg.ReplyToMessageID = update.Message.MessageID

		// // add keyboard markup
		// switch update.Message.Text {
		// case "open":
		// 	msg.ReplyMarkup = optionKeyboard
		// case "close":
		// 	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		// }

		// // message error handling
		// if _, err := bot.Send(msg); err != nil {
		// 	log.Panic(err)
		// }
	}
}
