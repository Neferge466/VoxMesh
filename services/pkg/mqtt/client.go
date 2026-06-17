package mqtt

import (
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps paho.mqtt.golang with reconnect and convenience helpers.
type Client struct {
	client mqtt.Client
	opts   *mqtt.ClientOptions
	mu     sync.Mutex
}

// ClientConfig holds connection parameters.
type ClientConfig struct {
	BrokerURL   string
	ClientID    string
	Username    string
	Password    string
	KeepAlive   time.Duration
	WillTopic   string
	WillPayload []byte
	WillQoS     byte
	WillRetain  bool
}

// NewClient creates and connects an MQTT client. Blocks until connected.
func NewClient(cfg ClientConfig, connectHandler mqtt.OnConnectHandler, disconnectHandler mqtt.ConnectionLostHandler) (*Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetKeepAlive(cfg.KeepAlive).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(10 * time.Second).
		SetConnectRetry(true).
		SetCleanSession(true)

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}
	if cfg.WillTopic != "" {
		opts.SetWill(cfg.WillTopic, string(cfg.WillPayload), cfg.WillQoS, cfg.WillRetain)
	}
	if connectHandler != nil {
		opts.SetOnConnectHandler(connectHandler)
	}
	if disconnectHandler != nil {
		opts.SetConnectionLostHandler(disconnectHandler)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(15 * time.Second) {
		return nil, fmt.Errorf("mqtt connect timeout")
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &Client{client: client, opts: opts}, nil
}

// Subscribe subscribes to a topic with the given QoS. Calls handler on each message.
func (c *Client) Subscribe(topic string, qos byte, handler mqtt.MessageHandler) error {
	token := c.client.Subscribe(topic, qos, handler)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("subscribe timeout for %s", topic)
	}
	return token.Error()
}

// Unsubscribe removes a subscription.
func (c *Client) Unsubscribe(topic string) error {
	token := c.client.Unsubscribe(topic)
	token.WaitTimeout(5 * time.Second)
	return token.Error()
}

// Publish sends a message to a topic.
func (c *Client) Publish(topic string, qos byte, retained bool, payload []byte) error {
	token := c.client.Publish(topic, qos, retained, payload)
	token.WaitTimeout(5 * time.Second)
	return token.Error()
}

// IsConnected returns the current connection state.
func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}

// Close disconnects the client gracefully (waits up to 5s).
func (c *Client) Close() {
	c.client.Disconnect(5000)
}

// SharedSubscribe subscribes using EMQX shared subscription prefix.
func SharedSubscribe(c *Client, group, topic string, qos byte, handler mqtt.MessageHandler) error {
	sharedTopic := fmt.Sprintf("%s/%s/%s", SharedPrefix, group, topic)
	return c.Subscribe(sharedTopic, qos, handler)
}

// LogHandler returns a message handler that logs the topic.
func LogHandler(prefix string) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		log.Printf("[%s] %s: %d bytes", prefix, msg.Topic(), len(msg.Payload()))
	}
}
