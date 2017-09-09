package listener

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/p2002eq/discordeq/discord"
	"github.com/xackery/eqemuconfig"
)

var disco *discord.Discord

func ListenToDiscord(config *eqemuconfig.Config, disc *discord.Discord) (err error) {
	var session *discordgo.Session
	var guild *discordgo.Guild
	disco = disc
	//log.Println("Listen to discord..")
	if session, err = disco.GetSession(); err != nil {
		log.Printf("[Discord] Failed to get instance %s: %s (Make sure bot is part of server)", config.Discord.ServerID, err.Error())
		return
	}

	if guild, err = session.Guild(config.Discord.ServerID); err != nil {
		log.Printf("[Discord] Failed to get server %s: %s (Make sure bot is part of server)", config.Discord.ServerID, err.Error())
		return
	}

	if guild.Unavailable {
		log.Printf("[Discord] Failed to get server %s: Server unavailable (Make sure bot is part of server, and has permission)", config.Discord.ServerID)
		return
	}

	session.StateEnabled = true
	session.AddHandler(onMessageEvent)
	log.Printf("[Discord] Connected\n")
	if err = session.Open(); err != nil {
		log.Printf("[Discord] Session closed: %s", err.Error())
		return
	}
	select {}
}

func onMessageEvent(s *discordgo.Session, m *discordgo.MessageCreate) {

	//Look for messages to be relayed to OOC in game.
	if m.ChannelID == config.Discord.ChannelID &&
		len(m.Message.Content) > 0 &&
		m.Message.Content[0:1] != "!" {
		messageCreate(s, m)
		return
	}

	//Look for any commands.
	if len(m.Message.Content) > 0 &&
		m.Message.Content[0:1] == "!" {
		commandParse(s, m)
	}

}

func commandParse(s *discordgo.Session, m *discordgo.MessageCreate) {
	//Verify user is allowed to send commands
	isAllowed := false

	member, err := s.State.Member(config.Discord.ServerID, m.Author.ID)

	if err != nil {
		log.Printf("[Discord] Failed to get member: %s (Make sure you have set the bot permissions to see members)", err.Error())
		return
	}

	roles, err := s.GuildRoles(config.Discord.ServerID)

	if err != nil {
		log.Printf("[Discord] Failed to get roles: %s (Make sure you have set the bot permissions to see roles)", err.Error())
		return
	}

	for _, role := range member.Roles {

		// Stop for loop as we found matching role and is allowed to send. Helpful for servers with a large amount of roles
		if isAllowed {
			break
		}

		// Start scanning for matching role
		for _, gRole := range roles {
			if strings.TrimSpace(gRole.ID) == strings.TrimSpace(role) {
				if strings.Contains(gRole.Name, config.Discord.Admingroup) {
					isAllowed = true
					break
				}
			}
		}
	}

	if !isAllowed {
		if _, err := disco.SendMessage(m.ChannelID, fmt.Sprintf("Sorry %s, access denied.", m.Author.Username)); err != nil {
			fmt.Printf("[Discord] Failed to send discord message: %s\n", err.Error())
			return
		}
		return
	}

	//figure out command, remove the ! bang
	command := strings.ToLower(m.Message.Content[1:])

	switch command {
	case "help":
		if _, err := disco.SendMessage(m.ChannelID, fmt.Sprintf("Available commands:\n **!lock** - Locks the server\n **!unlock** - Unlocks the server\n **!worldshutdown** - Starts the worldshutdown process 10 Minutes with 60 second notices\n **!cancel_shutdown** - Stops the worldshutdown process")); err != nil {
			fmt.Printf("[Discord] Failed to send discord help command: %s\n", err.Error())
			return
		}
	case "lock":
		{
			Sendln(fmt.Sprintf("lock"))
			Sendln(fmt.Sprintf("broadcast Server has been locked"))
			disco.SendMessage(m.ChannelID, fmt.Sprintf("Server has been locked"))
			return
		}
	case "unlock":
		{
			Sendln(fmt.Sprintf("unlock"))
			Sendln(fmt.Sprintf("broadcast Server has been unlocked"))
			disco.SendMessage(m.ChannelID, fmt.Sprintf("Server has been unlocked"))
			return
		}
	case "reloadworld":
		{
			Sendln(fmt.Sprintf("reloadworld"))
			Sendln(fmt.Sprintf("broadcast World has been Reloaded"))
			disco.SendMessage(m.ChannelID, fmt.Sprintf("World has been Reloaded"))
			return
		}
	case "worldshutdown":
		{
			Sendln(fmt.Sprintf("worldshutdown 600 60"))
			disco.SendMessage(m.ChannelID, fmt.Sprintf("Server shutdown has started - 10 Minutes"))
			return
		}

	case "cancel_shutdown":
		{
			Sendln(fmt.Sprintf("worldshutdown disable"))
			disco.SendMessage(m.ChannelID, fmt.Sprintf("Server shutdown has been stopped"))
			return
		}

	default:
		if _, err := disco.SendMessage(m.ChannelID, fmt.Sprintf("%s: Invalid command. Use !help to learn commands.", m.Author.Username)); err != nil {
			fmt.Printf("[Discord] Failed to send discord command message: %s\n", err.Error())
			return
		}
	}

}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	ign := ""
	member, err := s.State.Member(config.Discord.ServerID, m.Author.ID)
	if err != nil {
		log.Printf("[Discord] Failed to get member: %s (Make sure you have set the bot permissions to see members)", err.Error())
		return
	}

	roles, err := s.GuildRoles(config.Discord.ServerID)
	if err != nil {
		log.Printf("[Discord] Failed to get roles: %s (Make sure you have set the bot permissions to see roles)", err.Error())
		return
	}
	for _, role := range member.Roles {
		if ign != "" {
			break
		}
		for _, gRole := range roles {
			if ign != "" {
				break
			}
			if strings.TrimSpace(gRole.ID) == strings.TrimSpace(role) {
				if strings.Contains(gRole.Name, "IGN:") {
					splitStr := strings.Split(gRole.Name, "IGN:")
					if len(splitStr) > 1 {
						ign = strings.TrimSpace(splitStr[1])
					}
				}
			}
		}
	}
	if ign == "" {
		return
	}
	msg := m.ContentWithMentionsReplaced()
	//Maximum limit of 4k
	if len(msg) > 4000 {
		msg = msg[0:4000]
	}

	if len(msg) < 1 {
		return
	}

	ign = sanitize(ign)
	msg = sanitize(msg)

	//Send message.
	if err = Sendln(fmt.Sprintf("emote world 260 %s says from discord, '%s'", ign, msg)); err != nil {
		log.Printf("[Discord] Error sending message to telnet (%s:%s): %s\n", ign, msg, err.Error())
		return
	}

	log.Printf("[Discord] %s: %s\n", ign, msg)
}

func sanitize(data string) (sData string) {
	sData = data
	sData = strings.Replace(sData, `%`, "&PCT;", -1)
	re := regexp.MustCompile("[^\x00-\x7F]+")
	sData = re.ReplaceAllString(sData, "")
	return
}
