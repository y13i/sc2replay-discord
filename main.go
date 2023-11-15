package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const (
	entryPointPid      = 1
	envVarDiscordToken = "DISCORD_TOKEN"
)

var (
	logger Logger
)

func main() {
	isProd := os.Getenv("IS_PROD") == "true" || os.Getpid() == entryPointPid
	logger = getLogger(isProd)
	defer logger.Sync()

	logger.Info("Started")
	logger.Debug(os.Environ())

	discordToken := os.Getenv(envVarDiscordToken)

	if discordToken == "" {
		logger.Fatal("No " + envVarDiscordToken + " provided")
	}

	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		logger.Fatal("Cannot create Discord session, ", err)
	}

	dg.AddHandler(handleMessageCreateSafe)
	dg.AddHandler(handleMessageReactionSafe)

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions

	err = dg.Open()
	if err != nil {
		logger.Fatal("Cannot open connection, ", err)
	}

	logger.Info("Bot is now running.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	dg.Close()
}
