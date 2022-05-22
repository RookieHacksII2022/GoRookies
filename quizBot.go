package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func sendHelpMessage(chatID int64, bot *tgbotapi.BotAPI) {
	msg := tgbotapi.NewMessage(chatID, "")
	msg.Text = "I understand the following commands: \n" +
		"<strong>/help</strong>  - get list of commands\n" +
		"<strong>/addQuiz <i>quiz_name</i></strong> - add a new quiz\n" +
		"<strong>/addQns <i>quiz_name</i></strong> - add questions to a selected quiz\n" +
		"<strong>/removeQns <i>quiz_name</i></strong> - remove questions from a selected quiz\n" +
		"<strong>/tryQuiz <i>quiz_name</i></strong> - try a selected quiz\n" +
		"<strong>/deleteQuiz <i>quiz_name</i></strong> - delete a selected quiz\n" +
		"<strong>/listQuizzes</strong> - list all of your quizzes"
	msg.ParseMode = "HTML"

	if _, err := bot.Send(msg); err != nil {
		log.Panic(err)
	}
}

var yesNoKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Yes"),
		tgbotapi.NewKeyboardButton("No"),
	),
)

var questionReviewKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Keep"),
		tgbotapi.NewKeyboardButton("Toss"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Cancel"),
	),
)

func createTwoBtnRowKeyboard(btnTxt1 string, btnTxt2 string) tgbotapi.ReplyKeyboardMarkup {
	var optionsKeyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(btnTxt1),
			tgbotapi.NewKeyboardButton(btnTxt2),
		),
	)

	return optionsKeyboard
}

func sendQuestion(
	chatID int64,
	qnIndex int,
	bot *tgbotapi.BotAPI,
	questionsMap1 map[string]string,
	questionsMap2 map[int]string,
) {

	msg2 := tgbotapi.NewMessage(chatID, "")
	msg2.Text = "<strong>Q:</strong> " + questionsMap2[qnIndex] + "\n" +
		"<strong>A:</strong> " + questionsMap1[questionsMap2[qnIndex]] + "\n\n"
	msg2.ParseMode = "HTML"
	if _, err := bot.Send(msg2); err != nil {
		log.Panic(err)
	}
}

func confirmQnsRemove(
	chatID int64,
	qnIndex int,
	bot *tgbotapi.BotAPI,
	questionsMap1 map[string]string,
	questionsMap3 map[string]bool,
) bool {

	var msgCompilation string = ""
	var nextQn string = ""

	var haveTossed bool = false

	for question, isTossed := range questionsMap3 {
		if isTossed {
			haveTossed = true

			nextQn = "<strong>Q:</strong> " + question + "\n" +
				"<strong>A:</strong> " + questionsMap1[question] + "\n"

			if len(msgCompilation)+len(nextQn) < 4096 {
				// append and continue
				msgCompilation = msgCompilation + nextQn

			} else {
				// send partial message
				msg2 := tgbotapi.NewMessage(chatID, "")
				msg2.Text = msgCompilation
				msg2.ParseMode = "HTML"
				if _, err := bot.Send(msg2); err != nil {
					log.Panic(err)
				}

				msgCompilation = nextQn
			}
		}

	}

	if msgCompilation != "" {
		msg2 := tgbotapi.NewMessage(chatID, "")
		msg2.Text = msgCompilation
		msg2.ParseMode = "HTML"
		if _, err := bot.Send(msg2); err != nil {
			log.Panic(err)
		}
	}

	if haveTossed {
		msg2 := tgbotapi.NewMessage(chatID, "")
		msg2.Text = "Are you sure you want to remove all the above questions?"
		msg2.ReplyMarkup = yesNoKeyboard

		if _, err := bot.Send(msg2); err != nil {
			log.Panic(err)
		}
	} else {
		msg2 := tgbotapi.NewMessage(chatID, "")
		msg2.Text = "No questions selected for removal"
		msg2.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

		if _, err := bot.Send(msg2); err != nil {
			log.Panic(err)
		}
	}

	return haveTossed

}

func main() {
	// // check for env file
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }

	// fmt.Println("var1 = ", reflect.TypeOf(optionsKeyboard))

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

	// inactive -> current user has not be logged by bot. User is otherwise logged by the bot
	// idle -> user has been logged by bot and waiting command

	var inputExpected string = "none"

	var currentUsername string = ""
	var currentUserID string = ""
	var quizName string = ""

	questionsMap1 := make(map[string]string)
	questionsMap2 := make(map[int]string)
	questionsMap3 := make(map[string]bool)

	updateDocFields := make(map[string]interface{})

	var questionText = ""

	var numQns int = 0
	var qnsRemaining int = 0
	var scoreInt int = 0

	// var quizData map[string]interface{}

	fmt.Println(numQns, qnsRemaining, scoreInt)

	for update := range updates {
		// ignore non-Message updates
		if update.Message == nil {
			continue
		}

		currentUsername = update.Message.From.UserName
		currentUserID = fmt.Sprint(update.Message.From.ID)

		fmt.Printf("[%s, %s] %s\n", currentUsername, currentUserID, update.Message.Text)

		if update.Message.IsCommand() && update.Message.Command() == "start" {
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

			sendSimpleMsg(update.Message.Chat.ID, "Hello "+currentUsername+"!", bot)

			botState = "idle"

		} else if currentUserID == fmt.Sprint(update.Message.From.ID) {
			// check if message is from current user if not ignore other users

			switch botState {
			case "idle":
				if !update.Message.IsCommand() { // ignore any non-command Messages
					continue
				}

				switch update.Message.Command() {
				case "help":
					sendHelpMessage(update.Message.Chat.ID, bot)
				case "addQuiz":

					quizTitle := Parser(update.Message.Text)

					fmt.Println("SHOW QUIZ TITLE: " + quizTitle)

					if len(quizTitle) < 1 {

						sendSimpleMsg(
							update.Message.Chat.ID,
							"Quiz title cannot be empty, please try again!",
							bot,
						)

					} else {

						docRef := client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizTitle)
						_, err := docRef.Get(ctx)
						if err == nil {
							sendSimpleMsg(
								update.Message.Chat.ID,
								"Quiz title exists",
								bot,
							)
						} else {
							_, _ = client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizTitle).Set(ctx, map[string]interface{}{
								"numQns": "0",
								"score":  "none",
							})

							sendSimpleMsg(
								update.Message.Chat.ID,
								"New Quiz Title: "+quizTitle+" is added into your collection.",
								bot,
							)
						}

					}
					botState = "idle"

				case "addQns":
					// parse quiz name
					quizName = commandParse(update.Message.Text, "addQns")

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
							fmt.Println("Doc found:", doc.Ref.ID)

							fmt.Println("var1 = ", reflect.TypeOf(doc.Data()))

							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
							msg.Text = "Quiz titled " + quizName + " found!\n" +
								"Press <strong>Exit</strong> to save changes and end\n" +
								"Press <strong>Cancel</strong> to quit without saving\n" +
								"Please input new question:"
							msg.ParseMode = "HTML"
							msg.ReplyMarkup = createTwoBtnRowKeyboard("Exit", "Cancel")

							if _, err := bot.Send(msg); err != nil {
								log.Panic(err)
							}

							numQns = int(doc.Data()["numQns"].(int64))

							botState = "addQns_Qn"
							inputExpected = "qn"

						} else {
							sendSimpleMsg(
								update.Message.Chat.ID,
								"Quiz with name "+quizName+" not found.",
								bot,
							)
						}
					} else {
						sendSimpleMsg(
							update.Message.Chat.ID,
							"Please include a quiz name with this command.\n"+
								"Spaces in the quiz name are allowed.\n"+
								"e.g. `/addQns demo quiz`",
							bot,
						)
					}
				case "removeQns":
					// parse quiz name
					quizName = commandParse(update.Message.Text, "removeQns")

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
							fmt.Println("Doc found:", doc.Ref.ID)

							numQns = int(doc.Data()["numQns"].(int64))
							qnsRemaining = 0

							if numQns == 0 {
								sendSimpleMsg(
									update.Message.Chat.ID,
									"This quiz has no questions to remove!",
									bot,
								)
							} else {
								msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
								msg.Text = "Quiz titled " + quizName + " found!\n" +
									"For each question:\n" +
									"Press <strong>Keep</strong> to keep the question\n" +
									"Press <strong>Toss</strong> to remove the question\n" +
									"Press <strong>Cancel</strong> to revert changes\n"
								msg.ParseMode = "HTML"

								msg.ReplyMarkup = questionReviewKeyboard

								if _, err := bot.Send(msg); err != nil {
									log.Panic(err)
								}

								for question, answer := range doc.Data() {
									if question != "numQns" && question != "score" {
										questionsMap1[question] = answer.(string)
										qnsRemaining++
										questionsMap2[qnsRemaining] = question
									}
								}

								sendQuestion(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
								qnsRemaining--

								numQns = 0
								botState = "removeQns"
							}
						} else {
							sendSimpleMsg(
								update.Message.Chat.ID,
								"Quiz with name "+quizName+" not found.",
								bot,
							)
						}
					} else {
						sendSimpleMsg(
							update.Message.Chat.ID,
							"Please include a quiz name with this command.\n"+
								"Spaces in the quiz name are allowed.\n"+
								"e.g. `/addQns demo quiz`",
							bot,
						)
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

			case "addQns_Qn":
				switch update.Message.Text {
				case "Exit":
					questionsMap1["numQns"] = fmt.Sprint(numQns)
					questionsMap1["score"] = "none"

					_, err := client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizName).Set(ctx, questionsMap1, firestore.MergeAll)

					if err != nil {
						// Handle any errors in an appropriate way, such as returning them.
						log.Printf("An error has occurred: %s", err)
					}

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Questions added to quiz!"
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "idle"
					inputExpected = "none"

				case "Cancel":

					// to quit without saving
					msg2 := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg2.Text = "Are you sure you want to <strong>Cancel</strong> update?"
					msg2.ParseMode = "HTML"
					msg2.ReplyMarkup = yesNoKeyboard

					if _, err := bot.Send(msg2); err != nil {
						log.Panic(err)
					}

					botState = "addQns_cancel"

				default:
					if inputExpected == "qn" {
						// input expected is qn
						questionText = update.Message.Text
						inputExpected = "ans"

						sendSimpleMsg(
							update.Message.Chat.ID,
							"Please input the answer:",
							bot,
						)

					} else if inputExpected == "ans" {
						//input expected is answer

						// add ans to array
						questionsMap1[questionText] = update.Message.Text
						numQns++
						inputExpected = "qn"

						sendSimpleMsg(
							update.Message.Chat.ID,
							"Please input the next question:",
							bot,
						)
					} else {
						log.Panic("inputExpected should be qn or ans")
					}

				}

			case "addQns_cancel":
				switch update.Message.Text {
				case "Yes":
					// cancel all changes
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Changes to quiz cancelled."
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "idle"

				case "No":

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					if inputExpected == "qn" {
						msg.Text = "Please input next question"
					}

					msg.ReplyMarkup = createTwoBtnRowKeyboard("Exit", "Cancel")

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "addQns_Qn"

				default:
				}

			case "removeQns":
				switch update.Message.Text {
				case "Keep":
					// check for next qn to send
					questionsMap3[questionsMap2[qnsRemaining+1]] = false
					numQns++

					if qnsRemaining == 0 {
						haveTossed := confirmQnsRemove(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap3)

						if haveTossed {
							botState = "removeQns_confirm"
						} else {
							botState = "idle"
						}

					} else {
						sendQuestion(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
						qnsRemaining--
					}

				case "Toss":
					// add to questionsMap3
					questionsMap3[questionsMap2[qnsRemaining+1]] = true

					// check for next qn to send
					if qnsRemaining == 0 {
						haveTossed := confirmQnsRemove(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap3)

						if haveTossed {
							botState = "removeQns_confirm"
						} else {
							botState = "idle"
						}

					} else {
						sendQuestion(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
						qnsRemaining--
					}

				case "Cancel":
					// to quit without saving
					msg2 := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg2.Text = "Are you sure you want to <strong>Cancel</strong> update?"
					msg2.ParseMode = "HTML"
					msg2.ReplyMarkup = yesNoKeyboard

					if _, err := bot.Send(msg2); err != nil {
						log.Panic(err)
					}

					botState = "removeQns_cancel"

				default:
				}

			case "removeQns_cancel":
				switch update.Message.Text {
				case "Yes":
					// cancel all changes
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Changes to quiz cancelled."
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					// reset arrays
					questionsMap1 = make(map[string]string)
					questionsMap2 = make(map[int]string)
					questionsMap3 = make(map[string]bool)

					botState = "idle"

				case "No":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Continuing quiz review. Toss or keep previous question?"
					msg.ReplyMarkup = questionReviewKeyboard

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "removeQns"

				default:
				}

			case "removeQns_confirm":
				switch update.Message.Text {
				case "Yes":
					// update all the listed questions to remove in firebase
					updateDocFields = make(map[string]interface{})
					for question, isTossed := range questionsMap3 {
						if !isTossed {
							updateDocFields[question] = questionsMap1[question]
						}
					}
					// update score field to "none"
					updateDocFields["score"] = "none"

					// update numQns field to numQns
					updateDocFields["numQns"] = numQns

					_, err := client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizName).Set(ctx, updateDocFields)
					if err != nil {
						// Handle any errors in an appropriate way, such as returning them.
						log.Printf("An error has occurred: %s", err)
					}

					// cancel all changes
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Removed selected questions."
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					// reset arrays
					questionsMap1 = make(map[string]string)
					questionsMap2 = make(map[int]string)
					questionsMap3 = make(map[string]bool)

					botState = "idle"

				case "No":

					// cancel all changes
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Changes to quiz cancelled."
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					// reset arrays
					questionsMap1 = make(map[string]string)
					questionsMap2 = make(map[int]string)
					questionsMap3 = make(map[string]bool)

					botState = "idle"

				default:
				}

			default:
				sendHelpMessage(update.Message.Chat.ID, bot)
				fmt.Println("BOT STATE INVALID")
			}

		}

	}
}

func Parser(str string) string {
	arr := strings.SplitN(str, " ", 2)
	return arr[1]
}
