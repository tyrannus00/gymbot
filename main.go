package main

import (
	"GymBot/exercises"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

const KgToLbs = 2.20462
const CommandPrefix rune = '.'

func main() {
	sess, err := discordgo.New(fmt.Sprintf("Bot %v", getToken()))

	if err != nil {
		log.Fatal(err)
	}

	sess.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == sess.State.User.ID {
			return
		}

		handleUnitConversion(s, m)
		handleCommands(s, m)
	})

	sess.AddHandler(func(s *discordgo.Session, r *discordgo.MessageReactionAdd) {

		msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)

		if err != nil {
			log.Println(err)
			return
		}

		if msg.Author.ID != sess.State.User.ID || r.Emoji.Name != "🗑️" {
			return
		}

		s.ChannelMessageDelete(r.ChannelID, r.MessageID)
	})

	// Logic for making bot online

	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = sess.Open()

	if err != nil {
		log.Fatal(err)
	}

	defer sess.Close()
	defer saveLeaderBoards()

	fmt.Println("Online")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func handleCommands(s *discordgo.Session, m *discordgo.MessageCreate) {
	content := []rune(m.Content)
	if content[0] != CommandPrefix {
		return
	}

	split := strings.Split(m.Content, " ")
	test := fmt.Sprintf("%cpr", CommandPrefix)

	if split[0] == test {
		handlePrCommand(split, s, m)
	} else if split[0] == fmt.Sprintf("%cleaderboard", CommandPrefix) {
		handleLeaderboardCommand(split, s, m)
	}

}

func handleLeaderboardCommand(split []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	exerciseString := split[1]
	var exercise exercises.Exercise

	if exerciseString == "bench" {
		exercise = exercises.BENCH
	} else if exerciseString == "squat" {
		exercise = exercises.SQUAT
	} else if exerciseString == "deadlift" {
		exercise = exercises.DEADLIFT
	} else {
		s.ChannelMessageSend(m.ChannelID, "Invalid lift!\nValid options for lifts are 'bench', 'squat' or 'deadlift'")
		return
	}

	prs := liftPrs[exercise]
	keys := make([]string, 0.0, len(prs))

	for key := range prs {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return prs[keys[i]] > prs[keys[j]]
	})

	builder := strings.Builder{}

	for _, key := range keys {
		user, err := s.User(key)

		if err != nil {
			return
		}
		weight := prs[key]

		builder.WriteString(fmt.Sprintf("%s: %.2f kg (%.2f lbs)\n", user.GlobalName, weight, weight*KgToLbs))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Venomforce " + exerciseString + " leaderboard:",
		Color:       0x00ff00,
		Description: builder.String(),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func handlePrCommand(split []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(split) != 4 {
		s.ChannelMessageSend(m.ChannelID, "Correct syntax for 'pr' command is .pr <lift> <amount> <unit>\nValid options for lifts are 'bench', 'squat' or 'deadlift'.")
		return
	}

	lift := split[1]
	var exercise exercises.Exercise

	if lift == "bench" {
		exercise = exercises.BENCH
	} else if lift == "squat" {
		exercise = exercises.SQUAT
	} else if lift == "deadlift" {
		exercise = exercises.DEADLIFT
	} else {
		s.ChannelMessageSend(m.ChannelID, "Invalid lift!\nValid options for lifts are 'bench', 'squat' or 'deadlift'")
		return
	}

	amount, err := strconv.ParseFloat(split[2], 64)

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Invalid amount! You need to specify a positive number in a reasonable range, stop lying!")
		return
	}

	unit := split[3]

	if unit == "lbs" {
		amount /= KgToLbs
	} else if unit != "kg" {
		s.ChannelMessageSend(m.ChannelID, "Invalid unit! Only 'kg' and 'lbs' are valid units.")
		return
	}

	lbsAmount := amount * KgToLbs
	oldPr, exists := GetPr(m.Author.ID, exercise)
	AddPr(m.Author.ID, exercise, amount)

	message := fmt.Sprintf("Added new personal %v record of %.2f kg (%.2f lbs)! ", lift, amount, lbsAmount)

	if exists {
		message += fmt.Sprintf("Previous pr was %.2f kg.", oldPr)
	}

	s.ChannelMessageSend(m.ChannelID, message)
}

func handleUnitConversion(s *discordgo.Session, m *discordgo.MessageCreate) {
	content := []rune(m.Content)

	if content[0] == CommandPrefix {
		return
	}

	if kgResult := getUnitString("kg", m.Content, content); kgResult != "" {
		s.ChannelMessageSend(m.ChannelID, kgResult)
	}

	if lbsResult := getUnitString("lbs", m.Content, content); lbsResult != "" {
		s.ChannelMessageSend(m.ChannelID, lbsResult)
	}
}

/*
baseUnit must be either kg or lbs
*/
func getUnitString(baseUnit string, message string, content []rune) string {
	baseUnitLength := len(baseUnit)
	otherUnit := "lbs"
	if baseUnit == "lbs" {
		otherUnit = "kg"
	}

	lastIndex := strings.LastIndex(message, baseUnit)

	if lastIndex != -1 && (lastIndex == len(content)-baseUnitLength || content[lastIndex+baseUnitLength] == ' ') {
		numberIdx := -1
		hitBlank := false
		hitFloatingPoint := false

		for i := lastIndex - 1; i >= 0; i-- {
			var rune = content[i]
			if rune == ' ' {
				if hitBlank {
					break
				} else {
					hitBlank = true
				}
			} else if rune == '.' {
				if hitFloatingPoint {
					break
				} else {
					hitFloatingPoint = true
				}
			} else if !unicode.IsDigit(rune) {
				break
			}

			numberIdx = i
		}

		if numberIdx != -1 {
			numba := strings.TrimSpace(message[numberIdx:lastIndex])
			if len(numba) > 0 {
				if result, err := strconv.ParseFloat(numba, 64); err == nil {
					converted := result * KgToLbs
					if baseUnit == "lbs" {
						converted = result / KgToLbs
					}

					convertedString := strconv.FormatFloat(converted, 'f', 2, 64)

					return numba + baseUnit + " = " + convertedString + otherUnit
				} else {
					fmt.Println(err)
				}
			}
		}
	}

	return ""
}

func getToken() string {
	if token, set := os.LookupEnv("BOT_TOKEN"); set {
		return token
	}

	log.Fatal("Environment variable BOT_TOKEN is not set!")
	return ""
}
