package main

import (
	"GymBot/exercises"
	"encoding/json"
	"log"
	"os"
)

const benchLeaderboardsFile = "benchleaderboards.json"
const squatLeaderboardsFile = "squatleaderboards.json"
const deadliftLeaderboardsFile = "deadliftleaderboards.json"

// All prs are in kg
var benchPrs = map[string]float64{}
var squatPrs = map[string]float64{}
var deadliftPrs = map[string]float64{}

var liftPrs = map[exercises.Exercise]map[string]float64{
	exercises.BENCH:    benchPrs,
	exercises.SQUAT:    squatPrs,
	exercises.DEADLIFT: deadliftPrs,
}

func init() {
	loadLeaderBoards()
}

func AddPr(userId string, exercise exercises.Exercise, amount float64) {
	liftPrs[exercise][userId] = amount
}

func GetPr(userId string, exercise exercises.Exercise) (float64, bool) {
	val, exists := liftPrs[exercise][userId]
	return val, exists
}

func saveLeaderBoards() {
	save(benchLeaderboardsFile, benchPrs)
	save(squatLeaderboardsFile, squatPrs)
	save(deadliftLeaderboardsFile, deadliftPrs)
}

func loadLeaderBoards() {
	load(benchLeaderboardsFile, benchPrs)
	load(squatLeaderboardsFile, squatPrs)
	load(deadliftLeaderboardsFile, deadliftPrs)
}

func load(file string, prs map[string]float64) {
	jsonBytes, err := os.ReadFile(file)
	if err != nil {
		log.Println(err)
		return
	}

	err = json.Unmarshal(jsonBytes, &prs)
	if err != nil {
		log.Println(err)
	}
}

func save(file string, prs map[string]float64) {
	marshal, err := json.MarshalIndent(prs, "", "    ")
	if err != nil {
		log.Println(err)
		return
	}

	err = os.WriteFile(file, marshal, 0644)
	if err != nil {
		log.Println(err)
	}
}
