package main

import (
	"log"
	m "math"
	"os"
)

// Sphere function minimum is 0 reached in (0, ..., 0).
// Any search domain is fine.
func sphere(X []float64) float64 {
	sum := 0.0
	for _, x := range X {
		sum += m.Pow(x, 2)
	}
	return sum
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

	travelCosts := Evaluate(teamTravelCostMatrix, ss, []TeamCostID{12, 1, 2, 64, 4, 5, 67, 7, 8, 9})

	log.Print(travelCosts)

	//log.Print(distanceMartix.citiesTravelInformation)

	/*
		sb := LoadSchaakbondExcel("data\\Indeling.xlsx")
		log.Print("Number of teams: ", len(sb.teams))
		log.Print("Number of verenigingen: ", len(sb.verenigingen))

		for _, vfrom := range sb.verenigingen {

			for _, vto := range sb.verenigingen {

				if vfrom.id > vto.id {

					log.Print(vfrom.id, " <-> ", vto.id)

				}
			}
		}

		// Instantiate a GA with 2 variables and the fitness function

		var ga = presets.Float64(5, sphere)

		ga.Initialize()
		// Enhancement
		for i := 0; i < 10; i++ {
			ga.Enhance()
			// Display the current best solution
			fmt.Printf("The best obtained solution is %f\n", ga.Best.Fitness)
		}

	*/
}
