package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"

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

//PrintDescription info
func (X Vector) PrintDescription() {
	for _, tid := range X {

		teamInfo := optimizer.matrix.GetTeamInfoByCostID(tid)

		log.Printf("%v - %v %v %v %v %v ", teamInfo.teamCostID, teamInfo.team.id, teamInfo.team.klasse, teamInfo.team.naam, teamInfo.team.vereniging.naam, teamInfo.team.vereniging.plaats)
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

	var ga = gago.Generational(MakeVector)
	for i := 1; i < 10000; i++ {
		ga.Enhance()
		if i%10 == 0 {
			fmt.Printf("Best fitness at generation %d: %f\n", i, ga.Best.Fitness)

		}
	}
	fmt.Print(ga.Best)

	ga.Best.Genome.(Vector).PrintDescription()

}
