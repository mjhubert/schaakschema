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

	sb := LoadSchaakbondExcel("data\\Indeling.xlsx")

	plaatsen := make(map[string]bool)

	for _, ver := range sb.verenigingen {
		plaatsen[ver.plaats] = true
	}

	uniekePlaatsen := make([]string, 0, len(plaatsen))

	for plaats := range plaatsen {
		uniekePlaatsen = append(uniekePlaatsen, plaats+", Netherlands")
	}

	info, err := GetDistanceMatrix(uniekePlaatsen, "data\\distance.cache", os.Args[1])

	log.Print(info)
	log.Print(err)

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
