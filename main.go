package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/p2002eq/discordeq/applog"
	"github.com/p2002eq/discordeq/discord"
	"github.com/p2002eq/discordeq/listener"
	"github.com/xackery/eqemuconfig"
)

func main() {
	applog.StartupInteractive()
	log.SetOutput(applog.DefaultOutput)
	startService()
}

func startService() {
	log.Println("Starting DiscordEQ v0.51")
	var option string
	//Load config
	config, err := eqemuconfig.GetConfig()
	if err != nil {
		applog.Error.Println("Error while loading eqemu_config.xml to start:", err.Error())
		fmt.Println("press a key then enter to exit.")

		fmt.Scan(&option)
		os.Exit(1)
	}
	if config.Discord.RefreshRate == 0 {
		config.Discord.RefreshRate = 10
	}

	if !isNewTelnetConfig(config) && strings.ToLower(config.World.Tcp.Telnet) != "enabled" {
		log.Println("Telnet must be enabled for this tool to work. Check your eqemuconfig.xml, and please adjust.")
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
		os.Exit(1)
	}

	if config.Discord.Username == "" {
		applog.Error.Println("I don't see a username set in your <discord><username> section of eqemu_config.xml, please adjust.")
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
		os.Exit(1)
	}

	if config.Discord.Password == "" {
		applog.Error.Println("I don't see a password set in your <discord><password> section of eqemu_config.xml, please adjust.")
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
		os.Exit(1)
	}

	if config.Discord.ServerID == "" {
		applog.Error.Println("I don't see a serverid set in your <discord><serverid> section of eqemu_config.xml, please adjust.")
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
		os.Exit(1)
	}

	if config.Discord.ChannelID == "" {
		applog.Error.Println("I don't see a channelid set in your <discord><channelid> section of eqemu_config.xml, please adjust.")
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
		os.Exit(1)
	}

	if config.Discord.Admingroup == "" {
		applog.Error.Println("I don't see a admingroup set in your <discord><admingroup> section of eqemu_config.xml, please adjust.")
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
		os.Exit(1)
	}

	if config.Discord.Auction == "" {
		applog.Error.Println("I don't see a Auction channel set in your <discord><auction> section of eqemu_config.xml, please adjust.")
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
		os.Exit(1)
	}

	disco := discord.Discord{}
	err = disco.Connect(config.Discord.Username, config.Discord.Password)
	if err != nil {
		applog.Error.Println("Error connecting to discord:", err.Error())
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
		os.Exit(1)
	}
	go listenToDiscord(config, &disco)
	go listenToOOC(config, &disco)
	select {}
}

func isNewTelnetConfig(config *eqemuconfig.Config) bool {
	return strings.ToLower(config.World.Telnet.Enabled) == "true"
}

func listenToDiscord(config *eqemuconfig.Config, disco *discord.Discord) (err error) {
	for {
		applog.Info.Println("[Discord] Connecting as", config.Discord.Username, "...")
		if err = listener.ListenToDiscord(config, disco); err != nil {
			applog.Error.Println("[Discord] Disconnected with error:", err.Error())
		}

		applog.Info.Println("[Discord] Reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
		err = disco.Connect(config.Discord.Username, config.Discord.Password)
		if err != nil {
			applog.Error.Println("[Discord] Error connecting to discord:", err.Error())
		}
	}
}

func listenToOOC(config *eqemuconfig.Config, disco *discord.Discord) (err error) {
	for {
		listener.ListenToOOC(config, disco)
		applog.Info.Println("[OOC] Reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}
