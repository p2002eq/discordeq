package listener

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/p2002eq/eqemuconfig"
	"github.com/p2002eq/discordeq/discord"
	"github.com/xackery/rebuildeq/go/eqproto"
)

var auction string

//Guilds
var guild5 string
var guild8 string
var guild20 string
var guild26 string
var guild38 string
var guild40 string
var guild68 string

var (
	newNATS     bool
	nc          *nats.Conn
	isCronSet   bool
	chanType    string
	guildType	string
	ok          bool
	chans       = map[int]string{
		4:   "auctions:",
		5:   "OOC:",
		11:  "GMSay:", //GM Say
		13:  "", //encounter
		15:  "", //system wide message
		256: "says:",
		260: "OOC:",
		261: "auctions:",
		263: "announcement:",
		269: "", //broadcast
	}
	guilddbid       = map[int]string{
		5:    "", // Powerslave
		8:    "", // Tight Underpants
		20:   "", // Gods
		26:   "", // Ding
		38:   "", // Brethren of Norrath
		40:   "", // Unity
		68:   "", // TheoryCraft
	}
)

func GetNATS() (conn *nats.Conn) {
	conn = nc
	return
}

func ListenToNATS(eqconfig *eqemuconfig.Config, disco *discord.Discord) {
	var err error
	config = eqconfig

	channelID = config.Discord.ChannelID // OOC Channel
	auction = config.Discord.AuctionID // Auction Channel

	// Guilds
	guild5 = config.Discord.GuildID5
	guild8 = config.Discord.GuildID8
	guild20 = config.Discord.GuildID20
	guild26 = config.Discord.GuildID26
	guild38 = config.Discord.GuildID38
	guild40 = config.Discord.GuildID40
	guild68 = config.Discord.GuildID68

	if err = connectNATS(config); err != nil {
		log.Println("[NATS] Warning while getting NATS connection:", err.Error())
		return
	}

	if err = checkForNATSMessages(nc, disco); err != nil {
		log.Println("[NATS] Warning while checking for messages:", err.Error())
	}
	nc.Close()
	nc = nil
	return
}

func connectNATS(config *eqemuconfig.Config) (err error) {
	if nc != nil {
		return
	}
	if config.NATS.Host != "" && config.NATS.Port != "" {
		if nc, err = nats.Connect(fmt.Sprintf("nats://%s:%s", config.NATS.Host, config.NATS.Port)); err != nil {
			log.Fatal(err)
		}
	} else {
		if nc, err = nats.Connect(nats.DefaultURL); err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("[NATS] Connected\n")
	return
}

func checkForNATSMessages(nc *nats.Conn, disco *discord.Discord) (err error) {

	//nc.Subscribe("DailyGain", OnDailyGain)
	nc.Subscribe("world.>", OnChannelMessage)
	nc.Subscribe("global.admin_message.>", OnAdminMessage)

	for {
		if nc.Status() != nats.CONNECTED {
			log.Println("[NATS] Disconnected, status is", nc.Status())
			break
		}
		time.Sleep(6000 * time.Second)
	}
	return
}

func SendWorldCommand(author string, command string, parameters []string) (err error) {
	if nc == nil {
		err = fmt.Errorf("nats is not connected.")
		return
	}

	commandMessage := &eqproto.CommandMessage{
		Author:  author,
		Command: command,
		Params:  parameters,
	}
	log.Println(commandMessage)
	var cmd []byte
	if cmd, err = proto.Marshal(commandMessage); err != nil {
		err = fmt.Errorf("Failed to marshal command: %s", err.Error())
		return
	}

	var msg *nats.Msg
	if msg, err = nc.Request("world.command_message.in", cmd, 2*time.Second); err != nil {
		return
	}

	//now process reply
	if err = proto.Unmarshal(msg.Data, commandMessage); err != nil {
		err = fmt.Errorf("Failed to unmarshal response: %s", err.Error())
		return
	}

	if _, err = disco.SendMessage(config.Discord.CommandChannelID, fmt.Sprintf("**%s** %s: %s", commandMessage.Author, commandMessage.Command, commandMessage.Result)); err != nil {
		log.Printf("[NATS] Error sending message (%s: %s) %s", commandMessage.Author, commandMessage.Result, err.Error())
		err = nil
		return
	}
	return
}

/* func SendZoneCommand(author string, command string, parameters []string) (err error) {
	if nc == nil {
		err = fmt.Errorf("nats is not connected.")
		return
	}

	commandMessage := &eqproto.CommandMessage{
		Author:  author,
		Command: command,
		Params:  parameters,
	}
	log.Println(commandMessage)
	var cmd []byte
	if cmd, err = proto.Marshal(commandMessage); err != nil {
		err = fmt.Errorf("Failed to marshal command: %s", err.Error())
		return
	}

	var msg *nats.Msg
	if msg, err = nc.Request("Zone.command_message.in", cmd, 2*time.Second); err != nil {
		return
	}

	//now process reply
	if err = proto.Unmarshal(msg.Data, commandMessage); err != nil {
		err = fmt.Errorf("Failed to unmarshal response: %s", err.Error())
		return
	}

	if _, err = disco.SendMessage(config.Discord.CommandChannelID, fmt.Sprintf("**%s** %s: %s", commandMessage.Author, commandMessage.Command, commandMessage.Result)); err != nil {
		log.Printf("[NATS] Error sending message (%s: %s) %s", commandMessage.Author, commandMessage.Result, err.Error())
		err = nil
		return
	}
	return
} */

func OnAdminMessage(nm *nats.Msg) {
	var err error
	channelMessage := &eqproto.ChannelMessage{}
	proto.Unmarshal(nm.Data, channelMessage)

	if _, err = disco.SendMessage(config.Discord.CommandChannelID, fmt.Sprintf("**Admin:** %s", channelMessage.Message)); err != nil {
		log.Printf("[NATS] Error sending admin message (%s) %s", channelMessage.Message, err.Error())
		return
	}

	log.Printf("[NATS] AdminMessage: %s\n", channelMessage.Message)
}

func OnChannelMessage(nm *nats.Msg) {
	var err error
	channelMessage := &eqproto.ChannelMessage{}
	proto.Unmarshal(nm.Data, channelMessage)

	if channelMessage.IsEmote {
		channelMessage.ChanNum = channelMessage.Type
	}

	if chanType, ok = guilddbid[int(channelMessage.Guilddbid)]; !ok {
		log.Printf("[NATS] Unknown GuildID: %d with message: %s", channelMessage.Guilddbid, channelMessage.Message)
	}

	if chanType, ok = chans[int(channelMessage.ChanNum)]; !ok {
		log.Printf("[NATS] Unknown channel: %d with message: %s", channelMessage.ChanNum, channelMessage.Message)
	}

	channelMessage.From = strings.Replace(channelMessage.From, "_", " ", -1)

	if strings.Contains(channelMessage.From, " ") {
		channelMessage.From = fmt.Sprintf("%s [%s]", channelMessage.From[:strings.Index(channelMessage.From, " ")], channelMessage.From[strings.Index(channelMessage.From, " ")+1:])
	}
	channelMessage.From = alphanumeric(channelMessage.From) //purify name to be alphanumeric

	if strings.Contains(channelMessage.Message, "Summoning you to") { //GM messages are relaying to discord!
		return
	}

	//message = message[strings.Index(message, "says ooc, '")+11 : len(message)-padOffset]

	channelMessage.Message = convertLinks(config.Discord.ItemUrl, channelMessage.Message)

	if channelMessage.Guilddbid == 5 { // Guild: Powerslave
		if _, err = disco.SendMessage(guild5, fmt.Sprintf("**%s tells the guild:** %s", channelMessage.From, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	} else if channelMessage.Guilddbid == 8 { // Guild: Tight Underpants
		if _, err = disco.SendMessage(guild8, fmt.Sprintf("**%s tells the guild:** %s", channelMessage.From, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	} else if channelMessage.Guilddbid == 20 { // Guild: Gods
		if _, err = disco.SendMessage(guild20, fmt.Sprintf("**%s tells the guild:** %s", channelMessage.From, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	} else if channelMessage.Guilddbid == 26 { // Guild: Ding
		if _, err = disco.SendMessage(guild26, fmt.Sprintf("**%s tells the guild:** %s", channelMessage.From, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	} else if channelMessage.Guilddbid == 38 { // Guild: Brethren of Norrath
		if _, err = disco.SendMessage(guild38, fmt.Sprintf("**%s tells the guild:** %s", channelMessage.From, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	} else if channelMessage.Guilddbid == 40 { // Guild: Unity
		if _, err = disco.SendMessage(guild40, fmt.Sprintf("**%s tells the guild:** %s", channelMessage.From, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	} else if channelMessage.Guilddbid == 68 { // Guild: TheoryCraft
		if _, err = disco.SendMessage(guild68, fmt.Sprintf("**%s tells the guild:** %s", channelMessage.From, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	}

	if channelMessage.ChanNum == 4 { // Auctions
		if _, err = disco.SendMessage(auction, fmt.Sprintf("**%s %s** %s", channelMessage.From, chanType, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	} else if channelMessage.ChanNum == 5 { // OOC
		if _, err = disco.SendMessage(channelID, fmt.Sprintf("**%s %s** %s", channelMessage.From, chanType, channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
			return
		}
	}

	/* channelMessage.Message = convertLinks(config.Discord.ItemUrl, channelMessage.Message)

	if _, err = disco.SendMessage(channelID, fmt.Sprintf("**%s %s** %s", channelMessage.From, chanType, channelMessage.Message)); err != nil {
		log.Printf("[NATS] Error sending message (%s: %s) %s", channelMessage.From, channelMessage.Message, err.Error())
		return
	} */

	//log.Printf("[NATS] %d %s: %s\n", channelMessage.ChanNum, channelMessage.From, channelMessage.Message)
}

func sendNATSMessage(from string, message string) {
	if nc == nil {
		log.Println("[NATS] not connected?")
		return
	}
	channelMessage := &eqproto.ChannelMessage{
		//From:    from + " says from discord, '",
		IsEmote: true,
		Message: fmt.Sprintf("%s says from discord, '%s'", from, message),
		ChanNum: 260,
		Type:    260,
	}
	msg, err := proto.Marshal(channelMessage)
	if err != nil {
		log.Printf("[NATS] Error marshalling %s %s: %s", from, message, err.Error())
		return
	}
	err = nc.Publish("ChannelMessageWorld", msg)
	if err != nil {
		log.Printf("[NATS] Error publishing: %s", err.Error())
		return
	}
}