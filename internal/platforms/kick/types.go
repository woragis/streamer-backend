package kick

const PlatformName = "kick"

const DefaultPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAq/+l1WnlRrGSolDMA+A8
6rAhMbQGmQ2SapVcGM3zq8ANXjnhDWocMqfWcTd95btDydITa10kDvHzw9WQOqp2
MZI7ZyrfzJuz5nhTPCiJwTwnEtWft7nV14BYRDHvlfqPUaZ+1KR4OCaO/wWIk/rQ
L/TjY0M70gse8rlBkbo2a8rKhu69RQTRsoaf4DVhDPEeSeI5jVrRDGAMGL3cGuyY
6CLKGdjVEM78g3JfYOvDU/RvfqD7L89TZ3iN94jrmWdGz34JNlEI5hqK8dd7C5EF
BEbZ5jgB8s8ReQV8H+MkuffjdAj3ajDDX3DOJMIut1lBrUVD1AaSrGCKHooWoL2e
twIDAQAB
-----END PUBLIC KEY-----`

const (
	HeaderEventType      = "Kick-Event-Type"
	HeaderEventVersion   = "Kick-Event-Version"
	HeaderMessageID      = "Kick-Event-Message-Id"
	HeaderMessageTS      = "Kick-Event-Message-Timestamp"
	HeaderSignature      = "Kick-Event-Signature"
	HeaderSubscriptionID = "Kick-Event-Subscription-Id"
)

const (
	EventChatMessage      = "chat.message.sent"
	EventChannelFollowed  = "channel.followed"
	EventSubNew           = "channel.subscription.new"
	EventSubRenewal       = "channel.subscription.renewal"
	EventSubGifts         = "channel.subscription.gifts"
	EventKicksGifted      = "kicks.gifted"
	EventLivestreamStatus = "livestream.status.updated"
)

type User struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	ChannelSlug string `json:"channel_slug"`
	IsAnonymous bool   `json:"is_anonymous"`
}

type ChatMessagePayload struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
	Sender    User   `json:"sender"`
	Broadcaster User `json:"broadcaster"`
}

type FollowPayload struct {
	Broadcaster User `json:"broadcaster"`
	Follower    User `json:"follower"`
}

type SubscriptionPayload struct {
	Broadcaster User `json:"broadcaster"`
	Subscriber  User `json:"subscriber"`
	Duration    int  `json:"duration"`
}

type SubscriptionGiftsPayload struct {
	Broadcaster User   `json:"broadcaster"`
	Gifter      User   `json:"gifter"`
	Giftees     []User `json:"giftees"`
}

type KicksGiftedPayload struct {
	Broadcaster User `json:"broadcaster"`
	Sender      User `json:"sender"`
	Gift        struct {
		Amount  int    `json:"amount"`
		Name    string `json:"name"`
		Message string `json:"message"`
	} `json:"gift"`
}

type LivestreamStatusPayload struct {
	Broadcaster User   `json:"broadcaster"`
	IsLive      bool   `json:"is_live"`
	Title       string `json:"title"`
}
