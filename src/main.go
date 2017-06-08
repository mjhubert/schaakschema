package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"

	"github.com/MaxHalford/gago"
)

var optimizer *Optimizer

//A Vector contains byte (=TeamCostID)
type Vector []TeamCostID

//Evaluate a vector
func (X Vector) Evaluate() float64 {
	var result float64
	//log.Printf("Evaluate %v", X)
	for i := 0; i < len(X)/10; i++ {
		result += float64(optimizer.Evaluate(X[(i * 10):((i + 1) * 10)]).TotalCost)
	}

	//whish list evaluation
	//add penalities for not granted whishes

	return result
}

//Mutate a Vector
func (X Vector) Mutate(rng *rand.Rand) {
	//log.Printf("Mutate: %v", X)
	mutations := rng.Intn(2) + 1

	for m := 0; m < mutations; m++ {
		//random pick a position to pick a group
		absolutePosition := rng.Intn(len(optimizer.bond.teams))
		description := optimizer.descriptions[absolutePosition]
		groupPosition := absolutePosition - description.klasseGroup.begin

		var swapGroupPosition int
		for swapGroupPosition = groupPosition; swapGroupPosition == groupPosition; {
			swapGroupPosition = rand.Intn(len(description.klasseGroup.teams))
		}

		groupPosition += description.klasseGroup.begin
		swapGroupPosition += description.klasseGroup.begin

		x := X[groupPosition]
		X[groupPosition] = X[swapGroupPosition]
		X[swapGroupPosition] = x

	}

	//log.Print("Mutated: ", X)
}

func teamInSlice(team TeamCostID, list []TeamCostID) bool {
	for _, tid := range list {
		if team == tid {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

//Crossover a Vector
//http://www.rubicite.com/Tutorials/GeneticAlgorithms/CrossoverOperators/Order1CrossoverOperator.aspx
func (X Vector) Crossover(Y gago.Genome, rng *rand.Rand) (gago.Genome, gago.Genome) {

	totalTeams := len(optimizer.bond.teams)
	child1 := make([]TeamCostID, totalTeams, totalTeams)
	child2 := make([]TeamCostID, totalTeams, totalTeams)

	copy(child1, X)
	copy(child2, Y.(Vector))

	stopXPosition := rng.Intn(len(X))
	startXPosition := rng.Intn(len(X))

	if startXPosition == stopXPosition {
		stopXPosition++
		if stopXPosition > totalTeams-1 {
			stopXPosition = 0
		}
	}

	for ix := 0; ix < totalTeams; ix++ {
		if (startXPosition < stopXPosition && (ix < startXPosition || ix > stopXPosition)) ||
			(startXPosition > stopXPosition && (ix > stopXPosition && ix < startXPosition)) {
			child1[ix] = TeamCostID(0xFF)
			child2[ix] = TeamCostID(0xFF)
		}
	}

	var ixx, ixy int
	for ix := 0; ix < totalTeams; ix++ {
		if (startXPosition < stopXPosition && (ix < startXPosition || ix > stopXPosition)) ||
			(startXPosition > stopXPosition && (ix > stopXPosition && ix < startXPosition)) {

			for ; teamInSlice(Y.(Vector)[ixy], child1); ixy++ {
			}
			child1[ix] = Y.(Vector)[ixy]

			for ; teamInSlice(X[ixx], child2); ixx++ {
			}
			child2[ix] = X[ixx]

		}
	}

	return Vector(child1), Vector(child2)
}

//MakeVector return a new random solution
func MakeVector(rng *rand.Rand) gago.Genome {
	totalTeams := len(optimizer.bond.teams)
	vector := make([]TeamCostID, totalTeams, totalTeams)

	position := 0
	for k := Meester; k <= Derde; k++ {
		teams := optimizer.bond.klasses[k]
		perm := rng.Perm(len(teams))

		for _, v := range perm {
			vector[position] = optimizer.matrix.GetTeamCostID(teams[v].id)
			position++
		}
	}

	return Vector(vector)
}

func truncateString(str string, num int) string {
	bnoden := str
	if len(str) > num {

		bnoden = str[0:num]
	} else if len(str) < num {
		for i := 0; i < num-len(str); i++ {
			bnoden += " "
		}
	}
	return bnoden
}

//PrintDescription info
func (X Vector) PrintDescription() {
	log.Print("XXX")
	for ix, tid := range X {

		teamInfo := optimizer.matrix.GetTeamInfoByCostID(tid)
		if ix%10 == 0 {
			log.Print("\n")
		}
		log.Printf("%v\t%v\t%v\t%v\t%v\t%v\t%v", teamInfo.teamCostID, teamInfo.team.klasse, teamInfo.team.pd, teamInfo.team.id, truncateString(teamInfo.team.vereniging.plaats, 18), truncateString(teamInfo.team.naam, 18), teamInfo.team.vereniging.id)
	}
}

func main() {

	log.Print("Phact Schaakindeling Optimizer v0.1")
	if len(os.Args) != 5 {
		log.Fatal("usage: <EXCELSCHEMA> <EXCELTEAMS> <CACHEFILE> <APIKEY>")
		return
	}

	var excelSchemaFileName = os.Args[1]
	var excelTeamsFileName = os.Args[2]
	var distanceCacheFileName = os.Args[3]
	var googleDistanceMatrixAPIKey = os.Args[4]

	//0: load excel Schema
	ss, serr := LoadSpeelSchemaExcel(excelSchemaFileName)

	if serr != nil {
		log.Panic(serr)
	}

	log.Printf("Loaded %d rondes and %d loten", len(ss.Rondes), len(ss.Loten))

	//1: load excel Teams
	sb, lerr := LoadSchaakbondExcel(excelTeamsFileName)

	if lerr != nil {
		log.Panic(lerr)
	}

	log.Printf("Loaded %d verenigingen and %d teams", len(sb.verenigingen), len(sb.teams))

	if len(sb.teams) > 256 {
		log.Panic("Currently only a maximum of 256 teams allowed")
	}

	for klasse, teams := range sb.klasses {
		log.Printf("In Klasse %v are %d teams", klasse, len(teams))
	}

	//2: extract unique cities
	plaatsen := make(map[string]bool)

	for _, ver := range sb.verenigingen {
		plaatsen[ver.plaats] = true
	}

	uniekePlaatsen := make([]string, 0, len(plaatsen))

	for plaats := range plaatsen {
		uniekePlaatsen = append(uniekePlaatsen, plaats+", Netherlands")
	}

	log.Printf("Extracted %d unique city names", len(uniekePlaatsen))

	if len(uniekePlaatsen) > 256 {
		log.Panic("Currently only a maximum of 256 cities allowed")
	}

	//3: get travel information between cities
	info, err := GetTravelInformation(uniekePlaatsen, distanceCacheFileName, googleDistanceMatrixAPIKey)

	if err != nil {
		log.Panic(err)
	}

	log.Printf("Loaded %d travel information elements", len(info))

	//4: create a distance matrix and index city names
	distanceMartix := CreateDistanceMatrixWithTravelInformations(info)

	log.Printf("Created distance matrix for %d cities with %d pairs", len(distanceMartix.citiesByID), len(distanceMartix.citiesTravelInformation))

	//5: create a travel cost matrix for team-pairs and index team ids
	teamTravelCostMatrix := CreateTeamTravelCostInformationMatrix(sb, distanceMartix)

	log.Printf("Created team pair travel cost matrix for %d teams with %d pairs", len(teamTravelCostMatrix.teamCostIDByTeamID), len(teamTravelCostMatrix.teamCostMatrix))

	optimizer = NewOptimizer(teamTravelCostMatrix, ss, sb)

	lastYearGroup1A := []string{"0400691", "0100261", "0400891", "0900611", "0800071", "0400041", "0900081", "0300101", "0900231", "0200541"}

	lastYearGroup1ACostIDs, terr := teamTravelCostMatrix.TranslateToTeamCostIDs(lastYearGroup1A)

	if terr != nil {
		log.Panic(terr)
	}

	log.Print(lastYearGroup1A)
	log.Print(lastYearGroup1ACostIDs)

	travelCosts := optimizer.Evaluate(lastYearGroup1ACostIDs)

	log.Print(travelCosts)
	/*
		lastYear := []string{
			"1700401", "1100111", "0900711", "0600171", "1600131", "0200271", "0600081", "1400091", "0600821", "1400431", "0400691", "0100261", "0400891", "0900611", "0800071",
			"0400041", "0900081", "0300101", "0900231", "0200541", "1200101", "1900331", "0400611", "0800091", "1200071", "1700661", "0400531", "0900411", "0600521", "1100181",
			"0800431", "0200011", "0600571", "0300371", "0400351", "0200272", "0400532", "0300102", "0300161", "0200542", "0600041", "1100112", "0900712", "0800092", "0800072",
			"0600311", "0800211", "0900341", "1100291", "0200543", "0900441", "0800261", "0600561", "0400421", "1400511", "1700662", "1100121", "1400261", "0600522", "1400401",
			"1700402", "1700801", "1700611", "1600051", "1600132", "1900281", "1700241", "1700541", "1900051", "1400101", "0400741", "0100262", "0200151", "0300011", "0100331",
			"0200321", "0100291", "0100301", "0200221", "0400381", "0400411", "0200141", "0400892", "0800093", "0800031", "0200131", "0900082", "0300103", "0300111", "0300301",
			"1200102", "1100021", "0100081", "0800094", "0800073", "0400043", "0800381", "0900342", "0600523", "0900621", "0400692", "0900011", "0400612", "0600822", "0800032",
			"1100091", "0600082", "0800541", "0600602", "0900421", "0900422", "1100113", "1200381", "0800095", "1200231", "0400042", "0400061", "1400092", "0900232", "1400402",
			"1200271", "1100114", "1400481", "1200441", "1200072", "1100111", "1100221", "1700542", "1400221", "1100182", "1200103", "1400421", "1600011", "1600052", "1600133",
			"1700091", "1400331", "1400093", "1400071", "1600091", "1700831", "1900332", "0600191", "1700381", "1900211", "1700663", "1700242", "1700141", "1700841", "1900231"}

		lastYearCostIDs, lyerr := optimizer.matrix.TranslateToTeamCostIDs(lastYear)

		if lyerr != nil {
			panic(lyerr)
		}

		lastYearVector := Vector(lastYearCostIDs)

		e := lastYearVector.Evaluate()

		log.Print("Fitness lastYear", e)
	*/
	// open output file
	fo, err := os.Create("fitness.txt")
	if err != nil {
		panic(err)
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	var ga = gago.Generational(MakeVector)
	ga.Initialize()

	var lastFitness float64
	for i := 1; i < 5000000; i++ {
		ga.Enhance()

		if i%1000 == 0 {
			fo.WriteString(strconv.FormatFloat(ga.Best.Fitness, 'f', 6, 64) + "\n")
			fmt.Printf("Best fitness at generation %d: %f (%v)\n", i, ga.Best.Fitness, ga.Best.Fitness-lastFitness)
			ga.Best.Genome.(Vector).PrintDescription()
			lastFitness = ga.Best.Fitness
		}
	}
	fo.WriteString(strconv.FormatFloat(ga.Best.Fitness, 'f', 6, 64) + "\n")
	fmt.Print(ga.Best)

	ga.Best.Genome.(Vector).PrintDescription()

}
