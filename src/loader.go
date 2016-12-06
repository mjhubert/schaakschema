package main

import (
	"fmt"

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

				}
			}
		}
	}

	return sb, nil
}
