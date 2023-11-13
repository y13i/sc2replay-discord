package main

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/icza/s2prot/rep"
	"github.com/k0kubun/pp"
	"go.uber.org/zap"
)

const (
	entryPointPid = 1
	unknownEmoji  = ":question:"
	victoryEmoji  = ":trophy:"
	defeatEmoji   = ":skull:"
	tieEmoji      = ":infinity:"
)

var (
	logger Logger
)

type Logger struct {
	*zap.SugaredLogger
}

func (l Logger) Debug(args ...interface{}) {
	l.Debugf("", "\n"+pp.Sprint(args))
}

func getLogger(isProd bool) Logger {
	var (
		_logger *zap.Logger
		err     error
	)

	if isProd {
		_logger, err = zap.NewProduction()
	} else {
		_logger, err = zap.NewDevelopment()
	}

	if err != nil {
		panic(err)
	}

	return Logger{_logger.Sugar()}
}

func main() {
	isProd := os.Getenv("IS_PROD") == "true" || os.Getpid() == entryPointPid
	logger = getLogger(isProd)
	defer logger.Sync()

	logger.Info("Started")
	logger.Debug(os.Environ())

	discordToken := os.Getenv("DISCORD_TOKEN")

	if discordToken == "" {
		logger.Fatal("No DISCORD_TOKEN provided")
	}

	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		logger.Fatal("Cannot create Discord session, ", err)
	}

	dg.AddHandler(handleMessageCreate)
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

func handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	logger.Debug(m)

	if len(m.Message.Attachments) == 0 {
		logger.Debug("No attachments")
		return
	}

	filename := m.Message.Attachments[0].Filename
	logger.Debug(filename)

	if strings.HasSuffix(filename, ".SC2Replay") {
		logger.Info("Replay file detected, ", filename)

		url := m.Message.Attachments[0].URL
		logger.Debug("URL: ", url)

		resp, err := http.Get(url)
		if err != nil {
			logger.Error("Error requesting attachment, ", err)
			logger.Debug(err)
		}
		// logger.Debug(resp)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			logger.Error("Error reading response body, ", err)
			logger.Debug(err)
		}
		// logger.Debug(body)

		replay, err := rep.New(bytes.NewReader(body))
		if err != nil {
			logger.Error("Error opening replay, ", err)
			logger.Debug(err)
		}
		// logger.Debug(replay)
		logger.Debug(replay.Header)
		logger.Debug(replay.Details)
		// logger.Debug(replay.InitData)
		logger.Debug(replay.InitData.GameDescription)
		logger.Debug(replay.InitData.GameDescription.Region())
		logger.Debug(replay.Details.Players())

		embed := &discordgo.MessageEmbed{
			Title: m.Message.Attachments[0].Filename,
			URL:   url,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Version",
					Value: replay.Header.VersionString(),
				},
				{
					Name:  "Region",
					Value: replay.InitData.GameDescription.Region().Code,
				},
				{
					Name:  "Time",
					Value: replay.Details.Time().String(),
				},
				{
					Name:  "Duration",
					Value: replay.Header.Duration().String(),
				},
				{
					Name:  "Map",
					Value: replay.Details.Title(),
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Reactions:\n" + chartEmoji + " - Show analysis",
			},
		}

		teams := make([][]rep.Player, replay.InitData.GameDescription.MaxTeams())
		for _, player := range replay.Details.Players() {
			teams[player.TeamID()] = append(teams[player.TeamID()], player)
		}
		logger.Debug(teams)

		for teamIndex, players := range teams {
			if len(players) == 0 {
				continue
			}

			playerStrings := make([]string, len(players))

			result := unknownEmoji

			for playerIndex, player := range players {
				playerStrings[playerIndex] = fmt.Sprintf("%s (%s)", html.UnescapeString(player.Name), player.RaceString())

				if result == unknownEmoji {
					switch player.Result().Enum.Name {
					case rep.ResultUnknown.Name:
						result = unknownEmoji
					case rep.ResultVictory.Name:
						result = victoryEmoji
					case rep.ResultDefeat.Name:
						result = defeatEmoji
					case rep.ResultTie.Name:
						result = tieEmoji
					}
				}
			}

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  fmt.Sprintf("Team #%d %s", teamIndex+1, result),
				Value: strings.Join(playerStrings, "\n"),
			})
		}

		logger.Debug(embed)

		newMessage, err := s.ChannelMessageSendEmbed(m.ChannelID, embed)
		if err != nil {
			logger.Error("Error sending message, ", err)
			logger.Debug(err)
		}
		logger.Debug(newMessage)

		err = s.MessageReactionAdd(m.ChannelID, newMessage.ID, chartEmoji)
		if err != nil {
			logger.Error("Error adding reactions to message, ", err)
			logger.Debug(err)
		}
	}
}
