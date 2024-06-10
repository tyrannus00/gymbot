package main

import (
	"GymBot/exercises"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"net/http"
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
	go keepAlive()
	autoSave()

	sess, err := discordgo.New(fmt.Sprintf("Bot %v", getToken()))

	if err != nil {
		log.Fatal(err)
	}

	sess.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == sess.State.User.ID || len(m.Content) == 0 {
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
	defer saveAll()

	fmt.Println("Online")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func keepAlive() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "I am online")
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func handleCommands(s *discordgo.Session, m *discordgo.MessageCreate) {
	content := []rune(m.Content)
	if content[0] != CommandPrefix {
		return
	}

	split := strings.Split(m.Content, " ")

	if split[0] == fmt.Sprintf("%cpr", CommandPrefix) {
		handlePrCommand(split, s, m)
	} else if split[0] == fmt.Sprintf("%cleaderboard", CommandPrefix) {
		handleLeaderboardCommand(split, s, m)
	}

}

func handleLeaderboardCommand(split []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(split) != 2 {
		s.ChannelMessageSend(m.ChannelID, "Correct syntax for 'leaderboard' command is .leaderboard <lift>\nValid options for lifts are 'bench', 'squat', 'deadlift' or 'total'")
		return
	}

	exerciseString := split[1]
	var exercise exercises.Exercise

	if exerciseString == "bench" {
		exercise = exercises.BENCH
	} else if exerciseString == "squat" {
		exercise = exercises.SQUAT
	} else if exerciseString == "deadlift" {
		exercise = exercises.DEADLIFT
	} else if exerciseString != "total" {
		s.ChannelMessageSend(m.ChannelID, "Invalid lift!\nValid options for lifts are 'bench', 'squat', 'deadlift' or 'total'")
		return
	}

	var prs map[string]float64
	if exercise == nil { // total
		prs = map[string]float64{}

		for id, pr := range benchPrs {
			prs[id] = pr
		}

		for id, pr := range squatPrs {
			prs[id] += pr
		}

		for id, pr := range deadliftPrs {
			prs[id] += pr
		}
	} else {
		prs = liftPrs[exercise]
	}

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

		builder.WriteString(fmt.Sprintf("%s: %.2f kg (%.2f lbs)\n", user.Username, weight, weight*KgToLbs))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Venomforce " + exerciseString + " leaderboard:",
		Color:       0x00ff00,
		Description: builder.String(),
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func handlePrCommand(split []string, s *discordgo.Session, m *discordgo.MessageCreate) {
	if len(split) < 2 {
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
	} else if lift != "total" || len(split) == 4 {
		s.ChannelMessageSend(m.ChannelID, "Invalid lift!\nValid options for lifts are 'bench', 'squat' or 'deadlift'")
		return
	}

	if len(split) == 2 {
		if exercise == nil {
			total := liftPrs[exercises.BENCH][m.Author.ID]
			total += liftPrs[exercises.SQUAT][m.Author.ID]
			total += liftPrs[exercises.DEADLIFT][m.Author.ID]

			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Your total is %.2fkg (%.2f lbs).", total, total*KgToLbs))
		} else {
			pr := liftPrs[exercise][m.Author.ID]

			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Your %s pr is %.2fkg (%.2f lbs).", lift, pr, pr*KgToLbs))
		}

		return
	} else if len(split) == 3 {
		name := split[2]
		if search, err := s.GuildMembersSearch(m.GuildID, name, 1000); err == nil && len(search) > 0 {
			user := search[0].User

			if exercise == nil {
				total := liftPrs[exercises.BENCH][user.ID]
				total += liftPrs[exercises.SQUAT][user.ID]
				total += liftPrs[exercises.DEADLIFT][user.ID]

				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s's total is %.2fkg (%.2f lbs).", user.Username, total, total*KgToLbs))
			} else if pr, exists := liftPrs[exercise][user.ID]; exists {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s's %s pr is %.2fkg (%.2f lbs).", user.Username, lift, pr, pr*KgToLbs))
			} else if exercise == nil {
			} else {
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s hasn't set a pr for %s yet!", user.Username, lift))
			}
		} else {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Can't find user %s!", name))
		}
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
