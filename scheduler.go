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

// expresses days of the week
type DayOfWeek int
const (
	SUN DayOfWeek = iota
	MON
	TUES
	WED
	THURS
	FRI
	SAT
) // https://stackoverflow.com/questions/14426366/what-is-an-idiomatic-way-of-representing-enums-in-go

type Available struct {
	UserID string // leverage the fact that the discord library can get a User from their ID
	TimeStart int // transform a string, ex: "12:30", into a corresponding int value, 1230
	TimeEnd int // same as above, but must be an int value larger than timeStart
	// this must be checked before creating an Available
}

// TODO: figure out internal data structure that holds scheduling information
// use a map based structure:
// key = day of the week
// value = array of Availability objects
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
// Restrict bot access to a text channel named "scheduler"
// times are expected in military time, 0:00 - 23:59
// !schedule add [me | User] day startTime endTime (optional: notes)
// !schedule check User
// !schedule
func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// ignore messages that are sent by this bot
	if m.Author.ID == s.State.User.ID {
		return
	}
	channel, err := s.Channel(m.ChannelID) // get the channel this message is from
	if err != nil {
		fmt.Printf("Error finding channel with ID = %s", m.ChannelID)
	}

	if channel.Type != discordgo.ChannelTypeGuildText {
		return // we only want to **text channels** named "scheduler"
	}

	if !strings.EqualFold("scheduler", channel.Name) {
		// if the channel name is not "scheduler" (case insensitive), don't process the message
		return
	}

	// we only want to process messages that begin with our defined prefix, "!schedule"
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}
}

// function that returns a string that gives a basic help output for using the bot
func botHelp() (string) {
	return "Scheduler usage: " // TODO: determine bot functionality and finish help message
}
