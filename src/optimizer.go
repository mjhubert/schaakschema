package main

import (
	"log"
	"math"
)

//TeamCostID identiefier
type TeamCostID uint16

//TeamInfo for team cost
type TeamInfo struct {
	teamCostID TeamCostID
	team       Team
}

//TeamCostMatrix cost info
type TeamCostMatrix struct {
	teamCostIDByTeamID map[string]*TeamInfo
	teamCostMatrix     map[uint32]*TravelInformation
}

func combineTeamCostIDs(fromID TeamCostID, toID TeamCostID) uint32 {
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

//GetTeamsTravelCost between cities
func (matrix *TeamCostMatrix) GetTeamsTravelCost(fromID TeamCostID, toID TeamCostID) *TravelInformation {
	return matrix.teamCostMatrix[combineTeamCostIDs(fromID, toID)]
}

//GetOrAddTeamCostInfoByTeam of matrix
func (matrix *TeamCostMatrix) GetOrAddTeamCostInfoByTeam(team Team) *TeamInfo {

	gc := matrix.teamCostIDByTeamID[team.id]

	if gc != nil {
		return gc
	}

	teamInfo := new(TeamInfo)
	teamInfo.team = team
	teamInfo.teamCostID = TeamCostID(uint16(len(matrix.teamCostIDByTeamID)))

	matrix.teamCostIDByTeamID[team.id] = teamInfo

	return teamInfo
}

//CreateTeamTravelCostInformationMatrix between teams
func CreateTeamTravelCostInformationMatrix(sb *Schaakbond, distanceMatrix *DistanceMatrix) *TeamCostMatrix {
	matrix := new(TeamCostMatrix)
	matrix.teamCostIDByTeamID = make(map[string]*TeamInfo)
	matrix.teamCostMatrix = make(map[uint32]*TravelInformation)
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

//Optimizer info
type Optimizer struct {
	matrix *TeamCostMatrix
	schema *SpeelSchema
}

//NewOptimizer create a optimizer
func NewOptimizer(matrix *TeamCostMatrix, schema *SpeelSchema) *Optimizer {
	optimizer := new(Optimizer)
	optimizer.matrix = matrix
	optimizer.schema = schema
	return optimizer
}

//Evaluate cost of team loten
func (optimizer *Optimizer) Evaluate(teamLoten []TeamCostID) *TravelCosts {

	result := new(TravelCosts)

	for lotNR, teamID := range teamLoten {
		var totalDuration, totalDistance uint64

		travelInfos := make([]*TravelInformation, 0, 9)

		for ronde := 0; ronde < 9; ronde++ {
			if optimizer.schema.Loten[lotNR].Rondes[ronde].Verplaatsing == Uit {
				travelInfo := optimizer.matrix.GetTeamsTravelCost(teamID, teamLoten[optimizer.schema.Loten[lotNR].Rondes[ronde].Tegenstander])
				travelInfos = append(travelInfos, travelInfo)
				totalDistance += travelInfo.Distance
				totalDuration += travelInfo.Duration
			}
		}

		meanDistance := float64(totalDistance) / 9.0
		meanDuration := float64(totalDuration) / 9.0
		sdDistance := 0.0
		sdDuration := 0.0

		for _, ti := range travelInfos {
			sdDistance += math.Pow(float64(ti.Distance)-meanDistance, 2)
			sdDuration += math.Pow(float64(ti.Duration)-meanDuration, 2)
		}

		sdDistance = math.Sqrt(sdDistance / float64(len(travelInfos)-1))
		sdDuration = math.Sqrt(sdDuration / float64(len(travelInfos)-1))

		log.Printf("lotNr=%d, teamID=%v, meanDistance=%v, meanDuration=%v, sdDistance=%v, sdDuration=%v, totalDistance=%v, totalDuration=%v",
			lotNR, teamID, meanDistance, meanDuration, sdDistance, sdDuration, totalDistance, totalDuration)

		result.TotalDistance += totalDistance
		result.TotalDuration += totalDuration
		result.TotalCost += uint64((meanDistance * sdDistance) + (meanDuration * sdDuration))
	}

	return result
}
