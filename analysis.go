package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	analysisFieldName    = "Analysis"
	chartEmoji           = "📊"
	getReplayEndpoint    = "https://sc2replaystats.com/replay/"
	uploadReplayEndpoint = "https://drop.sc/uploadReplay"
)

var (
	cachedMe *discordgo.User
)

type DropScResponse struct {
	Success                bool   `json:"success"`
	HasReplayBeenProcessed bool   `json:"has_replay_been_processed"`
	ReplayID               string `json:"replay_id"`
}

func handleMessageReactionSafe(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	err := handleMessageReaction(s, r)
	if err != nil {
		logger.Error("Error handling reaction, ", err)
	}
}

func handleMessageReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) error {
	me, err := getMe(s)

	if err != nil {
		logger.Error("error getting me, ", err)
		return err
	}

	if r.UserID == me.ID {
		// Ignore own reaction
		return nil
	}

	message, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		logger.Error("error retrieving message when handling reaction, ", err)
		return err
	}

	if message == nil {
		logger.Debug("message was nil when handling reaction")
		return nil
	}

	if message.Author.ID != me.ID {
		// Ignore reaction to message from different author
		return nil
	}

	switch r.Emoji.Name {
	case chartEmoji:
		if len(message.Embeds) == 0 {
			return errors.New("reacted message has no embeds")
		}

		embed := message.Embeds[0]

		embedUrl, err := url.Parse(embed.URL)
		if err != nil {
			logger.Error("Error parsing embed URL, ", err)
			return err
		}

		if !strings.HasSuffix(embedUrl.Path, ".SC2Replay") {
			return errors.New("embed url for reacted message is wrong format")
		}

		if len(embed.Fields) == 0 {
			return errors.New("reacted message embed has no fields")
		}

		if embed.Fields[len(embed.Fields)-1].Name == analysisFieldName {
			// ignore message which already has analysis
			return nil
		}

		_, filename := path.Split(embedUrl.Path)

		resp, err := http.Get(embedUrl.String())
		if err != nil {
			logger.Error("Error requesting attachment, ", err)
			return err
		}

		defer resp.Body.Close()

		var b bytes.Buffer
		mw := multipart.NewWriter(&b)
		fw, err := mw.CreateFormFile("files[]", filename)
		if err != nil {
			logger.Error("Error creating upload request, ", err)
			return err
		}

		_, err = io.Copy(fw, resp.Body)
		if err != nil {
			logger.Error("Error copying upload data, ", err)
			return err
		}

		mw.Close()

		contentType := "multipart/form-data; boundary=" + mw.Boundary()

		resp, err = http.Post(uploadReplayEndpoint, contentType, &b)
		if err != nil {
			logger.Error("Error uploading replay, ", err)
			return err
		}

		defer resp.Body.Close()

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Error reading response, ", err)
			return err
		}

		var dropScResponse DropScResponse
		err = json.Unmarshal(content, &dropScResponse)
		if err != nil {
			logger.Error("Error unmarshalling drop sc response, ", err)
			return err
		}

		if !dropScResponse.Success {
			return errors.New("failed drop.sc upload")
		}

		replayLink := discordgo.MessageEmbedField{
			Name:  analysisFieldName,
			Value: getReplayEndpoint + dropScResponse.ReplayID,
		}
		embed.Fields = append(embed.Fields, &replayLink)

		_, err = s.ChannelMessageEditEmbed(message.ChannelID, message.ID, embed)
		if err != nil {
			logger.Debug("Error editing embed, ", err)
			return err
		}

	default:
		// Ignore unrecognised emoji
		return nil
	}

	return nil
}

func getMe(s *discordgo.Session) (*discordgo.User, error) {
	if cachedMe != nil {
		return cachedMe, nil
	}

	cachedMe, err := s.User("@me")
	if err != nil {
		return nil, err
	}

	if cachedMe == nil {
		return nil, errors.New("me data was nil")
	}

	return cachedMe, nil
}
