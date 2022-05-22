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
	if strings.Contains(msgTxt, "/"+keyword+"@go_quiz_test_bot ") {
		return strings.Replace(msgTxt, "/"+keyword+"@go_quiz_test_bot ", "", 1)
	} else if strings.Contains(msgTxt, "/"+keyword+" ") {
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
	msg.ParseMode = "HTML"
	msg.Text = "I understand the following commands: \n" +
		"<strong>/help</strong>  - get list of commands\n" +
		"<strong>/add_quiz <i>quiz_name</i></strong> - add a new quiz\n" +
		"<strong>/add_qns <i>quiz_name</i></strong> - add questions to a selected quiz\n" +
		"<strong>/remove_qns <i>quiz_name</i></strong> - remove questions from a selected quiz\n" +
		"<strong>/try_quiz</strong> - try a selected quiz\n" +
		"<strong>/delete_quiz <i>quiz_name</i></strong> - delete a selected quiz\n" +
		"<strong>/list_quizzes</strong> - list all of your quizzes\n" +
		"<strong>/get_my_id</strong> - get your telegram ID number"

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

var questionResultKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Correct"),
		tgbotapi.NewKeyboardButton("Wrong"),
	),
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("End Quiz"),
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

func sendQuestionAndAnswerSet(
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

func sendQuestion(
	chatID int64,
	qnIndex int,
	bot *tgbotapi.BotAPI,
	questionsMap1 map[string]string,
	questionsMap2 map[int]string,
) {

	msg2 := tgbotapi.NewMessage(chatID, "")
	msg2.Text = "<strong>Q:</strong> " + questionsMap2[qnIndex] + "\n"
	msg2.ParseMode = "HTML"
	msg2.ReplyMarkup = createTwoBtnRowKeyboard("Reveal Ans", "End Quiz")
	if _, err := bot.Send(msg2); err != nil {
		log.Panic(err)
	}
}

func sendAnswer(
	chatID int64,
	qnIndex int,
	bot *tgbotapi.BotAPI,
	questionsMap1 map[string]string,
	questionsMap2 map[int]string,
) {

	msg2 := tgbotapi.NewMessage(chatID, "")
	msg2.Text = "<strong>A:</strong> " + questionsMap1[questionsMap2[qnIndex]] + "\n"
	msg2.ParseMode = "HTML"
	msg2.ReplyMarkup = questionResultKeyboard
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

	var msgCompilation string = "QUESTIONS TO REMOVE:\n"
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
		msg2.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
			RemoveKeyboard: true,
			Selective:      false,
		}

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

	// inactive -> current user has not be logged by bot. User is otherwise logged by the bot
	// idle -> user has been logged by bot and waiting command
	var botState string = "inactive"
	var inputExpected string = "none"
	var tryingMyQuiz bool = false

	var friendUserID string = ""

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

		if currentUserID == "" {
			currentUserID = fmt.Sprint(update.Message.From.ID)
		}
		if currentUsername == "" {
			currentUsername = update.Message.From.UserName
		}

		fmt.Printf("[%s, %s] %s\n", currentUsername, currentUserID, update.Message.Text)

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			// Check if the focus user id is already in the USERS collection, else create new user
			currentUsername = update.Message.From.UserName
			currentUserID = fmt.Sprint(update.Message.From.ID)

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

				if doc.Data()["username"].(string) != currentUsername {
					// update username in database
					_, err = client.Collection("USERS").Doc(currentUserID).Update(ctx, []firestore.Update{
						{
							Path:  "username",
							Value: currentUsername,
						},
					})

					if err != nil {
						// Handle any errors in an appropriate way, such as returning them.
						log.Printf("An error has occurred trying to update username: %s", err)
					}

				}

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
					"numQns":                       1,
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
				case "add_quiz":

					quizTitle := commandParse(update.Message.Text, "add_quiz")

					fmt.Println("SHOW QUIZ TITLE: " + quizTitle)

					paramCharLen := len(quizTitle)

					if paramCharLen < 1 {

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
								"numQns": 0,
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

				case "add_qns":
					// parse quiz name
					quizName = commandParse(update.Message.Text, "add_qns")

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

							botState = "add_qns_Qn"
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
								"e.g. `/add_qns demo quiz`",
							bot,
						)
					}
				case "remove_qns":
					// parse quiz name
					quizName = commandParse(update.Message.Text, "remove_qns")

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

								sendQuestionAndAnswerSet(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
								qnsRemaining--

								numQns = 0
								botState = "remove_qns"
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
								"e.g. `/remove_qns demo quiz`",
							bot,
						)
					}
				case "delete_quiz":
					// parse quiz name
					quizName = commandParse(update.Message.Text, "delete_quiz")
					paramCharLen := len(quizName)
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.ParseMode = "HTML"

					if paramCharLen > 0 {
						docRef := client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizName)
						doc, err := docRef.Get(ctx)
						if doc.Exists() {
							docRef.Delete(ctx)
						}

						if err == nil {
							msg.Text = "Successfully deleted quiz: " + quizName
						}

						if err != nil {
							msg.Text = "Quiz could not be found. Error deleting quiz: " + quizName
						}

						if _, err := bot.Send(msg); err != nil {
							log.Panic(err)
						}
					} else {
						sendSimpleMsg(
							update.Message.Chat.ID,
							"Please include a quiz name with this command.\n"+
								"Spaces in the quiz name are allowed.\n"+
								"e.g. `/delete_quiz demo quiz`",
							bot,
						)
					}
				case "list_quizzes":
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
					if len(docNames) > 0 {
						msg.Text = "Here is the list of your quizzes: \n"
						for i, s := range docNames {
							msg.Text += "- " + s + "\n"
							fmt.Println(i, s)
						}
					} else {
						msg.Text = "No quizzes found. Create one with /add_quiz quiz name"
					}


					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

				case "get_my_id":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.ParseMode = "HTML"
					msg.Text = "Here is your user info: \n" +
						"<strong>id</strong>: " + currentUserID + "\n" +
						"<strong>firstname</strong> " + update.Message.From.FirstName + "\n" +
						"<strong>username</strong> " + currentUsername + "\n"

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

				case "try_quiz":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.ParseMode = "HTML"
					msg.Text = "Would you like to try your own quiz or a friend's quiz?"
					msg.ReplyMarkup = createTwoBtnRowKeyboard("My own quiz", "A friend's quiz")

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "try_quiz_select"

				default:
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Sorry I don't understand you! Type <strong>/help</strong> for a list of commands!"
					msg.ParseMode = "HTML"

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}
				}

			case "try_quiz_select":
				switch update.Message.Text {
				case "My own quiz":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Please input the quiz name:\n" +
						"(Press <strong>Cancel</strong> to exit)"
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Cancel"),
						),
					)
					msg.ParseMode = "HTML"

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "try_quiz_myQuiz"
					tryingMyQuiz = true

				case "A friend's quiz":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Please input your friend's user id number.\n" +
						"Your friend can get their id number using the <strong>/get_my_id</strong> command.\n" +
						"(Press <strong>Cancel</strong> to exit)"
					msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
						tgbotapi.NewKeyboardButtonRow(
							tgbotapi.NewKeyboardButton("Cancel"),
						),
					)
					msg.ParseMode = "HTML"

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "try_quiz_friend"
					tryingMyQuiz = false

				default:

				}
			case "try_quiz_myQuiz":
				switch update.Message.Text {
				case "Cancel":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Cancelling quiz attempt"
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
						RemoveKeyboard: true,
						Selective:      false,
					}

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "idle"

				default:
					quizName = update.Message.Text
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
						prevScore := doc.Data()["score"].(string)
						qnsRemaining = 0
						scoreInt = 0

						if prevScore != "none" {
							prevScore = "You previously got " + prevScore + " on this quiz.\n"
						} else {
							prevScore = ""
						}

						if numQns == 0 {
							sendSimpleMsg(
								update.Message.Chat.ID,
								"This quiz has no questions to try! Please enter another quiz name.",
								bot,
							)
						} else {
							// send quiz instructions
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
							msg.Text = "Quiz titled " + quizName + " found!\n" +
								prevScore +
								"For each question:\n" +
								"Press <strong>Reveal Answer</strong> to reveal the answer.\n" +
								"After that, press <strong>Correct</strong> if you answered correctly,\n" +
								"or press <strong>Wrong</strong> if you answered wrongly\n" +
								"Your score will be computed at the end of the quiz.\n" +
								"You may also <strong>End quiz</strong> at any time\n"
							msg.ParseMode = "HTML"

							if _, err := bot.Send(msg); err != nil {
								log.Panic(err)
							}

							// reset questionMaps
							questionsMap1 = make(map[string]string)
							questionsMap2 = make(map[int]string)
							questionsMap3 = make(map[string]bool)

							// save questions to question map
							for question, answer := range doc.Data() {
								if question != "numQns" && question != "score" {
									questionsMap1[question] = answer.(string)
									qnsRemaining++
									questionsMap2[qnsRemaining] = question
								}
							}

							// send first question
							sendQuestion(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)

							botState = "try_quiz_quizAttempt"
							inputExpected = "post-qn"
						}
					} else {
						sendSimpleMsg(
							update.Message.Chat.ID,
							"Quiz with name "+quizName+" not found. Please re-enter your quiz name",
							bot,
						)
					}
				}

			case "try_quiz_friend":
				switch update.Message.Text {
				case "Cancel":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Cancelling quiz attempt"
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
						RemoveKeyboard: true,
						Selective:      false,
					}

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "idle"

				default:
					friendUserID = update.Message.Text
					docRef := client.Collection("USERS").Doc(friendUserID)
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

						friendUsername := doc.Data()["username"].(string)

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
						msg.Text = "Friend with username " + friendUsername + " found! Please input the quiz name:\n" +
							"(Press <strong>Cancel</strong> to exit)"
						msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
							tgbotapi.NewKeyboardButtonRow(
								tgbotapi.NewKeyboardButton("Cancel"),
							),
						)
						msg.ParseMode = "HTML"

						if _, err := bot.Send(msg); err != nil {
							log.Panic(err)
						}

						botState = "try_quiz_friendQuiz"

					} else {
						sendSimpleMsg(
							update.Message.Chat.ID,
							"User with with ID "+friendUserID+" not found in our database. Please re-enter friend ID",
							bot,
						)
					}
				}

			case "try_quiz_friendQuiz":
				switch update.Message.Text {
				case "Cancel":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Cancelling quiz attempt"
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
						RemoveKeyboard: true,
						Selective:      false,
					}

					if _, err := bot.Send(msg); err != nil {
						log.Panic(err)
					}

					botState = "idle"

				default:
					quizName = update.Message.Text
					docRef := client.Collection("USERS").Doc(friendUserID).Collection("QUIZZES").Doc(quizName)
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
						scoreInt = 0

						if numQns == 0 {
							sendSimpleMsg(
								update.Message.Chat.ID,
								"This quiz has no questions to try! Please enter another quiz name.",
								bot,
							)
						} else {
							// send quiz instructions
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
							msg.Text = "Quiz titled " + quizName + " found!\n" +
								"For each question:\n" +
								"Press <strong>Reveal Answer</strong> to reveal the answer.\n" +
								"After that, press <strong>Correct</strong> if you answered correctly,\n" +
								"or press <strong>Wrong</strong> if you answered wrongly\n" +
								"Your score will be computed at the end of the quiz.\n" +
								"You may also <strong>End quiz</strong> at any time\n"
							msg.ParseMode = "HTML"

							if _, err := bot.Send(msg); err != nil {
								log.Panic(err)
							}

							// reset questionMaps
							questionsMap1 = make(map[string]string)
							questionsMap2 = make(map[int]string)
							questionsMap3 = make(map[string]bool)

							// save questions to question map
							for question, answer := range doc.Data() {
								if question != "numQns" && question != "score" {
									questionsMap1[question] = answer.(string)
									qnsRemaining++
									questionsMap2[qnsRemaining] = question
								}
							}

							// send first question
							sendQuestion(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)

							botState = "try_quiz_quizAttempt"
							inputExpected = "post-qn"
						}
					} else {
						sendSimpleMsg(
							update.Message.Chat.ID,
							"Quiz with name "+quizName+" not found. Please re-enter your friend's quiz name",
							bot,
						)
					}
				}
			case "try_quiz_quizAttempt":
				switch inputExpected {
				case "post-qn":
					switch update.Message.Text {
					case "Reveal Ans":
						sendAnswer(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
						qnsRemaining--
						inputExpected = "post-ans"

					case "End Quiz":
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
						msg.Text = "Cancelling quiz attempt"
						msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
							RemoveKeyboard: true,
							Selective:      false,
						}

						if _, err := bot.Send(msg); err != nil {
							log.Panic(err)
						}

						botState = "idle"
					}

				case "post-ans":
					inputError := false
					switch update.Message.Text {
					case "Correct":
						scoreInt++
						if qnsRemaining != 0 {
							sendQuestion(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
						}
						inputExpected = "post-qn"
					case "Wrong":
						if qnsRemaining != 0 {
							sendQuestion(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
						}
						inputExpected = "post-qn"

					case "End Quiz":
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
						msg.Text = "Cancelling quiz attempt"
						msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
							RemoveKeyboard: true,
							Selective:      false,
						}

						if _, err := bot.Send(msg); err != nil {
							log.Panic(err)
						}

						botState = "idle"

					default:
						inputError = true
					}

					if qnsRemaining == 0 && !inputError {
						if tryingMyQuiz {
							_, err = client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizName).Update(ctx, []firestore.Update{
								{
									Path:  "score",
									Value: fmt.Sprint(scoreInt) + "/" + fmt.Sprint(numQns),
								},
							})

							if err != nil {
								// Handle any errors in an appropriate way, such as returning them.
								log.Printf("An error has occurred trying to update score to firebase: %s", err)
							}

						}

						// TODO: link to html instead
						// define endMsg based on pass fail
						var endMsg string

						if scoreInt/numQns == 1 {
							endMsg = "Congrats perfect score!"
						} else if float64(scoreInt)/float64(numQns) > float64(0.5) {
							endMsg = "Congrats you passed!"
						} else {
							endMsg = "You failed! Better luck next time."
						}

						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
						msg.Text = "You scored " + fmt.Sprint(scoreInt) + "/" + fmt.Sprint(numQns) + "\n" + endMsg
						msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
							RemoveKeyboard: true,
							Selective:      false,
						}

						if _, err := bot.Send(msg); err != nil {
							log.Panic(err)
						}

						// send score
						botState = "idle"
					}
				default:

				}
			case "add_qns_Qn":
				switch update.Message.Text {
				case "Exit":
					if inputExpected == "ans" {
						delete(questionsMap1, questionText)
					}

					questionsMap1["score"] = "none"

					_, err := client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizName).Set(ctx, questionsMap1, firestore.MergeAll)

					if err != nil {
						// Handle any errors in an appropriate way, such as returning them.
						log.Printf("An error has occurred: %s", err)
					}

					_, err = client.Collection("USERS").Doc(currentUserID).Collection("QUIZZES").Doc(quizName).Update(ctx, []firestore.Update{
						{
							Path:  "numQns",
							Value: numQns,
						},
					})

					if err != nil {
						// Handle any errors in an appropriate way, such as returning them.
						log.Printf("An error has occurred trying to update numQns to firebase: %s", err)
					}

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Questions with answer inputs added to quiz!"
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
						RemoveKeyboard: true,
						Selective:      false,
					}

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

					botState = "add_qns_cancel"

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

			case "add_qns_cancel":
				switch update.Message.Text {
				case "Yes":
					// cancel all changes
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Changes to quiz cancelled."
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
						RemoveKeyboard: true,
						Selective:      false,
					}

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

					botState = "add_qns_Qn"

				default:
				}

			case "remove_qns":
				switch update.Message.Text {
				case "Keep":
					// check for next qn to send
					questionsMap3[questionsMap2[qnsRemaining+1]] = false
					numQns++

					if qnsRemaining == 0 {
						haveTossed := confirmQnsRemove(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap3)

						if haveTossed {
							botState = "remove_qns_confirm"
						} else {
							botState = "idle"
						}

					} else {
						sendQuestionAndAnswerSet(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
						qnsRemaining--
					}

				case "Toss":
					// add to questionsMap3
					questionsMap3[questionsMap2[qnsRemaining+1]] = true

					// check for next qn to send
					if qnsRemaining == 0 {
						haveTossed := confirmQnsRemove(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap3)

						if haveTossed {
							botState = "remove_qns_confirm"
						} else {
							botState = "idle"
						}

					} else {
						sendQuestionAndAnswerSet(update.Message.Chat.ID, qnsRemaining, bot, questionsMap1, questionsMap2)
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

					botState = "remove_qns_cancel"

				default:
				}

			case "remove_qns_cancel":
				switch update.Message.Text {
				case "Yes":
					// cancel all changes
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
					msg.Text = "Changes to quiz cancelled."
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
						RemoveKeyboard: true,
						Selective:      false,
					}

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

					botState = "remove_qns"

				default:
				}

			case "remove_qns_confirm":
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
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
						RemoveKeyboard: true,
						Selective:      false,
					}

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
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardRemove{
						RemoveKeyboard: true,
						Selective:      false,
					}

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
				msg2 := tgbotapi.NewMessage(update.Message.Chat.ID, "")
				msg2.Text = "User not logged in. Please run <strong>/start</strong> to log in user"
				msg2.ParseMode = "HTML"

				if _, err := bot.Send(msg2); err != nil {
					log.Panic(err)
				}
				fmt.Println("BOT STATE INVALID")
			}

		}

	}
}

func Parser(str string) string {
	arr := strings.SplitN(str, " ", 2)
	return arr[1]
}
