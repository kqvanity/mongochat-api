package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leverly/ChatGLM/client"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var conversationID convID

type Message struct {
	Message string `json:"message"`
}
type DeltaMsg struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type FinishedMsg struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type ReferencesMsg struct {
	Type string              `json:"type"`
	Data []ReferencesMsgData `json:"data"`
}
type ReferencesMsgData struct {
	Title    string `json:"title"`
	Url      string `json:"url"`
	Metadata struct {
		SourceName string   `json:"sourceName"`
		Tags       []string `json:"tags"`
		SourceType string   `json:"sourceType"`
	} `json:"metadata"`
}

// TODO: maybe embed it instead? i'm not gonna use it elsewhere
type ConversationSession struct {
	ID        string   `json:"_id"`
	CreatedAt int64    `json:"created_at"`
	Messages  []string `json:"messages"`
}

func applyHeaders(r *http.Request) {
	r.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0")
	r.Header.Set("Accept-Language", "en-US,en;q=0.5")
	//r.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	r.Header.Set("Referer", "https://www.mongodb.com/")
	r.Header.Set("content-type", "application/json")
	r.Header.Set("x-request-origin", "https://www.mongodb.com/docs/manual/reference/method/db.collection.findAndModify/")
	r.Header.Set("Origin", "https://www.mongodb.com")
	r.Header.Set("DNT", "1")
	r.Header.Set("Connection", "keep-alive")
	r.Header.Set("Sec-Fetch-Dest", "empty")
	r.Header.Set("Sec-Fetch-Mode", "cors")
	r.Header.Set("Sec-Fetch-Site", "same-site")
}

func createConversationSession() (convID, error) {

	baseUrl := "https://knowledge.mongodb.com/api/v1/conversations"
	request, err := http.NewRequest("POST", baseUrl, nil)
	if err != nil {
		return "", err
	}
	applyHeaders(request)
	request.Header.Set("Sec-GPC", "1")
	request.Header.Set("Priority", "u=4")
	request.Header.Set("Content-Length", "0")
	request.Header.Set("TE", "trailers")

	// TODO: use the same http client
	c := http.Client{}

	res, err := c.Do(request)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error creating conversation session, %s", res.Body)
	}

	var convSession ConversationSession
	defer res.Body.Close()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(responseBody, &convSession)
	if err != nil {
		return "", err
	}
	return convID(convSession.ID), nil
}

func mongoMain(m MongdoChatbotRequest, callback func(event client.StreamEvent)) error {
	msg := Message{
		//Message: "How is the weather today",
		Message: m.Message,
	}
	// convert the msg to json
	jsonMsg, err := json.Marshal(msg)

	// TODO: use stream instead of hardcoding it in the url
	baseUrl := fmt.Sprintf("https://knowledge.mongodb.com/api/v1/conversations/%s/messages?stream=true", m.ConversationID)
	request, err := http.NewRequest("POST", baseUrl, bytes.NewReader(jsonMsg))
	if err != nil {
		return err
	}

	applyHeaders(request)
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Priority", "u=0")
	request.Header.Set("TE", "trailers")

	c := http.Client{}
	response, err := c.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		// TODO: Check the type of response body, and if it's the 50Msg limit, then create a new conversation session
		return fmt.Errorf("error reading response body %s", string(responseBody))
	}
	//fmt.Println(string(responseBody))

	// the stream is compressed, so we need to decompress it
	reader := client.NewEventStreamReader(response.Body, -1)
	for {
		event, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error reading stream: %w", err)
		}
		callback(*event)
	}
}

func cliClient() {
	cliStreamCallback := &CLiStreamCallback{}
	mongoRequest := MongdoChatbotRequest{
		Message: "Which company made you",
		Stream:  true,
		// TODO: Pass convo id dynamically
		ConversationID: "",
	}
	// the process function should accept StreamCallback, but I sometimes want to either print the result or do something else
	// so I use a callback function to do that.
	// You can also use a closure to do that.
	mongoMain(mongoRequest, func(event client.StreamEvent) {
		err := process(&event, cliStreamCallback)
		if err != nil {
			log.Fatal(err)
		}
	})
}

func process(event *client.StreamEvent, callback StreamEventCallback) error {
	var data map[string]interface{}
	err := json.Unmarshal(event.Data, &data)
	if err != nil {
		return err
	}
	switch data["type"] {
	case "delta":
		var deltaMsg DeltaMsg
		err := json.Unmarshal(event.Data, &deltaMsg)
		if err != nil {
			return err
		}
		callback.OnDelta(deltaMsg)
	case "references":
		var referencesMsg ReferencesMsg
		err := json.Unmarshal(event.Data, &referencesMsg)
		if err != nil {
			return err
		}
		callback.OnReferences(referencesMsg)
	case "finished":
		var finishedMsg FinishedMsg
		err := json.Unmarshal(event.Data, &finishedMsg)
		if err != nil {
			return err
		}
		callback.OnFinished(finishedMsg)
	}

	return nil
}

type StreamEventCallback interface {
	OnDelta(data DeltaMsg)
	OnFinished(data FinishedMsg)
	OnReferences(data ReferencesMsg)
}

type CLiStreamCallback struct {
}

func (e *CLiStreamCallback) OnDelta(data DeltaMsg) {
	fmt.Print(data.Data)
}

func (e *CLiStreamCallback) OnFinished(msg FinishedMsg) {}
func (s *CLiStreamCallback) OnReferences(r ReferencesMsg) {
	resData := strings.Join([]string{"\n", "## References\n- ", parseRefMsgData(r.Data)}, "")
	fmt.Println(resData)
}

type RestStreamCallback struct {
	c *gin.Context
	f http.Flusher
	m OpenaiRequest
}

func (e *RestStreamCallback) OnDelta(data DeltaMsg) {
	e.c.SSEvent("message", fmt.Sprintf(`{"id":"chatcmpl-%s","object":"chat.completion.chunk","created":%d,"model":"%s","choices":[{"index":0,"delta":{"content":"%s"},"finish_reason":null}]}`,
		uuid.New().String(), time.Now().Unix(), "mongodb-1", escapeJSON(data.Data)))
	e.f.Flush()
}

func (e *RestStreamCallback) OnFinished(msg FinishedMsg) {
	e.c.SSEvent("message", fmt.Sprintf(`{"id":"chatcmpl-%s","object":"chat.completion.chunk","created":%d,"model":"%s","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
		uuid.New().String(), time.Now().Unix(), "mongodb-1"))
	e.f.Flush()
}

func (s *RestStreamCallback) OnReferences(r ReferencesMsg) {
	refs := strings.Join([]string{"\\n", "## References\\n- ", parseRefMsgData(r.Data)}, "")
	s.c.SSEvent("message", fmt.Sprintf(`{"id":"chatcmpl-%s","object":"chat.completion.chunk","created":%d,"model":"%s","choices":[{"index":0,"delta":{"content":"%s"},"finish_reason":null}]}`,
		uuid.New().String(), time.Now().Unix(), "mongodb-1", escapeJSON(refs)))
	s.f.Flush()
}

// FIXME: ai doesn't parse the output quite nicely with escape chars and whatnot
func escapeJSON(s string) string {
	//if !strings.Contains(s, " ") {
	//	//fmt.Println("contains", s)
	//	return s
	//}
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

func parseRefMsgData(rData []ReferencesMsgData) string {
	var urls []string
	for _, dataItem := range rData {
		urlMd := fmt.Sprintf("[%s](%s)", dataItem.Title, dataItem.Url)
		// convert urlMd to url
		urls = append(urls, urlMd)
	}
	var msg string = strings.Join(urls, "\\n- ")
	return msg
}

// make an http server compatible with openai /chat/completions api endpoint
// refactor the following to use net/http instead of gin
func MongoRestClient() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	//r.Use(cors.Default())

	var err error

	conversationID, err = createConversationSession()
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(conversationID)

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to Claude2OpenAI, Made by Vincent Yang. https://github.com/missuo/claude2openai",
		})
	})

	r.POST("/v1/chat/completions", cHandler)
	//r.GET("/v1/models", modelsHandler)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    http.StatusNotFound,
			"message": "Path not found",
		})
	})
	err = r.Run(":8800")
	if err != nil {
		log.Fatal(err)
	}
}
func cHandler(c *gin.Context) {
	var req OpenaiRequest
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": fmt.Errorf("invalid request body, %w", err).Error(),
		})
		return
	}

	restCallback := RestStreamCallback{
		c: c,
		f: flusher,
		m: OpenaiRequest{
			Model:    req.Model,
			Messages: req.Messages,
			Stream:   true,
		},
	}
	//fmt.Println(req.Messages)

	mongoRequest := MongdoChatbotRequest{
		Message:        req.Messages[0].Content,
		Stream:         true,
		ConversationID: conversationID,
	}
	err := mongoMain(mongoRequest, func(event client.StreamEvent) {
		err := process(&event, &restCallback)
		if err != nil {
			c.Error(err)
		}
	})
	if err != nil {
		c.Error(err)
	}
	//fmt.Println(req)
}

type OpenaiRequest struct {
	Model    string `json:"model"`
	Messages []RMsg `json:"messages"`
	Stream   bool   `json:"stream"`
}
type RMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MongdoChatbotRequest struct {
	Message        string `json:"message"`
	Stream         bool   `json:"stream"`
	ConversationID convID `json:"conversation_id"`
}
type convID string
