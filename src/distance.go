package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

//TravelInformation between cities
type TravelInformation struct {
	City [2]string
	//Distance in meters
	//Duration in seconds
	Distance, Duration uint64
}

type values struct {
	Text  string `json:"text"`
	Value uint64 `json:"value"`
}

type element struct {
	Distance values `json:"distance"`
	Duration values `json:"duration"`
	Status   string `json:"status"`
}

type row struct {
	Elements []element `json:"elements"`
}

type apiResponse struct {
	DestinationAddresses []string `json:"destination_addresses"`
	OriginAddresses      []string `json:"origin_addresses"`
	Rows                 []row    `json:"rows"`
	Status               string   `json:"status"`
}

func requestDistanceMatrix(apiKey string, origins []string, destinations []string) (*apiResponse, error) {
	//Example:
	//https://maps.googleapis.com/maps/api/distancematrix/json?origins=Apeldoorn&destinations=Venray&key=APIKEY

	//Prevent exceding query limit by sleeping:
	time.Sleep(1500 * time.Millisecond)

	var requestURL url.URL
	requestURL.Scheme = "https"
	requestURL.Host = "maps.googleapis.com"
	requestURL.Path = "maps/api/distancematrix/json"
	q := requestURL.Query()
	q.Set("key", apiKey)
	q.Set("origins", strings.Join(origins, "|"))
	q.Set("destinations", strings.Join(destinations, "|"))
	requestURL.RawQuery = q.Encode()

	httpResponse, err := http.Get(requestURL.String())
	if err != nil {
		return nil, err
	}
	var response = new(apiResponse)
	err = json.NewDecoder(httpResponse.Body).Decode(response)

	if err != nil {
		return nil, err
	}
	return response, nil
}

func getDistanceMatrix(apiKey string, cities []string, info *[]TravelInformation, position int, recursiveDistances bool, skip int) (int, error) {

	var err error
	var response *apiResponse

	totalCities := len(cities)

	//Limit to 25 elements max
	if recursiveDistances &&
		totalCities > 20 {
		chunkSize := 10
		var divided [][]string
		for i := 0; i < len(cities); i += chunkSize {
			end := i + chunkSize

			if end > len(cities) {
				end = len(cities)
			}

			divided = append(divided, cities[i:end])

			position, err = getDistanceMatrix(apiKey, cities[i:end], info, position, true, skip)

			if err != nil {
				return position, err
			}

		}

		//now none recursive chuncks comparison
		for x := 0; x < len(divided); x++ {
			for y := 0; y < len(divided); y++ {
				if x != y {
					chk := make([]string, 0, len(divided[x])+len(divided[y]))
					chk = append(chk, divided[x]...)
					chk = append(chk, divided[y]...)

					position, err = getDistanceMatrix(apiKey, chk, info, position, false, skip)

					if err != nil {
						return position, err
					}
				}
			}
		}

		return position, nil
	}

	half := totalCities / 2
	origins := cities[:half]
	destinations := cities[half:]

	if position >= skip {

		response, err = requestDistanceMatrix(apiKey, origins, destinations)

		if err != nil {
			return position, err
		}

		if response.Status == "OK" {
			for originNr, rw := range response.Rows {
				for destinationNr, el := range rw.Elements {
					if el.Status == "OK" {
						var ti TravelInformation
						ti.City[0] = origins[originNr]
						ti.City[1] = destinations[destinationNr]
						ti.Duration = el.Duration.Value
						ti.Distance = el.Distance.Value
						(*info)[position] = ti
						position++
						//log.Printf("%v, %v -> %v", origins[originNr], destinations[destinationNr], el)

					} else {
						return position, errors.New("Element status invalid: " + response.Status)
					}
				}
			}

		} else {
			return position, errors.New("Response status invalid: " + response.Status)
		}

	} else {
		position += len(origins) * len(destinations)
	}

	if recursiveDistances {
		//Recursive part
		if len(origins) > 1 {
			position, err = getDistanceMatrix(apiKey, origins, info, position, true, skip)

			if err != nil {
				return position, err
			}
		}

		if len(destinations) > 1 {
			position, err = getDistanceMatrix(apiKey, destinations, info, position, true, skip)

			if err != nil {
				return position, err
			}
		}
	}

	return position, nil
}

func loadCachedDistances(cacheFileName string) ([]TravelInformation, error) {
	file, err := ioutil.ReadFile(cacheFileName)
	if err != nil {
		return nil, err
	}

	var info []TravelInformation
	err = json.Unmarshal(file, &info)

	if err != nil {
		return nil, err
	}

	return info, nil
}

func saveCachedDistances(cacheFileName string, infos []TravelInformation, position int) error {

	b, err := json.Marshal(infos[:position])

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(cacheFileName, b, 0644)

	return err
}

//GetDistanceMatrix between cities by google maps API
func GetDistanceMatrix(cities []string, cacheFileName string, apiKey string) ([]TravelInformation, error) {

	sort.Strings(cities)

	totalCities := len(cities)
	totalCombinations := (totalCities * (totalCities - 1)) / 2
	var info = make([]TravelInformation, totalCombinations, totalCombinations)

	var position int
	var err error

	var cachedInfo []TravelInformation

	cachedInfo, err = loadCachedDistances(cacheFileName)

	if err != nil {
		return nil, err
	}

	copy(info[:len(cachedInfo)], cachedInfo)

	if len(cachedInfo) == len(info) {
		return info, nil
	}

	position, err = getDistanceMatrix(apiKey, cities, &info, 0, true, len(cachedInfo))

	log.Print(err)

	err = saveCachedDistances(cacheFileName, info, position)

	if err != nil {
		return nil, err
	}

	if position != totalCombinations {
		return nil, fmt.Errorf("Not enough results: %d of %d, run again", position, totalCombinations)
	}

	if err != nil {
		return nil, err
	}

	return info, nil
}
