package main

import (
	"fmt"

	"strconv"

	"github.com/tealeg/xlsx"
)

//LoadSchaakbondExcel Laad teams en verenigingen uit het excel-bestand
func LoadSchaakbondExcel(fileName string) (*Schaakbond, error) {
	xlFile, err := xlsx.OpenFile(fileName)

	if err != nil {
		return nil, err
	}

	sb := new(Schaakbond)
	sb.verenigingen = make(map[string]Vereniging)
	sb.teams = make(map[string]Team)
	sb.klasses = make(map[Klasse][]Team)

	for _, sheet := range xlFile.Sheets {
		if sheet.Name == "Indeling" {
			for _, row := range sheet.Rows {
				//0 - Team Id
				//1 - Team Klasse { M, 1, 2, 3 }
				//4 - Team Naam
				//5 - Vereniging Id
				//6 - Vereniging plaats
				//7 - Team D/P/_

				if row.Cells[0].Value != "" &&
					row.Cells[0].Value != "Teamid" {

					v, ok := sb.verenigingen[row.Cells[5].Value]

					if !ok {
						var nw Vereniging
						nw.id = row.Cells[5].Value
						nw.plaats = row.Cells[6].Value
						nw.teams = make(map[string]Team)
						sb.verenigingen[nw.id] = nw
						v = nw
					}

					var t Team
					t.id = row.Cells[0].Value
					t.naam = row.Cells[4].Value

					switch row.Cells[1].Value {
					case "M":
						t.klasse = Meester
					case "1":
						t.klasse = Eerste
					case "2":
						t.klasse = Tweede
					case "3":
						t.klasse = Derde
					default:
						return nil, fmt.Errorf("Unknown Klasse value of team %v (%v)", row.Cells[0].Value, row.Cells[1].Value)
					}

					switch row.Cells[7].Value {
					case "K":
						t.pd = Kampioen
					case "P":
						t.pd = Promotie
					case "D":
						t.pd = Degradatie
					case "":
						t.pd = Ongewijzigd
					default:
						return nil, fmt.Errorf("Unknown P/D value of team %v (%v)", row.Cells[0].Value, row.Cells[7].Value)
					}

					t.vereniging = v
					v.teams[t.id] = t
					sb.teams[t.id] = t

					klasseTeams := sb.klasses[t.klasse]

					if klasseTeams == nil {
						klasseTeams = make([]Team, 0, 256)
					}

					sb.klasses[t.klasse] = append(klasseTeams, t)
				}
			}
		}
	}

	return sb, nil
}

//LoadSpeelSchemaExcel Laad speel schema excel-bestand
func LoadSpeelSchemaExcel(fileName string) (*SpeelSchema, error) {
	xlFile, err := xlsx.OpenFile(fileName)

	if err != nil {
		return nil, err
	}

	ss := new(SpeelSchema)

	for _, sheet := range xlFile.Sheets {
		if len(sheet.Rows) >= 7 &&
			len(sheet.Rows[0].Cells) > 0 &&
			sheet.Rows[0].Cells[0].Value == "Ronde" { //should be the wright sheet
			for ix, row := range sheet.Rows {
				if ix > 1 && ix < 7 {
					for ronde := 0; ronde < 9; ronde++ {

						thuis, err := strconv.ParseUint(row.Cells[1+(3*ronde)].Value, 10, 16)

						if err != nil {
							return nil, err
						}

						uit, err := strconv.ParseUint(row.Cells[2+(3*ronde)].Value, 10, 16)

						if err != nil {
							return nil, err
						}

						lotThuis := LotNummer(thuis - 1)
						lotUit := LotNummer(uit - 1)

						ss.Rondes[ronde].Wedstrijden[ix-2].Thuis = lotThuis
						ss.Rondes[ronde].Wedstrijden[ix-2].Uit = lotUit
						ss.Loten[lotThuis].Rondes[ronde].Tegenstander = lotUit
						ss.Loten[lotThuis].Rondes[ronde].Verplaatsing = Thuis
						ss.Loten[lotUit].Rondes[ronde].Tegenstander = lotThuis
						ss.Loten[lotUit].Rondes[ronde].Verplaatsing = Uit

					}
				}
			}

			return ss, nil
		}
	}

	return nil, fmt.Errorf("No suitable data found")
}
