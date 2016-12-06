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

//CityID City identifier
type CityID uint16

//City information
type City struct {
	Name string
	ID   CityID
}

//DistanceMatrix contains City and Distance Information
type DistanceMatrix struct {
	citiesByName            map[string]*City
	citiesByID              map[CityID]*City
	citiesTravelInformation map[uint32]*TravelInformation
}

//GetCityByID of matrix by ID
func (distanceMatrix *DistanceMatrix) GetCityByID(ID CityID) *City {
	return distanceMatrix.citiesByID[ID]
}

//GetCityByName of matrix by ID
func (distanceMatrix *DistanceMatrix) GetCityByName(name string) *City {
	return distanceMatrix.citiesByName[name]
}

func combineCityIDs(fromID CityID, toID CityID) uint32 {
	var result uint32 = 0x0000

	if fromID > toID {
		to := toID
		toID = fromID
		fromID = to
	}

	result = uint32(fromID)
	result = result << 8
	result |= uint32(toID)

	return result
}

//AddTravelInformation between cities
func (distanceMatrix *DistanceMatrix) AddTravelInformation(fromID CityID, toID CityID, tinfo TravelInformation) {
	nti := new(TravelInformation)
	nti.City = tinfo.City
	nti.Distance = tinfo.Distance
	nti.Duration = tinfo.Duration
	distanceMatrix.citiesTravelInformation[combineCityIDs(fromID, toID)] = nti
}

//GetTravelInformation between cities
func (distanceMatrix *DistanceMatrix) GetTravelInformation(fromID CityID, toID CityID) *TravelInformation {
	return distanceMatrix.citiesTravelInformation[combineCityIDs(fromID, toID)]
}

//GetOrAddCity to matrix
func (distanceMatrix *DistanceMatrix) GetOrAddCity(name string) *City {

	gc := distanceMatrix.citiesByName[name]

	if gc != nil {
		return gc
	}

	var city = new(City)

	city.ID = CityID(uint16(len(distanceMatrix.citiesByID)))
	city.Name = name

	distanceMatrix.citiesByID[city.ID] = city
	distanceMatrix.citiesByName[city.Name] = city

	return city
}

//AddTravelInformations between cities in matrix
func (distanceMatrix *DistanceMatrix) AddTravelInformations(info []TravelInformation) {
	for _, ti := range info {
		var fromCity = distanceMatrix.GetOrAddCity(ti.City[0])
		var toCity = distanceMatrix.GetOrAddCity(ti.City[1])

		distanceMatrix.AddTravelInformation(fromCity.ID, toCity.ID, ti)
	}
}

//CreateDistanceMatrixWithTravelInformations between cities in matrix
func CreateDistanceMatrixWithTravelInformations(info []TravelInformation) *DistanceMatrix {
	matrix := new(DistanceMatrix)
	matrix.citiesByID = make(map[CityID]*City)
	matrix.citiesByName = make(map[string]*City)
	matrix.citiesTravelInformation = make(map[uint32]*TravelInformation)
	matrix.AddTravelInformations(info)
	return matrix
}

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
				if x < y {
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

//GetTravelInformation between cities by google maps API
func GetTravelInformation(cities []string, cacheFileName string, apiKey string) ([]TravelInformation, error) {

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
