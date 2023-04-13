package nostr

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
	"unsafe"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip42"
	"github.com/pkg/errors"
)

type (
	// TODO: break out into individual files

	// Server is a persistent client keeping open connections to relays.
	Server struct {
		connectedRelays     map[string][]*nostr.Relay
		clientSubscriptions map[string][]*Subscription

		mutex sync.RWMutex
	}

	NostrEvent = *nostr.Event

	// Client for nostr protocol
	Client struct {
		// Reference to the server we are using
		server *Server
		// Secret key
		sk string
		// Public key
		pk string
	}

	// Subscription for events on a relay
	Subscription struct {
		id     string
		events chan *nostr.Event
		buffer *eventBuffer
		cancel context.CancelFunc
	}
)

const (
	// size of a subscription id
	SUB_ID_LENGTH = 10
)

var (
	// The default duration to try and connect to a relay
	relayConnectTimeout = time.Second * 5
	// The default duration to try and authenticate to a relay
	relayAuthTimeout = time.Second * 5

	// ErrRelayAuthFailed indicates the authentication on a relay completed, but failed
	ErrRelayAuthFailed = errors.New("Failed to authenticate to the relay")
	// ErrRelayAuthTimeout indicates the authentication on a relay did not complete in time
	ErrRelayAuthTimeout = errors.New("Timeout authenticating to the relay")
	// ErrFailedToPublishEvent indicates the event could not be published to the relay
	ErrFailedToPublishEvent = errors.New("Failed to publish event to relay")
	/// ErrNoRelayConnected inidcates that we try to perform an action on a realay, but we aren't connected to any.
	ErrNoRelayConnected = errors.New("No relay connected currently")
)

// NewServer managing relay connections and subscriptions for possibly different peers
func NewServer() *Server {
	return &Server{
		connectedRelays:     make(map[string][]*nostr.Relay),
		clientSubscriptions: make(map[string][]*Subscription),
	}
}

// NewClient for a server, authenticated by the private key of the client. Private key is passed as hex bytes
func (s *Server) NewClient(sk string) (*Client, error) {
	pk, err := nostr.GetPublicKey(sk)
	if err != nil {
		return nil, errors.Wrap(err, "could not get public key from provided private key")
	}

	return &Client{
		server: s,
		sk:     sk,
		pk:     pk,
	}, nil
}

func GenerateKeyPair() string {
	return nostr.GeneratePrivateKey()
}

// Manage an active relay connection for a client
func (s *Server) manageRelay(id string, relay *nostr.Relay) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, r := range s.connectedRelays[id] {
		if r == relay {
			return
		}
	}
	s.connectedRelays[id] = append(s.connectedRelays[id], relay)
}

// Get the list of all relays managed for the given client. These relays must have been
// added first through the manageRelay method
func (s *Server) clientRelays(id string) []*nostr.Relay {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.connectedRelays[id]
}

// Manage an active subscription for a client
func (s *Server) manageSubscription(id string, sub *Subscription) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, oldSub := range s.clientSubscriptions[id] {
		if oldSub.id == sub.id {
			return
		}
	}

	s.clientSubscriptions[id] = append(s.clientSubscriptions[id], sub)
}

// Get a list of all the subscriptions being managed for a client
func (s *Server) subscriptions(id string) []*Subscription {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.clientSubscriptions[id]
}

// Remove an active subscription for a client by its ID
func (s *Server) removeSubscription(id string, subID string) *Subscription {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i, sub := range s.clientSubscriptions[id] {
		if sub.id == subID {
			s.clientSubscriptions[id] = append(s.clientSubscriptions[id][:i], s.clientSubscriptions[id][i+1:]...)
			return sub
		}
	}

	return nil
}

// Id of the client, this is the enocded public key in NIP19 format
func (c *Client) Id() string {
	id, err := nip19.EncodePublicKey(c.pk)
	if err != nil {
		panic(fmt.Sprintf("Can't encode public key, although this was previously validated. This should not happen (%s)", err))
	}

	return id
}

func (c *Client) ConnectRelay(ctx context.Context, relayURL string) error {
	ctxConnect, cancelFuncConnect := context.WithTimeout(ctx, relayConnectTimeout)
	defer cancelFuncConnect()

	relay, err := nostr.RelayConnect(ctxConnect, relayURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to the provided relay")
	}

	// Add relay to the list of managed relays
	c.server.manageRelay(c.Id(), relay)

	return nil
}

// ConnectAuthRelay connect and authenticates to a NIP42 authenticated relay
func (c *Client) ConnectAuthRelay(ctx context.Context, relayURL string) error {
	ctxConnect, cancelFuncConnect := context.WithTimeout(ctx, relayConnectTimeout)
	defer cancelFuncConnect()

	relay, err := nostr.RelayConnect(ctxConnect, relayURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to the provided relay")
	}

	// Wait for the challenge send by the relay
	challenge := <-relay.Challenges

	event := nip42.CreateUnsignedAuthEvent(challenge, c.pk, relayURL)
	event.Sign(c.sk)

	ctxAuth, cancelFuncAuth := context.WithTimeout(ctx, relayAuthTimeout)
	defer cancelFuncAuth()

	auth_status, err := relay.Auth(ctxAuth, event)
	if err != nil {
		return errors.Wrap(err, "could not authenticate to relay")
	}

	if auth_status != nostr.PublishStatusSucceeded {
		return ErrRelayAuthFailed
	}

	// Add relay to the list of managed relays
	c.server.manageRelay(c.Id(), relay)

	return nil
}

// Add function to publish events to a set of relays
func (c *Client) PublishEventToRelays(ctx context.Context, tags []string, content string) error {
	c.server.mutex.RLock()
	defer c.server.mutex.RUnlock()

	relays := c.server.connectedRelays[c.Id()]
	if len(relays) == 0 {
		return errors.New("No relays connected")
	}

	ev := nostr.Event{
		PubKey:    c.pk,
		CreatedAt: time.Now(),
		Kind:      1,
		Tags:      make(nostr.Tags, len(tags)),
		Content:   content,
	}

	// calling Sign sets the event ID field and the event Sig field
	ev.Sign(c.sk)

	for _, relay := range c.server.connectedRelays[c.Id()] {
		status, err := relay.Publish(ctx, ev)
		if err != nil {
			return errors.Wrap(err, ErrFailedToPublishEvent.Error())
		}

		if status == nostr.PublishStatusFailed {
			return ErrFailedToPublishEvent
		}
	}

	return nil
}

// Subscribe to events on a relay
func (c *Client) SubscribeRelays(ctx context.Context) (string, error) {
	relays := c.server.clientRelays(c.Id())
	if len(relays) == 0 {
		return "", ErrNoRelayConnected
	}

	var filters nostr.Filters
	if _, v, err := nip19.Decode(c.Id()); err == nil {
		pub := v.(string)
		filters = []nostr.Filter{{
			Kinds:   []int{1},
			Authors: []string{pub},
			Limit:   1000,
		}}
	} else {
		errors.New("could not create client filters")
	}

	eventChan := make(chan *nostr.Event)
	ctx, cancel := context.WithCancel(context.Background())

	for _, relay := range relays {
		sub, err := relay.Subscribe(ctx, filters)
		if err != nil {
			cancel()
			return "", errors.Wrapf(err, "could not subscribe to relay %s", relay.URL)
		}

		go func() {
			for ev := range sub.Events {
				eventChan <- ev
			}
		}()
	}

	buf := newEventBuffer()
	go func() {
		select {
		case ev := <-eventChan:
			buf.push(ev)
		case <-ctx.Done():
			return
		}
	}()

	sub := &Subscription{
		id:     randString(SUB_ID_LENGTH),
		events: eventChan,
		buffer: buf,
		cancel: cancel,
	}

	c.server.manageSubscription(c.Id(), sub)

	return sub.id, nil
}

// Get all historic events on active subscriptions for the client.
// Note that only a limited amount of events are kept. If the actual client waits
// too long to call this, events might be dropped.
func (c *Client) GetEvents() []nostr.Event {
	subs := c.server.subscriptions(c.Id())
	var events []nostr.Event
	for _, sub := range subs {
		events = append(events, sub.buffer.take()...)
	}
	return events
}

// Get the ID's of all active subscriptions
func (c *Client) SubscriptionIds() []string {
	subs := c.server.subscriptions(c.Id())
	var ids []string
	for _, sub := range subs {
		ids = append(ids, sub.id)
	}
	return ids
}

// CloseSubscription managed by the server for this client, based on its ID.
func (c *Client) CloseSubscription(id string) {
	sub := c.server.removeSubscription(c.Id(), id)
	if sub != nil {
		sub.Close()
	}
}

// Close an open subscription
func (s *Subscription) Close() {
	s.cancel()
	close(s.events)
}

// go random string, source: https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randString(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

// end random string code copy
