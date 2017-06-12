package main

import (
	"fmt"
	"log"
	"math"
)

//TeamCostID identiefier
type TeamCostID byte

//TeamCostPairID identiefier
type TeamCostPairID uint16

//TeamInfo for team cost
type TeamInfo struct {
	teamCostID TeamCostID
	team       Team
}

//TeamCostMatrix cost info
type TeamCostMatrix struct {
	teamCostIDByTeamID map[string]*TeamInfo
	teamInfoByCostID   map[TeamCostID]*TeamInfo
	teamCostMatrix     map[TeamCostPairID]*TravelInformation
}

//TranslateToTeamInfos translate teamids to team info
func (matrix *TeamCostMatrix) TranslateToTeamInfos(teamIDs []string) ([]*TeamInfo, error) {
	result := make([]*TeamInfo, len(teamIDs), len(teamIDs))

	for ix, id := range teamIDs {
		info := matrix.teamCostIDByTeamID[id]

		if info == nil {
			return nil, fmt.Errorf("Unknown team id %v", id)
		}

		result[ix] = info
	}

	return result, nil
}

//TranslateToTeamCostIDs translate teamids to TeamCostIDs
func (matrix *TeamCostMatrix) TranslateToTeamCostIDs(teamIDs []string) ([]TeamCostID, error) {
	result := make([]TeamCostID, len(teamIDs), len(teamIDs))

	infos, err := matrix.TranslateToTeamInfos(teamIDs)

	if err != nil {
		return nil, err
	}

	for ix, inf := range infos {
		result[ix] = inf.teamCostID
	}

	return result, nil
}

func combineTeamCostIDs(fromID TeamCostID, toID TeamCostID) TeamCostPairID {
	var result uint16 = 0x00

	if fromID > toID {
		to := toID
		toID = fromID
		fromID = to
	}

	result = uint16(fromID)
	result = result << 8
	result |= uint16(toID)

	id := TeamCostPairID(result)
	return id
}

//GetTeamsTravelCost between cities
func (matrix *TeamCostMatrix) GetTeamsTravelCost(fromID TeamCostID, toID TeamCostID) *TravelInformation {
	return matrix.teamCostMatrix[combineTeamCostIDs(fromID, toID)]
}

//GetTeamCostID of string ID
func (matrix *TeamCostMatrix) GetTeamCostID(teamID string) TeamCostID {
	return matrix.teamCostIDByTeamID[teamID].teamCostID
}

//GetTeamInfoByCostID info
func (matrix *TeamCostMatrix) GetTeamInfoByCostID(teamID TeamCostID) *TeamInfo {
	return matrix.teamInfoByCostID[teamID]
}

//GetOrAddTeamCostInfoByTeam of matrix
func (matrix *TeamCostMatrix) GetOrAddTeamCostInfoByTeam(team Team) *TeamInfo {

	gc := matrix.teamCostIDByTeamID[team.id]

	if gc != nil {
		return gc
	}

	teamInfo := new(TeamInfo)
	teamInfo.team = team
	teamInfo.teamCostID = TeamCostID(byte(len(matrix.teamCostIDByTeamID)))

	matrix.teamCostIDByTeamID[team.id] = teamInfo
	matrix.teamInfoByCostID[teamInfo.teamCostID] = teamInfo
	return teamInfo
}

//CreateTeamTravelCostInformationMatrix between teams
func CreateTeamTravelCostInformationMatrix(sb *Schaakbond, distanceMatrix *DistanceMatrix) *TeamCostMatrix {
	matrix := new(TeamCostMatrix)
	matrix.teamCostIDByTeamID = make(map[string]*TeamInfo)
	matrix.teamCostMatrix = make(map[TeamCostPairID]*TravelInformation)
	matrix.teamInfoByCostID = make(map[TeamCostID]*TeamInfo)

	for _, fromTeam := range sb.teams {
		for _, toTeam := range sb.teams {
			if fromTeam.id < toTeam.id {

				fromTeamInfo := matrix.GetOrAddTeamCostInfoByTeam(fromTeam)
				toTeamInfo := matrix.GetOrAddTeamCostInfoByTeam(toTeam)

				fromTeamCity := distanceMatrix.GetCityByName(fromTeam.vereniging.plaats + ", Netherlands")
				toTeamCity := distanceMatrix.GetCityByName(toTeam.vereniging.plaats + ", Netherlands")

				var info *TravelInformation
				if fromTeamCity.ID == toTeamCity.ID {
					info = new(TravelInformation)
					info.City[0] = fromTeamCity.Name
					info.City[1] = toTeamCity.Name
					info.Distance = 0
					info.Duration = 0
				} else {
					info = distanceMatrix.GetTravelInformation(fromTeamCity.ID, toTeamCity.ID)
				}

				matrix.teamCostMatrix[combineTeamCostIDs(fromTeamInfo.teamCostID, toTeamInfo.teamCostID)] = info
			}
		}
	}

	return matrix
}

//TravelCosts info
type TravelCosts struct {
	TotalDuration, TotalDistance, TotalCost uint64
}

//KlasseGroup info
type KlasseGroup struct {
	klasse     Klasse
	begin, end int
	teams      []TeamCostID
}

//Description of property of array position
type Description struct {
	klasseGroup *KlasseGroup
	groupNr     int
	begin, end  int
}

//Optimizer info
type Optimizer struct {
	matrix       *TeamCostMatrix
	schema       *SpeelSchema
	bond         *Schaakbond
	descriptions []*Description
}

//NewOptimizer create a optimizer
func NewOptimizer(matrix *TeamCostMatrix, schema *SpeelSchema, bond *Schaakbond) *Optimizer {
	optimizer := new(Optimizer)
	optimizer.matrix = matrix
	optimizer.schema = schema
	optimizer.bond = bond

	optimizer.descriptions = make([]*Description, len(bond.teams), len(bond.teams))

	ix := 0

	for k := Meester; k <= Derde; k++ {

		klasseGroup := new(KlasseGroup)

		teams := bond.klasses[k]
		klasseGroup.teams = make([]TeamCostID, len(teams), len(teams))

		tix := 0
		for _, t := range teams {
			klasseGroup.teams[tix] = matrix.GetTeamCostID(t.id)
			tix++
		}

		klasseGroup.begin = ix
		klasseGroup.end = ix + (len(teams) - 1)
		klasseGroup.klasse = k

		for i := 0; i < len(teams)/10; i++ {
			description := new(Description)
			description.klasseGroup = klasseGroup
			description.groupNr = i
			description.begin = klasseGroup.begin + (i * 10)
			description.end = klasseGroup.begin + (((i + 1) * 10) - 1)

			for x := description.begin; x <= description.end; x++ {
				optimizer.descriptions[x] = description
			}
		}

		ix = klasseGroup.end + 1
	}

	return optimizer
}

//Evaluate cost of team loten
func (optimizer *Optimizer) Evaluate(teams []TeamCostID) *TravelCosts {

	result := new(TravelCosts)

	//2 promovendi
	//1 degradant
	//penalty if samen vereniging

	var promovendi, degradanten, kampioenen uint
	promovendi = 0
	degradanten = 0
	kampioenen = 0
	verenigingen := make(map[string]int)

	for lotNR, teamID := range teams {
		var totalDuration, totalDistance uint64

		travelInfos := make([]*TravelInformation, 0, 9)
		uitCount := 0

		teamInfo := optimizer.matrix.GetTeamInfoByCostID(teamID)

		if teamInfo == nil {
			log.Panic("Unknown team", teamID, "!")
		}

		switch teamInfo.team.pd {
		case Promotie:
			promovendi++
			break
		case Degradatie:
			degradanten++
			break
		case Kampioen:
			kampioenen++
			break
		}

		vercount := verenigingen[teamInfo.team.vereniging.id]
		verenigingen[teamInfo.team.vereniging.id] = (vercount + 1)

		for ronde := 0; ronde < 9; ronde++ {
			//ronde 9 is on central location, for Meester klasse
			if ronde == 8 {
				teamInfo := optimizer.matrix.GetTeamInfoByCostID(teamID)

				if teamInfo != nil &&
					teamInfo.team.klasse == Meester {
					//special
				}
			}

			if optimizer.schema.Loten[lotNR].Rondes[ronde].Verplaatsing == Uit {
				travelInfo := optimizer.matrix.GetTeamsTravelCost(teamID, teams[optimizer.schema.Loten[lotNR].Rondes[ronde].Tegenstander])

				if travelInfo == nil {
					log.Panic("Unknown travelcosts for ", teamID, " <-> ", teams[optimizer.schema.Loten[lotNR].Rondes[ronde].Tegenstander])
				}

				travelInfos = append(travelInfos, travelInfo)
				totalDistance += travelInfo.Distance
				totalDuration += travelInfo.Duration
				uitCount++
			}
		}

		meanAllDistance := float64(totalDistance) / 8.0
		meanAllDuration := float64(totalDuration) / 8.0
		meanUitDistance := float64(totalDistance) / float64(len(travelInfos))
		meanUitDuration := float64(totalDuration) / float64(len(travelInfos))

		sdUitDistance := 0.0
		sdUitDuration := 0.0

		for _, ti := range travelInfos {
			sdUitDistance += math.Pow(float64(ti.Distance)-meanUitDistance, 2)
			sdUitDuration += math.Pow(float64(ti.Duration)-meanUitDuration, 2)
		}

		sdUitDistance = math.Sqrt(sdUitDistance / float64(len(travelInfos)-1))
		sdUitDuration = math.Sqrt(sdUitDuration / float64(len(travelInfos)-1))

		//log.Printf("lotNr=%d, teamID=%v, meanAllDistance=%v, meanAllDuration=%v, meanUitDistance=%v, meanUitDuration=%v, sdUitDistance=%v, sdUitDuration=%v, totalDistance=%v, totalDuration=%v, len(travelInfos)=%d",
		//	lotNR, teamID, meanAllDistance, meanAllDuration, meanUitDistance, meanUitDuration, sdUitDistance, sdUitDuration, totalDistance, totalDuration, len(travelInfos))

		result.TotalDistance += totalDistance
		result.TotalDuration += totalDuration
		result.TotalCost += uint64((meanAllDistance * sdUitDistance) + (meanAllDuration * sdUitDuration))
		//result.TotalCost += uint64(meanAllDistance * sdUitDistance)

	}

	//evaluate whish list
	//foreach whish not granted add cost penalty
	//for _,wish := wishlist
	//

	//penalties
	if (promovendi+kampioenen) != 2 || degradanten != 1 {
		//log.Println("Penalty: ", result.TotalCost, " => ", uint64(float64(result.TotalCost)*1.2))
		result.TotalCost = uint64(float64(result.TotalCost) * 1.9)
	}

	if len(verenigingen) != 10 {
		result.TotalCost = uint64(float64(result.TotalCost) * 2.5)
	}

	return result
}
