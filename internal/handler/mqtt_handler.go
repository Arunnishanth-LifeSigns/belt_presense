package handler

import (
	"log"

	"github.com/eclipse/paho.mqtt.golang"
	"belt-presense/internal/config"
)

func NewMessageHandler(processor *BeltProcessor) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

		switch msg.Topic() {
		case "arrhythmia/svc_start":
			processor.HandleSvcStartMessage(msg.Payload())
		case "arrhythmia/svc_action":
			processor.HandleSvcActionMessage(msg.Payload())
		default:
			log.Printf("Unknown topic: %s", msg.Topic())
		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
	subscribeToTopics(client)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

func InitializeMQTT(cfg *config.Config, processor *BeltProcessor) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTTBroker)
	opts.SetClientID(cfg.MQTTClientID)
	opts.SetUsername(cfg.MQTTUsername)
	opts.SetPassword(cfg.MQTTPassword)
	opts.SetDefaultPublishHandler(NewMessageHandler(processor))
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}

func subscribeToTopics(client mqtt.Client) {
	topics := []string{"arrhythmia/svc_start", "arrhythmia/svc_action"}
	for _, topic := range topics {
		token := client.Subscribe(topic, 1, nil)
		token.Wait()
		log.Printf("Subscribed to topic: %s", topic)
	}
}
