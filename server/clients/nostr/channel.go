package nostr

import (
	"context"
	"encoding/json"

	"github.com/nbd-wtf/go-nostr"
	"github.com/pkg/errors"
)

type (
	Channel struct {
		Name    string `json:"name"`
		About   string `json:"about"`
		Picture string `json:"picture"`
	}

	ChannelMessage struct {
		// Content of the message
		Content string `json:"content"`
		// ReplyTo is either the ID of a message to reply to, or the ID of the channel create message of the channel to post in
		// if this is a root message in the channel
		ReplyTo string `json:"replyTo"`
	}
)

const (
	// kindCreateChannel creates a channel
	kindCreateChannel = 40
	// kindSetChannelMetadata updates channel metadata
	kindSetChannelMetadata = 41
	// kindCreateChannelMessage creates a message in a channel
	kindCreateChannelMessage = 42
	// kindHideChannelMessage hides a message in the channel
	kindHideChannelMessage = 43
	// kindMuteChannelUser mutes a channel user for the sending user
	kindMuteChanneluser = 44
)

// CreateChannel creates a new channel
func (c *Client) CreateChannel(ctx context.Context, tags []string, content Channel) error {
	if content.Name == "" {
		return errors.New("Channel must have a name")
	}
	marshalledContent, err := json.Marshal(content)
	if err != nil {
		return errors.Wrap(err, "could not encode metadata")
	}
	return c.publishEventToRelays(ctx, kindCreateChannel, [][]string{tags}, string(marshalledContent))
}

// UpdateChannelMetadata updates the channel metdata. ChannelID is the event ID of the create channel event used to create the channel to update
func (c *Client) UpdateChannelMetadata(ctx context.Context, tags []string, channelID string, content Channel) error {
	if content.Name == "" {
		return errors.New("Channel must have a name")
	}
	marshalledContent, err := json.Marshal(content)
	if err != nil {
		return errors.Wrap(err, "could not encode metadata")
	}
	return c.publishEventToRelays(ctx, kindSetChannelMetadata, [][]string{tags, {"e", channelID}}, string(marshalledContent))
}

// CreateChannelMessage creates a message in channel. If replyTo is the empty string, it is marked as a root
func (c *Client) CreateChannelMessage(ctx context.Context, tags []string, message ChannelMessage) error {
	if message.Content == "" {
		return errors.New("Refusing to submit empty message")
	}
	return c.publishEventToRelays(ctx, kindSetChannelMetadata, [][]string{tags, {"e", message.ReplyTo}}, message.Content)
}

// HideMessage marks a message as hidden for the user. It should be noted that properly handling this is mostly up to the clients
func (c *Client) HideMessage(ctx context.Context, tags []string, messageID string, content string) error {
	return c.publishEventToRelays(ctx, kindSetChannelMetadata, [][]string{tags, {"e", messageID}}, content)
}

// MuteUser marks a user as muted for the current user. It should be noted that properly handling this is mostly up to the clients.
// The user to mute is identified by it's pubkey
func (c *Client) MuteUser(ctx context.Context, tags []string, user string, content string) error {
	return c.publishEventToRelays(ctx, kindSetChannelMetadata, [][]string{tags, {"p", user}}, content)
}

func (c *Client) SubscribeChannelCreation() (string, error) {
	var filters nostr.Filters
	filters = []nostr.Filter{{
		Kinds: []int{nostr.KindChannelCreation},
		Limit: DEFAULT_LIMIT,
	}}

	return c.subscribeWithFiler(filters)
}

// SubscribeChannelMessages subsribes to messages which are a reply to the given chanMessageId
func (c *Client) SubscribeChannelMessages(chanMessageId string) (string, error) {
	var filters nostr.Filters
	filters = []nostr.Filter{{
		Kinds: []int{nostr.KindChannelMessage},
		Limit: DEFAULT_LIMIT,
		Tags:  nostr.TagMap{"e": []string{chanMessageId}},
	}}

	return c.subscribeWithFiler(filters)
}

func (c *Client) FetchChannelCreation() ([]RelayEvent, error) {
	var filters nostr.Filters
	filters = []nostr.Filter{{
		Kinds: []int{nostr.KindChannelCreation},
		Limit: DEFAULT_LIMIT,
	}}

	return c.fetchEventsWithFilter(filters)
}

// SubscribeChannelMessages subsribes to messages which are a reply to the given chanMessageId
func (c *Client) FetchChannelMessages(chanMessageId string) ([]RelayEvent, error) {
	var filters nostr.Filters
	filters = []nostr.Filter{{
		Kinds: []int{nostr.KindChannelMessage},
		Limit: DEFAULT_LIMIT,
		Tags:  nostr.TagMap{"e": []string{chanMessageId}},
	}}

	return c.fetchEventsWithFilter(filters)
}
