package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.bug.st/serial.v1"
)

var (
	mqttFlag  = flag.String("m", "tcp://127.0.0.1:1883", "MQTT broker")
	portFlag  = flag.String("p", "/dev/cu.usbmodem34208131", "serial port")
	quietFlag = flag.Bool("q", false, "quiet mode, don't show in/out messages")
	topicFlag = flag.String("t", "bridge/%", "MQTT topic template")
	userFlag  = flag.String("u","user","mqtt-username (optional)")
	passFlag  = flag.String("P","passwd","mqtt-passwd (optional)")
)

func main() {
	flag.Parse()
	log.SetFlags(0) // omit timestamps

	dev, err := serial.Open(*portFlag, &serial.Mode{BaudRate: 115200})
	check(err)
	defer dev.Close()

	opts := mqtt.NewClientOptions().AddBroker(*mqttFlag)
	opts.SetUsername(*userFlag)
	opts.SetPassword(*passFlag)
	c := mqtt.NewClient(opts)
	t := c.Connect()
	mqttCheck(t)

	t = c.Subscribe(topic("out"), 0, func(c mqtt.Client, msg mqtt.Message) {
		if !*quietFlag {
			fmt.Printf("> %s\n", msg.Payload())
		}
		dev.Write(append(msg.Payload(), '\n'))
	})
	mqttCheck(t)
	
	var wg sync.WaitGroup
    	wg.Add(1)  

	go func() {
		scanner := bufio.NewScanner(dev)
		for scanner.Scan() {
			msg := scanner.Text()
			if !*quietFlag {
				fmt.Printf("< %s\n", msg)
			}
			if strings.HasPrefix(msg,">mqtt:") {
			        msg = strings.TrimPrefix(msg,">mqtt:")
				var mt []string
				mt = strings.SplitN(msg,",",2)
				if len(mt)==2 { 
				   t := c.Publish(mt[0],0,false,mt[1])
				   mqttCheck(t)
				}
			} else {
				t := c.Publish(topic("in"), 0, false, msg)
			   	mqttCheck(t)
			}
		}
		os.Exit(1)
	}()

	if !*quietFlag {
		reader := bufio.NewReader(os.Stdin)
		for {
		    text, err := reader.ReadString('\n')
		    if err != nil {
		    	break
		    }
		dev.Write([]byte(text))
		}
  	} else {
	       wg.Wait()
	}
}

func topic(s string) string {
	return strings.Replace(*topicFlag, "%", s, -1)
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func mqttCheck(t mqtt.Token) {
	t.Wait()
	check(t.Error())
}
