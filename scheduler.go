package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const prefix = "!schedule"      // prefix to all messages that interact with this bot
var Token string                // Bot token read from file
var m map[DayOfWeek][]Available // a map where key = Day of the week, and value = list of available structs

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
	UserID string 		// leverage the fact that the discord library can get a User from their ID
						// expects all strings in the format "xx:xx", so "04:00" is valid but "4:00" is not
						// this way, times can still be compared lexicographically
	TimeStart string 	// strips the colon character from a time. "12:30" => "1230"
	TimeEnd string 		// same as above, but value must be larger than timeStart
						// this must be checked before creating an Available
	Notes string		// any notes that the user has is stored here
}

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
	readBytes := make([]byte, 100)
	count, err := file.Read(readBytes) 	// read bytes of file into bytes array
										// count tells us the exact number of bytes read for us to unmarshal
	if err != nil {
		fmt.Println("error reading file", err)
		os.Exit(1)
	}
	var tokenJson Auth
	err = json.Unmarshal(readBytes[:count], &tokenJson)
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

	// instantiate our map after we successfully start the bot
	m = make(map[DayOfWeek][]Available)

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
// ex: !schedule add me Sunday 15:00 22:00 May be 15 minutes late
// !schedule check [me | User]
// ex: !schedule check @someUser
// !schedule update [me | User] day startTime endTime (optional: notes)
// ex: !schedule update @someUser Sunday 18:00 22:00 Need to run errands in the afternoon
// !schedule available
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

	// process the message into parts
	// if we made it this far we know that the first elem is "!schedule" so we can ignore it from now on
	msgParts := strings.Split(m.Content, " ")
	// if no other arguments are given, we can't process further. return after printing a help message
	if len(msgParts) < 2 {
		s.ChannelMessageSend(m.ChannelID, botHelp())
		return // return after sending the message
	}
	// we want to branch based on the second token
	op := msgParts[1] // process invoked is dependent on the operating statement
	if strings.EqualFold(op, "add") { // !schedule add [me | User] day timeStart timeEnd (opt.) notes
		// when the user specifies that they want to add, we have to check the number of args
		// it must be at least 6 args in total
		if len(msgParts) < 6 {
			s.ChannelMessageSend(m.ChannelID, botAddUsage())
		}
		notes := "" // variable placeholder in case the user specifies notes for their availability
		if len(msgParts) > 6 {
			notes = concatenateNotes(msgParts[6:]) // any string from the 7th index onwards is part of our notes
		}
		err = scheduleAdd(msgParts[2], msgParts[3], msgParts[4], msgParts[5], notes)
		if err != nil {
			fmt.Println(err)
			// also want to reprint add usage message
			s.ChannelMessageSend(m.ChannelID, botAddUsage())
			return
		}
		// err is nil, so we successfully added the user's availability. make this known to the channel
		s.ChannelMessageSend(m.ChannelID, "Successfully added availability.")
	} else {
		s.ChannelMessageSend(m.ChannelID, unrecognizedOp(op))
	}
}

// function that returns a string that gives a basic help output for using the bot
func botHelp() string {
	return "Scheduler usage: " // TODO: determine bot functionality and finish help message
}

// function that returns a string that details the usage for the add command
func botAddUsage() string {
	var buffer bytes.Buffer
	buffer.WriteString("Add usage: !schedule add [me | User] day timeStart timeEnd (opt.) notes")
	buffer.WriteString("\n")
	buffer.WriteString("Example: !schedule add @Username Sunday 15:00 18:00")
	return buffer.String()
}

// function that returns a string detailing that a given command is unrecognized
func unrecognizedOp(op string) string {
	return fmt.Sprintf("Unrecognized command: %s. Type !schedule help to see available commands", op)
}

// takes user input and adds a user's availability for a day to our map
// if an error occurs, we return it, else return nil
func scheduleAdd(user, day, timeStart, timeEnd, notes string) error {
	// first we need to check if the user already has determined availability for a given day
	// need to map day value to a corresponding DayOfTheWeek
	dayOfWeek, err := mapStrToDay(day)
	if err != nil {
		return err // if an error occurred, then our input string for the day was invalid
	}

	// day is validated so now check if the user already has availability defined for the given day
	users := m[dayOfWeek]
	if isUserInList(user, users) {
		return errors.New("User already defined availability for day. Please use [!schedule update] instead.")
	}

	timeStart, err = convertStrToMilitaryTime(timeStart) // convert our times to their military counterparts
	if err != nil {
		return err
	}
	timeEnd, err = convertStrToMilitaryTime(timeEnd)
	if err != nil {
		return err
	}
	// need to check to make sure that the start time is less than the end time
	if strings.Compare(timeStart, timeEnd) >= 0 {
		errorMsg := fmt.Sprintf("Start time %s must be less than end time %s.", timeStart, timeEnd)
		return errors.New(errorMsg)
	}

	// if the user is not in our map for the given day, we add them
	m[dayOfWeek] = append(users, Available{user, timeStart, timeEnd, notes})

	return nil
}

// takes user input and updates a user's availability for a day in our map
// if an error occurs, we return it, else return nil
func scheduleUpdate(user, day, timeStart, timeEnd, notes string) error {
	return nil
}

// takes user input for available time
// first validates that the string represents valid input, i.e.: "15:00"
// if valid, returns a resulting string that strips the colon character from the string
// the pattern [0-2]?[0-9]:[0-5][0-9] would allow for "35:00" to be validated
// rather than account for that case, keep the regexp simple
const pattern = "[0-2][0-9]:[0-5][0-9]"
// if we include "00:00" as a time, "24:00" cannot be included.
const timeBound = "2400" // lexicographically, all valid times will be less than this constant
func convertStrToMilitaryTime(time string) (string, error) {
	// first validate that our string input matches a pattern "00:00"
	matched, err := regexp.MatchString(pattern, time)
	if err != nil { // if an error occurred while attempting to match the string, return that error
		return time, err
	} else if !matched { // if no error, but not matched, return a different error
		errorMsg := fmt.Sprintf("Invalid format: %s. Time expected in 00:00 format, military time.", time)
		return time, errors.New(errorMsg)
	}
	// if no error occurred, and string input did match, then we can manipulate the string to remove the ':' character
	time = strings.Replace(time, ":", "", -1)
	// now check to make sure that the time is within our bound
	if strings.Compare(time, timeBound) >= 0 { // if the value is greater than our bounds, return an error
		errorMsg := fmt.Sprintf("Invalid value: %s. Time must be between 00:00 and 23:59 inclusive.", time)
		return time, errors.New(errorMsg)
	}
	return time, nil
}

// checks if a user is in a list of available structs, based on User ID
// returns true if the user is found, false otherwise
func isUserInList(user string, users []Available) bool {
	for _, elem := range users {
		if user == elem.UserID {
			return true
		}
	}
	return false
}

// takes a string and returns a mapping to a DayOfTheWeek type
// returns a non-nil error if the input string is not a valid day of the week
func mapStrToDay(day string) (DayOfWeek, error) {
	switch {
	case strings.EqualFold(day, "Sunday"): return SUN, nil
	case strings.EqualFold(day, "Monday"): return MON, nil
	case strings.EqualFold(day, "Tuesday"): return TUES, nil
	case strings.EqualFold(day, "Wednesday"): return WED, nil
	case strings.EqualFold(day, "Thursday"): return THURS, nil
	case strings.EqualFold(day, "Friday"): return FRI, nil
	case strings.EqualFold(day, "Saturday"): return SAT, nil
	default:
		errorMsg := fmt.Sprintf("Invalid day of the week: %s. Must be Sunday thru Saturday", day)
		return SUN, errors.New(errorMsg)
	}
}

// takes a slice of strings and returns the concatenation of them
func concatenateNotes(parts []string) string {
	// https://stackoverflow.com/questions/1760757/how-to-efficiently-concatenate-strings-in-go
	var buffer bytes.Buffer

	for index, elem := range parts {
		buffer.WriteString(elem)
		if index != len(parts) - 1 {
			buffer.WriteString(" ")
		}
	}
	return buffer.String()
}
