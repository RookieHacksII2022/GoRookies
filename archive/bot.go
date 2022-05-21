package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Some error occured. Err: %s", err)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_API-TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		}

		// Create a new MessageConfig. We don't have text yet,
		// so we leave it empty.
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		// Extract the command from the Message.
		switch update.Message.Command() {
		case "help":
			msg.Text = "I understand the following commands: \n" +
				"/help - give instructions\n" +
				"/sayhi - gogo says hi\n" +
				"/status - gogo is okay\n" +
				"/time - give current date and time\n" +
				"/copycat - echos back"
		case "sayhi":
			msg.Text = "Hi :)"
		case "status":
			msg.Text = "I'm ok."
		case "time":
			dt := time.Now()
			msg.Text = "The current time and date now is " + dt.String()
		case "copycat":
			msg.Text = "Echo back to you " + ParserCopycat(update.Message.Text)
		default:
			msg.Text = "I don't know that command"
		}

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}

func ParserCopycat(str string) string {
	return strings.Replace(str, "/copycat ", "", 1)
}
