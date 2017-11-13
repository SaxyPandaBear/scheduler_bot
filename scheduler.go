package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const prefix string = "!schedule" // prefix to all messages that interact with this bot
var Token string // Bot token read from file

// struct that defines the json blob read from "discord_token.json"
type Auth struct {
	Token string
}

// TODO: figure out internal data structure that holds scheduling information
// TODO: set up cron job for clearing data weekly
// TODO: set up cron job for determining user availability every Friday
// TODO: keep track of users, decide on tracking users who respond or those who don't

func init() {
	// read the token from the json blob "discord_token.json"
	file, err := os.Open("discord_token.json")
	if err != nil {
		fmt.Println("error reading file", err)
		os.Exit(1)
	}
	bytes := make([]byte, 100)
	count, err := file.Read(bytes) // read bytes of file into bytes array
	// count tells us the exact number of bytes read for us to unmarshal
	if err != nil {
		fmt.Println("error reading file", err)
		os.Exit(1)
	}
	var tokenJson Auth
	err = json.Unmarshal(bytes[:count], &tokenJson)
	if err != nil {
		fmt.Println("error decoding json blob", err)
		os.Exit(1)
	}
	defer file.Close()
	Token = tokenJson.Token
}

func main() {
	// instantiate the discord bot
	dg, err := discordgo.New("Bot " + Token)

	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// event handler for messages in channels that the bot is authorized to access
	dg.AddHandler(onMessage)

	// open a web socket connection to Discord
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// see https://github.com/bwmarrin/discordgo/blob/master/examples/pingpong/main.go example
	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	close(sc)
	fmt.Println("\nBot is now closing.")
	dg.Close()
}

// TODO: define bot functionality
func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// ignore messages that are sent by this bot
	if m.Author.ID == s.State.User.ID {
		return
	}
	if strings.HasPrefix(m.Content, prefix) {

	}
}

// function that returns a string that gives a basic help output for using the bot
func botHelp() (string) {
	return "Scheduler usage: " // TODO: determine bot functionality and finish help message
}
