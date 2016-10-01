package listener

import (
	"github.com/bwmarrin/discordgo"
	"github.com/xackery/discordeq/discord"
	"github.com/xackery/eqemuconfig"
	"log"
	"strings"
	//"time"
	_ "database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func ListenToDiscord(config *eqemuconfig.Config, disco *discord.Discord) (err error) {
	var session *discordgo.Session
	var guild *discordgo.Guild
	session = disco.GetSession()
	guild, err = session.Guild(config.Discord.ServerID)
	if err != nil {
		log.Printf("[Discord] Failed to get server %s: %s (Make sure bot is part of server)", config.Discord.ServerID, err.Error())
		return
	}
	isNotAvail := true
	if guild.Unavailable == &isNotAvail {
		log.Printf("[Discord] Failed to get server %s: Server unavailable (Make sure bot is part of server, and has permission)", config.Discord.ServerID, err.Error())
		return
	}
	session.StateEnabled = true
	session.AddHandler(messageCreate)
	select {}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.ChannelID != config.Discord.ChannelID {
		return
	}

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
	msg := m.Content
	//Maximum limit of 4k
	if len(msg) > 4000 {
		msg = msg[0:4000]
	}

	if len(msg) < 1 {
		return
	}

	//Insert entry
	db, err := sqlx.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true", config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Port, config.Database.Db))
	if err != nil {
		return
	}
	defer db.Close()
	_, err = db.NamedExec("INSERT INTO qs_player_speech (`from`, `to`, `message`,`type`, `guilddbid`, `minstatus`) VALUES (:ign, '!discord', :msg, 5, 0, 0)",
		map[string]interface{}{
			"ign": ign,
			"msg": msg,
		})
	if err != nil {
		log.Println("[Discord] Invalid insert:", err.Error())
		return
	}
	log.Printf("[Discord] %s: %s\n", ign, msg)

}