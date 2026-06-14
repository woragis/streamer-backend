package youtube

const PlatformName = "youtube"

type SearchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

type VideoListResponse struct {
	Items []struct {
		ID                 string `json:"id"`
		LiveStreamingDetails struct {
			ActiveLiveChatID string `json:"activeLiveChatId"`
		} `json:"liveStreamingDetails"`
	} `json:"items"`
}

type ChatMessagesResponse struct {
	NextPageToken          string `json:"nextPageToken"`
	PollingIntervalMillis  int    `json:"pollingIntervalMillis"`
	Items                  []ChatMessage `json:"items"`
}

type ChatMessage struct {
	ID            string `json:"id"`
	Snippet       ChatSnippet `json:"snippet"`
	AuthorDetails AuthorDetails `json:"authorDetails"`
}

type ChatSnippet struct {
	Type               string `json:"type"`
	DisplayMessage     string `json:"displayMessage"`
	TextMessageDetails struct {
		MessageText string `json:"messageText"`
	} `json:"textMessageDetails"`
	SuperChatDetails struct {
		AmountDisplayString string `json:"amountDisplayString"`
		AmountMicros          string `json:"amountMicros"`
		Currency              string `json:"currency"`
		UserComment           string `json:"userComment"`
	} `json:"superChatDetails"`
	SuperStickerDetails struct {
		AmountDisplayString string `json:"amountDisplayString"`
		AmountMicros          string `json:"amountMicros"`
		Currency              string `json:"currency"`
		SuperStickerMetadata  struct {
			AltText string `json:"altText"`
		} `json:"superStickerMetadata"`
	} `json:"superStickerDetails"`
}

type AuthorDetails struct {
	ChannelID   string `json:"channelId"`
	DisplayName string `json:"displayName"`
}
