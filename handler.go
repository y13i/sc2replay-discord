package main

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/icza/s2prot/rep"
)

const (
	unknownEmoji = ":question:"
	victoryEmoji = ":trophy:"
	defeatEmoji  = ":skull:"
	tieEmoji     = ":infinity:"
)

func handleMessageCreateSafe(s *discordgo.Session, m *discordgo.MessageCreate) {
	err := handleMessageCreate(s, m)
	if err != nil {
		logger.Error("Error handling message, ", err)
	}
}

func handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if m.Author.ID == s.State.User.ID {
		return nil
	}

	logger.Debug(m)

	if len(m.Message.Attachments) == 0 {
		logger.Debug("No attachments")
		return nil
	}

	for _, attachment := range m.Message.Attachments {
		logger.Debug(attachment)

		if !strings.HasSuffix(attachment.Filename, ".SC2Replay") {
			logger.Debug("Not a replay file")
			continue
		}

		logger.Info("Replay file " + attachment.Filename + " detected on message: " + buildMessageURL(m.GuildID, m.ChannelID, m.ID))

		fileURL := attachment.URL
		logger.Debug("Replay file URL: ", fileURL)

		resp, err := http.Get(fileURL)
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

		version := fmt.Sprintf(
			"%v.%v.%v",
			replay.Header.Version().Struct["major"],
			replay.Header.Version().Struct["minor"],
			replay.Header.Version().Struct["revision"],
		)

		embed := &discordgo.MessageEmbed{
			Title: attachment.Filename,
			URL:   fileURL,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name: "Version",
					Value: fmt.Sprintf(
						"[%v](https://liquipedia.net/starcraft2/index.php?search=%v).%v",
						version,
						url.QueryEscape(fmt.Sprintf("Patch %v", version)),
						replay.Header.Version().Struct["build"],
					),
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
					Name: "Map",
					Value: fmt.Sprintf("[%s](https://liquipedia.net/starcraft2/index.php?search=%s)", replay.Details.Title(),
						url.QueryEscape(replay.Details.Title())),
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
				var searchName string
				if strings.Contains(player.Name, "&gt;") {
					s := strings.Split(player.Name, "&gt;")
					searchName = s[len(s)-1]
				} else {
					searchName = player.Name
				}

				playerStrings[playerIndex] = fmt.Sprintf(
					"[%s](https://sc2pulse.nephest.com/sc2/?type=search&name=%s#search) (%s)",
					html.UnescapeString(player.Name),
					searchName,
					player.RaceString(),
				)

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

		logger.Info("Message sent: " + buildMessageURL(m.GuildID, m.ChannelID, newMessage.ID))

		err = s.MessageReactionAdd(m.ChannelID, newMessage.ID, chartEmoji)
		if err != nil {
			logger.Error("Error adding reactions to message, ", err)
			logger.Debug(err)
		}
	}

	return nil
}

func buildMessageURL(guildID string, channelID string, messageID string) string {
	return fmt.Sprintf(
		"https://discord.com/channels/%s/%s/%s", guildID, channelID, messageID,
	)
}
