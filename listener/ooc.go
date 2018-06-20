package listener

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/p2002eq/eqemuconfig"
	"github.com/ziutek/telnet"
)

var newTelnet bool

var lastId int
var channelID string


type UserMessage struct {
	Id         int       `db:"id"`
	From       string    `db:"from"`
	To         string    `db:"to"`
	Message    string    `db:"message"`
	Type       int       `db:"type"`
	CreateDate time.Time `db:"timerecorded"`
}

var userMessages []UserMessage
var config *eqemuconfig.Config

var t *telnet.Conn

func connectTelnet(config *eqemuconfig.Config) (err error) {
	if t != nil {
		return
	}
	ip := config.World.Telnet.Ip
	if ip == "" {
		ip = config.World.Tcp.Ip
	}
	port := config.World.Telnet.Port
	if port == "" {
		port = config.World.Tcp.Port
	}

	log.Printf("[OOC] Connecting to %s:%s...\n", ip, port)

	if t, err = telnet.Dial("tcp", fmt.Sprintf("%s:%s", ip, port)); err != nil {
		return
	}
	t.SetReadDeadline(time.Now().Add(10 * time.Second))
	t.SetWriteDeadline(time.Now().Add(10 * time.Second))
	index := 0
	skipAuth := false
	if index, err = t.SkipUntilIndex("Username:", "Connection established from localhost, assuming admin"); err != nil {
		return
	}
	if index != 0 {
		skipAuth = true
		log.Println("[OOC] Skipping auth")
		newTelnet = true
	}

	if !skipAuth {
		if err = Sendln(config.Discord.TelnetUsername); err != nil {
			return
		}

		if err = t.SkipUntil("Password:"); err != nil {
			return
		}
		if err = Sendln(config.Discord.TelnetPassword); err != nil {
			return
		}
	}

	if err = Sendln("echo off"); err != nil {
		return
	}

	if err = Sendln("acceptmessages on"); err != nil {
		return
	}

	t.SetReadDeadline(time.Time{})
	t.SetWriteDeadline(time.Time{})
	log.Printf("[OOC] Connected\n")
	return
}

func Sendln(s string) (err error) {

	buf := make([]byte, len(s)+1)
	copy(buf, s)
	buf[len(s)] = '\n'
	if t == nil {
		for {
			if err = connectTelnet(config); err != nil {
				return
			}
			fmt.Println("Telnet not connected, reconnecting...")
			time.Sleep(config.Discord.RefreshRate)
		}
	}
	_, err = t.Write(buf)
	return
}

func convertLinks(prefix string, message string) (messageFixed string) {
	messageFixed = message
	if strings.Count(message, "") > 1 {
		sets := strings.SplitN(message, "", 3)

		itemid, err := strconv.ParseInt(sets[1][0:6], 16, 32)
		if err != nil {
			itemid = 0
		}
		itemname := sets[1][56:]
		itemlink := prefix
		if itemid > 0 && len(prefix) > 0 {
			itemlink = fmt.Sprintf(" %s%d (%s)", itemlink, itemid, itemname)
		} else {
			itemlink = fmt.Sprintf(" *%s* ", itemname)
		}
		messageFixed = sets[0] + itemlink + sets[2]
		if strings.Count(message, "") > 1 {
			messageFixed = convertLinks(prefix, messageFixed)
		}
	}
	return
}
