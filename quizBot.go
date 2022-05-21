package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	firebase "firebase.google.com/go"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/api/iterator"
)

func commandParse(msgTxt string, keyword string) string {
	if strings.Contains(msgTxt, "/"+keyword+" ") {
		return strings.Replace(msgTxt, "/"+keyword+" ", "", 1)
	} else {
		return ""
	}
}

func sendSimpleMsg(chatID int64, msgTxt string, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, msgTxt)

	if _, err := bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

var optionKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Yes"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("No"),
	),
)

func main() {

	// init firebase client
	opt := option.WithCredentialsFile("firebase_service_acct.json")
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		fmt.Errorf("error initializing app: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	defer client.Close()

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

	var currentUsername string = ""
	var currentUserID string = ""

	for update := range updates {
		// ignore non-Message updates
		if update.Message == nil {
			continue
		}

		currentUsername = update.Message.From.UserName
		currentUserID = fmt.Sprint(update.Message.From.ID)

		fmt.Printf("[%s, %s] %s", currentUsername, currentUserID, update.Message.Text)

		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		} else if update.Message.Command() == "start" {
			// Check if the focus user id is already in the USERS collection, else create new user

			docRef := client.Collection("USERS").Doc(currentUserID)
			doc, err := docRef.Get(ctx)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					// // Handle document not existing here
					// _, err := docRef.Set(ctx /* custom object here */)
					// if err != nil {
					// 	return err
					// }
				} else {
					// return err
				}
			}

			if doc.Exists() {
				// Handle document existing here
				fmt.Println("User found")

			} else {
				// Create new user document
				_, err := client.Collection("USERS").Doc(currentUserID).Set(ctx, map[string]interface{}{
					"username": currentUsername,
				})

				if err != nil {
					log.Fatalf("Failed adding [%s]: %v", currentUsername, err)
				}

				// Create new user quizzes collection
				_, err2 := client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc("demo quiz").Set(ctx, map[string]interface{}{
					"numQns":                       "1",
					"score":                        "none",
					"this is a demo quiz question": "this is a demo quiz answer",
				})

				if err2 != nil {
					log.Fatalf("Failed adding quizzes collection for [%s]: %v", currentUsername, err2)
				}
			}

			// // Return message: "Hello <username>!"
			// msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			// msg.Text = "Hello " + currentUsername + "!"

			// if _, err := bot.Send(msg); err != nil {
			// 	log.Panic(err)
			// }

			sendSimpleMsg(update.Message.Chat.ID, "Hello "+currentUsername+"!", bot)

			botState = "idle"

		} else if currentUserID == fmt.Sprint(update.Message.From.ID) {
			// check if message is from current user if not ignore other users

			if botState == "idle" {
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

				case "addQns":
					// parse quiz name
					quizName := commandParse(update.Message.Text, "addQns")

					fmt.Println("SEARCHING FOR QUIZ: " + quizName)

					paramCharLen := len(quizName)

					if paramCharLen > 0 {
						docRef := client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizName)
						doc, err := docRef.Get(ctx)
						if err != nil {
							if status.Code(err) == codes.NotFound {
								// // Handle document not existing here
								// _, err := docRef.Set(ctx /* custom object here */)
								// if err != nil {
								// 	return err
								// }
							} else {
								// return err
							}
						}

						if doc.Exists() {
							// Handle document existing here
							fmt.Println("User found")

						} else {

						}
					} else {

					}
				case "listQuizzes":
					var docNames []string
					iter := client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Documents(ctx)
					for {
						doc, err := iter.Next()
						if err == iterator.Done {
							break
						}
						if err != nil {
							//return err
						}
						docNames = append(docNames, doc.Ref.ID)
					}

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.ParseMode = "HTML"
					msg.Text = "Here is the list of your quizzes: \n" 
					for i, s := range docNames {
						msg.Text += "- " + s + "\n"
						fmt.Println(i, s)
					}
					msg.ParseMode = "HTML" 

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

				default:
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Sorry I don't understand you! Type <strong>/help</strong> for a list of commands!"
					msg.ParseMode = "HTML"

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}
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
